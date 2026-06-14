//go:build js

package audio

import (
	"bytes"
	"log"
	"sync"
	"sync/atomic"
	"syscall/js"
	"time"

	"github.com/sinshu/go-meltysynth/meltysynth"
)

// webMidiDriver is the js midiSink: tracks are pre-rendered to AudioBuffer
// chunks (static buffers are immune to main-thread synthesis bursts) and
// scheduled onto the Web Audio clock.
//
// The shared audioLoop (audioloop.go) owns all fade/latch sequencing and
// drives the four midiSink primitives from its 50ms ticker; volume —
// including the stepped ±8 fade — lands on the single musicGain node as the
// linear vol/128 gain.
//
// A track plays once and then stops — music never loops (see midiSink.play
// for why; the TS reference disables looping). The chunks are scheduled
// gaplessly across the Web Audio clock and that is the whole of playback.

// renderChunkFrames is how many frames are synthesized per scheduled chunk
// (~250ms of audio). Each chunk becomes its own AudioBufferSourceNode
// started at a precise time, so playback can begin after the FIRST chunk
// renders instead of waiting for the whole track; the render then races
// ahead of playback so the rest is scheduled before it's needed. A chunk is
// also one frame's worth of synthesis CPU, keeping the per-chunk yield
// (see streamRender) responsive.
const renderChunkFrames = SampleRate / 4

// musicStream is one track's playback: every source node scheduled for it.
// Stopping it stops them all, including ones scheduled in the future.
type musicStream struct {
	mu      sync.Mutex
	sources []js.Value
	stopped bool
}

// schedule starts buf at AudioContext time `at` on this stream. If the
// stream was already stopped (a newer command superseded it between
// chunks), the source is stopped immediately so it neither plays nor leaks.
func (s *musicStream) schedule(buf js.Value, at float64) {
	src := ac.Call("createBufferSource")
	src.Set("buffer", buf)
	src.Set("loop", false)
	src.Call("connect", musicGain)
	src.Call("start", at)
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		safeStop(src)
		src.Call("disconnect")
		return
	}
	s.sources = append(s.sources, src)
	s.mu.Unlock()
}

// stopAll stops + disconnects every scheduled source. Idempotent (both a
// superseding command and a superseded render goroutine may call it).
func (s *musicStream) stopAll() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	srcs := s.sources
	s.sources = nil
	s.mu.Unlock()
	for _, src := range srcs {
		safeStop(src)
		src.Call("disconnect")
	}
}

// safeStop calls stop() on a source, swallowing the InvalidStateError some
// browsers throw when the source has already ended naturally. A thrown JS
// exception would otherwise panic the Go side via syscall/js.
func safeStop(src js.Value) {
	defer func() { _ = recover() }()
	src.Call("stop")
}

type webMidiDriver struct {
	soundFont *meltysynth.SoundFont
	loadOnce  sync.Once

	mu         sync.Mutex
	cur        *musicStream // current track's stream (nil before first play / after stop)
	musicalEnd float64      // AudioContext time when the track's musical length elapses

	// One-track render cache: a jingle→track resume re-schedules instantly
	// instead of re-synthesizing. AudioBuffers are immutable and may back
	// any number of source nodes, so the cache holds the same buffers the
	// first play used — no second copy of the PCM.
	cacheKey     string
	cacheChunks  []js.Value
	cacheSeconds float64 // musical length (sans tail) of the cached track

	// gen is bumped on every play/stop. In-flight render and loop
	// goroutines snapshot it and abandon when superseded.
	gen atomic.Uint64
}

func newWebMidiDriver() *webMidiDriver { return &webMidiDriver{} }

func (d *webMidiDriver) ensureSoundFont() *meltysynth.SoundFont {
	d.loadOnce.Do(func() {
		sf, err := loadSoundFont()
		if err != nil {
			log.Printf("audio/midi: soundfont unavailable, music silent: %v", err)
			return
		}
		d.soundFont = sf
	})
	return d.soundFont
}

// play (midiSink) replaces the current track immediately; it plays once and
// then stops — music never loops (see midiSink.play). Any audible crossfade
// is the audioLoop's stepped setVolume, which has already run a full
// fade-out before this is called on a fading change. Rendering runs on a
// background goroutine so the 50ms tick isn't blocked by synthesis.
func (d *webMidiDriver) play(midData []byte, vol int) {
	d.setVolume(vol)
	gen := d.gen.Add(1)
	d.mu.Lock()
	old := d.cur
	d.cur = nil
	d.mu.Unlock()
	if old != nil {
		old.stopAll()
	}
	go d.startTrack(midData, gen)
}

// setVolume (midiSink): linear vol/128 on the shared music gain node.
// Java: MidiPlayer.setVolume(0, volume) — see linearVolume for how the CC
// rescale collapses to a linear post-gain and the /128 calibration.
func (d *webMidiDriver) setVolume(vol int) {
	if !musicGain.Truthy() {
		return
	}
	musicGain.Get("gain").Set("value", linearVolume(vol))
}

// stop (midiSink): Java MidiPlayer.stop() (MidiPlayer.java:46-49). The gen
// bump makes in-flight render/loop goroutines abandon.
func (d *webMidiDriver) stop() {
	d.gen.Add(1)
	d.mu.Lock()
	s := d.cur
	d.cur = nil
	d.musicalEnd = 0
	d.mu.Unlock()
	if s != nil {
		s.stopAll()
	}
}

// running (midiSink): Java Sequencer.isRunning() — a track ends when its
// musical length elapses on the context clock.
// INVARIANT: a fresh track's cur/musicalEnd are set asynchronously by
// startTrack, so there is a window after play() where this returns false
// for the track just started. Safe because the only caller
// (audioLoop.playMidi) consults running() to decide whether to fade out the
// CURRENTLY-PLAYING track — i.e. before any new play() — at which point the
// old track's cur/musicalEnd are still live and accurate. If the audioLoop
// ever starts consulting running() right after a play(), revisit this.
func (d *webMidiDriver) running() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cur == nil {
		return false
	}
	return ac.Get("currentTime").Float() < d.musicalEnd
}

// startTrack makes a new stream current and feeds it: cached chunks for a
// repeat of the cached track, or a fresh streaming render.
func (d *webMidiDriver) startTrack(midData []byte, gen uint64) {
	now := ac.Get("currentTime").Float()
	s := &musicStream{}

	d.mu.Lock()
	if d.gen.Load() != gen { // superseded before we became current
		d.mu.Unlock()
		return
	}
	d.cur = s
	cacheHit := d.cacheKey == string(midData) && len(d.cacheChunks) > 0
	var cached []js.Value
	if cacheHit {
		cached = d.cacheChunks
		d.musicalEnd = now + d.cacheSeconds
	}
	d.mu.Unlock()

	if cacheHit {
		d.replayChunks(s, cached, now, gen)
		return
	}
	d.streamRender(s, midData, now, gen)
}

// replayChunks schedules already-rendered chunks gapless from startAt,
// advancing by each chunk's own frame count. Yields between chunks
// (single-threaded wasm) so the game loop keeps drawing; abandons if
// superseded.
func (d *webMidiDriver) replayChunks(s *musicStream, chunks []js.Value, startAt float64, gen uint64) {
	at := startAt
	for _, buf := range chunks {
		if d.gen.Load() != gen {
			return // superseded: the new command stops this stream
		}
		s.schedule(buf, at)
		at += float64(buf.Get("length").Int()) / float64(SampleRate)
		time.Sleep(time.Millisecond)
	}
}

// streamRender synthesizes the track chunk by chunk, scheduling each to
// play seamlessly as it is produced (so playback starts after chunk 0), and
// caches the chunk buffers for instant replay. Renders through a small
// reusable scratch pair so the Go heap peak stays flat regardless of track
// length; the rendered PCM lives only in the JS AudioBuffers (Go wasm
// linear memory never shrinks, so a full-track []float32 would permanently
// inflate it). Yields between chunks so the game loop keeps drawing.
// Abandons if superseded.
func (d *webMidiDriver) streamRender(s *musicStream, midData []byte, startAt float64, gen uint64) {
	sf := d.ensureSoundFont()
	if sf == nil {
		return
	}
	midiFile, err := meltysynth.NewMidiFile(bytes.NewReader(midData))
	if err != nil {
		log.Printf("audio/midi: parse: %v", err)
		return
	}
	settings := meltysynth.NewSynthesizerSettings(SampleRate)
	settings.EnableReverbAndChorus = false
	synth, err := meltysynth.NewSynthesizer(sf, settings)
	if err != nil {
		log.Printf("audio/midi: synth init: %v", err)
		return
	}
	seq := meltysynth.NewMidiFileSequencer(synth)
	// One rendering pass; the track plays once (music never loops — see
	// midiSink.play). The second arg is meltysynth's own loop flag, false.
	seq.Play(midiFile, false)

	seconds := midiFile.GetLength().Seconds()
	d.mu.Lock()
	d.musicalEnd = startAt + seconds
	d.mu.Unlock()

	frames := renderFrameCount(midiFile.GetLength())
	// One reusable chunk-sized scratch pair, not two full-track slices.
	// Safe to reuse because f32ToJSFloat32Array copies the bytes into the
	// JS buffer before the next Render overwrites the scratch.
	left := make([]float32, renderChunkFrames)
	right := make([]float32, renderChunkFrames)
	chunks := make([]js.Value, 0, (frames+renderChunkFrames-1)/renderChunkFrames)
	at := startAt
	for off := 0; off < frames; off += renderChunkFrames {
		if d.gen.Load() != gen {
			return
		}
		n := renderChunkFrames
		if off+n > frames {
			n = frames - off
		}
		ls, rs := left[:n], right[:n]
		seq.Render(ls, rs)
		buf := ac.Call("createBuffer", ChannelCount, n, SampleRate)
		buf.Call("copyToChannel", f32ToJSFloat32Array(ls), 0)
		buf.Call("copyToChannel", f32ToJSFloat32Array(rs), 1)
		s.schedule(buf, at)
		chunks = append(chunks, buf)
		at += float64(n) / float64(SampleRate)
		time.Sleep(time.Millisecond)
	}

	if d.gen.Load() != gen {
		return
	}
	d.mu.Lock()
	d.cacheKey = string(midData)
	d.cacheChunks = chunks
	d.cacheSeconds = seconds
	d.mu.Unlock()
}

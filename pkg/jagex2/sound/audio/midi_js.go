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

	"github.com/zsrv/goscape-client/pkg/sign/signlink"
)

// fadeDuration matches the native 2s fade-out window.
const fadeDuration = 2 * time.Second

// fadeTimeConstant is the exponential time constant for setTargetAtTime,
// matching the native per-sample smoother's ~0.5s (gainSmoothingAlpha).
const fadeTimeConstant = 0.5

// renderChunkFrames is how many frames are synthesized per scheduled chunk
// (~250ms of audio). Each chunk becomes its own AudioBufferSourceNode started
// at a precise time, so playback can begin after the FIRST chunk renders
// instead of waiting for the whole track; the render then races ahead of
// playback (~realtime×many) so the rest is scheduled before it's needed. A
// chunk is also one frame's worth of synthesis CPU, keeping the per-chunk
// yield (see streamRender) responsive.
const renderChunkFrames = SampleRate / 4

// PlayMIDI plays an in-memory MIDI track, replacing the current one.
// fade=true reproduces the native fade-out-then-start. Returns immediately;
// synthesis + scheduling run on a background goroutine.
func PlayMIDI(midData []byte, fade bool) {
	d := getMidiDriver()
	if d == nil {
		return
	}
	d.playFromBytes(midData, fade)
}

// musicStream is one track's playback: a fade GainNode plus the chunk
// AudioBufferSourceNodes scheduled onto it. Stopping it stops every scheduled
// source (including ones scheduled in the future) and disconnects the gain.
type musicStream struct {
	fadeGain js.Value

	mu      sync.Mutex
	sources []js.Value
	stopped bool
}

// schedule starts buf at AudioContext time `at` on this stream. If the stream
// was already stopped (a newer track superseded it between chunks), the source
// is stopped immediately so it neither plays nor leaks.
func (s *musicStream) schedule(buf js.Value, at float64) {
	src := ac.Call("createBufferSource")
	src.Set("buffer", buf)
	src.Set("loop", false)
	src.Call("connect", s.fadeGain)
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

// stopAll stops + disconnects every scheduled source and the fade gain.
// Idempotent (both the superseding command and a superseded render goroutine
// may call it).
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
	s.fadeGain.Call("disconnect")
}

// safeStop calls stop() on a source, swallowing the InvalidStateError some
// browsers throw when the source has already ended naturally (e.g. a cache-hit
// full-track source that finished before a loop re-issue's stopAll). A thrown
// JS exception would otherwise panic the Go side via syscall/js.
func safeStop(src js.Value) {
	defer func() { _ = recover() }()
	src.Call("stop")
}

// webMidiDriver owns the current playing stream, a one-track full-buffer cache
// (so the game's loop re-issue replays instantly without re-synthesizing), and
// a generation counter so a rapid track change abandons an in-flight render.
type webMidiDriver struct {
	soundFont *meltysynth.SoundFont
	loadOnce  sync.Once

	mu       sync.Mutex
	cur      *musicStream // current track's stream (nil before first play)
	cacheKey string       // identity of the cached fully-rendered track
	cacheBuf js.Value     // full AudioBuffer for cacheKey

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

// handle dispatches a signlink Midi command (same protocol as native).
func (d *webMidiDriver) handle(cmd string) {
	switch cmd {
	case "stop":
		d.stop(signlink.ReadMidiFade() == 1)
	case "voladjust":
		d.applyMidiVolume()
	default:
		log.Printf("audio/midi: ignoring unexpected command %q", cmd)
	}
}

// applyMidiVolume sets the shared music gain from MidiVol (called on
// voladjust and every track change).
func (d *webMidiDriver) applyMidiVolume() {
	v := float64(volumeFromCentibels(signlink.ReadMidiVol()))
	musicGain.Get("gain").Set("value", v)
}

// playFromBytes is the play entry point. gen is bumped now so a newer
// play/stop arriving during the (background) render makes this goroutine
// abandon. The whole thing runs off the game-loop goroutine so the loop keeps
// drawing during synthesis.
func (d *webMidiDriver) playFromBytes(midData []byte, fade bool) {
	d.applyMidiVolume()
	gen := d.gen.Add(1)
	go d.playTrack(midData, fade, gen)
}

// playTrack fades out the previous stream (fade-out-then-start, matching
// native: the new track starts only after the old has faded), creates the new
// stream, then plays the cached full buffer (loop re-issue) or stream-renders
// a new track onto it.
func (d *webMidiDriver) playTrack(midData []byte, fade bool, gen uint64) {
	now := ac.Get("currentTime").Float()
	startAt := now

	d.mu.Lock()
	old := d.cur
	d.mu.Unlock()
	if old != nil {
		if fade {
			old.fadeGain.Get("gain").Call("setTargetAtTime", 0.0, now, fadeTimeConstant)
			startAt = now + fadeDuration.Seconds()
			go func() {
				time.Sleep(fadeDuration)
				old.stopAll()
			}()
		} else {
			old.stopAll()
		}
	}

	s := &musicStream{fadeGain: ac.Call("createGain")}
	s.fadeGain.Get("gain").Set("value", 1.0)
	s.fadeGain.Call("connect", musicGain)

	d.mu.Lock()
	if d.gen.Load() != gen { // superseded before we became current
		d.mu.Unlock()
		s.fadeGain.Call("disconnect")
		return
	}
	d.cur = s
	cacheHit := d.cacheKey == string(midData) && d.cacheBuf.Truthy()
	cbuf := d.cacheBuf
	d.mu.Unlock()

	if cacheHit {
		s.schedule(cbuf, startAt) // instant replay of the cached full buffer
		return
	}
	d.streamRender(s, midData, startAt, gen)
}

// streamRender synthesizes the track chunk by chunk, scheduling each chunk to
// play seamlessly as it is produced (so playback starts after chunk 0), and
// caches the assembled full buffer for instant loop replay. Yields between
// chunks so the game loop keeps drawing. Abandons if superseded.
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
	seq.Play(midiFile, false) // no synth-side loop; the game re-issues SetMidi

	frames := renderFrameCount(midiFile.GetLength())
	left := make([]float32, frames)
	right := make([]float32, frames)
	at := startAt
	for off := 0; off < frames; off += renderChunkFrames {
		if d.gen.Load() != gen {
			return // superseded: the new command fades+stops this stream
		}
		end := off + renderChunkFrames
		if end > frames {
			end = frames
		}
		seq.Render(left[off:end], right[off:end])
		n := end - off
		buf := ac.Call("createBuffer", ChannelCount, n, SampleRate)
		buf.Call("copyToChannel", f32ToJSFloat32Array(left[off:end]), 0)
		buf.Call("copyToChannel", f32ToJSFloat32Array(right[off:end]), 1)
		s.schedule(buf, at)
		at += float64(n) / float64(SampleRate)
		time.Sleep(time.Millisecond)
	}

	if d.gen.Load() != gen {
		return
	}
	full := ac.Call("createBuffer", ChannelCount, frames, SampleRate)
	full.Call("copyToChannel", f32ToJSFloat32Array(left), 0)
	full.Call("copyToChannel", f32ToJSFloat32Array(right), 1)
	d.mu.Lock()
	d.cacheKey = string(midData)
	d.cacheBuf = full
	d.mu.Unlock()
}

// stop silences music. With fade, ramp the current stream's gain to 0 then
// stop it after fadeDuration; without, stop immediately. gen-guarded so a
// play arriving during the fade isn't clobbered.
func (d *webMidiDriver) stop(fade bool) {
	d.gen.Add(1) // supersede any in-flight render so its goroutine abandons
	d.mu.Lock()
	s := d.cur
	d.cur = nil
	d.mu.Unlock()
	if s == nil {
		return
	}
	if !fade {
		s.stopAll()
		return
	}
	now := ac.Get("currentTime").Float()
	s.fadeGain.Get("gain").Call("setTargetAtTime", 0.0, now, fadeTimeConstant)
	go func() {
		time.Sleep(fadeDuration)
		// No gen check: stop already set d.cur=nil, so a PlayMIDI arriving
		// during the fade builds a separate stream — stopping this faded-out
		// one is always correct, and stopAll is idempotent.
		s.stopAll()
	}()
}

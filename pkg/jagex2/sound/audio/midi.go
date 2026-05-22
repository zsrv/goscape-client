package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/sinshu/go-meltysynth/meltysynth"

	"goscape-client/pkg/sign/signlink"
)

// runMidiWatcher polls signlink.ConsumeMidi every midiPollInterval. The
// poll cadence (50ms) matches the Java signlink.run loop and the
// existing client.RunMidi poll — fast enough that title-screen and
// gameplay music feel responsive, slow enough to be invisible to CPU.
//
// On nil ctx (oto init failed), still drains commands so the channel
// doesn't back up — but doesn't actually play. This keeps Java's
// fire-and-forget protocol semantics intact even with audio disabled.
const midiPollInterval = 50 * time.Millisecond

func runMidiWatcher(ctx *oto.Context) {
	d := newMidiDriver(ctx)
	for {
		cmd := signlink.ConsumeMidi()
		if cmd != "" {
			d.handle(cmd)
		}
		time.Sleep(midiPollInterval)
	}
}

// midiDriver owns the live meltysynth sequencer and the oto Player that
// pulls bytes from it. There is at most one active player at a time
// (matching the original Java single-Sequencer model); a "stop" tears it
// down, a new track path swaps the sequencer's midi file under lock.
type midiDriver struct {
	ctx *oto.Context

	// soundFont is loaded once. nil if loading failed (no SF2 available);
	// in that case handle() still consumes commands but plays no audio.
	soundFont *meltysynth.SoundFont
	loadOnce  sync.Once

	// mu guards the live source/player handoff. handle() can swap them;
	// the source's Read() reads them under lock too.
	mu     sync.Mutex
	src    *midiSource
	player *oto.Player
}

func newMidiDriver(ctx *oto.Context) *midiDriver {
	return &midiDriver{ctx: ctx}
}

// handle dispatches one signlink Midi command. The three shapes are:
//   - "stop":      tear down the current player, honoring MidiFade.
//   - "voladjust": adjust gain on the live stream without restarting.
//   - <path>:      load the .mid at this path and start playing it.
func (d *midiDriver) handle(cmd string) {
	switch cmd {
	case "stop":
		d.stop(signlink.ReadMidiFade() == 1)
	case "voladjust":
		d.setGain(volumeFromCentibels(signlink.ReadMidiVol()))
	default:
		d.play(cmd, signlink.ReadMidiFade() == 1, volumeFromCentibels(signlink.ReadMidiVol()))
	}
}

// play loads the .mid at path and starts it. If the SoundFont isn't
// loaded yet, it's loaded on first call; if loading fails, play logs
// and returns without crashing the game.
func (d *midiDriver) play(path string, fade bool, gain float32) {
	if d.ctx == nil {
		return
	}
	sf := d.ensureSoundFont()
	if sf == nil {
		return
	}

	midData, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("audio/midi: read %q: %v\n", path, err)
		return
	}
	midiFile, err := meltysynth.NewMidiFile(bytes.NewReader(midData))
	if err != nil {
		fmt.Printf("audio/midi: parse %q: %v\n", path, err)
		return
	}

	settings := meltysynth.NewSynthesizerSettings(SampleRate)
	settings.EnableReverbAndChorus = false
	synth, err := meltysynth.NewSynthesizer(sf, settings)
	if err != nil {
		fmt.Printf("audio/midi: synth init: %v\n", err)
		return
	}
	seq := meltysynth.NewMidiFileSequencer(synth)
	seq.Play(midiFile, true)

	src := newMidiSource(seq, gain)

	d.mu.Lock()
	old := d.player
	oldSrc := d.src
	d.src = src
	d.player = d.ctx.NewPlayer(src)
	d.player.Play()
	d.mu.Unlock()

	if old != nil {
		go retireOldPlayer(old, oldSrc, fade)
	}
}

// stop tears down the active player. If fade is true, it ramps the
// player's gain to silence over the TS-matching 2-second fade window
// before closing.
func (d *midiDriver) stop(fade bool) {
	d.mu.Lock()
	old := d.player
	oldSrc := d.src
	d.player = nil
	d.src = nil
	d.mu.Unlock()

	if old != nil {
		go retireOldPlayer(old, oldSrc, fade)
	}
}

// setGain adjusts volume on the currently-playing track without
// restarting. No-op if no track is playing.
func (d *midiDriver) setGain(g float32) {
	d.mu.Lock()
	src := d.src
	d.mu.Unlock()
	if src != nil {
		src.setGain(g)
	}
}

// retireOldPlayer fades (or hard-stops) and closes a player. Called on a
// goroutine so handle() returns promptly. fadeDuration matches the TS
// _tinyMidiStop timeout (tinymidipcm.js:140).
const fadeDuration = 2 * time.Second

func retireOldPlayer(p *oto.Player, src *midiSource, fade bool) {
	if fade && src != nil {
		const steps = 40
		stepDur := fadeDuration / steps
		start := src.gain()
		for i := 1; i <= steps; i++ {
			g := start * (1 - float32(i)/float32(steps))
			src.setGain(g)
			time.Sleep(stepDur)
		}
	}
	p.Pause()
}

// ensureSoundFont lazy-loads the SF2. Returns nil if loading failed.
func (d *midiDriver) ensureSoundFont() *meltysynth.SoundFont {
	d.loadOnce.Do(func() {
		sf, err := loadSoundFont()
		if err != nil {
			fmt.Printf("audio/midi: soundfont unavailable, music will be silent: %v\n", err)
			return
		}
		d.soundFont = sf
	})
	return d.soundFont
}

// midiSource is the io.Reader that oto pulls PCM bytes from. It owns a
// pair of float32 scratch buffers, calls the sequencer's Render to fill
// them, applies gain, then interleaves and converts to int16 LE for oto.
//
// Concurrent access: oto pulls on a dedicated audio goroutine. Gain is
// adjusted from handle() goroutines. seq itself is not thread-safe;
// nobody else touches it once we hand it to the source.
type midiSource struct {
	seq *meltysynth.MidiFileSequencer

	// gainBits is the linear gain (0..1) as float32 bits, manipulated
	// via atomic so setGain doesn't need to coordinate with Read.
	gainBits atomic.Uint32

	// scratch buffers reused across Read calls.
	left  []float32
	right []float32
}

func newMidiSource(seq *meltysynth.MidiFileSequencer, initialGain float32) *midiSource {
	s := &midiSource{seq: seq}
	s.setGain(initialGain)
	return s
}

func (s *midiSource) setGain(g float32) {
	s.gainBits.Store(math.Float32bits(g))
}

func (s *midiSource) gain() float32 {
	return math.Float32frombits(s.gainBits.Load())
}

// Read fills p with interleaved stereo int16 LE PCM. It never returns
// io.EOF — the sequencer loops, and even if it didn't we'd render
// silence. oto treats a never-ending reader as a perpetual source,
// which is what we want for a music track that may be hot-swapped.
//
// p's length is determined by oto. It is expected to be a multiple of
// 4 (one stereo int16 frame), though we tolerate odd remainders by
// rounding down.
func (s *midiSource) Read(p []byte) (int, error) {
	frames := len(p) / 4
	if frames == 0 {
		return 0, nil
	}
	if cap(s.left) < frames {
		s.left = make([]float32, frames)
		s.right = make([]float32, frames)
	} else {
		s.left = s.left[:frames]
		s.right = s.right[:frames]
	}
	s.seq.Render(s.left, s.right)
	g := s.gain()
	for i := range frames {
		l := clipInt16(s.left[i] * g)
		r := clipInt16(s.right[i] * g)
		off := i * 4
		binary.LittleEndian.PutUint16(p[off:], uint16(l))
		binary.LittleEndian.PutUint16(p[off+2:], uint16(r))
	}
	return frames * 4, nil
}

// clipInt16 quantizes a float32 sample (nominally -1..1) to int16 with
// hard clipping at the rails. meltysynth's output stays roughly in
// range but transient peaks can exceed 1.0; without the clip you'd hear
// a wraparound buzz.
func clipInt16(f float32) int16 {
	v := f * 32767
	if v > 32767 {
		return 32767
	}
	if v < -32768 {
		return -32768
	}
	return int16(v)
}

// volumeFromCentibels maps signlink.MidiVol's centibel scale (e.g. -400
// for -4 dB, 0 for full) to a linear amplitude factor. dB = cb/100;
// linear = 10^(dB/20). Matches the TS client's `Math.pow(10, dB / 20)`
// in tinymidipcm.js:300.
func volumeFromCentibels(cb int) float32 {
	if cb >= 0 {
		return 1.0
	}
	return float32(math.Pow(10, float64(cb)/100.0/20.0))
}


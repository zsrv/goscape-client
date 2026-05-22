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

// midiDriver runs a single persistent oto Player attached to a
// midiSource whose internal sequencer is hot-swappable. Track changes,
// stops, and volume adjustments mutate the source in place rather than
// tearing down the player. This mirrors the TS reference client's
// architecture (tinymidipcm.js:226-295): one gainNode + one
// BufferSource line, the source's content gets swapped via setTimeout
// after a fade-out, never overlapping the previous track.
//
// Before this design, faded track changes spawned a second Player in
// parallel with the fading-out first, producing audible overlap for
// the 2-second fade window. Live-reported bug fixed by collapsing to
// a single persistent player.
type midiDriver struct {
	ctx *oto.Context

	// soundFont is loaded once. nil if loading failed (no SF2 available);
	// in that case handle() still consumes commands but plays no audio.
	soundFont *meltysynth.SoundFont
	loadOnce  sync.Once

	// mu guards the player handoff during first-play initialization.
	// After the player exists, all further work happens through src,
	// which has its own internal locking.
	mu     sync.Mutex
	src    *midiSource
	player *oto.Player

	// gen is incremented on every command. In-flight fade goroutines
	// snapshot it at start and abandon if it diverges, so a rapid
	// region change can't leave a stale fade clobbering the new track's
	// gain. Atomic so abandon-checks are lock-free.
	gen atomic.Uint64
}

func newMidiDriver(ctx *oto.Context) *midiDriver {
	return &midiDriver{ctx: ctx}
}

// handle dispatches one signlink Midi command. The three shapes are:
//   - "stop":      stop sequencing, honoring MidiFade.
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

// play loads the .mid at path and swaps it into the live source. The
// first ever play() creates the persistent oto Player; subsequent
// plays reuse it. If fade is true and a track is already playing,
// the gain ramps to 0, the sequencer is swapped, then the gain ramps
// back to the target — matching the TS reference's "fade-out, then
// start" sequencing with no audible overlap.
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

	gen := d.gen.Add(1)

	d.mu.Lock()
	if d.src == nil {
		d.src = newMidiSource(seq, gain)
		d.player = d.ctx.NewPlayer(d.src)
		d.player.Play()
		d.mu.Unlock()
		return
	}
	src := d.src
	d.mu.Unlock()

	if !fade {
		src.swap(seq, gain)
		return
	}

	go d.fadeAndSwap(src, seq, gain, gen)
}

// fadeAndSwap ramps src's gain to 0 over fadeDuration, swaps in the
// new sequencer, then ramps the gain back to target. Abandons silently
// if a newer command supersedes it (gen mismatch).
func (d *midiDriver) fadeAndSwap(src *midiSource, newSeq *meltysynth.MidiFileSequencer, target float32, gen uint64) {
	startGain := src.gain()
	if !d.rampGain(src, startGain, 0, gen) {
		return
	}
	if d.gen.Load() != gen {
		return
	}
	src.swap(newSeq, 0)
	if !d.rampGain(src, 0, target, gen) {
		return
	}
}

// stop silences the live source. If fade is true, the gain ramps to 0
// over fadeDuration first; afterwards the sequencer is cleared so no
// further notes are produced and the player synthesizes silence. The
// player itself stays alive — the next play() reuses it.
func (d *midiDriver) stop(fade bool) {
	d.mu.Lock()
	src := d.src
	d.mu.Unlock()
	if src == nil {
		return
	}

	gen := d.gen.Add(1)

	if !fade {
		src.swap(nil, 0)
		return
	}
	go func() {
		if !d.rampGain(src, src.gain(), 0, gen) {
			return
		}
		if d.gen.Load() != gen {
			return
		}
		src.swap(nil, 0)
	}()
}

// setGain is the "voladjust" handler. It does not restart the track.
// In-flight fade ramps will see the gen they captured no longer
// matches and abandon; voladjust's caller is expected to also drive
// SetMidi("voladjust") frequently enough to keep the live gain correct
// after the abandoned ramp leaves it stranded.
func (d *midiDriver) setGain(g float32) {
	d.mu.Lock()
	src := d.src
	d.mu.Unlock()
	if src == nil {
		return
	}
	d.gen.Add(1)
	src.setGain(g)
}

// rampGain interpolates src's gain from start to end over fadeDuration
// in `fadeSteps` discrete time slices, polling d.gen each step and
// returning false the moment it diverges from the captured gen — that
// signal means a newer command took over and this ramp must abandon.
// Returns true after a successful full ramp.
const (
	fadeDuration = 2 * time.Second
	fadeSteps    = 40
)

func (d *midiDriver) rampGain(src *midiSource, start, end float32, gen uint64) bool {
	step := fadeDuration / fadeSteps
	for i := 1; i <= fadeSteps; i++ {
		if d.gen.Load() != gen {
			return false
		}
		t := float32(i) / float32(fadeSteps)
		src.setGain(start + (end-start)*t)
		time.Sleep(step)
	}
	return true
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
// swappable sequencer pointer (under seqMu) and a separate atomic gain.
// When seq is nil, Read fills its output with silence so the player
// stays alive across "stop" commands.
//
// Concurrent access: oto's audio goroutine calls Read; driver goroutines
// (handle, fadeAndSwap, stop) call swap and setGain. The seqMu critical
// section in Read is one pointer copy long; meltysynth's Render runs
// outside it so a long render can't block a swap.
type midiSource struct {
	seqMu sync.Mutex
	seq   *meltysynth.MidiFileSequencer

	// gainBits is the linear gain (0..1) as float32 bits, manipulated
	// via atomic so setGain doesn't need to coordinate with Read.
	gainBits atomic.Uint32

	// scratch buffers reused across Read calls. Read is single-reader
	// (oto's audio goroutine) so these need no lock.
	left  []float32
	right []float32
}

func newMidiSource(seq *meltysynth.MidiFileSequencer, initialGain float32) *midiSource {
	s := &midiSource{seq: seq}
	s.setGain(initialGain)
	return s
}

// swap atomically replaces the sequencer and sets the gain. Passing nil
// for newSeq leaves the source emitting silence until the next swap.
// The old sequencer is dropped — meltysynth has no Close, GC handles
// it.
func (s *midiSource) swap(newSeq *meltysynth.MidiFileSequencer, newGain float32) {
	s.seqMu.Lock()
	s.seq = newSeq
	s.seqMu.Unlock()
	s.setGain(newGain)
}

func (s *midiSource) setGain(g float32) {
	s.gainBits.Store(math.Float32bits(g))
}

func (s *midiSource) gain() float32 {
	return math.Float32frombits(s.gainBits.Load())
}

// Read fills p with interleaved stereo int16 LE PCM. It never returns
// io.EOF — when the sequencer is nil ("stop"), it emits silence so the
// player stays alive and ready for the next play().
//
// p's length is determined by oto. It is expected to be a multiple of
// 4 (one stereo int16 frame); odd remainders are truncated.
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

	s.seqMu.Lock()
	seq := s.seq
	s.seqMu.Unlock()

	if seq == nil {
		clear(s.left)
		clear(s.right)
	} else {
		seq.Render(s.left, s.right)
	}

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

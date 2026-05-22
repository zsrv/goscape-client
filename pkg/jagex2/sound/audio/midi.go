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

// gainSmoothingAlpha is the per-sample low-pass coefficient used to
// smooth gain changes inside midiSource.Read. It implements the same
// exponential-approach automation as Web Audio's setTargetAtTime, which
// the TS reference client uses for its fade-out (tinymidipcm.js:229
// with a 0.5s time constant). For each sample:
//
//	current = current*α + target*(1-α)
//
// where α = exp(-1 / (τ × sampleRate)). With τ = 0.5s and sampleRate =
// 22050 Hz that's exp(-1/11025) ≈ 0.9999093. At t = 4τ = 2s the gain
// has decayed to e⁻⁴ ≈ 1.83 % of its starting value — what TS
// considers "faded out" before it swaps tracks (fadeseconds = 2 at
// tinymidipcm.js:140).
//
// Moving gain interpolation from the driver (stepped every 50 ms in a
// goroutine) into the per-sample Read loop eliminates the audible
// zipper noise of discrete steps and matches TS's smoothness.
const gainSmoothingAlpha float32 = 0.9999093

// runMidiWatcher polls signlink.ConsumeMidi every midiPollInterval.
// The cadence matches the game's tick rate (20ms) so a Logout-issued
// "stop" never has to wait longer than a single tick before the
// audio side reacts — important because the TS reference's
// browser-side equivalent is essentially synchronous, and any extra
// latency here shows up as audible "music still playing after the
// title screen appeared". Doubled-up with player.Pause inside
// stop() to flush oto's internal buffer.
//
// Track data (paths/bytes) does NOT travel through signlink anymore —
// c.SaveMidi now calls audio.PlayMIDI directly, shaving ~70ms of
// polling + disk-write latency off the title-to-game transition.
// The watcher only handles "stop" and "voladjust" sentinels here.
//
// On nil ctx (oto init failed), still drains commands so the channel
// doesn't back up — but doesn't actually play. This keeps Java's
// fire-and-forget protocol semantics intact even with audio disabled.
const midiPollInterval = 20 * time.Millisecond

func runMidiWatcher(ctx *oto.Context, d *midiDriver) {
	for {
		cmd := signlink.ConsumeMidi()
		if cmd != "" {
			d.handle(cmd)
		}
		time.Sleep(midiPollInterval)
	}
}

// midiDriverRegistry holds the singleton midiDriver and a close-once
// channel that PlayMIDI callers wait on. This decouples the audio
// subsystem's startup (oto init may take ~100ms+ on first run) from
// the client's startup, which can call SetMidi/SaveMidi as soon as
// c.Load runs. Without the wait, the very first scape_main play could
// race ahead of audio.Start and be silently dropped.
var (
	driverMu    sync.Mutex
	driver      *midiDriver
	driverReady = make(chan struct{})
)

// registerMidiDriver publishes the driver and unblocks waiters. Called
// exactly once from audio.Start — either with a real driver (success)
// or with nil (oto init failed). Nil drivers still close the channel
// so PlayMIDI returns silently instead of hanging the caller.
func registerMidiDriver(d *midiDriver) {
	driverMu.Lock()
	driver = d
	driverMu.Unlock()
	close(driverReady)
}

// getMidiDriver waits until the driver is available, then returns it.
// May return nil if audio.Start failed; callers must handle that.
func getMidiDriver() *midiDriver {
	<-driverReady
	driverMu.Lock()
	defer driverMu.Unlock()
	return driver
}

// PlayMIDI feeds an in-memory Standard MIDI File to the synthesizer.
// Called from c.SaveMidi instead of the historical signlink.MidiSave
// detour, which required a temporary disk file (a Java-applet
// artifact). fade matches signlink.MidiFade semantics: true for a
// crossfade, false for an instant cut-in.
//
// User volume is read from signlink.MidiVol inside playFromBytes and
// applied to the persistent oto.Player via SetVolume — separate from
// the source's per-sample fade gain. Splitting the two means the
// volume slider takes effect AFTER oto's internal buffer (instant),
// while track-change crossfades still ramp smoothly via the source's
// per-sample smoother.
//
// Blocks briefly on first call only, until audio.Start finishes
// initializing oto. Subsequent calls are non-blocking — playFromBytes
// hands off to the audio thread via the driver's internal channels.
func PlayMIDI(midData []byte, fade bool) {
	d := getMidiDriver()
	if d == nil {
		return
	}
	d.playFromBytes(midData, fade)
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
//   - "voladjust": adjust user volume on the persistent Player.
//   - <path>:      load the .mid at this path and start playing it.
func (d *midiDriver) handle(cmd string) {
	switch cmd {
	case "stop":
		d.stop(signlink.ReadMidiFade() == 1)
	case "voladjust":
		d.setUserVolume(volumeFromCentibels(signlink.ReadMidiVol()))
	default:
		d.play(cmd, signlink.ReadMidiFade() == 1)
	}
}

// play is the path-based entry point — kept as a defensive fallback
// for any signlink consumer that still publishes paths via
// signlink.MidiSave. The primary entry point is now PlayMIDI (via
// c.SaveMidi), which bypasses the disk roundtrip entirely.
func (d *midiDriver) play(path string, fade bool) {
	midData, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("audio/midi: read %q: %v\n", path, err)
		return
	}
	d.playFromBytes(midData, fade)
}

// playFromBytes is the actual play implementation. Reused by play()
// (file-path entry) and by PlayMIDI (direct in-memory entry from
// c.SaveMidi). The expensive bits — MIDI parsing, Synthesizer
// allocation, SoundFont voicing setup — happen on the caller's
// goroutine; only the brief player/source handoff takes d.mu.
//
// User volume is read from signlink.MidiVol and applied via the
// Player's SetVolume on every track change so a "music off → on"
// option toggle doesn't leak old volume into the new player.
func (d *midiDriver) playFromBytes(midData []byte, fade bool) {
	if d.ctx == nil {
		return
	}
	sf := d.ensureSoundFont()
	if sf == nil {
		return
	}

	midiFile, err := meltysynth.NewMidiFile(bytes.NewReader(midData))
	if err != nil {
		fmt.Printf("audio/midi: parse: %v\n", err)
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
	// loop=false: the game re-issues SetMidi via the NextMusicDelay
	// countdown at client.go:6997 when it wants the track restarted.
	// The TS reference reached the same conclusion — its synth-side
	// looping is commented out at tinymidipcm.js:190-194 with the note
	// "this was buggy with some midi files". Synth-side looping is
	// the wrong layer; the game decides when to repeat.
	seq.Play(midiFile, false)

	userVol := float64(volumeFromCentibels(signlink.ReadMidiVol()))

	gen := d.gen.Add(1)

	d.mu.Lock()
	if d.src == nil {
		// First-ever play creates the persistent player and source.
		// Source's fade-gain starts at 1.0 (full) — the source's gain
		// is the FADE multiplier, not the user volume.
		d.src = newMidiSource(seq, 1.0)
		d.player = d.ctx.NewPlayer(d.src)
		d.player.SetVolume(userVol)
		d.player.Play()
		d.mu.Unlock()
		return
	}
	src := d.src
	player := d.player
	d.mu.Unlock()

	// Re-apply user volume on every track change. Covers the case
	// where MidiVol changed while music was disabled (no voladjust
	// publish) and then the option flipped back on — without this,
	// the player would still carry the pre-disable volume.
	player.SetVolume(userVol)

	// Always Play() in case a previous hard-stop reset the player to
	// flush oto's internal buffer. Play() is idempotent on a
	// currently-playing player.
	player.Play()

	if !fade {
		// No fade requested: swap the sequencer and snap the fade-
		// gain back to full. From the listener's perspective the new
		// track replaces the old immediately. The snap (rather than
		// setGainTarget) keeps a coming-out-of-stop start from
		// suffering an unintended ~0.5s fade-in via the smoother.
		src.swap(seq)
		src.snapGain(1.0)
		return
	}

	go d.fadeAndSwap(src, seq, gen)
}

// fadeAndSwap drives the TS-style "fade-out, then start" sequencing:
//
//  1. Sets src's fade-gain target to 0 — the per-sample smoother in
//     Read exponentially decays it toward 0 over the 0.5s time
//     constant. At t = 2s (= 4τ) the effective fade-gain is ~1.8%,
//     audibly silent.
//  2. Sleeps fadeDuration.
//  3. If still the current generation, swaps the sequencer and snaps
//     the fade-gain back to 1.0 — matching TS's instant onset of the
//     new track at the end of its setTimeout. The Player's user
//     volume is untouched throughout, so the new track resumes at
//     whatever the slider was last set to.
//
// Abandons silently if a newer command supersedes the in-flight ramp.
func (d *midiDriver) fadeAndSwap(src *midiSource, newSeq *meltysynth.MidiFileSequencer, gen uint64) {
	src.setGainTarget(0)
	time.Sleep(fadeDuration)
	if d.gen.Load() != gen {
		return
	}
	src.swap(newSeq)
	src.snapGain(1.0)
}

// stop silences the live source. If fade is true, the gain target
// ramps to 0 over fadeDuration first via the source's smoother;
// afterwards the sequencer is cleared so no further notes are
// produced and Read synthesizes silence.
//
// Stops must FLUSH oto's internal buffer, not just halt it. oto pulls
// ~100ms of audio from the source's Read in advance and queues it
// internally; if we only Pause(), that queue stays frozen and replays
// on the next Play() — audible as a brief snippet of pre-stop music
// before the new track starts (the symptom that prompted this code:
// logout → silence → login → small piece of the old track → new
// track). Reset() pauses AND clears the queue, which is what we want.
//
// Reset is marked deprecated in oto v3.4 with the note "use Pause or
// Seek instead" — but neither alternative achieves "halt and flush"
// for a non-seekable streaming source like midiSource. The
// deprecation is misleading for this use case; if a future oto
// version removes Reset we will need to either implement io.Seeker
// on midiSource (returning 0 to discard the queue) or rebuild the
// player. For now Reset is the correct tool.
func (d *midiDriver) stop(fade bool) {
	d.mu.Lock()
	src := d.src
	player := d.player
	d.mu.Unlock()
	if src == nil {
		return
	}

	gen := d.gen.Add(1)

	if !fade {
		src.swap(nil)
		src.snapGain(0)
		if player != nil {
			player.Reset()
		}
		return
	}
	go func() {
		src.setGainTarget(0)
		time.Sleep(fadeDuration)
		if d.gen.Load() != gen {
			return
		}
		src.swap(nil)
		src.snapGain(0)
		// Faded stop: gain has already smoothed to ~0 over the
		// fade window, but oto's internal queue still holds those
		// (near-silent) samples from the fade. Reset flushes them
		// so a quick re-play has a clean baseline — same rationale
		// as the hard-stop path above.
		if player != nil {
			player.Reset()
		}
	}()
}

// setUserVolume is the "voladjust" handler — the in-game audio
// settings slider. Volume lives on the persistent oto.Player rather
// than the source so the change is applied AFTER oto's internal
// buffer (~100ms of pre-Read audio that's already in the queue),
// matching TS's gainNode-after-BufferSource topology. Source-side
// gain (snapGain/setGainTarget) only affects samples not yet pulled
// from Read, so a volume slider routed through there has a ~100ms
// tail at the old volume — exactly the "delay" the live tester
// reported.
//
// Player.SetVolume is sample-accurate at oto's audio thread, so the
// user hears the new volume essentially immediately (the 20ms
// watcher poll is now the dominant latency, matching TS).
func (d *midiDriver) setUserVolume(linear float32) {
	d.mu.Lock()
	player := d.player
	d.mu.Unlock()
	if player == nil {
		return
	}
	player.SetVolume(float64(linear))
}

// fadeDuration is the audible fade-out window. Matches TS's
// fadeseconds = 2 (tinymidipcm.js:140) and gives the 0.5 s smoothing
// time constant 4 τ to converge to ~1.8 % of the starting gain.
const fadeDuration = 2 * time.Second

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
// swappable sequencer pointer (under seqMu) and a smoothed gain pair
// (current → target via gainSmoothingAlpha). When seq is nil, Read
// fills its output with silence so the player stays alive across
// "stop" commands.
//
// Concurrent access: oto's audio goroutine calls Read; driver goroutines
// (handle, fadeAndSwap, stop) call swap, setGainTarget, and snapGain.
// gainMu held during Read covers the per-sample smoothing loop, which
// is microseconds for typical oto buffer sizes — short enough that
// setters never wait noticeably.
type midiSource struct {
	seqMu sync.Mutex
	seq   *meltysynth.MidiFileSequencer

	// gainMu guards currentGain and targetGain. currentGain is updated
	// per sample inside Read; targetGain is updated by driver
	// goroutines. snapGain sets both at once so a track change can
	// match TS's instant onset.
	gainMu      sync.Mutex
	currentGain float32
	targetGain  float32

	// scratch buffers reused across Read calls. Read is single-reader
	// (oto's audio goroutine) so these need no lock.
	left  []float32
	right []float32
}

func newMidiSource(seq *meltysynth.MidiFileSequencer, initialGain float32) *midiSource {
	return &midiSource{
		seq:         seq,
		currentGain: initialGain,
		targetGain:  initialGain,
	}
}

// swap atomically replaces the sequencer. Passing nil leaves the
// source emitting silence until the next swap. Gain state is
// untouched — the per-sample smoother handles any transition
// continuously. Callers wanting an instant gain change at the swap
// point follow with snapGain; callers wanting a smooth one use
// setGainTarget.
func (s *midiSource) swap(newSeq *meltysynth.MidiFileSequencer) {
	s.seqMu.Lock()
	s.seq = newSeq
	s.seqMu.Unlock()
}

// setGainTarget glides the gain toward g via the per-sample smoother.
// Used for voladjust and fade-outs. Time constant is ~0.5 s
// (gainSmoothingAlpha); the audible effect is an exponential approach
// reaching ~63 % at 0.5 s and ~98 % at 2 s.
func (s *midiSource) setGainTarget(g float32) {
	s.gainMu.Lock()
	s.targetGain = g
	s.gainMu.Unlock()
}

// snapGain sets BOTH current and target gain to g, skipping the
// smoother. Used after a fade-out completes so the new track's first
// samples are at full volume — matching TS's setValueAtTime restore
// in tinymidipcm.js:248.
func (s *midiSource) snapGain(g float32) {
	s.gainMu.Lock()
	s.currentGain = g
	s.targetGain = g
	s.gainMu.Unlock()
}

// gain returns the most recent currentGain — exposed only for tests.
func (s *midiSource) gain() float32 {
	s.gainMu.Lock()
	defer s.gainMu.Unlock()
	return s.currentGain
}

// Read fills p with interleaved stereo int16 LE PCM. It never returns
// io.EOF — when the sequencer is nil ("stop"), it emits silence so the
// player stays alive and ready for the next play().
//
// p's length is determined by oto. It is expected to be a multiple of
// 4 (one stereo int16 frame); odd remainders are truncated.
//
// The per-sample gain smoothing loop runs under gainMu so snapGain and
// setGainTarget see consistent state. The lock is held for the
// duration of one oto buffer's worth of samples (typically 1024
// frames, ~46 ms of audio but a few hundred µs of CPU); driver
// goroutines waiting on the lock for a setter call see no noticeable
// delay.
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

	s.gainMu.Lock()
	current := s.currentGain
	target := s.targetGain
	const oneMinusAlpha = 1 - gainSmoothingAlpha
	for i := range frames {
		current = current*gainSmoothingAlpha + target*oneMinusAlpha
		l := clipInt16(s.left[i] * current)
		r := clipInt16(s.right[i] * current)
		off := i * 4
		binary.LittleEndian.PutUint16(p[off:], uint16(l))
		binary.LittleEndian.PutUint16(p[off+2:], uint16(r))
	}
	s.currentGain = current
	s.gainMu.Unlock()
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

//go:build !js

package audio

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/sinshu/go-meltysynth/meltysynth"
)

// midiDriver is the native midiSink: a single persistent oto Player attached
// to a midiSource whose internal sequencer is hot-swappable. Track changes,
// stops, and volume changes mutate the source/player in place rather than
// tearing the player down, so oto's device stream stays open for the
// process lifetime (collapsing to one player fixed an audible-overlap bug
// in the pre-244 design; that property is preserved).
//
// All fade/latch sequencing lives in the shared audioLoop (audioloop.go) —
// the faithful port of the SignLink wrapper consumer — which calls these
// four primitives from its 50ms tick goroutine:
//
//	Java MidiPlayer (MidiPlayer.java)      midiDriver
//	play(seq, loop, volume)  :36-44   →    play(midData, loop, vol)
//	stop()                   :46-49   →    stop()
//	setVolume(0, volume)     :32-34   →    setVolume(vol)
//	running()                :51-53   →    running()
type midiDriver struct {
	ctx *oto.Context

	// soundFont is loaded once. nil if loading failed (no SF2 available);
	// play() then consumes commands but produces no audio.
	soundFont *meltysynth.SoundFont
	loadOnce  sync.Once

	// mu guards the player/source handoff and the running() bookkeeping.
	mu       sync.Mutex
	src      *midiSource
	player   *oto.Player
	looping  bool
	playedAt time.Time
	trackLen time.Duration
}

func newMidiDriver(ctx *oto.Context) *midiDriver { return &midiDriver{ctx: ctx} }

// play parses and starts a Standard MIDI File at linear gain vol/256.
//
// Java: MidiPlayer.play(Sequence, int loop, int volume) (MidiPlayer.java:
// 36-44) — sequencer.setSequence (the new track replaces the old
// immediately; any audible crossfade is the audioLoop's doing, not play's),
// setLoopCount(loop == 1 ? -1 : 0) (the fade flag doubles as loop-forever:
// MIDI_SONG region tracks loop, jingles play once), and setVolume runs
// before sequencer.start() so the first samples are at the right gain.
// A parse/synth failure maps to Java's swallowed InvalidMidiDataException.
func (d *midiDriver) play(midData []byte, loop bool, vol int) {
	if d.ctx == nil {
		return
	}
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
	// Java: sequencer.setLoopCount(loop == 1 ? -1 : 0) (MidiPlayer.java:39).
	seq.Play(midiFile, loop)

	d.mu.Lock()
	d.looping = loop
	d.playedAt = time.Now()
	d.trackLen = midiFile.GetLength()
	if d.src == nil {
		// First-ever play creates the persistent player and source.
		d.src = newMidiSource(seq)
		d.player = d.ctx.NewPlayer(d.src)
		d.player.SetVolume(linearVolume(vol))
		d.player.Play()
		d.mu.Unlock()
		return
	}
	src, player := d.src, d.player
	d.mu.Unlock()

	player.SetVolume(linearVolume(vol))
	src.swap(seq)
	// Play() is idempotent on a playing player and revives one paused by a
	// previous stop()'s haltAndFlush.
	player.Play()
}

// stop halts playback immediately.
//
// Java: MidiPlayer.stop() (MidiPlayer.java:46-49) — sequencer.stop() plus
// the setTick(-1) broadcast (all-notes-off / all-sound-off / reset
// controllers on all 16 channels, :59-83). swap(nil) makes the source
// render silence, covering the note kill; haltAndFlush discards oto's
// ~100ms pre-rendered queue so no stale audio replays on the next play.
func (d *midiDriver) stop() {
	d.mu.Lock()
	src, player := d.src, d.player
	d.looping = false
	d.trackLen = 0
	d.mu.Unlock()
	if src == nil {
		return
	}
	src.swap(nil)
	if player != nil {
		haltAndFlush(player)
	}
}

// setVolume applies linear gain vol/256 to the persistent player.
//
// Java: MidiPlayer.setVolume(0, volume) (MidiPlayer.java:32-34, 113-121) —
// rescales every channel's 14-bit CC7/CC39 by sqrt(volume/256), which
// through meltysynth's GM-quadratic channel curve is exactly a linear
// vol/256 output gain (see linearVolume). Player-side volume applies after
// oto's internal buffer, so each 50ms fade step from the audioLoop is heard
// ~immediately — matching javax.sound's immediate CC handling. The 8/256 ≈
// 3% amplitude steps are the same zipper the Java wrapper produced.
func (d *midiDriver) setVolume(vol int) {
	d.mu.Lock()
	player := d.player
	d.mu.Unlock()
	if player == nil {
		return
	}
	player.SetVolume(linearVolume(vol))
}

// running reports whether a sequence is still playing.
//
// Java: Sequencer.isRunning() (via MidiPlayer.running(), MidiPlayer.java:
// 51-53) — true from start() until the sequence's musical end or stop(); a
// looping sequence never ends. meltysynth exposes no equivalent getter, so
// the musical end is tracked by wall clock: play() records the MIDI file's
// length; stop() zeroes it.
func (d *midiDriver) running() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.looping || (d.trackLen > 0 && time.Since(d.playedAt) < d.trackLen)
}

// haltAndFlush stops player AND discards oto's internal pre-read buffer —
// the post-v3.4 replacement for the now-deprecated Player.Reset(), which
// did both in one call. Pause() alone only halts: oto keeps ~100ms of
// already-rendered PCM queued, which replays on the next Play(). Seeking
// flushes that queue — mux.Player.Seek resets its internal buffer before
// delegating to the source. Pause MUST come first: Seek resumes playback
// if the player was playing, so moving to the paused state before the
// flush is what makes this equivalent to Reset (paused + empty buffer).
// The Seek lands on midiSource.Seek, a no-op that exists only to satisfy
// oto's io.Seeker requirement.
func haltAndFlush(player *oto.Player) {
	player.Pause()
	_, _ = player.Seek(0, io.SeekStart)
}

// ensureSoundFont lazy-loads the SF2. Returns nil if loading failed.
func (d *midiDriver) ensureSoundFont() *meltysynth.SoundFont {
	d.loadOnce.Do(func() {
		sf, err := loadSoundFont()
		if err != nil {
			log.Printf("audio/midi: soundfont unavailable, music will be silent: %v", err)
			return
		}
		d.soundFont = sf
	})
	return d.soundFont
}

// midiSource is the io.Reader oto pulls PCM from. It owns a swappable
// sequencer pointer; when seq is nil it renders silence so the persistent
// player stays alive across stops. Gain is NOT applied here: the Java
// volume model is channel-CC based and maps onto oto Player.SetVolume (see
// setVolume above); the old per-sample fade smoother went away with the
// TS-style exponential fades it served — the 244 fade is the audioLoop's
// stepped setVolume.
type midiSource struct {
	seqMu sync.Mutex
	seq   *meltysynth.MidiFileSequencer

	// scratch buffers reused across Read calls. Read is single-reader
	// (oto's audio goroutine) so these need no lock.
	left  []float32
	right []float32
}

func newMidiSource(seq *meltysynth.MidiFileSequencer) *midiSource {
	return &midiSource{seq: seq}
}

// swap atomically replaces the sequencer. Passing nil leaves the source
// emitting silence until the next swap.
func (s *midiSource) swap(newSeq *meltysynth.MidiFileSequencer) {
	s.seqMu.Lock()
	s.seq = newSeq
	s.seqMu.Unlock()
}

// Read fills p with interleaved stereo int16 LE PCM. It never returns
// io.EOF — when the sequencer is nil ("stop"), it emits silence so the
// player stays alive and ready for the next play.
//
// p's length is determined by oto. It is expected to be a multiple of 4
// (one stereo int16 frame); odd remainders are truncated.
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

	for i := range frames {
		l := clipInt16(s.left[i])
		r := clipInt16(s.right[i])
		off := i * 4
		binary.LittleEndian.PutUint16(p[off:], uint16(l))
		binary.LittleEndian.PutUint16(p[off+2:], uint16(r))
	}
	return frames * 4, nil
}

// Seek is a no-op that exists solely so *oto.Player.Seek can flush oto's
// internal buffer on a stop (see haltAndFlush). A MIDI synthesizer has no
// seekable byte position — Read always renders the current sequencer — so
// there is nothing to reposition; returning (0, nil) just satisfies the
// io.Seeker assertion mux.Player.Seek makes before truncating its buffer.
func (s *midiSource) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
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

// Package audio is the playback half of the signlink audio protocol.
//
// In the original Jagex architecture, the signed-applet wrapper held a
// thread that watched signlink.midi (a file path, or one of the sentinels
// "stop" / "voladjust") and signlink.wave (a path to a WAV) and fed them
// to javax.sound. The LostCityRS Java repo only kept the publisher half
// of signlink — the consumer ran in a separate process boundary that
// wasn't ported.
//
// This package supplies the missing consumer in Go. It owns the single
// process-wide oto audio context, runs two poll-and-dispatch goroutines
// (one for MIDI, one for SFX), and drives a meltysynth-based SoundFont
// synthesizer for the MIDI side.
//
// Format unification: oto allows exactly one context per process, with a
// fixed sample format. We pick 22050 Hz stereo signed 16-bit LE because:
//   - It matches the TS reference client (tinymidipcm.js:137).
//   - The Wave/SFX pipeline already produces 22050 Hz mono 8-bit PCM
//     (see sound/wave.GetWave), upconversion to 16-bit stereo is cheap.
//   - meltysynth renders stereo float32; quantizing to int16 with volume
//     bake-in matches the TS GainNode model.
package audio

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/oto/v3"
)

// Format constants for the shared oto context. Changing these affects
// every player in the process; do not branch per-source.
const (
	SampleRate   = 22050
	ChannelCount = 2
)

// otoCtx is the lazily-initialized, process-wide audio context. nil if
// initialization failed (no audio device, permission denied, etc.) — in
// which case the watcher goroutines still run but produce no sound.
var (
	otoMu  sync.Mutex
	otoCtx *oto.Context

	// readyCtx publishes the oto context lock-free once it is ready, for the
	// SFX path (PlayWave). It is separate from otoCtx because ensureContext
	// holds otoMu across its blocking <-ready; PlayWave must not take otoMu or
	// it would block the game-update goroutine until the first user gesture.
	// nil = not ready (pre-gesture / low-memory / init failed) -> SFX dropped.
	readyCtx atomic.Pointer[oto.Context]
)

// Start boots the audio subsystem: it brings up the oto context,
// publishes it for the SFX path, and kicks off the MIDI watcher goroutine.
// It is safe to call even if audio init fails — Start logs a warning and
// returns; the game continues silently.
//
// Intended to be called once from cmd/client/main.go on a dedicated
// goroutine. Returns when the watchers are running; the goroutines then
// run for the process lifetime.
func Start() {
	ctx, err := ensureContext()
	if err != nil {
		log.Printf("audio: oto init failed, game will run silently: %v", err)
		// Unblock any PlayMIDI callers waiting on the driver — with a
		// nil driver, they'll return silently and the game continues
		// without music rather than hanging forever.
		registerMidiDriver(nil)
		return
	}

	// Create and register the driver SYNCHRONOUSLY before spawning the
	// watcher. This guarantees PlayMIDI callers (e.g. c.SaveMidi from
	// the c.RunMidi goroutine) find a live driver as soon as
	// audio.Start returns. Without sync registration, c.SaveMidi
	// could race ahead of audio.Start during boot and lose the first
	// scape_main play.
	d := newMidiDriver(ctx)
	registerMidiDriver(d)

	// Publish the ready context lock-free so PlayWave can use it without
	// taking otoMu (which ensureContext holds across <-ready).
	readyCtx.Store(ctx)

	go runMidiWatcher(d)
}

// DisableForLowMemory is the low-memory counterpart of Start: it brings
// up no oto context, loads no SoundFont, and spawns no watcher
// goroutines. It exists so the audio subsystem matches the Java client's
// lowMemory behavior, where the MIDI thread is never started, sounds.dat
// is never unpacked, and every runtime playback path is gated behind
// !lowMemory (deob/client.java:5949, 6163, 7374/9656/9868/9889). Doing
// any audio work there would contradict the whole point of low-memory
// mode.
//
// It still registers a nil MIDI driver — identical to Start's failed-init
// path above — so that any stray PlayMIDI caller returns silently instead
// of blocking forever on driverReady. In lowMemory every SetMidi/SaveMidi
// site is provably gated out and c.RunMidi never starts, so PlayMIDI
// should be unreachable; the nil driver is belt-and-suspenders matching
// the rest of this package.
//
// The lowMemory decision lives in cmd/client/main.go rather than here
// because the audio package cannot import client (client imports audio),
// so the flag is read at the one call site that knows both.
func DisableForLowMemory() {
	registerMidiDriver(nil)
}

// ensureContext lazily initializes oto on first call. Subsequent calls
// return the cached context. The function may return (nil, err) if the
// audio device is unavailable; callers must handle a nil context.
func ensureContext() (*oto.Context, error) {
	otoMu.Lock()
	defer otoMu.Unlock()
	if otoCtx != nil {
		return otoCtx, nil
	}

	op := &oto.NewContextOptions{
		SampleRate:   SampleRate,
		ChannelCount: ChannelCount,
		Format:       oto.FormatSignedInt16LE,
		BufferSize:   100 * time.Millisecond,
	}
	ctx, ready, err := oto.NewContext(op)
	if err != nil {
		return nil, err
	}
	<-ready
	otoCtx = ctx
	return ctx, nil
}

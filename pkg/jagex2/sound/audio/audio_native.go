//go:build !js

package audio

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
// process-wide oto audio context, runs the shared audioLoop ticker (the
// faithful SignLink consumer, see audioloop.go) for MIDI (driving a
// meltysynth-based SoundFont synthesizer), and plays SFX synchronously on
// demand via PlayWave (the game hands it WAV bytes directly; no watcher or
// scratch file).
//
// Format unification: oto allows exactly one context per process, with a
// fixed sample format. We pick 22050 Hz stereo signed 16-bit LE because:
//   - It matches the TS reference client (tinymidipcm.js:137).
//   - The Wave/SFX pipeline already produces 22050 Hz mono 8-bit PCM
//     (see sound/wave.GetWave), upconversion to 16-bit stereo is cheap.
//   - meltysynth renders stereo float32; quantizing to int16 with volume
//     bake-in matches the TS GainNode model.

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/oto/v3"
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

// Start boots the audio subsystem: it brings up the oto context, publishes
// it for the SFX path, and spawns the shared audioLoop ticker (the faithful
// SignLink consumer, see audioloop.go) driving the native MIDI sink. Safe to
// call even if audio init fails — the loop then runs with a no-op sink so
// latched signlink commands are still drained.
//
// Intended to be called once from cmd/client/main.go on a dedicated
// goroutine. Returns once the loop goroutine is spawned; that goroutine
// runs for the process lifetime.
func Start() {
	ctx, err := ensureContext()
	if err != nil {
		log.Printf("audio: oto init failed, game will run silently: %v", err)
		go runAudioLoop(nullSink{})
		return
	}

	// Publish the ready context lock-free so PlayWave can use it without
	// taking otoMu (which ensureContext holds across <-ready).
	readyCtx.Store(ctx)

	go runAudioLoop(newMidiDriver(ctx))
}

// DisableForLowMemory is the low-memory counterpart of Start: no oto
// context, no SoundFont, no audio device. It matches the Java client's
// lowMemory behavior, where every playback path is gated behind !lowMemory.
// A nullSink loop still drains the signlink command slot — the client's
// publish sites are lowmem-gated, but logout's stopMidi is unconditional,
// and a latched command should not sit in the slot forever.
func DisableForLowMemory() {
	go runAudioLoop(nullSink{})
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

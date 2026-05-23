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
)

// Start boots the audio subsystem: it brings up the oto context and
// kicks off the MIDI and Wave watcher goroutines. It is safe to call
// even if audio init fails — Start logs a warning and returns; the game
// continues silently.
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

	go runMidiWatcher(d)
	go runWaveWatcher(ctx)
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

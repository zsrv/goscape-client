//go:build !js

package audio

import (
	"encoding/binary"
	"io"
	"log"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/sign/signlink"
)

// waveMu guards lastWave, the most recently played SFX bytes — replayed by
// ReplayWave (Java replaywave).
var (
	waveMu   sync.Mutex
	lastWave []byte
)

// PlayWave plays a one-shot sound effect from in-memory WAV bytes (the 22050 Hz
// mono 8-bit format sound/wave.GetWave emits). It caches a copy for ReplayWave,
// then plays it if the audio context is ready.
//
// SFX are fire-and-forget: if the context is not ready — pre-gesture (oto only
// resumes the browser AudioContext after the first user interaction), low-memory
// mode, or a failed init — the clip is dropped, never queued or blocked. The
// call runs on the game-update path, which must not block, and a clip queued
// before the first gesture would otherwise fire as a stale burst on that
// gesture. readyCtx is a lock-free signal precisely because ensureContext holds
// otoMu across its blocking <-ready, so taking otoMu here would block until the
// first gesture (see readyCtx in audio.go).
func PlayWave(data []byte) {
	// Cache a defensive copy regardless of readiness: upstream var5.Data is a
	// reused buffer, and a later ReplayWave should still work.
	cp := make([]byte, len(data))
	copy(cp, data)
	waveMu.Lock()
	lastWave = cp
	waveMu.Unlock()

	ctx := readyCtx.Load()
	if ctx == nil {
		return
	}
	playWaveBytes(ctx, cp)
}

// ReplayWave replays the most recently played SFX (Java replaywave). No-op if
// nothing has played yet or the context is not ready.
func ReplayWave() {
	ctx := readyCtx.Load()
	if ctx == nil {
		return
	}
	waveMu.Lock()
	data := lastWave
	waveMu.Unlock()
	if data != nil {
		playWaveBytes(ctx, data)
	}
}

// lastWaveForTest exposes the replay cache for white-box tests.
func lastWaveForTest() []byte {
	waveMu.Lock()
	defer waveMu.Unlock()
	return lastWave
}

// playWaveBytes converts a 22050 Hz mono 8-bit WAV to the shared context's
// 22050 Hz stereo 16-bit LE format and spawns a one-shot Player.
//
// CRITICAL: oto's Player relies on a finalizer for cleanup (oto/v3 player.go:93:
// "(*mux.Player).Close() is called by the finalizer. Let's rely on it"). If the
// Player goes GC-unreachable before playback finishes, the finalizer closes it
// mid-stream and the SFX is cut off. We anchor the Player in a goroutine that
// polls IsPlaying() until the source drains, then drops the reference. Without
// this anchor even a 1-second clip can be silenced after a few ms.
func playWaveBytes(ctx *oto.Context, data []byte) {
	stereo, ok := wave8MonoToStereoInt16(data)
	if !ok {
		log.Printf("audio/wave: unsupported WAV format")
		return
	}
	p := ctx.NewPlayer(&byteSliceReader{b: stereo})
	// DEVIATION: Java 244's wavevol is dead — SignLink.audioLoop's wave
	// branch applies no gain at all (SignLink.java:427-475), and the 245.2
	// deob drops the wrapper consumer entirely. The Go port keeps the slider
	// working, mapping the published volume (245.2 centibel scale via
	// centibelToVol128) through the same vol/128 curve as music — so the
	// slider max (0 cB = internal 128) is unity gain, which IS Java's audible
	// full-gain SFX, and lower positions attenuate (the TS reference model,
	// audio.js:64). Per-Player so only new sounds pick it up, matching the
	// slider dispatch (UpdateVarp clientCode 4).
	p.SetVolume(linearVolume(centibelToVol128(signlink.ReadWaveVol())))
	p.Play()
	// Hold the only strong reference until the reader EOFs and oto stops, so the
	// finalizer can't close the Player mid-playback. Unbounded by design (SFX
	// are short, <=441 KB); a wedged IsPlaying() would leak one Player.
	go func() {
		for p.IsPlaying() {
			time.Sleep(50 * time.Millisecond)
		}
	}()
}

// wave8MonoToStereoInt16 parses a RIFF/WAV file emitted by sound.wave
// (22050 Hz, 1 ch, 8-bit unsigned PCM — see sound/wave.GetWave) and returns
// interleaved stereo 16-bit signed LE samples. Returns false if the header
// doesn't match: any deviation means the file wasn't produced by our own tone
// synthesizer and we'd rather skip than play garbage.
func wave8MonoToStereoInt16(data []byte) ([]byte, bool) {
	samples, ok := parseWave8Mono(data)
	if !ok {
		return nil, false
	}

	// 8-bit unsigned PCM uses 0x80 as the silent midpoint; (s-128)<<8 maps
	// 0x80->0, 0xFF->+32512, 0x00->-32768, preserving zero-crossings. Doubled to
	// both channels for the stereo context.
	out := make([]byte, len(samples)*4)
	for i, s := range samples {
		v := int16(int(s)-128) << 8
		off := i * 4
		u := uint16(v)
		binary.LittleEndian.PutUint16(out[off:], u)
		binary.LittleEndian.PutUint16(out[off+2:], u)
	}
	return out, true
}

// byteSliceReader is the one-shot io.Reader an SFX Player drains. It returns
// io.EOF after exhausting the slice, signalling oto to release the player.
type byteSliceReader struct {
	b   []byte
	pos int
}

func (r *byteSliceReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.pos:])
	r.pos += n
	return n, nil
}

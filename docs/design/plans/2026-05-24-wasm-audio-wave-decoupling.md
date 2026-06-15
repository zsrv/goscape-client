# WASM Sub-project 3: Browser Audio (wave SFX decoupling) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Play wave SFX from in-memory WAV bytes (new `audio.PlayWave`/`ReplayWave`) instead of the signlink scratch-file + `os.ReadFile` path, so SFX work in the browser and stop polluting the cache. Native plays from memory too (consistent with the existing MIDI refactor).

**Architecture:** `c.SaveWave`/`c.ReplayWave` call `audio.PlayWave`/`ReplayWave` directly with the bytes they already hold. `PlayWave` caches the bytes for replay, then plays via a **lock-free** `readyCtx atomic.Pointer[oto.Context]` (set in `Start` once oto is ready) — dropping the clip if not ready, never blocking. The signlink wave scratch-path + watcher + `os.ReadFile` are retired.

**Tech Stack:** Go 1.26, `github.com/ebitengine/oto/v3`. Spec: `docs/superpowers/specs/2026-05-24-wasm-audio-wave-decoupling-design.md`.

**Sandbox note:** prefix `go` commands with `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache`. The `GOOS=js GOARCH=wasm` build prints a harmless `writing stat cache: ... read-only file system` line — filter with `| grep -v 'stat cache'` and check exit codes. Commit with `git commit --no-gpg-sign`.

---

## File Structure

| File | Change |
|---|---|
| `pkg/jagex2/sound/audio/wave.go` | `PlayWave`/`ReplayWave`/`playWaveBytes` + `lastWave` cache + `lastWaveForTest`; remove `runWaveWatcher`/`playWaveFile`; drop `os` import, add `sync` |
| `pkg/jagex2/sound/audio/audio.go` | add `sync/atomic` import + `readyCtx` var; `readyCtx.Store(ctx)` in `Start`; remove `go runWaveWatcher(ctx)` |
| `pkg/jagex2/client/client.go` | `SaveWave`/`ReplayWave` → `audio.PlayWave`/`ReplayWave` |
| `pkg/jagex2/sound/audio/wave_test.go` | **new** — conversion + replay-cache tests |

---

## Task 1: Decouple wave SFX from the scratch-file path

**Files:** as above.

- [ ] **Step 1: Write the failing tests**

Create `pkg/jagex2/sound/audio/wave_test.go`:

```go
package audio

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// makeWAV builds the 22050 Hz / mono / 8-bit unsigned RIFF WAV that
// wave8MonoToStereoInt16 accepts (the format sound/wave.GetWave emits).
func makeWAV(samples []byte) []byte {
	buf := make([]byte, 44+len(samples))
	copy(buf[0:], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:], uint32(36+len(samples)))
	copy(buf[8:], "WAVE")
	copy(buf[12:], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:], 16) // subchunk1 size
	binary.LittleEndian.PutUint16(buf[20:], 1)  // PCM
	binary.LittleEndian.PutUint16(buf[22:], 1)  // mono
	binary.LittleEndian.PutUint32(buf[24:], SampleRate)
	binary.LittleEndian.PutUint32(buf[28:], SampleRate) // byte rate
	binary.LittleEndian.PutUint16(buf[32:], 1)          // block align
	binary.LittleEndian.PutUint16(buf[34:], 8)          // bits per sample
	copy(buf[36:], "data")
	binary.LittleEndian.PutUint32(buf[40:], uint32(len(samples)))
	copy(buf[44:], samples)
	return buf
}

func TestWave8MonoToStereoInt16(t *testing.T) {
	// 0x80 is the 8-bit silent midpoint -> 0; 0xFF -> +32512; 0x00 -> -32768.
	out, ok := wave8MonoToStereoInt16(makeWAV([]byte{0x80, 0xFF, 0x00}))
	if !ok {
		t.Fatal("valid WAV rejected")
	}
	if len(out) != 12 { // 3 samples * 2 channels * 2 bytes
		t.Fatalf("len = %d, want 12", len(out))
	}
	want := []int16{0, 0, 32512, 32512, -32768, -32768}
	for i, w := range want {
		if got := int16(binary.LittleEndian.Uint16(out[i*2:])); got != w {
			t.Errorf("int16 %d: got %d, want %d", i, got, w)
		}
	}
}

func TestWave8MonoToStereoInt16Rejects(t *testing.T) {
	if _, ok := wave8MonoToStereoInt16([]byte("short")); ok {
		t.Error("accepted too-short input")
	}
	wav := makeWAV([]byte{0x80})
	binary.LittleEndian.PutUint16(wav[34:], 16) // 16-bit, not our 8-bit format
	if _, ok := wave8MonoToStereoInt16(wav); ok {
		t.Error("accepted non-8-bit WAV")
	}
}

func TestPlayWaveCachesForReplay(t *testing.T) {
	// Start() is never called in tests, so readyCtx is nil: PlayWave drops
	// playback but must still cache a defensive copy for ReplayWave.
	in := makeWAV([]byte{0x10, 0x20, 0x30})
	PlayWave(in)

	cached := lastWaveForTest()
	if !bytes.Equal(cached, in) {
		t.Fatal("PlayWave did not cache the input bytes")
	}
	in[44] = 0x99
	if lastWaveForTest()[44] == 0x99 {
		t.Error("cache aliases the caller's slice (missing defensive copy)")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/sound/audio/ -run 'TestWave8|TestPlayWave' -v`
Expected: FAIL — compile error (`PlayWave`/`lastWaveForTest` undefined).

- [ ] **Step 3: Rewrite wave.go**

Replace the entire contents of `pkg/jagex2/sound/audio/wave.go` with:

```go
package audio

import (
	"encoding/binary"
	"io"
	"log"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"

	"github.com/zsrv/goscape-client/pkg/sign/signlink"
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
	// SFX volume is per-Player (each clip is short; the slider only affects new
	// sounds). Reading WaveVol once at spawn matches the slider dispatch
	// (client.go:3823-3833).
	p.SetVolume(float64(volumeFromCentibels(signlink.ReadWaveVol())))
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
	if len(data) < 44 {
		return nil, false
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, false
	}
	if string(data[12:16]) != "fmt " {
		return nil, false
	}
	audioFormat := binary.LittleEndian.Uint16(data[20:22])
	channels := binary.LittleEndian.Uint16(data[22:24])
	sampleRate := binary.LittleEndian.Uint32(data[24:28])
	bitsPerSample := binary.LittleEndian.Uint16(data[34:36])
	if audioFormat != 1 || channels != 1 || sampleRate != SampleRate || bitsPerSample != 8 {
		return nil, false
	}
	if string(data[36:40]) != "data" {
		return nil, false
	}
	dataLen := int(binary.LittleEndian.Uint32(data[40:44]))
	if 44+dataLen > len(data) {
		dataLen = len(data) - 44
	}
	samples := data[44 : 44+dataLen]

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
```

NOTE: this drops the `os` import and the `runWaveWatcher`/`playWaveFile` functions; `wave8MonoToStereoInt16` and `byteSliceReader` are carried over unchanged.

- [ ] **Step 4: Wire readyCtx into audio.go**

In `pkg/jagex2/sound/audio/audio.go`, add `"sync/atomic"` to the import block (after `"sync"`):

```go
import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/oto/v3"
)
```

Add `readyCtx` to the var block (alongside `otoMu`/`otoCtx`):

```go
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
```

In `Start`, after the `ensureContext()` error check and before `newMidiDriver`, publish the context and remove the wave watcher. The body becomes:

```go
func Start() {
	ctx, err := ensureContext()
	if err != nil {
		log.Printf("audio: oto init failed, game will run silently: %v", err)
		registerMidiDriver(nil)
		return
	}

	// Publish the ready context for the lock-free SFX path (PlayWave).
	readyCtx.Store(ctx)

	d := newMidiDriver(ctx)
	registerMidiDriver(d)

	go runMidiWatcher(d)
}
```

(Delete the `go runWaveWatcher(ctx)` line.) Also update `Start`'s doc comment, which currently says it "kicks off the MIDI and Wave watcher goroutines" — that becomes false. Change that clause to: "brings up the oto context, publishes it for the SFX path, and kicks off the MIDI watcher goroutine." Leave the rest of the comment (the "safe to call even if init fails" paragraph) intact.

- [ ] **Step 5: Route the client through the new API**

In `pkg/jagex2/client/client.go`, replace `SaveWave` and `ReplayWave` (currently at ~5317-5326):

```go
func (c *Client) SaveWave(arg0 []byte, arg1 int) bool {
	if arg0 == nil {
		return true
	}
	audio.PlayWave(arg0[:arg1])
	return true
}

func (c *Client) ReplayWave() bool {
	audio.ReplayWave()
	return true
}
```

(`audio` is already imported — `client.go:53`, used by `c.SaveMidi`. Always returning `true` is correct: there is no save-slot to back-pressure on, so the wave is consumed rather than retried.)

- [ ] **Step 6: Run tests and builds**

Run:
```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/sound/audio/ -run 'TestWave8|TestPlayWave' -v 2>&1 | grep -v 'stat cache'; echo "wave test ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... 2>&1 | grep -v 'stat cache'; echo "native build ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/sound/audio/ ./pkg/sign/signlink/ 2>&1 | grep -v 'stat cache'; echo "pkg tests ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm ${PIPESTATUS[0]}"
gofmt -l pkg/jagex2/sound/audio/wave.go pkg/jagex2/sound/audio/audio.go pkg/jagex2/sound/audio/wave_test.go pkg/jagex2/client/client.go
```
Expected: wave tests PASS; native build OK; audio + signlink tests PASS; `wasm 0`; gofmt prints nothing.

- [ ] **Step 7: Commit**

```bash
git add pkg/jagex2/sound/audio/wave.go pkg/jagex2/sound/audio/audio.go pkg/jagex2/sound/audio/wave_test.go pkg/jagex2/client/client.go
git commit --no-gpg-sign -m "feat(audio): play wave SFX from memory (browser-capable, no scratch file)"
```

---

## Task 2: Verification

- [ ] **Step 1: Native gate**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... && TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./... && TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go vet ./...`
Expected: all pass, vet clean.

- [ ] **Step 2: WASM build + vet**

Run:
```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "exit ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go vet ./pkg/jagex2/sound/audio/ 2>&1 | grep -v 'stat cache'; echo "vet exit ${PIPESTATUS[0]}"
```
Expected: both `exit 0`.

- [ ] **Step 3: Manual browser smoke (human-run, outside the sandbox)**

Performed by the user on the host. Document the result.

1. `make wasm && make wasm-serve` (cache/WS backend on `localhost:8888`, which must also serve `SCC1_Florestan.sf2` for MIDI).
2. Open `http://localhost:8080/?argv=10 0 highmem members`, log in, and click around to trigger SFX (UI clicks, combat, etc.).
3. Confirm: SFX are audible (after the first interaction), MIDI music plays, and in DevTools → Application → IndexedDB → `goscape` → `cache` there are **no `sound<n>.wav` entries** (SFX no longer hit the cache).
4. Sanity-check native still plays audio: `make run ARGS="10 0 highmem members ws://..."` or the usual desktop run, confirm SFX + music.

---

## Notes / deferred

- Vestigial after this: `signlink.WaveSave`/`WaveReplay`/`ConsumeWave`/`Wave` (dead but harmless, left in place like `signlink.MidiSave`); the `os.ReadFile` in `midi.go:185` (`play(path)`, never the live path). Not removed here.
- Sub-project 4 (lifecycle/polish) remains: Gio frame-loop review, `os.Exit` on wasm, the `signlink.OpenURL` reporterror hardcoded `127.0.0.1:8888`.

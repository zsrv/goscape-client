# Decouple wasm audio (pre-rendered Web Audio) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** On js/wasm, play MIDI music and SFX through pre-rendered static Web Audio buffers so playback runs on the browser audio thread, immune to main-thread CPU bursts — fixing the music skips. Native (oto) is unchanged.

**Architecture:** Build-tag the audio backend by platform. Oto-free helpers (format constants, volume math, WAV header parse, soundfont load, full-track MIDI→PCM render) move to untagged shared files. The existing oto code gets `//go:build !js` (behavior identical). A new `//go:build js` backend renders each MIDI track to a complete PCM `AudioBuffer` played via a looping-free `AudioBufferSourceNode`, with a `GainNode` fade chain that reproduces the current 2s fade-out-then-start; SFX decode to mono `AudioBuffer` one-shots. oto is not reachable from the js build.

**Tech Stack:** Go (GOOS=js GOARCH=wasm), syscall/js, Web Audio API, go-meltysynth (synthesis), existing signlink command protocol.

**Reference spec:** `docs/superpowers/specs/2026-05-25-wasm-audio-prerender-design.md`

**Build/test prefix (sandbox):** all `go` commands below assume the env prefix `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache`. Commit with `git commit --no-gpg-sign`.

---

## File Structure

Package `pkg/jagex2/sound/audio`. After this plan:

**Untagged (shared, oto-free):**
- `format.go` — `SampleRate`, `ChannelCount` consts + `volumeFromCentibels`. (moved out of audio.go/midi.go)
- `wavparse.go` — `parseWave8Mono(data) (samples []byte, ok bool)`: validate the RIFF/WAV header and return the raw 8-bit unsigned mono PCM. (extracted from `wave8MonoToStereoInt16`)
- `render.go` — `renderFrameCount(length time.Duration) int` and `renderMidiToPCM(sf, midData) (left, right []float32, err error)`. (new)
- `soundfont.go` — `loadSoundFont` (already untagged, oto-free; unchanged).

**Native (`//go:build !js`), behavior-identical to today:**
- `audio_native.go` — was `audio.go` minus moved consts: `Start`, `DisableForLowMemory`, `ensureContext`, `otoCtx`/`readyCtx`.
- `midi_native.go` — was `midi.go` minus moved `volumeFromCentibels`: driver, watcher, registry, `PlayMIDI`, fade/stop/volume, `midiSource`.
- `wave_native.go` — was `wave.go`: `PlayWave`, `ReplayWave`, `playWaveBytes`; `wave8MonoToStereoInt16` now calls `parseWave8Mono`.

**Browser (`//go:build js`), new:**
- `audio_js.go` — `Start`, `DisableForLowMemory`, the `AudioContext`, autoplay-gesture resume, MIDI driver registry + watcher (js).
- `midi_js.go` — `PlayMIDI` + the js music driver: render→`AudioBuffer` cache, `AudioBufferSourceNode` play, fade-out-then-start, stop, volume.
- `wave_js.go` — `PlayWave`, `ReplayWave`: mono `AudioBuffer` one-shots.
- `webaudio_js.go` — small syscall/js helpers (`f32ToJSFloat32Array`).

Public symbols (`Start`, `DisableForLowMemory`, `PlayMIDI`, `PlayWave`, `ReplayWave`, `SampleRate`, `ChannelCount`) exist exactly once per build, so callers (`cmd/client/main.go`, `client.SaveMidi`, SFX) are unchanged.

---

## Task 1: Extract shared oto-free helpers into untagged files

No behavior change. Just relocate symbols so the js build can reuse them.

**Files:**
- Create: `pkg/jagex2/sound/audio/format.go`
- Create: `pkg/jagex2/sound/audio/wavparse.go`
- Modify: `pkg/jagex2/sound/audio/audio.go` (remove `SampleRate`/`ChannelCount` consts)
- Modify: `pkg/jagex2/sound/audio/midi.go` (remove `volumeFromCentibels`)
- Modify: `pkg/jagex2/sound/audio/wave.go` (`wave8MonoToStereoInt16` calls `parseWave8Mono`)

- [ ] **Step 1: Create `format.go`** with the moved constants and volume math. Copy `volumeFromCentibels`' FULL body from `midi.go` (it currently lives there ending around line 558+; reproduce it verbatim including the `cb >= 0` branch).

```go
package audio

// Format constants for the audio pipeline. 22050 Hz stereo matches the TS
// reference client and the Wave/SFX pipeline (sound/wave.GetWave).
const (
	SampleRate   = 22050
	ChannelCount = 2
)

// volumeFromCentibels maps signlink's centibel scale (e.g. -400 for -4 dB,
// 0 for full) to a linear amplitude factor: dB = cb/100; linear = 10^(dB/20).
// Matches the TS client's Math.pow(10, dB/20) (tinymidipcm.js:300).
func volumeFromCentibels(cb int) float32 {
	if cb >= 0 {
		return 1.0
	}
	db := float64(cb) / 100.0
	return float32(math.Pow(10, db/20.0))
}
```

(Add `import "math"`. Verify the moved body matches `midi.go`'s original exactly — open `midi.go:554+` and copy the real implementation; the snippet above is the known shape, confirm the `cb >= 0` early return and the `math.Pow` line match before deleting the original.)

- [ ] **Step 2: Remove the moved symbols from their old homes.**
  - In `audio.go`: delete the `const ( SampleRate ... ChannelCount ... )` block.
  - In `midi.go`: delete the `volumeFromCentibels` function (and drop the now-unused `"math"` import from `midi.go` only if nothing else there uses it — `grep -n 'math\.' midi.go` first; `midi.go` does not otherwise use math, so remove it).

- [ ] **Step 3: Create `wavparse.go`** — extract the header validation + raw-sample slice from `wave8MonoToStereoInt16`.

```go
package audio

import "encoding/binary"

// parseWave8Mono validates a RIFF/WAV file emitted by sound/wave.GetWave
// (22050 Hz, 1 ch, 8-bit unsigned PCM) and returns the raw 8-bit unsigned
// mono sample bytes. ok is false if the header doesn't match exactly — any
// deviation means the file wasn't produced by our tone synthesizer.
func parseWave8Mono(data []byte) (samples []byte, ok bool) {
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
	return data[44 : 44+dataLen], true
}
```

- [ ] **Step 4: Rewrite `wave8MonoToStereoInt16` in `wave.go`** to call `parseWave8Mono` (keep its int16-stereo conversion; behavior identical):

```go
func wave8MonoToStereoInt16(data []byte) ([]byte, bool) {
	samples, ok := parseWave8Mono(data)
	if !ok {
		return nil, false
	}
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
```

- [ ] **Step 5: Verify native build + tests + fmt (behavior unchanged)**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go build ./... && go test ./pkg/jagex2/sound/audio/... && gofmt -l pkg/jagex2/sound/audio/`
Expected: build OK; tests PASS (existing `midi_test.go`/`wave_test.go` cover behavior); gofmt prints nothing.

- [ ] **Step 6: Commit**

```bash
git add pkg/jagex2/sound/audio/format.go pkg/jagex2/sound/audio/wavparse.go pkg/jagex2/sound/audio/audio.go pkg/jagex2/sound/audio/midi.go pkg/jagex2/sound/audio/wave.go
git commit --no-gpg-sign -m "refactor(audio): extract oto-free helpers to untagged files"
```

---

## Task 2: Shared full-track MIDI→PCM render (+ frame-count unit test)

**Files:**
- Create: `pkg/jagex2/sound/audio/render.go`
- Test: `pkg/jagex2/sound/audio/render_test.go`

- [ ] **Step 1: Write the failing test** for the pure frame-count math (the synth render itself needs an SF2 and is browser-verified; the length math is the unit-testable part).

```go
package audio

import (
	"testing"
	"time"
)

func TestRenderFrameCount(t *testing.T) {
	// 2.0s track at 22050 Hz = 44100 frames + 1s (22050) release tail.
	got := renderFrameCount(2 * time.Second)
	if got != 44100+SampleRate {
		t.Fatalf("renderFrameCount(2s) = %d; want %d", got, 44100+SampleRate)
	}
	// A zero-length track still renders at least the tail (never 0).
	if z := renderFrameCount(0); z != SampleRate {
		t.Fatalf("renderFrameCount(0) = %d; want %d (tail only)", z, SampleRate)
	}
}
```

- [ ] **Step 2: Run it, expect failure**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./pkg/jagex2/sound/audio/ -run TestRenderFrameCount -v`
Expected: FAIL — `undefined: renderFrameCount`.

- [ ] **Step 3: Implement `render.go`**

```go
package audio

import (
	"bytes"
	"time"

	"github.com/sinshu/go-meltysynth/meltysynth"
)

// renderTailFrames is the extra silence rendered after a track's musical
// length so trailing note releases aren't cut off. Reverb/chorus are
// disabled, so 1s is ample.
const renderTailFrames = SampleRate

// renderFrameCount returns how many PCM frames to render for a track of the
// given musical length: length rounded to frames, plus the release tail.
func renderFrameCount(length time.Duration) int {
	frames := int(length.Seconds()*float64(SampleRate)) + renderTailFrames
	if frames < renderTailFrames {
		frames = renderTailFrames
	}
	return frames
}

// renderMidiToPCM synthesizes an entire MIDI track to left/right float32 PCM
// at SampleRate (one Render call covers any length; the sequencer renders the
// decay tail past the last event with loop=false). Reverb/chorus disabled to
// match the native path. Returns equal-length left/right slices.
func renderMidiToPCM(sf *meltysynth.SoundFont, midData []byte) (left, right []float32, err error) {
	midiFile, err := meltysynth.NewMidiFile(bytes.NewReader(midData))
	if err != nil {
		return nil, nil, err
	}
	settings := meltysynth.NewSynthesizerSettings(SampleRate)
	settings.EnableReverbAndChorus = false
	synth, err := meltysynth.NewSynthesizer(sf, settings)
	if err != nil {
		return nil, nil, err
	}
	seq := meltysynth.NewMidiFileSequencer(synth)
	seq.Play(midiFile, false) // no synth-side loop; game re-issues SetMidi
	frames := renderFrameCount(midiFile.GetLength())
	left = make([]float32, frames)
	right = make([]float32, frames)
	seq.Render(left, right)
	return left, right, nil
}
```

- [ ] **Step 4: Run test, expect pass**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./pkg/jagex2/sound/audio/ -run TestRenderFrameCount -v`
Expected: PASS. Also confirm both targets compile: `GOOS=js GOARCH=wasm go build ./pkg/jagex2/sound/audio/ && go build ./pkg/jagex2/sound/audio/`.

- [ ] **Step 5: Commit**

```bash
git add pkg/jagex2/sound/audio/render.go pkg/jagex2/sound/audio/render_test.go
git commit --no-gpg-sign -m "feat(audio): shared full-track MIDI->PCM render"
```

---

## Task 3: Build-tag the native files

Pure tagging. After Task 1 the shared symbols are gone from `audio.go`/`midi.go`/`wave.go`; now restrict the remaining (oto) code to non-js builds so the js build won't pull oto.

**Files:** rename + tag (use `git mv` to preserve history):
- `audio.go` → `audio_native.go`
- `midi.go` → `midi_native.go`
- `wave.go` → `wave_native.go`

- [ ] **Step 1: Rename and add build tags**

```bash
cd pkg/jagex2/sound/audio
git mv audio.go audio_native.go
git mv midi.go midi_native.go
git mv wave.go wave_native.go
```

Add as the FIRST line of each of the three renamed files (before `package audio`), then a blank line:

```go
//go:build !js
```

- [ ] **Step 2: Verify native still builds + tests + js build excludes oto**

Run:
```
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache CGO_ENABLED=1 go build ./... && go test ./pkg/jagex2/sound/audio/...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOOS=js GOARCH=wasm go build ./pkg/jagex2/sound/audio/
```
Expected: native build + tests PASS. The js build of the audio package will FAIL now with "undefined: Start / PlayMIDI / PlayWave / ReplayWave / DisableForLowMemory" — that's expected; Tasks 4–6 add the js implementations. (If you want a green checkpoint here, skip the js build line until Task 6.)

- [ ] **Step 3: Commit**

```bash
git add -A pkg/jagex2/sound/audio/
git commit --no-gpg-sign -m "refactor(audio): restrict oto backend to !js builds"
```

---

## Task 4: js audio context, Start, registry + watcher

**Files:**
- Create: `pkg/jagex2/sound/audio/webaudio_js.go`
- Create: `pkg/jagex2/sound/audio/audio_js.go`

- [ ] **Step 1: `webaudio_js.go` — float32→JS Float32Array helper**

```go
//go:build js

package audio

import (
	"syscall/js"
	"unsafe"
)

// f32ToJSFloat32Array copies a Go []float32 into a new JS Float32Array via a
// byte view (one bulk CopyBytesToJS, no per-element boundary crossings).
func f32ToJSFloat32Array(s []float32) js.Value {
	if len(s) == 0 {
		return js.Global().Get("Float32Array").New(0)
	}
	b := unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), len(s)*4)
	u8 := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(u8, b)
	return js.Global().Get("Float32Array").New(u8.Get("buffer"))
}
```

- [ ] **Step 2: `audio_js.go` — context, Start, DisableForLowMemory, gesture resume, watcher/registry**

```go
//go:build js

package audio

import (
	"log"
	"sync"
	"syscall/js"
	"time"

	"github.com/zsrv/goscape-client/pkg/sign/signlink"
)

// One AudioContext for the process. Created in Start; resumed on the first
// user gesture (browsers block autoplay until then).
var (
	ac        js.Value // AudioContext
	musicGain js.Value // GainNode: user music volume (MidiVol)
	sfxGain   js.Value // GainNode: user SFX volume (WaveVol)
	gestures  []js.Func
)

// midiPollInterval matches the native watcher cadence (one game tick).
const midiPollInterval = 20 * time.Millisecond

// driver registry: PlayMIDI callers wait until Start has created the driver
// (or registered nil on disable), so the first SaveMidi can't race startup.
var (
	driverMu    sync.Mutex
	driver      *webMidiDriver
	driverReady = make(chan struct{})
)

func registerMidiDriver(d *webMidiDriver) {
	driverMu.Lock()
	driver = d
	driverMu.Unlock()
	close(driverReady)
}

func getMidiDriver() *webMidiDriver {
	<-driverReady
	driverMu.Lock()
	defer driverMu.Unlock()
	return driver
}

// Start brings up the Web Audio context and the MIDI driver + watcher.
// Called once from cmd/client/main.go on a goroutine.
func Start() {
	ctor := js.Global().Get("AudioContext")
	if !ctor.Truthy() {
		ctor = js.Global().Get("webkitAudioContext")
	}
	if !ctor.Truthy() {
		log.Printf("audio: Web Audio unavailable, game will run silently")
		registerMidiDriver(nil)
		return
	}
	ac = ctor.New(map[string]any{"sampleRate": SampleRate})

	musicGain = ac.Call("createGain")
	musicGain.Call("connect", ac.Get("destination"))
	sfxGain = ac.Call("createGain")
	sfxGain.Call("connect", ac.Get("destination"))

	installGestureResume()

	d := newWebMidiDriver()
	registerMidiDriver(d)
	go runMidiWatcher(d)
}

// DisableForLowMemory matches the native low-memory path: no context, no
// soundfont, nil driver so any stray PlayMIDI returns silently.
func DisableForLowMemory() {
	registerMidiDriver(nil)
}

// installGestureResume resumes the AudioContext on the first user gesture
// (touchend/keyup/mouseup), mirroring oto's autoplay handling.
func installGestureResume() {
	var resume js.Func
	resume = js.FuncOf(func(this js.Value, args []js.Value) any {
		ac.Call("resume")
		for _, e := range []string{"touchend", "keyup", "mouseup"} {
			js.Global().Call("removeEventListener", e, resume)
		}
		resume.Release()
		return nil
	})
	gestures = append(gestures, resume)
	for _, e := range []string{"touchend", "keyup", "mouseup"} {
		js.Global().Call("addEventListener", e, resume)
	}
}

func runMidiWatcher(d *webMidiDriver) {
	for {
		if cmd := signlink.ConsumeMidi(); cmd != "" {
			d.handle(cmd)
		}
		time.Sleep(midiPollInterval)
	}
}
```

- [ ] **Step 3: This task does not build standalone** (it references `webMidiDriver` from Task 5). No checkpoint build here; commit after Task 5. Continue.

---

## Task 5: js music driver (PlayMIDI, render→AudioBuffer, fade, stop, volume)

**Files:**
- Create: `pkg/jagex2/sound/audio/midi_js.go`

- [ ] **Step 1: `midi_js.go`**

```go
//go:build js

package audio

import (
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

// PlayMIDI renders an in-memory MIDI track and plays it (replacing the
// current track). fade=true reproduces the native fade-out-then-start.
func PlayMIDI(midData []byte, fade bool) {
	d := getMidiDriver()
	if d == nil {
		return
	}
	d.playFromBytes(midData, fade)
}

// webMidiDriver owns the current music AudioBufferSourceNode + its fadeGain,
// a one-track render cache, and a generation counter so a rapid track change
// abandons an in-flight fade.
type webMidiDriver struct {
	soundFont *meltysynth.SoundFont
	loadOnce  sync.Once

	mu       sync.Mutex
	curSrc   js.Value // current AudioBufferSourceNode (or undefined)
	curFade  js.Value // current per-track fade GainNode (or undefined)
	cacheKey string   // identity of the cached rendered track
	cacheBuf js.Value // cached AudioBuffer for cacheKey

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

// renderToBuffer renders midData to an AudioBuffer, reusing the cache when the
// same track is re-issued (the game's NextMusicDelay restart).
func (d *webMidiDriver) renderToBuffer(midData []byte) js.Value {
	key := string(midData) // track identity; cheap vs re-rendering
	if d.cacheKey == key && d.cacheBuf.Truthy() {
		return d.cacheBuf
	}
	sf := d.ensureSoundFont()
	if sf == nil {
		return js.Undefined()
	}
	left, right, err := renderMidiToPCM(sf, midData)
	if err != nil {
		log.Printf("audio/midi: render: %v", err)
		return js.Undefined()
	}
	buf := ac.Call("createBuffer", ChannelCount, len(left), SampleRate)
	buf.Call("copyToChannel", f32ToJSFloat32Array(left), 0)
	buf.Call("copyToChannel", f32ToJSFloat32Array(right), 1)
	d.cacheKey = key
	d.cacheBuf = buf
	return buf
}

// playFromBytes renders + plays the track. With fade, the OLD source's fade
// gain ramps to 0 and is stopped after fadeDuration, THEN the new source
// starts at full — the native fade-out-then-start (no overlap), gen-guarded.
func (d *webMidiDriver) playFromBytes(midData []byte, fade bool) {
	d.applyMidiVolume()
	buf := d.renderToBuffer(midData)
	if !buf.Truthy() {
		return
	}
	gen := d.gen.Add(1)

	d.mu.Lock()
	oldSrc, oldFade := d.curSrc, d.curFade
	d.mu.Unlock()

	startNew := func() {
		fadeGain := ac.Call("createGain")
		fadeGain.Get("gain").Set("value", 1.0)
		fadeGain.Call("connect", musicGain)
		src := ac.Call("createBufferSource")
		src.Set("buffer", buf)
		src.Set("loop", false)
		src.Call("connect", fadeGain)
		src.Call("start")
		d.mu.Lock()
		d.curSrc, d.curFade = src, fadeGain
		d.mu.Unlock()
	}

	if !fade || !oldSrc.Truthy() {
		stopSource(oldSrc)
		startNew()
		return
	}

	// Fade the old track out over fadeDuration, then (if not superseded)
	// stop it and start the new one.
	now := ac.Get("currentTime").Float()
	oldFade.Get("gain").Call("setTargetAtTime", 0.0, now, fadeTimeConstant)
	go func() {
		time.Sleep(fadeDuration)
		if d.gen.Load() != gen {
			return // superseded by a newer command
		}
		stopSource(oldSrc)
		startNew()
	}()
}

// stop silences music. With fade, ramp the current source's fade gain to 0
// then stop it after fadeDuration; without, stop immediately. gen-guarded.
func (d *webMidiDriver) stop(fade bool) {
	gen := d.gen.Add(1)
	d.mu.Lock()
	src, fadeGain := d.curSrc, d.curFade
	d.curSrc, d.curFade = js.Undefined(), js.Undefined()
	d.mu.Unlock()
	if !src.Truthy() {
		return
	}
	if !fade {
		stopSource(src)
		return
	}
	now := ac.Get("currentTime").Float()
	fadeGain.Get("gain").Call("setTargetAtTime", 0.0, now, fadeTimeConstant)
	go func() {
		time.Sleep(fadeDuration)
		if d.gen.Load() != gen {
			return
		}
		stopSource(src)
	}()
}

// stopSource stops + disconnects an AudioBufferSourceNode if present.
func stopSource(src js.Value) {
	if src.Truthy() {
		src.Call("stop")
		src.Call("disconnect")
	}
}
```

- [ ] **Step 2: Commit Tasks 4+5 together** (they compile as a unit once the wave_js stubs from Task 6 are NOT yet needed — but `Start` references nothing from wave_js, so the package still lacks `PlayWave`/`ReplayWave` for js. Do NOT build-check the js audio package yet; commit and proceed to Task 6.)

```bash
git add pkg/jagex2/sound/audio/webaudio_js.go pkg/jagex2/sound/audio/audio_js.go pkg/jagex2/sound/audio/midi_js.go
git commit --no-gpg-sign -m "feat(audio/js): Web Audio context + pre-rendered MIDI music driver"
```

---

## Task 6: js SFX path + full build verification

**Files:**
- Create: `pkg/jagex2/sound/audio/wave_js.go`

- [ ] **Step 1: `wave_js.go`**

```go
//go:build js

package audio

import (
	"log"
	"sync"
	"syscall/js"

	"github.com/zsrv/goscape-client/pkg/sign/signlink"
)

var (
	waveMu   sync.Mutex
	lastWave []byte
)

// PlayWave plays a one-shot SFX from 22050 Hz mono 8-bit WAV bytes and caches
// a copy for ReplayWave. Dropped silently if the context isn't ready yet.
func PlayWave(data []byte) {
	cp := make([]byte, len(data))
	copy(cp, data)
	waveMu.Lock()
	lastWave = cp
	waveMu.Unlock()
	playWaveBytes(cp)
}

// ReplayWave replays the most recent SFX (Java replaywave).
func ReplayWave() {
	waveMu.Lock()
	data := lastWave
	waveMu.Unlock()
	if data != nil {
		playWaveBytes(data)
	}
}

func playWaveBytes(data []byte) {
	if !ac.Truthy() {
		return // pre-gesture / disabled
	}
	samples, ok := parseWave8Mono(data)
	if !ok {
		log.Printf("audio/wave: unsupported WAV format")
		return
	}
	// 8-bit unsigned (0x80 = silence) -> float32 [-1,1): (s-128)/128.
	f := make([]float32, len(samples))
	for i, s := range samples {
		f[i] = (float32(int(s)-128)) / 128.0
	}
	buf := ac.Call("createBuffer", 1, len(f), SampleRate)
	buf.Call("copyToChannel", f32ToJSFloat32Array(f), 0)

	// SFX volume read once at spawn (matches native per-clip volume).
	sfxGain.Get("gain").Set("value", float64(volumeFromCentibels(signlink.ReadWaveVol())))

	src := ac.Call("createBufferSource")
	src.Set("buffer", buf)
	src.Call("connect", sfxGain)
	src.Call("start")
	// One-shot: the browser reclaims the node after it ends; no Go anchor
	// needed (unlike oto's finalizer-based Player).
}
```

- [ ] **Step 2: Verify the js audio package builds + the whole js client builds**

Run:
```
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOOS=js GOARCH=wasm go build ./pkg/jagex2/sound/audio/
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOOS=js GOARCH=wasm go build ./...
```
Expected: both PASS.

- [ ] **Step 3: Verify oto is NOT reachable from the js build**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOOS=js GOARCH=wasm go list -deps ./cmd/client | grep ebitengine/oto || echo "oto NOT in js build (correct)"`
Expected: prints `oto NOT in js build (correct)`.

- [ ] **Step 4: Verify native unchanged + lint + fmt both targets**

Run:
```
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache CGO_ENABLED=1 go build ./... && go test ./pkg/... ./cmd/...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go vet ./... && GOOS=js GOARCH=wasm go vet ./pkg/jagex2/sound/audio/
gofmt -l pkg/jagex2/sound/audio/
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOLANGCI_LINT_CACHE=$TMPDIR/golangci-cache go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run ./pkg/jagex2/sound/audio/...
```
Expected: native build + tests PASS; vet both targets OK; gofmt prints nothing; lint 0 issues.

- [ ] **Step 5: Commit**

```bash
git add pkg/jagex2/sound/audio/wave_js.go
git commit --no-gpg-sign -m "feat(audio/js): Web Audio SFX one-shots; drop oto from js build"
```

- [ ] **Step 6: Browser verification (manual, on host — cannot be done in sandbox)**

`make wasm && make wasm-serve`, open the client, and confirm:
- Music plays after the first click/keypress (autoplay resume), and restarts via the game's NextMusicDelay re-issue.
- Area-change music transition fades out then starts (2s), matching the desktop feel; rapid area changes don't leave a stuck/clobbered track.
- Music volume slider works; SFX play and respect the SFX volume.
- **The goal:** music does NOT skip during cache download, scene load, or area changes with music playing.

---

## Notes for the implementer

- **No oto in js files.** If `GOOS=js go build` ever pulls `ebitengine/oto`, an oto symbol leaked into an untagged or js-tagged file — find it with the `go list -deps` check in Task 6 Step 3.
- **AudioBuffer memory:** only the current track's buffer is cached (`cacheBuf`); a new track replaces it and the old buffer becomes unreferenced for GC. Do not accumulate a map of buffers.
- **`copyToChannel` availability:** all target browsers support it (it's the same API oto's ScriptProcessor fallback guards for Safari 11; our minimum is WebGL-capable browsers, well past that). If a regression appears on an ancient browser, fall back to `getChannelData(ch).set(...)`.
- **Fade fidelity:** the design is fade-OUT-then-START (no crossfade/overlap), matching the current native `fadeAndSwap`. Keep the `gen` guard so a region change during a fade abandons the stale fade goroutine.

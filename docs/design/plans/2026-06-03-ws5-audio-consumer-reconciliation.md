# WS5 — Audio Consumer Reconciliation (rev-244) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the Java-244 wrapper-side audio consumer (`SignLink.audioLoop` fade state machine + `MidiPlayer` volume/loop semantics) faithfully onto the existing Go audio sinks (oto native / Web Audio prerender), plus the [M] side scope (linear volume scale, `midivol/wavevol=96` defaults, `storeid` 32–34).

**Architecture:** A new shared `audioLoop` (50ms ticker in `pkg/jagex2/sound/audio/audioloop.go`) is the faithful port of `SignLink.audioLoop`/`playMidi` (SignLink.java:359-425): linear ±8-per-tick fade on the `0..midivol` integer scale, single-slot latched command protocol, fade-flag-doubles-as-loop. Both backends (`midiDriver` native, `webMidiDriver` js) are reduced to a 4-primitive `midiSink` interface mirroring Java's `MidiPlayer` surface: `play(data, loop, vol)` / `stop()` / `setVolume(vol)` / `running()`. All TS-style exponential-fade machinery is deleted.

**Tech Stack:** Go 1.26, oto/v3, go-meltysynth, syscall/js (wasm build), signlink slot protocol.

**Branch:** `rev-244`. Commit with `git commit --no-gpg-sign`. Build/test prefix: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache`.

---

## Context (read before Task 1)

### Java references (NEVER read the Client-Java working tree — it has local edits)

```bash
cd $HOME/Code/github.com/LostCityRS/Client-Java
git show 01f16088:src/main/java/jagex2/client/sign/SignLink.java    # the consumer
git show 01f16088:src/main/java/jagex2/client/sign/MidiPlayer.java  # the synth wrapper
git show 01f16088:src/main/java/jagex2/client/Client.java           # client call sites
```

### The volume-math proof (why no CC interception is ported)

Java `MidiPlayer.setVolume(0, vol)` rescales every channel's 14-bit CC7/CC39 volume:
`getVolume(ch) = sqrt(((cc14*vol) >>> 8) * cc14) ≈ cc14 * sqrt(vol/256)` (MidiPlayer.java:123-126).
The `check()` interceptor (MidiPlayer.java:134-160) applies the same rescale to the *file's own*
CC7/CC39/CC121 messages before the synth sees them.

go-meltysynth squares channel volume per the GM spec
(`channelGain = (volume×expression)²`, meltysynth/voice.go:195-197). Composing:
audible gain = `(cc14·sqrt(vol/256))² = cc14²·(vol/256)` — i.e. meltysynth's
native rendering times a **linear `vol/256` post-gain**. So `linearVolume(vol) = vol/256`
applied at the player/gain-node reproduces Java's curve exactly, and the entire CC
machinery does not need porting.

**Documented deviation:** Java's `check()` resets its tracked channel volume to 12800
on CC121 (Reset All Controllers); meltysynth handles CC121 per MIDI RP-015 (volume NOT
reset). Spec-correct on our side; only audible for files sending CC121 after non-default CC7.

### Faithful semantics being ported (vs current Go behavior)

| Aspect | Java 244 (target) | Current Go (to be replaced) |
|---|---|---|
| Fade | linear ±8 per 50ms tick on 0..midivol (≈600ms @ 96); 3 states (out→in→steady) | 2s exponential `setTargetAtTime`-style smoothing |
| Volume scale | linear, client sends 128/96/64/32, default 96, gain = vol/256 | centibels (0/-400/-800/-1200), `10^(cb/2000)` |
| Looping | `setLoopCount(loop==1 ? -1 : 0)` — MIDI_SONG tracks (fade=1) loop forever; jingles play once | never loops (225 re-issue mechanism, gone in 244 → music dies after one pass — a live bug) |
| Stop | always immediate (`stopMidi` zeroes midifade first) | faded stop when MidiFade==1 |
| Command slot | ONE `midi` variable, latest-wins, **latched while fading out** (`if (!midiFadingOut) midi="none"`, SignLink.java:422-424) | `ConsumeMidi` clears unconditionally; track bytes bypass the slot entirely |
| Consumer cadence | 50ms (SignLink.java:198) | 20ms watcher |
| First play / jingle | no fade-in: straight to full midivol | same (incidentally) |

### Documented deviations (intentional, keep)

1. Track bytes travel **in-memory** through the signlink slot (no `jingle<pos>.mid` disk round-trip — applet process-boundary artifact).
2. The **wave branch** of audioLoop (SignLink.java:427-475) is NOT moved into the ticker: SFX keep playing directly/overlapping via `PlayWave` instead of synchronously blocking the consumer thread (in Java a playing SFX stalls any in-flight MIDI fade).
3. **wavevol is dead in Java 244** (no gain applied anywhere). Per user decision: keep the SFX slider functional via the same `linearVolume` curve.
4. CC121 handling (see proof above).
5. The `FileStream` sector cache (`main_file_cache.dat/.idx0-4`, >50MB delete) is NOT ported — the Go port replaced it with the plain-name storage seam + `/ondemand.zip` bundle (`ondemand.New(..., nil)`).
6. `MidiPlayer.setSoundfont` has no caller in Java 244 (dead) — Go keeps its own SF2 loading.
7. `setVolume`'s `velocity` parameter is always 0 at every call site (the `0.1^(velocity*0.0005)` factor is always 1) — not ported.
8. Java 244 **removed** the async `cacheload`/`cachesave` string protocol (archives go through `FileStream` instead). The Go port keeps `CacheLoad`/`CacheSave` — they front the plain-name storage seam that *replaces* the unported `FileStream` (see deviation 5). Nothing to do; do not delete them.
9. The wave branch's `FloatControl.Type.PAN` block is dead in Java (`curPosition` is hardwired `NORMAL`, SignLink.java:363) — not ported.

---

## File map

| File | Action |
|---|---|
| `pkg/jagex2/client/sign/signlink/signlink.go` | Modify: defaults 96, single-slot `Midi`/`MidiData`, `SetMidiTrack`/`PeekMidi`/`ClearMidi`, `StoreID` var; delete `ConsumeMidi` (Task 4) |
| `pkg/jagex2/client/sign/signlink/signlink_audio_helpers_test.go` | Modify: new slot tests; migrate off `ConsumeMidi` |
| `pkg/jagex2/client/sign/signlink/storage_disk.go` | Modify: `storeDirName()` clamp |
| `pkg/jagex2/client/sign/signlink/storage_disk_test.go` | Modify: clamp test |
| `pkg/jagex2/sound/audio/audioloop.go` | **Create**: `midiSink`, `audioLoop`, `runAudioLoop`, `nullSink` |
| `pkg/jagex2/sound/audio/audioloop_test.go` | **Create**: state-machine tests w/ fake sink |
| `pkg/jagex2/sound/audio/format.go` | Modify: add `linearVolume`, later delete `volumeFromCentibels` |
| `pkg/jagex2/sound/audio/midi_native.go` | Rewrite: sink primitives, no fades/gen/watcher/registry |
| `pkg/jagex2/sound/audio/audio_native.go` | Modify: `Start`/`DisableForLowMemory` spawn `runAudioLoop` |
| `pkg/jagex2/sound/audio/midi_js.go` | Rewrite: sink primitives + chunk-loop scheduler |
| `pkg/jagex2/sound/audio/audio_js.go` | Modify: same wiring as native |
| `pkg/jagex2/sound/audio/midi_test.go` | Modify: drop gain-smoother tests, adjust `newMidiSource` |
| `pkg/jagex2/sound/audio/wave_native.go` | Modify: `linearVolume` for WaveVol |
| `pkg/jagex2/sound/audio/wave_js.go` | Modify: same |
| `pkg/jagex2/client/client.go` | Modify: `SaveMidi` (sig + reroute), `SetMidiVolume` (2-arg), `UpdateVarp` constants, OnDemand call site |
| `cmd/client/main.go` | Modify: `-store-id` flag; comment refresh |
| `LOGIC-DELTA-SCOPE.md`, `.claude/resume/` | Docs close-out |

Gates after every task (all must pass before commit):

```bash
cd .
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go vet ./...
gofmt -l pkg cmd   # expect empty
```

---

### Task 1: signlink — 244 defaults + single-slot midi track protocol

**Files:**
- Modify: `pkg/jagex2/client/sign/signlink/signlink.go`
- Test: `pkg/jagex2/client/sign/signlink/signlink_audio_helpers_test.go`

- [ ] **Step 1.1: Write the failing test**

Append to `signlink_audio_helpers_test.go` (add `"bytes"` to its imports):

```go
// TestMidiSlotSingleSlotClobber pins the Java single-slot protocol: SignLink
// has ONE `midi` field (SignLink.java:45), so a "stop" issued while a track
// is pending replaces it (the track is lost), and vice versa. PeekMidi must
// NOT clear — the consumer clears separately (the fade-out latch,
// SignLink.java:422-424).
func TestMidiSlotSingleSlotClobber(t *testing.T) {
	t.Cleanup(ClearMidi)

	SetMidiTrack([]byte{1, 2})
	SetMidiCommand("stop")
	cmd, data := PeekMidi()
	if cmd != "stop" || data != nil {
		t.Fatalf("stop should clobber the pending track: got %q %v", cmd, data)
	}

	SetMidiTrack([]byte{3})
	cmd, data = PeekMidi()
	if cmd != "play" || !bytes.Equal(data, []byte{3}) {
		t.Fatalf("track should clobber the pending stop: got %q %v", cmd, data)
	}
	if again, _ := PeekMidi(); again != "play" {
		t.Fatalf("PeekMidi must not clear the slot: got %q", again)
	}

	ClearMidi()
	if cmd, data := PeekMidi(); cmd != "" || data != nil {
		t.Fatalf("ClearMidi should empty the slot: got %q %v", cmd, data)
	}
}
```

- [ ] **Step 1.2: Run it to verify it fails**

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/client/sign/signlink/ -run TestMidiSlotSingleSlotClobber
```
Expected: FAIL (compile error: `SetMidiTrack`, `PeekMidi`, `ClearMidi` undefined).

- [ ] **Step 1.3: Implement in `signlink.go`**

In the `var` block, change the `Midi`, `MidiVol`, `WaveVol` declarations and add `MidiData`:

```go
	// Midi is the single-slot audio command: "" (none), "stop", "voladjust",
	// or "play" (track bytes pending in MidiData). Latest write wins —
	// exactly Java's lone `midi` field, where a command can clobber a
	// pending track and vice versa.
	// Java: midi = "none" (SignLink.java:45); "play" stands in for the
	// jingle<pos>.mid path of the disk protocol (SignLink.java:179-182),
	// which the Go port replaces with in-memory bytes.
	Midi string
	// MidiData holds the pending track bytes when Midi == "play".
	MidiData []byte
```

```go
	MidiVol int = 96 // Java: midivol = 96 (SignLink.java:59)
```

```go
	WaveVol int = 96 // Java: wavevol = 96 (SignLink.java:71)
```

Replace the doc comments + bodies of `ConsumeMidi`/`SetMidiCommand` and add the new accessors. Keep `ConsumeMidi` for now (the js watcher still uses it until Task 4):

```go
// PeekMidi returns the pending command slot without clearing it. The
// consumer (audio.runAudioLoop) clears via ClearMidi only when not fading
// out, porting the latch `if (!midiFadingOut) midi = "none"`
// (SignLink.java:422-424).
func PeekMidi() (string, []byte) {
	mu.Lock()
	defer mu.Unlock()
	return Midi, MidiData
}

// ClearMidi empties the command slot (Java: midi = "none").
func ClearMidi() {
	mu.Lock()
	defer mu.Unlock()
	Midi = ""
	MidiData = nil
}

// SetMidiTrack publishes track bytes for the audio consumer. Single slot,
// latest-wins: it clobbers any pending command, exactly like Java's lone
// `midi` field. Java: midisave → run loop → midi = cachedir + savereq
// (SignLink.java:179-182, 327-337); the Go port hands the bytes over
// in-memory instead of via jingle<pos>.mid.
func SetMidiTrack(data []byte) {
	mu.Lock()
	defer mu.Unlock()
	Midi = "play"
	MidiData = data
}

// SetMidiCommand publishes the "stop" or "voladjust" sentinel. Clobbers a
// pending track (single slot, see SetMidiTrack).
func SetMidiCommand(s string) {
	mu.Lock()
	defer mu.Unlock()
	Midi = s
	MidiData = nil
}

// ConsumeMidi atomically reads and clears the command slot.
//
// Deprecated: superseded by PeekMidi/ClearMidi (the faithful consumer needs
// the fade-out latch). Still used by the js watcher until it moves onto
// runAudioLoop; remove with it.
func ConsumeMidi() string {
	mu.Lock()
	defer mu.Unlock()
	s := Midi
	Midi = ""
	MidiData = nil
	return s
}
```

- [ ] **Step 1.4: Run the signlink tests**

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/client/sign/signlink/
```
Expected: PASS (the existing `TestConsumeMidiClearsAndReturns` still passes; it migrates in Task 4). If `TestMidiFadeAndVolRoundTrip`'s cleanup resets `MidiVol = 0` (line ~84), change that reset to `MidiVol = 96` to restore the new default.

- [ ] **Step 1.5: Run full gates, then commit**

```bash
git add pkg/jagex2/client/sign/signlink/
git commit --no-gpg-sign -m "feat(rev-244): signlink 244 audio defaults + single-slot midi track protocol (WS5)"
```

---

### Task 2: audio — `linearVolume` + the faithful audioLoop state machine

**Files:**
- Modify: `pkg/jagex2/sound/audio/format.go`
- Create: `pkg/jagex2/sound/audio/audioloop.go`
- Create: `pkg/jagex2/sound/audio/audioloop_test.go`

- [ ] **Step 2.1: Write the failing tests**

Create `pkg/jagex2/sound/audio/audioloop_test.go`:

```go
package audio

import (
	"slices"
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/sign/signlink"
)

// fakeSink records midiSink calls so tests can assert the exact sequence the
// audioLoop state machine produces.
type playCall struct {
	data []byte
	loop bool
	vol  int
}

type fakeSink struct {
	playCalls []playCall
	stopCalls int
	volCalls  []int
	runningV  bool
}

func (f *fakeSink) play(d []byte, loop bool, vol int) {
	f.playCalls = append(f.playCalls, playCall{data: d, loop: loop, vol: vol})
	f.runningV = true
}
func (f *fakeSink) stop()          { f.stopCalls++; f.runningV = false }
func (f *fakeSink) setVolume(v int) { f.volCalls = append(f.volCalls, v) }
func (f *fakeSink) running() bool  { return f.runningV }

// resetSignlinkAudio restores the signlink audio fields the loop reads to
// their 244 defaults, before and after each test.
func resetSignlinkAudio(t *testing.T) {
	t.Helper()
	reset := func() {
		signlink.ClearMidi()
		signlink.SetMidiFade(0)
		signlink.SetMidiVol(96)
	}
	reset()
	t.Cleanup(reset)
}

// TestTrackChangeFadesOutThenIn pins the full Java fade cycle for a track
// change with midifade=1 while music is playing (SignLink.java:369-411):
// fade-out trigger (track latched), 12 ticks of -8 steps to 0, the new track
// starting at volume 0 and looping, then 12 ticks of +8 steps back to 96.
func TestTrackChangeFadesOutThenIn(t *testing.T) {
	resetSignlinkAudio(t)
	signlink.SetMidiFade(1)
	sink := &fakeSink{runningV: true}
	l := &audioLoop{sink: sink}

	signlink.SetMidiTrack([]byte{0xB})
	l.tick() // playMidi: running + fade → start fade-out; track stays latched
	if len(sink.playCalls) != 0 || !l.midiFadingOut {
		t.Fatalf("tick 1: want fade-out latched, got plays=%d fadingOut=%v",
			len(sink.playCalls), l.midiFadingOut)
	}
	if cmd, _ := signlink.PeekMidi(); cmd != "play" {
		t.Fatalf("track must stay latched during fade-out, got %q", cmd)
	}

	for range 12 { // 96/8 = 12 steps down; the 12th flips to fade-in + plays
		l.tick()
	}
	wantDown := []int{88, 80, 72, 64, 56, 48, 40, 32, 24, 16, 8, 0}
	if !slices.Equal(sink.volCalls, wantDown) {
		t.Fatalf("fade-out steps: got %v, want %v", sink.volCalls, wantDown)
	}
	if len(sink.playCalls) != 1 || sink.playCalls[0].vol != 0 || !sink.playCalls[0].loop {
		t.Fatalf("new track must start at vol 0 looping: %+v", sink.playCalls)
	}
	if cmd, _ := signlink.PeekMidi(); cmd != "" {
		t.Fatalf("slot must clear once the fade-out ends, got %q", cmd)
	}

	sink.volCalls = nil
	for range 12 {
		l.tick()
	}
	wantUp := []int{8, 16, 24, 32, 40, 48, 56, 64, 72, 80, 88, 96}
	if !slices.Equal(sink.volCalls, wantUp) {
		t.Fatalf("fade-in steps: got %v, want %v", sink.volCalls, wantUp)
	}
	if l.midiFadingIn || l.midiFadingOut {
		t.Fatalf("fade flags must settle: in=%v out=%v", l.midiFadingIn, l.midiFadingOut)
	}
}

// TestFirstPlayNoFadeStartsAtFullVolume: with nothing running, even a
// midifade=1 track starts immediately at full midivol (the fade only kicks
// in on a CHANGE — Java: the `midiPlayer.running()` condition).
func TestFirstPlayNoFadeStartsAtFullVolume(t *testing.T) {
	resetSignlinkAudio(t)
	signlink.SetMidiFade(1)
	sink := &fakeSink{runningV: false}
	l := &audioLoop{sink: sink}

	signlink.SetMidiTrack([]byte{0xA})
	l.tick()
	if len(sink.playCalls) != 1 {
		t.Fatalf("want immediate play, got %d calls", len(sink.playCalls))
	}
	got := sink.playCalls[0]
	if got.vol != 96 || !got.loop {
		t.Fatalf("first play: vol=%d loop=%v, want vol=96 loop=true", got.vol, got.loop)
	}
	if cmd, _ := signlink.PeekMidi(); cmd != "" {
		t.Fatalf("slot must clear, got %q", cmd)
	}
}

// TestJingleReplacesImmediately: midifade=0 (jingles) replaces the playing
// track at once, non-looping, full volume — no fade at all.
func TestJingleReplacesImmediately(t *testing.T) {
	resetSignlinkAudio(t)
	sink := &fakeSink{runningV: true}
	l := &audioLoop{sink: sink}

	signlink.SetMidiTrack([]byte{0xC})
	l.tick()
	if len(sink.playCalls) != 1 || sink.playCalls[0].loop || sink.playCalls[0].vol != 96 {
		t.Fatalf("jingle: %+v, want one immediate non-loop play at 96", sink.playCalls)
	}
}

// TestStopAndVoladjust: both sentinels dispatch on the next tick and clear
// when not fading out. Stop is immediate (stopMidi zeroes midifade first).
func TestStopAndVoladjust(t *testing.T) {
	resetSignlinkAudio(t)
	sink := &fakeSink{}
	l := &audioLoop{sink: sink}

	signlink.SetMidiCommand("stop")
	l.tick()
	if sink.stopCalls != 1 {
		t.Fatalf("stop: got %d calls, want 1", sink.stopCalls)
	}
	if cmd, _ := signlink.PeekMidi(); cmd != "" {
		t.Fatalf("stop must clear when not fading, got %q", cmd)
	}

	signlink.SetMidiVol(64)
	signlink.SetMidiCommand("voladjust")
	l.tick()
	if !slices.Equal(sink.volCalls, []int{64}) {
		t.Fatalf("voladjust: got %v, want [64]", sink.volCalls)
	}
}

// TestCommandClobbersLatchedTrack pins two coupled Java behaviors: (1) the
// single slot — a stop issued during a fade-out replaces the latched track,
// which then never plays; (2) the latch — the stop re-dispatches every tick
// until the fade-out completes, and only then clears.
func TestCommandClobbersLatchedTrack(t *testing.T) {
	resetSignlinkAudio(t)
	signlink.SetMidiFade(1)
	sink := &fakeSink{runningV: true}
	l := &audioLoop{sink: sink}

	signlink.SetMidiTrack([]byte{0xB})
	l.tick() // latch track, enter fade-out

	signlink.SetMidiFade(0) // Java stopMidi: midifade = 0, then midi = "stop"
	signlink.SetMidiCommand("stop")
	l.tick()
	l.tick()
	if sink.stopCalls < 2 {
		t.Fatalf("latched stop must re-run during fade-out: got %d", sink.stopCalls)
	}
	if cmd, _ := signlink.PeekMidi(); cmd != "stop" {
		t.Fatalf("stop must stay latched while fading out, got %q", cmd)
	}
	if len(sink.playCalls) != 0 {
		t.Fatalf("clobbered track must never play, got %+v", sink.playCalls)
	}

	for range 12 { // drain the rest of the fade-out
		l.tick()
	}
	if cmd, _ := signlink.PeekMidi(); cmd != "" {
		t.Fatalf("slot must clear after the fade-out completes, got %q", cmd)
	}
}

// TestLinearVolume pins the vol/256 mapping (see linearVolume's doc for the
// equivalence proof against Java's CC rescale).
func TestLinearVolume(t *testing.T) {
	cases := []struct {
		vol  int
		want float64
	}{
		{0, 0}, {-8, 0}, {32, 0.125}, {64, 0.25}, {96, 0.375},
		{128, 0.5}, {256, 1}, {300, 1},
	}
	for _, c := range cases {
		if got := linearVolume(c.vol); got != c.want {
			t.Errorf("linearVolume(%d) = %v, want %v", c.vol, got, c.want)
		}
	}
}
```

- [ ] **Step 2.2: Run to verify failure**

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/sound/audio/ -run 'TestTrackChange|TestFirstPlay|TestJingle|TestStopAnd|TestCommandClobbers|TestLinearVolume'
```
Expected: FAIL (compile: `audioLoop`, `linearVolume` undefined).

- [ ] **Step 2.3: Add `linearVolume` to `format.go`** (keep `volumeFromCentibels` for now — wave files still use it until Task 5):

```go
// linearVolume maps the 244 linear volume scale to an amplitude gain vol/256.
// The client sends 128/96/64/32 (Client.java:11372-11414); SignLink defaults
// midivol/wavevol to 96 (SignLink.java:59,71).
//
// Faithfulness proof: Java's MidiPlayer rescales each channel's 14-bit
// CC7/CC39 volume by sqrt(vol/256) before the synth sees it (getVolume:
// sqrt(((cc*vol)>>>8)*cc), MidiPlayer.java:123-126, applied to the file's
// own CC messages via check(), :134-160). meltysynth squares channel volume
// per the GM spec (voice.go:195-197: channelGain = ve*ve), so the audible
// composition is cc²·(vol/256) — meltysynth's native rendering times a
// linear vol/256 post-gain. Applying that gain at the player/gain node
// reproduces Java's volume curve exactly, and MidiPlayer's CC interception
// machinery does not need porting. Deviation: meltysynth handles the file's
// own CC121 per MIDI RP-015 (channel volume NOT reset), where Java's
// wrapper reset its tracked volume to the 12800 default.
func linearVolume(vol int) float64 {
	if vol <= 0 {
		return 0
	}
	if vol >= 256 {
		return 1
	}
	return float64(vol) / 256
}
```

- [ ] **Step 2.4: Create `pkg/jagex2/sound/audio/audioloop.go`**

```go
package audio

import (
	"time"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/sign/signlink"
)

// This file ports the wrapper-side MIDI consumer that the LostCityRS 244
// deob reconstructs in SignLink (SignLink.java:359-425, "adapted from
// play_members.html's JS loop") — the half of the audio pipeline that ran
// outside the signed Java applet and was never in the 225 sources.
//
// Deviations from the Java reference (everything audible is faithful):
//   - Track bytes travel in-memory through the signlink slot
//     (signlink.SetMidiTrack) instead of the jingle<pos>.mid disk round-trip
//     (SignLink.java:327-337) — an applet process-boundary artifact.
//   - The wave branch of audioLoop (SignLink.java:427-475) is not ported
//     into this ticker: SFX play directly (and may overlap) via PlayWave,
//     instead of synchronously blocking the consumer thread for the clip's
//     duration — in Java a playing SFX stalls any in-flight MIDI fade.
//   - MidiPlayer's CC7/CC39/CC121 interception collapses to the linear
//     vol/256 post-gain — see linearVolume.

// midiSink is the per-backend playback primitive set the audioLoop drives,
// mirroring the MidiPlayer surface SignLink uses. Implemented by *midiDriver
// (native: oto + meltysynth) and *webMidiDriver (js: Web Audio prerender);
// nullSink stands in when no audio backend is available.
type midiSink interface {
	// play starts a track at linear gain vol/256, replacing any current
	// one. loop mirrors Java MidiPlayer.play's setLoopCount(loop==1 ? -1 : 0)
	// (MidiPlayer.java:36-44): the fade flag doubles as loop-forever.
	play(midData []byte, loop bool, vol int)
	// stop halts playback immediately (Java: MidiPlayer.stop(),
	// MidiPlayer.java:46-49).
	stop()
	// setVolume applies linear gain vol/256 to the current track
	// (Java: MidiPlayer.setVolume(0, volume), MidiPlayer.java:32-34).
	setVolume(vol int)
	// running reports whether a track is still playing
	// (Java: Sequencer.isRunning() via MidiPlayer.running()).
	running() bool
}

// nullSink drains commands when no backend is available (oto init failure,
// low-memory mode, missing Web Audio). Contrast with Java, where a command
// arriving after a failed MidiPlayer constructor would NPE the wrapper
// thread (SignLink.run has no try/catch around audioLoop) — draining
// harmlessly is strictly better.
type nullSink struct{}

func (nullSink) play([]byte, bool, int) {}
func (nullSink) stop()                  {}
func (nullSink) setVolume(int)          {}
func (nullSink) running() bool          { return false }

// audioLoopInterval is the consumer cadence AND the fade step rate: the Java
// wrapper thread sleeps 50ms per iteration (SignLink.java:197-200), so the
// ±8 fade steps tick every 50ms — a 600ms linear ramp at the default
// midivol 96.
const audioLoopInterval = 50 * time.Millisecond

// audioLoop holds the MIDI fade state machine. Java: SignLink's instance
// fields midiFadingIn/midiFadingOut/midiFadeVol (SignLink.java:360-362).
// Single-goroutine (the runAudioLoop ticker); the signlink slot and the
// sink do their own locking.
type audioLoop struct {
	sink midiSink

	midiFadingIn  bool // Java: midiFadingIn (SignLink.java:360)
	midiFadingOut bool // Java: midiFadingOut (SignLink.java:361)
	midiFadeVol   int  // Java: midiFadeVol (SignLink.java:362)
}

// runAudioLoop ticks the consumer for the process lifetime.
// Java: the tail of SignLink.run's while-loop — audioLoop();
// Thread.sleep(50L) (SignLink.java:195-200).
func runAudioLoop(sink midiSink) {
	l := &audioLoop{sink: sink}
	for {
		l.tick()
		time.Sleep(audioLoopInterval)
	}
}

// tick is one consumer iteration: fade step first, then command dispatch.
// Java: SignLink.audioLoop() (SignLink.java:391-425; the wave branch at
// :427-475 is handled by PlayWave — see the file comment).
func (l *audioLoop) tick() {
	midivol := signlink.ReadMidiVol()

	// Fade step — Java: SignLink.java:392-411.
	if l.midiFadingIn {
		l.midiFadeVol += 8
		if l.midiFadeVol > midivol {
			l.midiFadeVol = midivol
		}
		l.sink.setVolume(l.midiFadeVol)
		if l.midiFadeVol == midivol {
			l.midiFadingIn = false
		}
	} else if l.midiFadingOut {
		l.midiFadeVol -= 8
		if l.midiFadeVol < 0 {
			l.midiFadeVol = 0
		}
		l.sink.setVolume(l.midiFadeVol)
		if l.midiFadeVol == 0 {
			l.midiFadingOut = false
			l.midiFadingIn = true
		}
	}

	// Command dispatch — Java: SignLink.java:413-425. The slot clears only
	// when not fading out: a command latched mid-fade re-dispatches every
	// tick until the fade-out completes (for a track, playMidi then starts
	// it; for stop/voladjust the repeats are idempotent — a faithful quirk).
	cmd, data := signlink.PeekMidi()
	if cmd == "" {
		return
	}
	switch cmd {
	case "stop":
		l.sink.stop()
	case "voladjust":
		l.sink.setVolume(midivol)
	default: // "play"
		l.playMidi(data)
	}
	if !l.midiFadingOut {
		signlink.ClearMidi()
	}
}

// playMidi decides how a pending track starts: ignored (stays latched)
// during fade-out; triggers a fade-out first when fading is on and a track
// is already playing; starts at volume 0 (then ramps in) when arriving out
// of a fade-out; starts at full midivol otherwise — first plays and jingles
// skip the fade entirely.
// Java: SignLink.playMidi (SignLink.java:369-388). The Java catch around
// the play maps to the sink logging-and-returning on a parse failure.
func (l *audioLoop) playMidi(data []byte) {
	midifade := signlink.ReadMidiFade()
	midivol := signlink.ReadMidiVol()
	if l.midiFadingOut {
		return
	}
	if !l.midiFadingIn && midifade != 0 && l.sink.running() {
		l.midiFadingOut = true
		l.midiFadeVol = midivol
		return
	}
	if midifade != 0 && l.midiFadingIn {
		l.midiFadingOut = false
		l.midiFadeVol = 0
		l.sink.play(data, midifade == 1, l.midiFadeVol)
	} else {
		l.sink.play(data, midifade == 1, midivol)
	}
}
```

- [ ] **Step 2.5: Run the new tests**

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/sound/audio/
```
Expected: PASS (new tests green; existing tests untouched).

- [ ] **Step 2.6: Full gates, then commit**

```bash
git add pkg/jagex2/sound/audio/format.go pkg/jagex2/sound/audio/audioloop.go pkg/jagex2/sound/audio/audioloop_test.go
git commit --no-gpg-sign -m "feat(rev-244): port the SignLink audioLoop fade state machine (WS5)"
```

---

### Task 3: native sink rewrite + client SaveMidi reroute

**Files:**
- Rewrite: `pkg/jagex2/sound/audio/midi_native.go`
- Modify: `pkg/jagex2/sound/audio/audio_native.go`
- Modify: `pkg/jagex2/sound/audio/midi_test.go`
- Modify: `pkg/jagex2/client/client.go:3418-3441` (SaveMidi), `client.go:9062` (call site)
- Modify: `cmd/client/main.go:110-130` (comment)

- [ ] **Step 3.1: Rewrite `midi_native.go`** with this full content:

```go
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
```

Deleted relative to the old file (verify none remain): `gainSmoothingAlpha`, `midiPollInterval`, `runMidiWatcher`, `driverMu`/`driver`/`driverReady`, `registerMidiDriver`, `getMidiDriver`, `PlayMIDI`, `handle`, `playFromBytes`, `fadeAndSwap`, `setUserVolume`, `fadeDuration`, gain fields/methods on `midiSource` (`currentGain`/`targetGain`/`gainMu`/`setGainTarget`/`snapGain`/`gain()`), the `gen` field.

- [ ] **Step 3.2: Update `audio_native.go`** — replace `Start` and `DisableForLowMemory` (keep the package doc paragraph about format unification; refresh the consumer description; keep `otoMu`/`otoCtx`/`readyCtx`/`ensureContext` unchanged):

```go
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
```

Remove the now-unused `"sync/atomic"`/`"time"` imports from `audio_native.go` **only if** unused after the edit (`readyCtx` still needs `sync/atomic`; `oto.NewContextOptions` still needs `time`). Let the compiler arbitrate.

- [ ] **Step 3.3: Update `client.go` SaveMidi (lines ~3418-3441) and its call site (line ~9062)**

Replace the whole `SaveMidi` method with:

```go
// SaveMidi hands a downloaded MIDI track to the audio consumer.
// Java: saveMidi(boolean fade, byte[] src) (Client.java:1447-1450) —
// SignLink.midifade = fade ? 1 : 0; SignLink.midisave(src, src.length).
// The Go port publishes the bytes in-memory through the signlink slot
// instead of midisave's jingle<pos>.mid disk round-trip (an applet
// process-boundary artifact); the consumer (audio.runAudioLoop) picks the
// slot up on its next 50ms tick, exactly like the Java wrapper thread did
// with the path. The fade flag must be published BEFORE the track so the
// consumer's playMidi reads the matching value (same ordering as Java).
func (c *Client) SaveMidi(fade bool, src []byte) {
	if fade {
		signlink.SetMidiFade(1)
	} else {
		signlink.SetMidiFade(0)
	}
	signlink.SetMidiTrack(src)
}
```

Call site at `client.go:9062` (inside `UpdateOnDemand`'s `req.Archive == 2` case):

```go
		case req.Archive == 2 && c.MidiSong == req.File && req.Data != nil:
			// Java: this.saveMidi(this.midiFading, req.data) (Client.java:2445).
			c.SaveMidi(c.MidiFading, req.Data)
```

If `client.go` no longer references the `audio` package after this… it still does (`SaveWave`/`ReplayWave` use `audio.PlayWave`/`audio.ReplayWave`) — leave the import.

- [ ] **Step 3.4: Update `midi_test.go`**

Delete `TestMidiSourceSnapGainAppliesImmediately` and `TestMidiSourceSetGainTargetIsSmoothed` (the smoother is gone). Update the file's header comment (it references the fade-overlap bug pinned by gain tests — reword to note the sequencing now lives in audioloop.go and these tests cover the source's silence/clip behavior). Update remaining `newMidiSource(nil, 1.0)`-style calls to `newMidiSource(nil)` and remove assertions on `s.gain()`. The surviving silence test should look like:

```go
func TestMidiSourceNilSeqEmitsSilence(t *testing.T) {
	s := newMidiSource(nil)
	buf := make([]byte, 64)
	n, err := s.Read(buf)
	if err != nil || n != 64 {
		t.Fatalf("Read: n=%d err=%v, want 64 <nil>", n, err)
	}
	for i, b := range buf {
		if b != 0 {
			t.Fatalf("byte %d = %#x, want silence", i, b)
		}
	}
}
```

`TestMidiSourceSwapToNilSilencesActiveStream` becomes a swap test without gain:

```go
func TestMidiSourceSwapToNilSilencesActiveStream(t *testing.T) {
	s := newMidiSource(nil)
	s.swap(nil) // stop(): the source must render silence from the next Read
	buf := make([]byte, 32)
	if _, err := s.Read(buf); err != nil {
		t.Fatal(err)
	}
	for i, b := range buf {
		if b != 0 {
			t.Fatalf("byte %d = %#x, want silence after swap(nil)", i, b)
		}
	}
}
```

Keep `TestMidiSourceShortBufferReturnsZero`, `TestClipInt16Saturates`, `TestVolumeFromCentibels` (dies in Task 5), and the wave-conversion tests as-is (fix compile errors only).

- [ ] **Step 3.5: Update the `cmd/client/main.go` comment** (lines ~110-118). Replace the stale watcher/ConsumeMidi sentence so the block reads:

```go
	wg.Go(func() {
		// audio.Start brings up the oto context and spawns the shared
		// audioLoop ticker (the faithful SignLink consumer) for the
		// lifetime of the process; SFX play synchronously via
		// audio.PlayWave (no watcher). Started after signlink so the
		// soundfont fetch (via signlink.OpenURL) doesn't race the
		// protocol coming up.
		//
		// In low-memory mode we bring up no audio at all, matching the
		// Java client: it never starts the MIDI thread, never unpacks
		// sounds.dat, and gates every playback path behind !lowMemory
		// (deob/client.java:5949/6163/7374/...). Initializing oto there
		// would open an audio device for a queue nothing ever fills.
		// client.LowMemory is set synchronously by SetLowMem above,
		// well before this goroutine reads it.
		if client.LowMemory {
			audio.DisableForLowMemory()
			return
		}
		audio.Start()
	})
```

- [ ] **Step 3.6: Run full gates** (native + js build + tests + vet + gofmt). The js build must still pass: `midi_js.go` still defines its own `PlayMIDI`/watcher and `ConsumeMidi` still exists.

- [ ] **Step 3.7: Commit**

```bash
git add pkg/jagex2/sound/audio/ pkg/jagex2/client/client.go cmd/client/main.go
git commit --no-gpg-sign -m "feat(rev-244): drive native MIDI through the faithful audioLoop (WS5)"
```

---

### Task 4: wasm sink rewrite + retire ConsumeMidi

**Files:**
- Rewrite: `pkg/jagex2/sound/audio/midi_js.go`
- Modify: `pkg/jagex2/sound/audio/audio_js.go`
- Modify: `pkg/jagex2/client/sign/signlink/signlink.go` (delete `ConsumeMidi`)
- Modify: `pkg/jagex2/client/sign/signlink/signlink_audio_helpers_test.go`

- [ ] **Step 4.1: Rewrite `midi_js.go`** with this full content:

```go
//go:build js

package audio

import (
	"bytes"
	"log"
	"sync"
	"sync/atomic"
	"syscall/js"
	"time"

	"github.com/sinshu/go-meltysynth/meltysynth"
)

// webMidiDriver is the js midiSink: tracks are pre-rendered to AudioBuffer
// chunks (static buffers are immune to main-thread synthesis bursts) and
// scheduled onto the Web Audio clock.
//
// The shared audioLoop (audioloop.go) owns all fade/latch sequencing and
// drives the four midiSink primitives from its 50ms ticker; volume —
// including the stepped ±8 fade — lands on the single musicGain node as the
// linear vol/256 gain.
//
// Looping (Java: setLoopCount(-1) for MIDI_SONG tracks, MidiPlayer.java:39)
// re-schedules the rendered chunks one musical length apart, staying
// loopLookahead ahead of the context clock (keepLooping). The 1s release
// tail rendered after the musical end overlaps the next iteration's start —
// a sequencer loop cuts releases at the loop point; the overlap is the
// closest prerender equivalent.

// renderChunkFrames is how many frames are synthesized per scheduled chunk
// (~250ms of audio). Each chunk becomes its own AudioBufferSourceNode
// started at a precise time, so playback can begin after the FIRST chunk
// renders instead of waiting for the whole track; the render then races
// ahead of playback so the rest is scheduled before it's needed. A chunk is
// also one frame's worth of synthesis CPU, keeping the per-chunk yield
// (see streamRender) responsive.
const renderChunkFrames = SampleRate / 4

// loopLookahead is how far ahead of the AudioContext clock the next loop
// iteration is scheduled: comfortably above keepLooping's poll period so
// the schedule never starves, small enough that little stale audio is ever
// queued (stopAll cancels scheduled sources anyway).
const loopLookahead = 5 * time.Second

// scheduledSrc pairs a started AudioBufferSourceNode with the AudioContext
// time it ends, so long loops can prune finished nodes.
type scheduledSrc struct {
	node js.Value
	end  float64
}

// musicStream is one track's playback: every source node scheduled for it.
// Stopping it stops them all, including ones scheduled in the future.
type musicStream struct {
	mu      sync.Mutex
	sources []scheduledSrc
	stopped bool
}

// schedule starts buf at AudioContext time `at` on this stream. If the
// stream was already stopped (a newer command superseded it between
// chunks), the source is stopped immediately so it neither plays nor leaks.
func (s *musicStream) schedule(buf js.Value, at float64) {
	src := ac.Call("createBufferSource")
	src.Set("buffer", buf)
	src.Set("loop", false)
	src.Call("connect", musicGain)
	src.Call("start", at)
	end := at + float64(buf.Get("length").Int())/float64(SampleRate)
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		safeStop(src)
		src.Call("disconnect")
		return
	}
	s.sources = append(s.sources, scheduledSrc{node: src, end: end})
	s.mu.Unlock()
}

// pruneEnded drops references to sources that finished before `now`,
// keeping the slice bounded during indefinite loops. Ended nodes are inert
// — the audio graph has already released them — so dropping the reference
// is safe; only still-playing/future nodes must stay reachable by stopAll.
func (s *musicStream) pruneEnded(now float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.sources[:0]
	for _, e := range s.sources {
		if e.end > now {
			kept = append(kept, e)
		}
	}
	clear(s.sources[len(kept):])
	s.sources = kept
}

// stopAll stops + disconnects every scheduled source. Idempotent (both a
// superseding command and a superseded render goroutine may call it).
func (s *musicStream) stopAll() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	srcs := s.sources
	s.sources = nil
	s.mu.Unlock()
	for _, e := range srcs {
		safeStop(e.node)
		e.node.Call("disconnect")
	}
}

// safeStop calls stop() on a source, swallowing the InvalidStateError some
// browsers throw when the source has already ended naturally. A thrown JS
// exception would otherwise panic the Go side via syscall/js.
func safeStop(src js.Value) {
	defer func() { _ = recover() }()
	src.Call("stop")
}

type webMidiDriver struct {
	soundFont *meltysynth.SoundFont
	loadOnce  sync.Once

	mu         sync.Mutex
	cur        *musicStream // current track's stream (nil before first play / after stop)
	looping    bool
	musicalEnd float64 // AudioContext time when a non-loop track's musical length elapses

	// One-track render cache: a jingle→track resume re-schedules instantly
	// instead of re-synthesizing. AudioBuffers are immutable and may back
	// any number of source nodes, so the cache holds the same buffers the
	// first play used — no second copy of the PCM.
	cacheKey     string
	cacheChunks  []js.Value
	cacheSeconds float64 // musical length (sans tail) of the cached track

	// gen is bumped on every play/stop. In-flight render and loop
	// goroutines snapshot it and abandon when superseded.
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

// play (midiSink) replaces the current track immediately — Java:
// sequencer.setSequence (MidiPlayer.java:38); any audible crossfade is the
// audioLoop's stepped setVolume, which has already run a full fade-out
// before this is called on a fading change. Rendering runs on a background
// goroutine so the 50ms tick isn't blocked by synthesis.
func (d *webMidiDriver) play(midData []byte, loop bool, vol int) {
	d.setVolume(vol) // Java: setVolume before sequencer.start() (MidiPlayer.java:40-41)
	gen := d.gen.Add(1)
	d.mu.Lock()
	old := d.cur
	d.cur = nil
	d.mu.Unlock()
	if old != nil {
		old.stopAll()
	}
	go d.startTrack(midData, loop, gen)
}

// setVolume (midiSink): linear vol/256 on the shared music gain node.
// Java: MidiPlayer.setVolume(0, volume) — see linearVolume for the proof
// that the CC rescale collapses to this.
func (d *webMidiDriver) setVolume(vol int) {
	if !musicGain.Truthy() {
		return
	}
	musicGain.Get("gain").Set("value", linearVolume(vol))
}

// stop (midiSink): Java MidiPlayer.stop() (MidiPlayer.java:46-49). The gen
// bump makes in-flight render/loop goroutines abandon.
func (d *webMidiDriver) stop() {
	d.gen.Add(1)
	d.mu.Lock()
	s := d.cur
	d.cur = nil
	d.looping = false
	d.musicalEnd = 0
	d.mu.Unlock()
	if s != nil {
		s.stopAll()
	}
}

// running (midiSink): Java Sequencer.isRunning() — a looping track never
// ends; a one-shot ends when its musical length elapses on the context
// clock.
func (d *webMidiDriver) running() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cur == nil {
		return false
	}
	return d.looping || ac.Get("currentTime").Float() < d.musicalEnd
}

// startTrack makes a new stream current and feeds it: cached chunks for a
// repeat of the cached track, or a fresh streaming render.
func (d *webMidiDriver) startTrack(midData []byte, loop bool, gen uint64) {
	now := ac.Get("currentTime").Float()
	s := &musicStream{}

	d.mu.Lock()
	if d.gen.Load() != gen { // superseded before we became current
		d.mu.Unlock()
		return
	}
	d.cur = s
	d.looping = loop
	cacheHit := d.cacheKey == string(midData) && len(d.cacheChunks) > 0
	var cached []js.Value
	var seconds float64
	if cacheHit {
		cached, seconds = d.cacheChunks, d.cacheSeconds
		d.musicalEnd = now + seconds
	}
	d.mu.Unlock()

	if cacheHit {
		d.replayChunks(s, cached, now, gen)
		if loop {
			d.keepLooping(s, cached, now+seconds, seconds, gen)
		}
		return
	}
	d.streamRender(s, midData, now, loop, gen)
}

// replayChunks schedules already-rendered chunks gapless from startAt,
// advancing by each chunk's own frame count. Yields between chunks
// (single-threaded wasm) so the game loop keeps drawing; abandons if
// superseded.
func (d *webMidiDriver) replayChunks(s *musicStream, chunks []js.Value, startAt float64, gen uint64) {
	at := startAt
	for _, buf := range chunks {
		if d.gen.Load() != gen {
			return // superseded: the new command stops this stream
		}
		s.schedule(buf, at)
		at += float64(buf.Get("length").Int()) / float64(SampleRate)
		time.Sleep(time.Millisecond)
	}
}

// streamRender synthesizes the track chunk by chunk, scheduling each to
// play seamlessly as it is produced (so playback starts after chunk 0), and
// caches the chunk buffers for instant replay. Renders through a small
// reusable scratch pair so the Go heap peak stays flat regardless of track
// length; the rendered PCM lives only in the JS AudioBuffers (Go wasm
// linear memory never shrinks, so a full-track []float32 would permanently
// inflate it). Yields between chunks so the game loop keeps drawing.
// Abandons if superseded. When loop is set, hands off to keepLooping after
// the full render.
func (d *webMidiDriver) streamRender(s *musicStream, midData []byte, startAt float64, loop bool, gen uint64) {
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
	// One rendering pass; looping is keepLooping's chunk re-scheduling
	// (Java setLoopCount(-1) has no prerender equivalent).
	seq.Play(midiFile, false)

	seconds := midiFile.GetLength().Seconds()
	d.mu.Lock()
	d.musicalEnd = startAt + seconds
	d.mu.Unlock()

	frames := renderFrameCount(midiFile.GetLength())
	// One reusable chunk-sized scratch pair, not two full-track slices.
	// Safe to reuse because f32ToJSFloat32Array copies the bytes into the
	// JS buffer before the next Render overwrites the scratch.
	left := make([]float32, renderChunkFrames)
	right := make([]float32, renderChunkFrames)
	chunks := make([]js.Value, 0, (frames+renderChunkFrames-1)/renderChunkFrames)
	at := startAt
	for off := 0; off < frames; off += renderChunkFrames {
		if d.gen.Load() != gen {
			return
		}
		n := renderChunkFrames
		if off+n > frames {
			n = frames - off
		}
		ls, rs := left[:n], right[:n]
		seq.Render(ls, rs)
		buf := ac.Call("createBuffer", ChannelCount, n, SampleRate)
		buf.Call("copyToChannel", f32ToJSFloat32Array(ls), 0)
		buf.Call("copyToChannel", f32ToJSFloat32Array(rs), 1)
		s.schedule(buf, at)
		chunks = append(chunks, buf)
		at += float64(n) / float64(SampleRate)
		time.Sleep(time.Millisecond)
	}

	if d.gen.Load() != gen {
		return
	}
	d.mu.Lock()
	d.cacheKey = string(midData)
	d.cacheChunks = chunks
	d.cacheSeconds = seconds
	d.mu.Unlock()

	if loop {
		d.keepLooping(s, chunks, startAt+seconds, seconds, gen)
	}
}

// keepLooping schedules iteration after iteration of the rendered chunks,
// each one musical length after the previous, staying loopLookahead ahead
// of the context clock. Runs until superseded — a stop or track change
// bumps gen AND stopAlls the stream, cancelling anything pre-scheduled.
// Java: Sequencer.setLoopCount(-1) (MidiPlayer.java:39).
func (d *webMidiDriver) keepLooping(s *musicStream, chunks []js.Value, nextAt, seconds float64, gen uint64) {
	if seconds <= 0 {
		return // degenerate zero-length track: nothing meaningful to loop
	}
	for d.gen.Load() == gen {
		now := ac.Get("currentTime").Float()
		if nextAt-now < loopLookahead.Seconds() {
			d.replayChunks(s, chunks, nextAt, gen)
			nextAt += seconds
			s.pruneEnded(now)
			continue
		}
		time.Sleep(time.Second)
	}
}
```

- [ ] **Step 4.2: Update `audio_js.go`** — replace `Start`, `DisableForLowMemory`, delete `runMidiWatcher`, `midiPollInterval`, and the driver registry (`driverMu`/`driver`/`driverReady`/`registerMidiDriver`/`getMidiDriver`); drop the now-unused `signlink` and `time`/`sync` imports if nothing else uses them:

```go
// Start brings up the Web Audio context and spawns the shared audioLoop
// (the faithful SignLink consumer, audioloop.go) ticking the web MIDI sink.
// Called once from cmd/client/main.go on a goroutine.
func Start() {
	ctor := js.Global().Get("AudioContext")
	if !ctor.Truthy() {
		ctor = js.Global().Get("webkitAudioContext")
	}
	if !ctor.Truthy() {
		log.Printf("audio: Web Audio unavailable, game will run silently")
		go runAudioLoop(nullSink{})
		return
	}
	ac = ctor.New(map[string]any{"sampleRate": SampleRate})

	musicGain = ac.Call("createGain")
	musicGain.Call("connect", ac.Get("destination"))
	sfxGain = ac.Call("createGain")
	sfxGain.Call("connect", ac.Get("destination"))

	installGestureResume()

	go runAudioLoop(newWebMidiDriver())
}

// DisableForLowMemory matches the native low-memory path: no context, no
// soundfont; a nullSink loop drains the signlink slot (logout's stopMidi is
// not lowmem-gated).
func DisableForLowMemory() {
	go runAudioLoop(nullSink{})
}
```

Also update the `musicGain` var comment: `// GainNode: MIDI volume — the audioLoop's stepped vol/256 lands here`.

- [ ] **Step 4.3: Delete `ConsumeMidi` from `signlink.go`** (no consumers remain) and update `signlink_audio_helpers_test.go`: delete `TestConsumeMidiClearsAndReturns` (its job is now covered by `TestMidiSlotSingleSlotClobber`), and in the concurrency-hammer test (`TestSetMidiCommandIsRaceFree`) replace the consumer goroutine body `_ = ConsumeMidi()` with:

```go
				if cmd, _ := PeekMidi(); cmd != "" {
					ClearMidi()
				}
```

Also update the hammer test's doc comment to name PeekMidi/ClearMidi instead of ConsumeMidi.

- [ ] **Step 4.4: Run full gates** — js build is the critical one:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... && TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./...
```

- [ ] **Step 4.5: Commit**

```bash
git add pkg/jagex2/sound/audio/ pkg/jagex2/client/sign/signlink/
git commit --no-gpg-sign -m "feat(rev-244): drive wasm MIDI through the faithful audioLoop (WS5)"
```

---

### Task 5: migrate volumes to the 244 linear scale

**Files:**
- Modify: `pkg/jagex2/client/client.go` (`SetMidiVolume` ~3465, `UpdateVarp` ~3990-4040)
- Modify: `pkg/jagex2/sound/audio/wave_native.go:93`, `wave_js.go:55`
- Modify: `pkg/jagex2/sound/audio/format.go` (delete `volumeFromCentibels`), `midi_test.go` (delete its test)

- [ ] **Step 5.1: Replace `SetMidiVolume`** (the 225-deob 3-arg form with the `PacketSize` dummy-arg artifact dies; Java 244 is a clean 2-arg):

```go
// SetMidiVolume publishes the music volume (244 linear scale: 128/96/64/32)
// and, when music is active, tells the consumer to re-read it.
// Java: setMidiVolume(int volume, boolean active) (Client.java:1459-1464).
// 244 drops the 225-deob dummy first arg and its packetSize side effect.
func (c *Client) SetMidiVolume(volume int, active bool) {
	signlink.SetMidiVol(volume)
	if active {
		signlink.SetMidiCommand("voladjust")
	}
}
```

- [ ] **Step 5.2: Update `UpdateVarp`'s clientCode 3 and 4 blocks** to the 244 values (Java: Client.java:11371-11414):

```go
	if var3 == 3 {
		var5 := c.MidiActive
		switch var4 {
		case 0:
			c.SetMidiVolume(128, c.MidiActive)
			c.MidiActive = true
		case 1:
			c.SetMidiVolume(96, c.MidiActive)
			c.MidiActive = true
		case 2:
			c.SetMidiVolume(64, c.MidiActive)
			c.MidiActive = true
		case 3:
			c.SetMidiVolume(32, c.MidiActive)
			c.MidiActive = true
		case 4:
			c.MidiActive = false
		}
		// Java: Client.java:11390-11399 — gated by !lowMem, and reactivation
		// re-requests the song by id over OnDemand archive 2.
		if c.MidiActive != var5 && !LowMemory {
			if c.MidiActive {
				c.MidiSong = c.NextMidiSong
				c.MidiFading = false
				c.OnDemand.Request(2, c.MidiSong)
			} else {
				c.StopMidi()
			}
			c.NextMusicDelay = 0
		}
	}
	if var3 == 4 {
		switch var4 {
		case 0:
			c.WaveEnabled = true
			c.SetWaveVolume(128)
		case 1:
			c.WaveEnabled = true
			c.SetWaveVolume(96)
		case 2:
			c.WaveEnabled = true
			c.SetWaveVolume(64)
		case 3:
			c.WaveEnabled = true
			c.SetWaveVolume(32)
		case 4:
			c.WaveEnabled = false
		}
	}
```

- [ ] **Step 5.3: Switch the wave volume curve.** `wave_native.go` (in `playWaveBytes`, ~line 90-94) — replace the SetVolume call + its comment:

```go
	// DEVIATION: Java 244's wavevol is dead — SignLink.audioLoop's wave
	// branch applies no gain at all (SignLink.java:427-475), so the in-game
	// SFX slider does nothing there. The Go port keeps the slider working,
	// mapping the 244 linear scale (128/96/64/32, default 96) through the
	// same vol/256 curve as music. Per-Player so only new sounds pick it up,
	// matching the slider dispatch (UpdateVarp clientCode 4).
	p.SetVolume(linearVolume(signlink.ReadWaveVol()))
```

`wave_js.go` line ~55 (same deviation comment, condensed to one line above it):

```go
	// DEVIATION: 244's wavevol is dead in Java; Go keeps the slider working
	// via the same linear vol/256 curve as music (see wave_native.go).
	sfxGain.Get("gain").Set("value", linearVolume(signlink.ReadWaveVol()))
```

- [ ] **Step 5.4: Delete `volumeFromCentibels`** from `format.go` (and its now-unused `"math"` import) and `TestVolumeFromCentibels` from `midi_test.go`.

- [ ] **Step 5.5: Run full gates** (native + js + tests + vet + gofmt). Watch for client tests that exercise `UpdateVarp`/audio paths — fix only compile breakage, never test semantics, unless a test pinned the old centibel values (then update the pinned values to the 244 scale with a Java ref comment).

- [ ] **Step 5.6: Commit**

```bash
git add pkg/jagex2/client/client.go pkg/jagex2/sound/audio/
git commit --no-gpg-sign -m "feat(rev-244): migrate audio volumes to the 244 linear scale (WS5)"
```

---

### Task 6: configurable storeid 32–34

**Files:**
- Modify: `pkg/jagex2/client/sign/signlink/signlink.go` (var), `storage_disk.go`
- Modify: `cmd/client/main.go`
- Test: `pkg/jagex2/client/sign/signlink/storage_disk_test.go`

- [ ] **Step 6.1: Write the failing test** (append to `storage_disk_test.go`):

```go
// TestStoreDirNameClamp pins the Java storeid window: values outside 32..34
// are clamped back to 32 (and written back to the field), valid values pick
// their own .file_store_<id> directory. Java: SignLink.java:206-210.
func TestStoreDirNameClamp(t *testing.T) {
	t.Cleanup(func() { StoreID = 32 })

	StoreID = 99
	if got := storeDirName(); got != ".file_store_32" || StoreID != 32 {
		t.Fatalf("clamp: got %q (StoreID=%d), want .file_store_32 (32)", got, StoreID)
	}

	StoreID = 33
	if got := storeDirName(); got != ".file_store_33" || StoreID != 33 {
		t.Fatalf("valid id: got %q (StoreID=%d), want .file_store_33 (33)", got, StoreID)
	}
}
```

- [ ] **Step 6.2: Run to verify failure** (compile: `StoreID`, `storeDirName` undefined).

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/client/sign/signlink/ -run TestStoreDirNameClamp
```

- [ ] **Step 6.3: Implement.** In `signlink.go`'s var block:

```go
	// StoreID selects the .file_store_<id> disk cache directory; clamped to
	// 32..34 by storeDirName. Set from the -store-id flag at boot, before
	// StartPriv brings the store up. Java: storeid (SignLink.java:19),
	// settable on the applet loader. The browser build's IndexedDB store
	// does not use it.
	StoreID int = 32
```

In `storage_disk.go`, add `"strconv"` to imports, add `storeDirName`, and use it in `FindCacheDir`:

```go
// storeDirName clamps StoreID to the valid window and returns the cache
// directory name. Java: findcachedir's clamp writes back to storeid before
// building the target (SignLink.java:206-210).
func storeDirName() string {
	if StoreID < 32 || StoreID > 34 {
		StoreID = 32
	}
	return ".file_store_" + strconv.Itoa(StoreID)
}
```

In `FindCacheDir`, replace `var1 := ".file_store_32"` with `var1 := storeDirName()`.

In `cmd/client/main.go`, add the flag after `showVersion` and apply it right after the `*showVersion` early-return block:

```go
	storeID := flag.Int("store-id", 32, "disk cache directory id (.file_store_<id>, clamped to 32-34)")
```

```go
	// Java: SignLink.storeid — selects .file_store_<id>; must be set before
	// the signlink store first resolves its directory (lazily, on first use).
	signlink.StoreID = *storeID
```

(Note: `flag.Int` must be declared BEFORE `flag.Parse()` — place the declaration with the other flags, the assignment after the version check.)

- [ ] **Step 6.4: Run the test + full gates.** Expected: PASS.

- [ ] **Step 6.5: Commit**

```bash
git add pkg/jagex2/client/sign/signlink/ cmd/client/main.go
git commit --no-gpg-sign -m "feat(rev-244): configurable storeid 32-34 + .file_store dir selection (WS5)"
```

---

### Task 7: gates, lint, docs close-out

- [ ] **Step 7.1: Full gate run**

```bash
cd .
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go vet ./...
gofmt -l pkg cmd
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOLANGCI_LINT_CACHE=/tmp/claude-1000/lint GOFLAGS=-mod=mod go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run --max-issues-per-linter=0 --max-same-issues=0 ./...
```
Expected: all clean. Fix new-code findings; pre-existing faithful-port findings get per-line `//nolint` + Java ref only if newly surfaced by this change.

- [ ] **Step 7.2: Update `LOGIC-DELTA-SCOPE.md`** Workstream 5 heading (line ~154): append ` — **DONE 2026-06-03**` and add a closing bullet:

```markdown
- **Status (2026-06-03): DONE.** Faithful audioLoop state machine + linear
  vol/256 volume model (proof in `pkg/jagex2/sound/audio/format.go`) on both
  backends; loop-on-fade semantics restored (music no longer dies after one
  pass). Deviations documented in `pkg/jagex2/sound/audio/audioloop.go`:
  in-memory track handoff, non-blocking wave path, functional wavevol slider
  (dead in Java 244), CC121 per-spec, no FileStream sector cache.
```

- [ ] **Step 7.3: Write the resume note** `.claude/resume/2026-06-03-rev244-ws5-done.md` superseding `2026-06-03-rev244-open-workstreams.md`: WS5 code-complete (all gates green, commits listed), remaining = host smoke test with audio (native: title music at 96/256 gain, region track change = 600ms out + 600ms in, looping background music, jingle + resume, SFX slider; server Engine-TS `244-GOSCAPE`).

- [ ] **Step 7.4: Commit docs**

```bash
git add LOGIC-DELTA-SCOPE.md .claude/resume/ docs/superpowers/plans/
git commit --no-gpg-sign -m "docs(rev-244): close out WS5 audio consumer reconciliation"
```

---

## Out of scope / explicitly not done

- Host smoke test (audio device + display required — hand back to user).
- `FileStream` sector cache port (established bundle/store deviation).
- Replicating the Java wave-branch thread-blocking or the wrapper's NPE-on-failed-MidiPlayer behavior.
- `MidiPlayer.setSoundfont` (dead in Java 244).

## Verification crib for the host smoke test (user-run)

1. Title screen: scape_main plays at moderate volume (96/256 ≈ 0.375 linear — quieter than pre-WS5 full gain; correct).
2. Log in, walk between regions with different tracks: old track fades out ~600ms, new starts silent, fades in ~600ms.
3. Stay in one region past a track's length: music repeats (the loop fix).
4. Level-up/quest jingle: cuts in instantly, plays once, background track resumes after the jingle delay.
5. Options: music volume slider steps audibly across 4 levels; off stops immediately; on resumes the region track. SFX slider affects newly triggered sounds.

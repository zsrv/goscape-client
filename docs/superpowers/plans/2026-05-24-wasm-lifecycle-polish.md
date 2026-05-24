# WASM Sub-project 4: Lifecycle & Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the last same-origin bug (browser MIDI music — the SF2 soundfont fetch was CORS-blocked), remove the dead wave/MIDI code earlier sub-projects left behind, and record the lifecycle/frame-loop review conclusions.

**Architecture:** A (real fix): build-tagged `urlBase()` so `signlink.OpenURL` is same-origin in the browser, mirroring `codebase_js.go`. B (cleanup): delete provably-dead `signlink.MidiSave`/`WaveSave`/`WaveReplay`/`ConsumeWave`, the path-based `midi.go:play()`, the dead fields, and the now-dead `cacheStore.cacheDir()`. C/D: documented review findings, no code change (one comment).

**Tech Stack:** Go 1.26, `syscall/js`. Spec: `docs/superpowers/specs/2026-05-24-wasm-lifecycle-polish-design.md`.

**Sandbox note:** prefix `go` commands with `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache`. The `GOOS=js GOARCH=wasm` build prints a harmless `writing stat cache: ... read-only file system` line — filter `| grep -v 'stat cache'`, check `${PIPESTATUS[0]}`. Commit with `git commit --no-gpg-sign`.

**Sequencing:** Tasks run in order A → B1 → B2 → C; each ends building + testing green before the next. B2 is the riskiest (signlink concurrency core) and comes last.

---

## Task 1 (A): `signlink.OpenURL` same-origin in the browser

**Files:** create `signlink_url_native.go`, `signlink_url_js.go`; modify `signlink.go`.

- [ ] **Step 1: Create the native urlBase**

`pkg/sign/signlink/signlink_url_native.go`:
```go
//go:build !js

package signlink

import (
	"strconv"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

// urlBase is the scheme://host[:port] that signlink.OpenURL fetches against.
// Native standalone uses the loopback data server (Java's literal
// http://127.0.0.1:<portOffset+8888>); see signlink_url_js.go for the browser
// origin-derived variant.
func urlBase() string {
	return "http://127.0.0.1:" + strconv.Itoa(clientextras.PortOffset+8888)
}
```

- [ ] **Step 2: Create the browser urlBase**

`pkg/sign/signlink/signlink_url_js.go`:
```go
//go:build js

package signlink

import "syscall/js"

// urlBase returns the page's own origin so signlink.OpenURL fetches cache
// resources (the SF2 soundfont, reporterror) same-origin — no CORS. Mirrors
// client.GetCodeBase (codebase_js.go) and signlink.ConfigureTransport, which
// derive from the same window.location. The serving origin (e.g. wasmserve)
// proxies these to the data backend.
func urlBase() string {
	return js.Global().Get("location").Get("origin").String()
}
```

- [ ] **Step 3: Use urlBase in the polling loop**

In `pkg/sign/signlink/signlink.go`, the `case urlReq != "":` branch, replace:
```go
			resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(clientextras.PortOffset+8888) + "/" + urlReq)
```
with:
```go
			resp, err := http.Get(urlBase() + "/" + urlReq)
```
Leave the surrounding comment and non-2xx handling. (`strconv` is still used elsewhere in signlink.go — reporterror — so the import stays; the compiler confirms.)

- [ ] **Step 4: Verify**
```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... 2>&1 | grep -v 'stat cache'; echo "native ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/sign/signlink/ 2>&1 | grep -v 'stat cache'; echo "test ${PIPESTATUS[0]}"
gofmt -l pkg/sign/signlink/signlink_url_native.go pkg/sign/signlink/signlink_url_js.go pkg/sign/signlink/signlink.go
```
Expected: native 0, wasm 0, signlink tests pass, gofmt clean.

- [ ] **Step 5: Commit**
```bash
git add pkg/sign/signlink/signlink_url_native.go pkg/sign/signlink/signlink_url_js.go pkg/sign/signlink/signlink.go
git commit --no-gpg-sign -m "fix(signlink): OpenURL fetches same-origin in the browser (fixes SF2 soundfont)"
```

---

## Task 2 (B1): remove the dead path-based MIDI play()

**Files:** modify `pkg/jagex2/sound/audio/midi.go`.

- [ ] **Step 1: Update handle() and remove play()**

Replace `handle()` (its doc comment + body) and the entire `play()` method with:
```go
// handle dispatches one signlink Midi command. Only two shapes occur now:
//   - "stop":      stop sequencing, honoring MidiFade.
//   - "voladjust": adjust user volume on the persistent Player.
// Track data reaches the synth as bytes via PlayMIDI (c.SaveMidi), so the
// command channel no longer carries file paths.
func (d *midiDriver) handle(cmd string) {
	switch cmd {
	case "stop":
		d.stop(signlink.ReadMidiFade() == 1)
	case "voladjust":
		d.setUserVolume(volumeFromCentibels(signlink.ReadMidiVol()))
	default:
		// A path here would be unexpected (signlink.MidiSave is gone) and there
		// is no filesystem in the browser — log rather than read a file.
		log.Printf("audio/midi: ignoring unexpected command %q", cmd)
	}
}
```
(Delete the old `play()` function entirely.)

- [ ] **Step 2: Drop the now-unused `os` import**

`os` was used only by `play()`. Remove `"os"` from `midi.go`'s import block.

- [ ] **Step 3: Verify**
```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... 2>&1 | grep -v 'stat cache'; echo "native ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/sound/audio/ 2>&1 | grep -v 'stat cache'; echo "test ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm ${PIPESTATUS[0]}"
gofmt -l pkg/jagex2/sound/audio/midi.go
```
Expected: native 0, audio tests pass, wasm 0, gofmt clean. (If `go build` reports `os` still used, something else uses it — investigate before removing.)

- [ ] **Step 4: Commit**
```bash
git add pkg/jagex2/sound/audio/midi.go
git commit --no-gpg-sign -m "refactor(audio): remove dead path-based MIDI play() (PlayMIDI is the live path)"
```

---

## Task 3 (B2): remove the dead wave/MIDI surface from signlink + the cacheDir seam method

**Files:** modify `signlink.go`, `storage.go`, `storage_disk.go`, `storage_mem.go`, `storage_idb_js.go`, `storage_mem_test.go`.

- [ ] **Step 1: Delete the dead signlink functions**

In `pkg/sign/signlink/signlink.go`, delete these four functions entirely: `WaveSave`, `WaveReplay`, `MidiSave`, and `ConsumeWave`. (Keep `ConsumeMidi`, `DNSLookup`, `SetMidiCommand`, `SetMidiFade`, and everything cache/socket/URL.)

- [ ] **Step 2: Simplify the Run() save-branch and loop snapshot**

In `Run()`, remove the `var1 := store.cacheDir()` line at startup (keep `uid := store.uid()`):
```go
	uid := store.uid()
	mu.Lock()
	UID = uid
	mu.Unlock()
```

Remove the `wavePlay`/`midiPlay` snapshot reads from the loop top:
```go
		mu.Lock()
		dnsReq := DNSReq
		loadReq := LoadReq
		saveReq := SaveReq
		saveBuf := SaveBuf
		saveLen := SaveLen
		urlReq := URLReq
		loopRate := LoopRate
		mu.Unlock()
```

Replace the `case saveReq != "":` branch with:
```go
		case saveReq != "":
			if saveBuf != nil {
				store.save(saveReq, saveBuf[0:saveLen])
			}
			mu.Lock()
			SaveReq = ""
			cond.Broadcast()
			mu.Unlock()
```

- [ ] **Step 3: Remove the dead fields**

In the package `var` block, delete `Wave`, `WavePos`, `MidiPos`, `MidiPlay`, `WavePlay`. The block becomes (gofmt will realign):
```go
var (
	DNSReq        string
	DNS           string
	LoadReq       string
	LoadBuf       []byte
	SaveReq       string
	SaveBuf       []byte
	URLReq        string
	URLStream     []byte // this was DataInputStream in java
	LoopRate      int    = 50
	Midi          string
	Save          string
	ReportError   bool = true
	ErrorName     string
	ClientVersion int = 225
	MidiFade      int
	MidiVol       int
	SaveLen       int
	//ThreadLiveID  int // not needed in go
	UID     int
	WaveVol int
	//MainApp Applet
	//SocketIP net.IPAddr // not needed in go
	SunJava bool
)
```

- [ ] **Step 4: Drop the now-unused `path` import**

`path` was used only in the removed save-branch `path.Join(var1, saveReq)`. Remove `"path"` from `signlink.go`'s imports. (Build will confirm; if `path` is still used elsewhere, leave it.)

- [ ] **Step 5: Remove cacheDir() from the cacheStore interface and all impls**

`storage.go`: delete the `cacheDir() string` method and its doc comment from the `cacheStore` interface.

`storage_disk.go`: delete `func (d *diskStore) cacheDir() string { ... }` (keep the `dir` field and `ensure()` — `load`/`save` still use them).

`storage_mem.go`: delete `func (s *memStore) cacheDir() string { return "" }`.

`storage_idb_js.go`: delete `func (s *idbStore) cacheDir() string { return "" }`.

- [ ] **Step 6: Update the storage test**

In `pkg/sign/signlink/storage_mem_test.go`, `TestMemStoreUIDAndDir` asserts `s.cacheDir() != ""`. Delete that assertion block (keep the `uid()` assertion) and rename the test to `TestMemStoreUID`.

- [ ] **Step 7: Verify (the race/stress tests are the guardrail for the Run() edit)**
```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... 2>&1 | grep -v 'stat cache'; echo "native ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test -race ./pkg/sign/signlink/ 2>&1 | grep -v 'stat cache'; echo "signlink -race ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm ${PIPESTATUS[0]}"
gofmt -l pkg/sign/signlink/signlink.go pkg/sign/signlink/storage.go pkg/sign/signlink/storage_disk.go pkg/sign/signlink/storage_mem.go pkg/sign/signlink/storage_idb_js.go pkg/sign/signlink/storage_mem_test.go
```
Expected: native 0; signlink tests PASS under `-race` (TestCacheLoadRace, TestSignlinkConcurrentStress, TestFindCacheDir, TestDiskStoreRoundTrip, TestMemStore*, TestResolveWSTarget); wasm 0; gofmt clean. If the build flags any of the removed fields/funcs as still-referenced, that reference is a live caller — restore it and report.

- [ ] **Step 8: Commit**
```bash
git add pkg/sign/signlink/signlink.go pkg/sign/signlink/storage.go pkg/sign/signlink/storage_disk.go pkg/sign/signlink/storage_mem.go pkg/sign/signlink/storage_idb_js.go pkg/sign/signlink/storage_mem_test.go
git commit --no-gpg-sign -m "refactor(signlink): remove dead wave/MIDI publishing surface and cacheDir seam"
```

---

## Task 4 (C/D + final verification)

- [ ] **Step 1: Document the wasm os.Exit review (C)**

In `pkg/jagex2/client/gameshell.go`, at the `os.Exit(0)` inside `Shutdown` (the one near the `Unload()` / `time.Sleep` sequence), add a comment above it:
```go
	// os.Exit halts the Go program cleanly on wasm too (handled by the
	// wasm_exec.js exit callback). Reviewed for the browser and intentionally
	// unchanged: DestroyEvent only fires on tab/canvas teardown, when the page
	// is going away regardless, so the best-effort Shutdown above is sufficient.
	os.Exit(0)
```
(No other change for C; D — the Gio frame loop — was reviewed and needs no change, documented in the spec.)

- [ ] **Step 2: Full native gate**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... && TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./... && TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go vet ./...`
Expected: all pass, vet clean.

- [ ] **Step 3: WASM build + vet**
```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "exit ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go vet ./pkg/sign/signlink/ ./pkg/jagex2/sound/audio/ 2>&1 | grep -v 'stat cache'; echo "vet ${PIPESTATUS[0]}"
```
Expected: both exit 0.

- [ ] **Step 4: Commit**
```bash
git add pkg/jagex2/client/gameshell.go
git commit --no-gpg-sign -m "docs(client): note wasm os.Exit/DestroyEvent reviewed, intentionally unchanged"
```

- [ ] **Step 5: Manual browser smoke (human-run, outside the sandbox)**

The real proof for A. With `make wasm && make wasm-serve` (backend serving `SCC1_Florestan.sf2`):
1. Log in. Confirm **MIDI music now plays** (it was silent before A).
2. In DevTools → Network, confirm the soundfont fetch goes to the page origin (`localhost:8080/SCC1_Florestan.sf2`), not `127.0.0.1:8888`, and succeeds (no CORS error).
3. Confirm SFX still play and the game is otherwise unchanged (boot, connect, cache persistence).
4. Sanity-check native still runs (desktop build) with audio.

---

## Notes / deferred

- `GetHash` stays (dead since the SP2 plain-name deviation, retained as a documented Java reference — not part of this cleanup).
- The vestigial-but-not-removed earlier note is now resolved: the wave/MIDI publishing surface is gone. `signlink.OpenURL`/`CacheLoad`/`CacheSave`/`DNS`/socket and `ConsumeMidi`/`SetMidiCommand`/`SetMidiFade` remain the live signlink surface.
- This completes the 4-part WASM-parity roadmap.

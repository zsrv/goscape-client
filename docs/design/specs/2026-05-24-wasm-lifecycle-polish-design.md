# WASM Sub-project 4: Lifecycle & Polish — Design

**Date:** 2026-05-24
**Status:** Approved (pending implementation)
**Branch:** rev-225

## Goal

Close out the WASM-parity work: fix the one remaining same-origin bug (the SF2
soundfont / reporterror fetch), remove the dead code the earlier sub-projects
left behind, and record the lifecycle/frame-loop review conclusions. Sub-projects
1-3 (boot, IndexedDB, audio) are complete and live-validated.

This sub-project has four items of very different weight: **A** is a real bug,
**B** is earned dead-code cleanup, and **C**/**D** are review items whose honest
conclusion is *no code change needed*.

## Item A: `signlink.OpenURL` must be same-origin in the browser (real bug)

### Problem
There are two independent `OpenURL` paths. `client.OpenURL` → `GetCodeBase()`
was made origin-aware in sub-project 1 (`codebase_js.go`), but `signlink.OpenURL`
is a separate function that still hardcodes the URL in the polling loop:

```go
// signlink.go:210
resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(clientextras.PortOffset+8888) + "/" + urlReq)
```

The **SF2 soundfont fetch** (`soundfont.go:40` → `signlink.OpenURL("SCC1_Florestan.sf2")`)
and `reporterror` go through this path. In the browser `http://127.0.0.1:8888`
is cross-origin to the `http://localhost:8080` page (different host *and* port),
so the fetch is CORS-blocked. Without the soundfont, `meltysynth` has nothing to
render with → **browser MIDI music is silent**. (SFX are unaffected — they don't
fetch.)

### Fix
Mirror the `GetCodeBase` split with a build-tagged `urlBase()`:
- `signlink_url_native.go` (`//go:build !js`):
  `func urlBase() string { return "http://127.0.0.1:" + strconv.Itoa(clientextras.PortOffset+8888) }`
  — native behavior **unchanged** (still `127.0.0.1`; the pre-existing native
  inconsistency with `GetCodeBase` using `clientextras.Host` is intentionally
  left alone to keep this minimal).
- `signlink_url_js.go` (`//go:build js`):
  `func urlBase() string { return js.Global().Get("location").Get("origin").String() }`.

`signlink.go:210` becomes `http.Get(urlBase() + "/" + urlReq)`. In the browser
the fetch is same-origin → through `wasmserve`'s proxy → backend. Fixes both the
soundfont (music) and reporterror. (`strconv` stays used elsewhere in
`signlink.go`; confirm.)

## Item B: dead-code removal

All of this is provably dead — left behind when sub-project 3 routed audio
through `audio.PlayWave`/`PlayMIDI` directly and retired the watcher.

### B1 (audio package, low risk)
`midi.go`'s path-based `play()` and its `os.ReadFile` are unreachable: the only
upstream that sets `signlink.Midi` to a path is `Run()`'s save-branch under
`midiPlay`, which is set only by `MidiSave` — and `MidiSave` has no live callers
(`c.SaveMidi` calls `audio.PlayMIDI` directly). `Midi` is only ever `"stop"` /
`"voladjust"` via `SetMidiCommand`. Remove `play()`; change `handle()`'s default
case from `d.play(cmd, …)` to a `log.Printf` of the unexpected command (defensive;
unreachable in practice). Drops `os` from `midi.go`'s imports if otherwise unused
(verify).

### B2 (signlink core, test-guarded)
Remove the now-dead wave/MIDI *playback-publishing* surface:
- **Functions:** `MidiSave`, `WaveSave`, `WaveReplay`, `ConsumeWave` (no live
  callers — `c.SaveWave`/`ReplayWave` were rerouted to `audio` in SP3; the wave
  watcher was deleted).
- **`Run()` save-branch:** drop the `waveOut`/`midiOut` + `Wave`/`Midi`-path
  assignment and the `wavePlay`/`midiPlay` snapshot reads. The branch reduces to:
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
- **Fields:** remove those only the above touched — `Wave`, `WavePlay`,
  `MidiPlay`, `WavePos`, `MidiPos`. Verify each is dead with the compiler before
  removing. **Keep** all live state: `LoadReq`/`LoadBuf`, `SaveReq`/`SaveBuf`/
  `SaveLen`, `URLReq`/`URLStream`, `DNS*`, `Midi`, `MidiVol`/`MidiFade`/`WaveVol`
  and their readers, `UID`, etc.
- **`cacheStore.cacheDir()` becomes dead** (it existed only to build the wave/MIDI
  scratch paths in `Run()`). Remove it from the `cacheStore` interface
  (`storage.go`) and from `diskStore` (`storage_disk.go`), `memStore`
  (`storage_mem.go`), and `idbStore` (`storage_idb_js.go`). `diskStore` keeps its
  internal `dir` field for its own `load`/`save`. Update
  `storage_mem_test.go`'s `TestMemStoreUIDAndDir` to drop the `cacheDir()`
  assertion.
- The now-unused `path` import in `signlink.go` is dropped if nothing else uses
  it (verify).

**Keep (explicitly not removed):** `ConsumeMidi`, `SetMidiCommand`, `SetMidiFade`,
`GetHash` (already dead per the SP2 deviation, retained as a reference — leave as
documented), and the whole cache/DNS/socket protocol.

**Guardrail:** `signlink_race_test.go` and `signlink_stress_test.go` exercise the
`CacheLoad`/`CacheSave` slot and the mu/cond/slotMu protocol — they must stay
green, proving the `Run()`-loop simplification preserved the concurrency
contract. They don't reference the removed wave/MIDI funcs.

## Item C: `os.Exit` / `DestroyEvent` on wasm — review finding (no change)

Reviewed `gameshell.go:48` (the draw goroutine's `os.Exit(0)` after `draw`
returns), `:224` (`Shutdown`'s `os.Exit(0)`), and the `DestroyEvent` handler
(`:70`). Conclusion: **no change.** On wasm, `os.Exit` halts the Go program
cleanly via the `wasm_exec.js` `exit` callback. `DestroyEvent` only fires on
tab/canvas teardown — the page is being destroyed regardless, so the graceful
`Shutdown` (close `c.Stream`, `StopMidi`, drop cache refs) is best-effort and
harmless if the browser truncates it. No leak, no crash, no observed issue.
Deliverable: a one-line comment at the `os.Exit` site (or the `DestroyEvent`
handler) noting the wasm path was reviewed and is intentionally unchanged, so a
future reader doesn't re-investigate.

## Item D: Gio frame loop on wasm — review finding (no change)

Reviewed `RunGameShell` (the `time.Sleep`-paced tick loop) and the `draw`
`FrameEvent` loop. Conclusion: **no change.** On wasm, Gio drives frames via the
browser's `requestAnimationFrame`; the per-tick `time.Sleep` blocks the game
goroutine but *yields* cooperatively to the JS event loop (the dangerous wasm
pattern is a CPU-bound loop that never blocks, which this is not). The client
runs correctly in-browser (validated in SP1-3). The `OpsMu`-held-across-`e.Frame`
stall recorded in memory is a desktop-Wayland-minimize issue with no browser
analog (no minimize). Deliverable: this finding documented here; no code change.

## Files changed

| File | Change |
|---|---|
| `pkg/sign/signlink/signlink_url_native.go` | **new** (`!js`) — `urlBase()` → 127.0.0.1:port |
| `pkg/sign/signlink/signlink_url_js.go` | **new** (`js`) — `urlBase()` → window.location.origin |
| `pkg/sign/signlink/signlink.go` | A: `http.Get(urlBase()+...)`; B2: remove dead funcs/fields, simplify save-branch, drop `path` import if unused |
| `pkg/jagex2/sound/audio/midi.go` | B1: remove `play()`+`os.ReadFile`; `handle()` default → log |
| `pkg/sign/signlink/storage.go` | B2: remove `cacheDir()` from the interface |
| `pkg/sign/signlink/storage_disk.go` | B2: remove `diskStore.cacheDir()` |
| `pkg/sign/signlink/storage_mem.go` | B2: remove `memStore.cacheDir()` |
| `pkg/sign/signlink/storage_idb_js.go` | B2: remove `idbStore.cacheDir()` |
| `pkg/sign/signlink/storage_mem_test.go` | B2: drop the `cacheDir()` assertion |
| `pkg/jagex2/client/gameshell.go` | C: one-line "reviewed for wasm" comment |

## Error handling

A: `signlink.OpenURL` already handles non-2xx / network errors (sets
`URLStream = nil`, caller logs); the URL change doesn't alter that. B: pure
removal of dead code — no behavior change on any live path.

## Testing

- **Native gate:** `go build ./...`, `go test ./...` (the signlink race/stress
  tests are the B2 guardrail; the storage tests cover the `cacheDir` removal),
  `go vet ./...` — all green.
- **wasm:** `GOOS=js GOARCH=wasm go build ./cmd/client` + `go vet` on the touched
  js-tagged packages.
- **No new unit test needed for A** — `urlBase` js is `syscall/js` (browser-only,
  like `codebase_js.go`, which has no native test either); native `urlBase` is a
  trivial constant-format. The fix is verified by manual browser smoke.
- **Manual browser smoke (the real proof for A):** with `make wasm && make
  wasm-serve` (backend serving `SCC1_Florestan.sf2`), log in and confirm **MIDI
  music now plays** (it was silent before) and the soundfont fetch in the Network
  panel goes to the page origin (`localhost:8080/SCC1_Florestan.sf2`), not
  `127.0.0.1:8888`. Confirm SFX still play and the game is otherwise unchanged.

## Open items for the implementation plan

- Field-by-field dead-confirmation for B2 (`Wave`/`WavePlay`/`MidiPlay`/`WavePos`/
  `MidiPos`) — rely on the compiler (remove, build, restore any that turn out
  live). Expected all dead.
- Whether `path` and `os`/other imports drop out of `signlink.go`/`midi.go` after
  the removals — let the compiler decide.
- Sequence the plan so A and B1 (low-risk) land before B2 (the signlink-core
  edit), each independently building + testing green.

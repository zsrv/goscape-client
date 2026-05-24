# WASM Sub-project 1: Storage Seam + Browser Boot — Design

**Date:** 2026-05-24
**Status:** Approved (pending implementation)
**Branch:** rev-225

## Goal

Make the `GOOS=js GOARCH=wasm` build of the client **boot, render, and connect
to a game server over WebSockets in a browser**. Today the build *compiles*
(after the profiling SIGUSR1 split) but fails at runtime: signlink's filesystem
calls (`FindCacheDir`, `GetUID`, cache load/save) hit the browser's empty wasm
filesystem and return `ENOSYS`, so the cache never works and every request
error-logs.

This sub-project introduces a **platform storage seam** in signlink — a small
interface with a real-disk implementation on native (behavior unchanged) and a
volatile in-memory implementation in the browser — and makes the browser build
**auto-derive its server target from the page origin** so `OpenSocket` takes the
already-working WebSocket path.

This is **sub-project 1 of a 4-part WASM-parity roadmap** (see *Roadmap
context* below). It is the foundation the later sub-projects plug into.

## Non-goals

- **Durable persistence.** The browser store is in-memory only; cache and uid do
  not survive a page reload. IndexedDB-backed persistence is **sub-project 2**,
  and is designed to drop into the same seam without reshaping it.
- **Audio.** oto already plays on wasm via Web Audio (no backend needed), but
  decoupling wave/MIDI from disk paths, loading the SF2 soundfont without a real
  filesystem, and the autoplay-gesture `Resume()` are **sub-project 3**.
- **Lifecycle/perf polish.** Gio frame-loop integration review, `os.Exit`
  cleanup, and any main-loop refactor are **sub-project 4**. `os.Exit` already
  halts acceptably under the Go wasm runtime, so it is not a blocker here.
- **Changing the RS2 protocol, `ClientStream`, or the mu/cond/slotMu signlink
  concurrency protocol.** Only the *leaf* I/O calls inside `signlink.Run()` move
  behind the seam.

## Roadmap context

Full-parity WASM was decomposed into four dependency-ordered sub-projects, each
with its own spec → plan → implementation cycle:

| # | Sub-project | Delivers |
|---|---|---|
| **1 (this)** | Storage seam + browser boot | Boots, renders, connects over WebSocket; signlink filesystem behind a seam; volatile store; origin-derived server target. |
| 2 | Durable persistence (IndexedDB) | Cache + uid survive reloads, behind the #1 seam. |
| 3 | Browser audio | SF2 load without FS, wave byte-decoupling, autoplay-gesture `Resume()`. oto already plays via Web Audio. |
| 4 | Lifecycle & polish | Gio frame-loop review, clean shutdown, perf. |

## Background: what runs where today

### signlink's filesystem usage

`signlink.Run()` (`pkg/sign/signlink/signlink.go`) is a single polling goroutine
servicing four request slots (DNS, cache load, cache save, URL fetch). Three
touch the filesystem:

```
Run() startup:
  var1 := FindCacheDir()        // signlink.go:254 — probes c:/windows, ~/, /tmp, … then os.Mkdir(.file_store_32)
  uid  := GetUID(var1)          // signlink.go:287 — reads/writes uid.dat (persistent client id)

Run() loop:
  loadReq: os.ReadFile(path.Join(var1, loadReq))         // signlink.go:178-193 — nil on miss
  saveReq: os.WriteFile(path.Join(var1, saveReq), …)     // signlink.go:194-219
```

On wasm in a browser, `FindCacheDir` finds nothing and `os.Mkdir` fails, so
`var1 == ""`; loads miss, saves error-log. HTTP asset fetches (`OpenURL`,
`GetCodeBase`) already work — Go's wasm `net/http` transparently uses `fetch()`.

### The networking seam already exists

`OpenSocket` (`signlink.go:382`) already branches on `clientextras.Transport`:
WS/WSS → `openWebSocket` (`signlink_ws.go`, using `github.com/coder/websocket`,
which has a first-class wasm backend over the browser `WebSocket`); otherwise
`net.DialTimeout("tcp", …)`. The TCP branch is unusable in a browser. See
`2026-05-24-websocket-transport-design.md`. The only gap for wasm is that
*something must select the WS branch* — today that requires a `ws://` host arg.

### Client-TS confirms the target patterns

The browser-native TypeScript client (`Client-TS`) validates each choice here:

- **Storage** (`src/io/Database.ts`): one IndexedDB object store keyed by string
  name — `cacheload(name)` → `store.get(name)`, `cachesave(name, bytes)` →
  `store.put(bytes, name)`. This is the exact shape of the seam below, so
  sub-project 2 maps onto `Database.ts` almost 1:1.
- **Client id** (`src/client/Client.ts:1729`): sends a hardcoded `1337` and
  persists nothing. A constant/session uid in the browser store is *parity*.
- **Server target** (`src/io/ClientStream.ts`): `openSocket(window.location.host,
  location.protocol === 'https:')` — derives host from the serving origin and
  upgrades to `wss://` under HTTPS, dodging mixed-content. Subprotocol `binary`,
  matching our `coder/websocket` transport.

## Design

### 1. The storage seam

A new file `pkg/sign/signlink/storage.go` (build-neutral) defines an unexported
interface and the package-level handle:

```go
// cacheStore is signlink's persistence backend. The disk implementation
// (storage_disk.go, //go:build !js) preserves the Java file-store behavior;
// the browser implementation (storage_mem.go, //go:build js) keeps data in
// memory for the session.
//
// Methods are synchronous. A future IndexedDB implementation (sub-project 2)
// may block its goroutine awaiting a JS promise — safe because store methods
// are only ever called from signlink.Run()'s own goroutine, and a goroutine
// blocked on a channel yields to the browser event loop under js/wasm.
type cacheStore interface {
	// load returns the bytes stored under name, or nil on a miss. Mirrors the
	// current os.Stat-then-ReadFile behavior, which returns nil (not an error)
	// for an absent file.
	load(name string) []byte
	// save stores data under name. Best-effort: failures are logged, never
	// returned, matching the current os.WriteFile error handling.
	save(name string, data []byte)
	// uid returns the persistent client id (Java: GetUID). Browser
	// implementations may return a session-stable value.
	uid() int
	// cacheDir returns the on-disk base path used to build wave/MIDI scratch
	// paths in Run(). "" in the browser (no filesystem); audio there reads
	// bytes via load() in sub-project 3.
	cacheDir() string
}

// store is the active backend, selected at build time by newCacheStore
// (storage_disk.go / storage_mem.go), mirroring the profiling Start() split.
var store cacheStore = newCacheStore()
```

Only the **selector** `newCacheStore()` is build-tagged; the store *types* live
where they compile everywhere they're testable.

`storage_mem.go` (**build-neutral**) defines `memStore`: a `sync.Mutex`-guarded
`map[string][]byte` with **no `syscall/js` dependency**, so it compiles and is
unit-testable on native. `load` returns a copy (or nil); `save` stores a copy
(defensive against caller mutation, since the real `os` path also decouples by
serializing to disk). `uid()` returns a **fixed constant placeholder** (parity
with `Client-TS`'s punted `1337`; a random-per-session value is a trivial later
change if a server ever needs distinct ids). `cacheDir()` returns `""`.

`storage_disk.go` (`//go:build !js`) defines `diskStore` and the native
selector:

```go
func newCacheStore() cacheStore { return newDiskStore() }
```

`diskStore` holds the resolved cache directory and uid (computed once via the
*existing* `FindCacheDir`/`GetUID` logic, moved into this file) and implements
`load`/`save` with the current `os.Stat`/`os.ReadFile`/`os.WriteFile` code,
unchanged. Native behavior — including the shared-cache `uid.dat` format and the
`.file_store_32` directory — is byte-for-byte preserved. It is `!js` because its
disk-probing is dead/wrong on wasm; keeping it out of the wasm binary also drops
the `os`-filesystem code path there.

`storage_js.go` (`//go:build js`) is just the browser selector:

```go
func newCacheStore() cacheStore { return newMemStore() }  // memStore from the neutral file
```

### 2. Routing Run() through the seam

In `signlink.go`:

- `Run()` startup: replace `var1 := FindCacheDir(); uid := GetUID(var1)` with a
  read of `store.uid()`; publish to the `UID` field as today. The `var1`
  threading collapses — the cache base now lives inside `store`.
- Load slot: `store.load(loadReq)` replaces the `os.Stat`/`os.ReadFile` block.
- Save slot: `store.save(saveReq, saveBuf[0:saveLen])` replaces `os.WriteFile`.
- The wave/MIDI scratch-path construction (`path.Join(var1, saveReq)` at
  `signlink.go:200-216`) sources its base from `store.cacheDir()`. On native
  this is unchanged; in the browser it yields a bare name. Audio is disabled in
  #1, so this path is dormant in the browser until #3 reworks it to read bytes
  via `store.load(name)`. **No path-vs-bytes audio change happens in #1.**

The mu/cond/slotMu protocol, the "release lock during I/O" structure, the
polling loop, and `cond.Broadcast()` handoffs are untouched.

`FindCacheDir`/`GetUID` currently exported: the plan must check for callers
outside signlink (tests, client). If any exist, keep thin exported wrappers
that delegate into `diskStore`; otherwise relocate the logic into
`storage_disk.go` and unexport.

### 3. Browser server-target derivation

New file `signlink_socket_js.go` (`//go:build js`). On startup (an `init()`, or
an explicit call early in the browser boot path — the plan picks the exact
hook), read `window.location` via `syscall/js`:

```
host   := js.Global().Get("location").Get("host").String()    // "example.com:8080"
secure := js.Global().Get("location").Get("protocol").String() == "https:"
```

Populate `clientextras`:

- `Transport = TransportWSS` if `secure` else `TransportWS`
- `Host`, `WSPort`, `WSPath` derived from `location` (host already includes the
  port; `WSPath` defaults to `/` or a configured game endpoint — the plan fixes
  the server's expected upgrade path).

`OpenSocket` then takes the existing WS branch unchanged. The browser build
**ignores `PortOffset` for the socket**: it connects back to the serving origin,
assuming the server multiplexes static assets and the WS upgrade on one origin
(the `Client-TS` model). This also removes the host from the `?argv=` launch
string and avoids mixed-content by construction.

**Guard:** also add a `//go:build js` override (or a check in the derivation
path) so that if derivation fails or `Transport` is somehow not WS/WSS, the
socket attempt returns a clear error — `"browser build requires a ws:// or
wss:// origin"` — instead of dialing the dead TCP branch. Native `OpenSocket`
is unchanged.

### 4. Build, serve, and launch

- **Makefile `wasm` target** building the client for `js/wasm`. Document the
  `gogio -target js ./cmd/client` flow as the default (it generates the
  `?argv=` URL glue and `wasm.js` shim); note the plain `GOOS=js GOARCH=wasm
  go build` + `wasm_exec.js` alternative.
- **Static dev server**: a tiny `net/http` `FileServer` (Go maps `.wasm` →
  `application/wasm`, which `WebAssembly.instantiateStreaming` requires).
  Document `python3 -m http.server` (≥3.12) as an alternative.
- **Launch**: `?argv=` now carries only `node-id port-offset lowmem|highmem
  free|members` — the server host/scheme are auto-derived. Example:
  `http://localhost:8080/?argv=10 0 highmem members`.

### Data flow (browser, after this sub-project)

```
page load (origin = http://localhost:8080)
  └─ wasm.js: go.argv = ["js","10","0","highmem","members"]
       └─ main(): parse argv → NodeID/PortOffset/mem/world
            ├─ signlink.StartPriv() goroutine
            │    ├─ signlink_socket_js init: Transport=WS, Host=localhost:8080
            │    └─ Run(): uid := store.uid(); loads/saves via memStore (in-RAM)
            ├─ client boot: assets via fetch() (net/http), cached in memStore
            └─ LoginFunc → OpenSocket → openWebSocket(ws://localhost:8080/…)
                 └─ coder/websocket (browser WebSocket) → ClientStream (unchanged)
```

## Error handling

- Cache **load miss** → `nil` (unchanged contract; not an error).
- Cache **save failure** (browser: effectively never) → logged, not returned.
- **WS derivation failure** (e.g. `syscall/js` `location` unavailable in a
  non-browser wasm host) → `OpenSocket` returns a descriptive error; the client
  surfaces it through its normal connection-failure path. No silent TCP attempt.
- All existing signlink error logging (`log.Printf("signlink: …")`) is
  preserved on native.

## Testing

**Native (`go test ./...`), the primary gate:**

- `memStore`: round-trip `save`→`load`, miss returns nil, `load`/`save` copy
  semantics, `uid()` constant stability. Testable on native because `memStore`
  lives in the build-neutral `storage_mem.go` (only `newCacheStore` is
  build-tagged), so a native test constructs it directly.
- `diskStore`: against a `t.TempDir()`, assert parity with current behavior
  (write then read; `uid.dat` byte format unchanged; missing file → nil).
- Socket guard: with `Transport` unset, the browser guard returns the expected
  error (host the guard logic in a testable, build-neutral function where
  feasible).
- **All existing signlink tests stay green** — the seam must not change native
  semantics. The race/stress tests (`signlink_race_test.go`,
  `signlink_stress_test.go`) are the regression backstop for the Run() edit.

**Browser (manual smoke):**

- Build wasm, serve via the static server, open
  `?argv=10 0 highmem members`, confirm: the client renders the login screen,
  the WS connects back to the origin, and no `ENOSYS`/cache error spam appears.
- Automated headless-wasm browser testing is **out of scope**, consistent with
  the repo's existing live-window smoke-test limitation.

## Files

| File | Change |
|---|---|
| `pkg/sign/signlink/storage.go` | **new** (build-neutral) — `cacheStore` interface + `var store` |
| `pkg/sign/signlink/storage_mem.go` | **new** (build-neutral) — `memStore` (pure Go, no `syscall/js`) |
| `pkg/sign/signlink/storage_disk.go` | **new** (`//go:build !js`) — `diskStore` + native `newCacheStore`; absorbs `FindCacheDir`/`GetUID` |
| `pkg/sign/signlink/storage_js.go` | **new** (`//go:build js`) — browser `newCacheStore` returning `memStore` |
| `pkg/sign/signlink/signlink.go` | **edit** — `Run()` routes cache/uid/cacheDir through `store`; remove direct `os` calls in those slots |
| `pkg/sign/signlink/signlink_socket_js.go` | **new** (`//go:build js`) — derive WS host/scheme from `window.location`; TCP guard |
| `pkg/sign/signlink/storage_*_test.go` | **new** — store + guard unit tests |
| `Makefile` | **edit** — `wasm` build target |
| `cmd/wasmserve/` or `scripts/` | **new** (optional) — static dev server with `application/wasm` MIME |
| `README.md` / docs | **edit** — browser build + serve + `?argv=` launch instructions |

## Open items for the implementation plan

- Exact hook for the WS-derivation `init` vs. an explicit call in the browser
  boot sequence (avoid the `go`-vs-goroutine-start race called out in prior
  porting work).
- The server's expected WS upgrade path (`WSPath`) — confirm against the server
  repo so derivation targets the right endpoint.
- Whether `FindCacheDir`/`GetUID` have external callers (keep exported wrappers
  if so).

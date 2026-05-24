# WASM Sub-project 2: IndexedDB Persistence — Design

**Date:** 2026-05-24
**Status:** Approved (pending implementation)
**Branch:** rev-225

## Goal

Make the browser client's cache **survive page reloads**. Today the `js/wasm`
build uses a volatile in-memory store (`memStore`), so every reload re-fetches
all game data (cache archives, MIDI) through the dev proxy. This sub-project
replaces the browser backend with an **IndexedDB-backed `cacheStore`** so cached
data persists across reloads — implemented entirely behind the existing
`cacheStore` interface, with **no interface change** and no change to
`signlink.Run()`.

This is **sub-project 2 of the 4-part WASM-parity roadmap**; sub-project 1
(storage seam + browser boot) is complete and live-validated. The IndexedDB
store drops into the seam sub-project 1 built.

## Non-goals

- **Native behavior.** `idbStore` is `//go:build js` only; the native build is
  untouched (`diskStore` remains the native backend).
- **Persisting the client uid.** The browser uid stays a constant (`1337`),
  matching `Client-TS`, which persists no client id. (See *Decisions* below.)
- **A new caching layer / in-memory hot cache.** Reads and writes go straight to
  IndexedDB on demand (blocking), faithful to the disk store's semantics. No
  read-through map, no startup hydrate-all.
- **Audio decoupling, lifecycle polish** — sub-projects 3 and 4.
- **Cache eviction / quota management.** IndexedDB's own storage limits and the
  browser's eviction apply; we add no explicit eviction.

## Background

### The seam (from sub-project 1)

`pkg/sign/signlink` defines an unexported `cacheStore` interface:

```go
type cacheStore interface {
	load(name string) []byte   // nil on miss
	save(name string, data []byte)
	uid() int
	cacheDir() string
}
```

`var store cacheStore = newCacheStore()` is selected per build:
`storage_disk.go` (`//go:build !js`) → `diskStore`; `storage_js.go`
(`//go:build js`) → currently `newMemStore()`. `signlink.Run()` calls
`store.load`/`store.save` **outside** the `mu` lock (mu released before the I/O,
re-acquired to publish), and `store.uid()`/`store.cacheDir()` once at startup.

### Why a synchronous interface can wrap async IndexedDB

IndexedDB is callback/event-based; `cacheStore.load` returns `[]byte`
synchronously. On `js/wasm` this is bridgeable because `signlink.Run()` runs in
its **own goroutine**: when it blocks on a Go channel awaiting an IDB
`onsuccess` callback, the Go scheduler yields to the JS event loop; the callback
fires, sends on the channel, and Run resumes. This is safe only because the
blocked goroutine is not the one pumping the event loop — exactly the property
sub-project 1's interface doc pre-authorized. `store.load`/`save` already run
outside `mu`, so blocking there mirrors how `os.ReadFile`/`os.WriteFile` blocked
on the native path.

**Critical timing constraint:** the IndexedDB database must NOT be opened at
package-init time (`var store = newCacheStore()` runs before `main`, before the
event loop pumps — blocking there would deadlock). Open must be **lazy**, on
first use from Run's goroutine, via `sync.Once` (mirroring `diskStore.ensure`).

### Reference: Client-TS

`Client-TS/src/io/Database.ts` is the proven browser pattern: one IndexedDB
database, one object store named `cache`, keyed by string name —
`cacheload(name)` → `store.get(name)`, `cachesave(name, bytes)` →
`store.put(bytes, name)` — with graceful fallback to no-persistence when IndexedDB
is unavailable (incognito). This design maps onto it 1:1.

## Design

### Components

| File | Build tag | Change |
|---|---|---|
| `pkg/sign/signlink/storage_idb_js.go` | `//go:build js` | **new** — `idbStore`, the async-bridge helper, `newIDBStore` |
| `pkg/sign/signlink/storage_js.go` | `//go:build js` | **edit** — `newCacheStore()` returns `newIDBStore()` (was `newMemStore()`) |
| `pkg/sign/signlink/storage_mem.go` | build-neutral | **unchanged** — `memStore` becomes the fallback backend |

`storage.go`, `signlink.go`, `storage_disk.go`, and all native code are
untouched.

### The async-bridge helper

```go
// await attaches onsuccess/onerror handlers to an IDBRequest and blocks the
// calling goroutine until one fires. Returns req.result on success. The
// callbacks are released after firing. MUST be called from a goroutine that is
// not pumping the JS event loop (signlink.Run's goroutine qualifies).
func await(req js.Value) (js.Value, error)
```

Implementation outline: a `chan struct{ val js.Value; err error }` of size 1;
`onsuccess` sends `{req.Get("result"), nil}`; `onerror` sends
`{undefined, error(...)}`; both `Release()` the two `js.Func`s; the function
blocks on `<-ch`. Reused by open, get, and put.

### idbStore

```go
type idbStore struct {
	once      sync.Once
	db        js.Value   // IDBDatabase; .IsNull()/.IsUndefined() when unavailable
	available bool
	fallback  *memStore  // used when available == false
}

func newIDBStore() *idbStore { return &idbStore{fallback: newMemStore()} }
```

- **`ensure()`** (`sync.Once`): get `js.Global().Get("indexedDB")`; if undefined →
  `available = false`, log once, done (fallback used). Else
  `indexedDB.open("goscape", 1)`; in the `onupgradeneeded` handler call
  `db.createObjectStore("cache")`; `await` the open request. On success store
  `db`, set `available = true`. On error → `available = false`, log once.
- **`load(name)`**: `ensure()`; if `!available` → `fallback.load(name)`. Else open
  a `"readonly"` transaction on `"cache"`, `objectStore.get(name)`, `await` it.
  Result `undefined`/`null` → return `nil` (miss). Otherwise allocate a `[]byte`
  of `result.Get("length")` and `js.CopyBytesToGo` from the `Uint8Array`. On
  error: log, return `nil`.
- **`save(name, data)`**: `ensure()`; if `!available` → `fallback.save(name, data)`.
  Else build a `Uint8Array` of `len(data)` and `js.CopyBytesToJS`, open a
  `"readwrite"` transaction, `objectStore.put(arr, name)`, `await` the request
  (commit). On error: log, return (best-effort; never propagated).
- **`uid()`**: return `browserUID` (the existing `1337` constant).
- **`cacheDir()`**: return `""`.

A compile-time assertion `var _ cacheStore = (*idbStore)(nil)` guards the
interface.

### Data flow (browser, with persistence)

```
first visit:
  Run() → store.load("title<crc>") → ensure() opens IDB → get → MISS (nil)
        → client fetches via proxy → store.save("title<crc>", bytes) → IDB put
reload:
  Run() → store.load("title<crc>") → ensure() (DB already exists) → get → HIT
        → returns persisted bytes, no network fetch
```

### Byte conversion

- **Read:** `n := result.Get("length").Int(); buf := make([]byte, n);
  js.CopyBytesToGo(buf, result)`.
- **Write:** `arr := js.Global().Get("Uint8Array").New(len(data));
  js.CopyBytesToJS(arr, data); objectStore.Call("put", arr, name)`.

`js.CopyBytesToGo`/`CopyBytesToJS` require a `Uint8Array` (not `ArrayBuffer`);
IndexedDB round-trips a stored `Uint8Array` as a `Uint8Array`, so this holds.

## Decisions (baked in, per design approval)

1. **uid stays constant `1337`.** No persistence; `Client-TS` parity. Persisting
   a real id later is one extra IDB key if ever needed.
2. **Fallback to in-memory** when IndexedDB is unavailable (incognito/disabled):
   `idbStore` delegates every method to its embedded `memStore`. The session
   works, just without cross-reload persistence — matching `Client-TS`.
3. **`save` blocks until the transaction commits** (faithful to `os.WriteFile`'s
   synchronous semantics; prevents a save-then-load race), rather than
   fire-and-forget.
4. **Schema:** DB `"goscape"`, object store `"cache"`, version `1`.

## Error handling

- **IDB unavailable / open error** → log once, degrade to `memStore`. Never fatal.
- **`get` error** → log, return `nil` (a cache miss; the client re-fetches).
- **`put`/commit error** → log, return (best-effort; the next session re-fetches).
- All consistent with the current contract (load returns nil on miss; save is
  best-effort) and with `signlink`'s existing `log.Printf("signlink: …")` style.

## Testing

- **Native (`go test ./...`)** — stays green. `idbStore` is `js`-only, excluded
  from native builds. The `memStore` fallback keeps its existing unit tests, so
  the fallback path is covered on the native gate.
- **No native unit test for `idbStore`** — it is `syscall/js` + IndexedDB,
  browser-only, the same constraint as the WS-origin / codebase / soft-keyboard
  work. Verified by manual browser smoke (below).
- **Manual browser smoke:** with `make wasm && make wasm-serve`, load the client
  once (observe cache fetches in the Network panel), then **reload** and confirm
  the cache archives are served from IndexedDB (no/repeat network fetches) and
  the entries appear under Application → IndexedDB → `goscape` → `cache`. Confirm
  an incognito window (IndexedDB blocked) still boots via the in-memory fallback.
- **Compile gate:** `GOOS=js GOARCH=wasm go build ./cmd/client` must pass.

## Files changed

| File | Change |
|---|---|
| `pkg/sign/signlink/storage_idb_js.go` | **new** (`//go:build js`) — `idbStore` + `await` + `newIDBStore` + interface assertion |
| `pkg/sign/signlink/storage_js.go` | **edit** — `newCacheStore()` → `newIDBStore()` |

## Open items for the implementation plan

- Exact handling of the `onupgradeneeded` callback lifetime relative to the
  `await` on the open request (the upgrade handler fires before `onsuccess`).
- Whether to set the IDB value as a plain `Uint8Array` vs `ArrayBuffer` — spec
  chooses `Uint8Array` so `CopyBytesToGo` works directly on read.
- `js.Func` release discipline in `await` (release in both success and error
  paths; guard against double-fire).

# WASM Sub-project 2: IndexedDB Persistence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the browser build's volatile in-memory cache with an IndexedDB-backed `cacheStore` so cached game data survives page reloads, implemented behind the existing seam with no interface change.

**Architecture:** A new `//go:build js` `idbStore` implements `cacheStore`. IndexedDB is async; each op blocks `signlink.Run`'s goroutine on a channel until the JS callback fires (safe — Run isn't the event-loop goroutine). The DB opens lazily via `sync.Once` on first use (never at package init). When IndexedDB is unavailable (private browsing), `idbStore` delegates to an embedded `memStore` fallback. Only the browser selector changes; native is untouched.

**Tech Stack:** Go 1.26, `syscall/js`, IndexedDB. Spec: `docs/superpowers/specs/2026-05-24-wasm-indexeddb-persistence-design.md`.

**Sandbox note:** prefix `go` commands with `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache`. The `GOOS=js GOARCH=wasm` build prints a harmless `writing stat cache: ... read-only file system` line — filter with `| grep -v 'stat cache'` and check exit codes. Commit with `git commit --no-gpg-sign`.

**Testing note:** `idbStore` is `syscall/js` + IndexedDB — browser-only, with no feasible native unit test (same constraint as the WS-origin / codebase / soft-keyboard work in sub-project 1). The native gate confirms `idbStore` is excluded from native builds and that everything still compiles/passes; the `memStore` fallback keeps its existing native unit tests. End-to-end behavior is verified by manual browser smoke (Task 2).

---

## File Structure

| File | Build tag | Responsibility |
|---|---|---|
| `pkg/sign/signlink/storage_idb_js.go` | `//go:build js` | **new** — `idbStore`, the `await` async-bridge helper, `newIDBStore`, interface assertion |
| `pkg/sign/signlink/storage_js.go` | `//go:build js` | **edit** — `newCacheStore()` returns `newIDBStore()` (was `newMemStore()`) |
| `pkg/sign/signlink/storage_mem.go` | build-neutral | **unchanged** — `memStore` is now the fallback backend |

---

## Task 1: Implement the IndexedDB store and wire it in

**Files:**
- Create: `pkg/sign/signlink/storage_idb_js.go`
- Modify: `pkg/sign/signlink/storage_js.go`

- [ ] **Step 1: Write the IndexedDB store**

Create `pkg/sign/signlink/storage_idb_js.go`:

```go
//go:build js

package signlink

import (
	"errors"
	"log"
	"sync"
	"syscall/js"
)

var _ cacheStore = (*idbStore)(nil)

const (
	idbName      = "goscape"
	idbStoreName = "cache"
)

// idbStore is the browser cacheStore backed by IndexedDB, so cached game data
// survives page reloads. IndexedDB is asynchronous; each operation blocks the
// calling goroutine (signlink.Run's) on a channel until the JS callback fires —
// safe because Run is not the goroutine pumping the JS event loop, so the
// scheduler yields to the loop while it waits. The database is opened lazily on
// first use (NOT at package init, which runs before the event loop and would
// deadlock). When IndexedDB is unavailable (e.g. private browsing), idbStore
// delegates to an in-memory memStore so the session still works, just without
// cross-reload persistence — matching the Client-TS reference.
type idbStore struct {
	once      sync.Once
	db        js.Value
	available bool
	fallback  *memStore
}

func newIDBStore() *idbStore { return &idbStore{fallback: newMemStore()} }

// await attaches success/error handlers to an IDBRequest and blocks until one
// fires, returning req.result on success. Both js.Funcs are released before
// returning. Must be called only from a goroutine that does not pump the JS
// event loop (signlink.Run's goroutine qualifies).
func await(req js.Value) (js.Value, error) {
	type result struct {
		val js.Value
		err error
	}
	ch := make(chan result, 1)
	var onOK, onErr js.Func
	onOK = js.FuncOf(func(this js.Value, args []js.Value) any {
		ch <- result{val: req.Get("result")}
		return nil
	})
	onErr = js.FuncOf(func(this js.Value, args []js.Value) any {
		ch <- result{err: errors.New("indexeddb request failed")}
		return nil
	})
	req.Set("onsuccess", onOK)
	req.Set("onerror", onErr)
	r := <-ch
	onOK.Release()
	onErr.Release()
	return r.val, r.err
}

// ensure opens the IndexedDB database exactly once, creating the object store on
// first run. Any failure (no indexedDB, a thrown SecurityError in private
// browsing, or an open error) leaves available=false so every op uses the
// in-memory fallback. The recover guards against synchronous JS exceptions,
// which syscall/js surfaces as panics.
func (s *idbStore) ensure() {
	s.once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("signlink: indexedDB unavailable (%v); cache will not persist across reloads", r)
				s.available = false
			}
		}()

		idb := js.Global().Get("indexedDB")
		if !idb.Truthy() {
			log.Printf("signlink: indexedDB unavailable; cache will not persist across reloads")
			return
		}

		req := idb.Call("open", idbName, 1)
		// onupgradeneeded fires (before onsuccess) when the DB is created or
		// version-bumped; create the object store there.
		onUpgrade := js.FuncOf(func(this js.Value, args []js.Value) any {
			db := req.Get("result")
			if !db.Get("objectStoreNames").Call("contains", idbStoreName).Bool() {
				db.Call("createObjectStore", idbStoreName)
			}
			return nil
		})
		req.Set("onupgradeneeded", onUpgrade)
		db, err := await(req)
		onUpgrade.Release()
		if err != nil || !db.Truthy() {
			log.Printf("signlink: indexedDB open failed; cache will not persist: %v", err)
			return
		}
		s.db = db
		s.available = true
	})
}

// load returns the bytes stored under name, or nil on a miss — mirroring the
// disk store's os.Stat-then-ReadFile (nil for absent). A get that resolves to
// undefined is a miss; otherwise the stored Uint8Array is copied to a []byte.
func (s *idbStore) load(name string) (out []byte) {
	s.ensure()
	if !s.available {
		return s.fallback.load(name)
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("signlink: indexedDB load %q panicked: %v", name, r)
			out = nil
		}
	}()
	tx := s.db.Call("transaction", idbStoreName, "readonly")
	req := tx.Call("objectStore", idbStoreName).Call("get", name)
	res, err := await(req)
	if err != nil {
		log.Printf("signlink: indexedDB get %q: %v", name, err)
		return nil
	}
	if !res.Truthy() {
		return nil // miss: get resolved to undefined
	}
	n := res.Get("length").Int()
	buf := make([]byte, n)
	js.CopyBytesToGo(buf, res)
	return buf
}

// save stores data under name as a Uint8Array. Best-effort: errors are logged,
// never returned (matching the cacheStore contract). It blocks until the
// readwrite request resolves, mirroring os.WriteFile's synchronous semantics so
// a subsequent load observes the write.
func (s *idbStore) save(name string, data []byte) {
	s.ensure()
	if !s.available {
		s.fallback.save(name, data)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("signlink: indexedDB save %q panicked: %v", name, r)
		}
	}()
	arr := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(arr, data)
	tx := s.db.Call("transaction", idbStoreName, "readwrite")
	req := tx.Call("objectStore", idbStoreName).Call("put", arr, name)
	if _, err := await(req); err != nil {
		log.Printf("signlink: indexedDB put %q: %v", name, err)
	}
}

// uid returns the constant browser client id (browserUID). The browser has no
// persistent uid.dat and Client-TS sends a fixed value, so a constant is parity.
func (s *idbStore) uid() int { return browserUID }

// cacheDir returns "" — there is no on-disk scratch directory in the browser.
func (s *idbStore) cacheDir() string { return "" }
```

- [ ] **Step 2: Verify the new file compiles for wasm and native stays green**

At this point `idbStore` exists but is not yet selected (`storage_js.go` still returns `newMemStore`). `newIDBStore` is an unused package-level function, which is legal Go.

Run:
```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm exit ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... 2>&1 | grep -v 'stat cache'; echo "native exit ${PIPESTATUS[0]}"
```
Expected: both `exit 0`. (The interface assertion `var _ cacheStore = (*idbStore)(nil)` forces all four methods to be present, so a wasm-compile success confirms `idbStore` satisfies `cacheStore`.)

- [ ] **Step 3: Flip the browser selector to the IndexedDB store**

Replace the body of `pkg/sign/signlink/storage_js.go`:

```go
//go:build js

package signlink

// newCacheStore returns the IndexedDB-backed store for the browser build, so
// cache survives reloads. It degrades to an in-memory store when IndexedDB is
// unavailable (see idbStore).
func newCacheStore() cacheStore { return newIDBStore() }
```

- [ ] **Step 4: Verify both targets after wiring**

Run:
```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm exit ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... && TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/sign/signlink/... 2>&1 | grep -v 'stat cache'; echo "native test exit ${PIPESTATUS[0]}"
gofmt -l pkg/sign/signlink/storage_idb_js.go pkg/sign/signlink/storage_js.go
```
Expected: `wasm exit 0`; native signlink tests PASS (the existing `memStore`/`diskStore`/socket tests, now exercising `memStore` as the fallback type); `gofmt -l` prints nothing.

- [ ] **Step 5: Commit**

```bash
git add pkg/sign/signlink/storage_idb_js.go pkg/sign/signlink/storage_js.go
git commit --no-gpg-sign -m "feat(signlink): IndexedDB-backed browser cache (persists across reloads)"
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
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go vet ./pkg/sign/signlink/ 2>&1 | grep -v 'stat cache'; echo "vet exit ${PIPESTATUS[0]}"
```
Expected: both `exit 0`.

- [ ] **Step 3: Manual browser smoke (human-run, outside the sandbox)**

Performed by the user on the host (the sandbox cannot open a browser). Document the result in the PR/commit notes.

1. `make wasm && make wasm-serve` (with the cache/WS backend running on `localhost:8888`).
2. Open `http://localhost:8080/?argv=10 0 highmem members`, let it load past the login screen once. In DevTools → Network, observe the cache archives (`crc…`, `title<crc>`, `config<crc>`, …) being fetched.
3. In DevTools → Application → IndexedDB, confirm a `goscape` database with a `cache` object store populated with those keys.
4. **Reload** the page. Confirm the cache archives are now served from IndexedDB — the prior `crc…`/archive fetches do not repeat (or return instantly from cache) — and the client still boots and logs in.
5. Open the same URL in a **private/incognito** window (IndexedDB blocked or ephemeral). Confirm the client still boots (in-memory fallback) and logs a single "indexedDB unavailable" line rather than crashing.

---

## Notes / deferred

- **Sub-project 3 (browser audio)** and **#4 (lifecycle/polish)** remain. The `signlink.OpenURL` reporterror path still hardcodes `127.0.0.1:8888` (non-boot; noted in earlier work).
- No cache eviction/quota handling is added; IndexedDB's own limits and browser eviction apply.

# WASM Sub-project 1: Storage Seam + Browser Boot — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the `GOOS=js GOARCH=wasm` client boot, render, and connect to a server over WebSocket in a browser, by putting signlink's filesystem behind a platform seam (disk on native, in-memory in the browser) and auto-deriving the WebSocket target from the page origin.

**Architecture:** A small unexported `cacheStore` interface inside `pkg/sign/signlink`. Pure-Go types (`memStore`, the `resolveWSTarget` helper) live in build-neutral files so they're unit-testable on the native gate; only the build-tagged *selectors* (`newCacheStore`), the disk-probing `diskStore`, and the `syscall/js` location-reading differ per `//go:build`. `signlink.Run()`'s concurrency protocol and `ClientStream` are untouched — only leaf I/O calls move behind the seam.

**Tech Stack:** Go 1.26, `//go:build` tags, `syscall/js`, `github.com/coder/websocket` (existing), `gioui.org/cmd/gogio` (browser packaging). Spec: `docs/superpowers/specs/2026-05-24-wasm-storage-boot-design.md`.

**Sandbox note:** All `go` commands in this plan must be prefixed for the sandbox:
`TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go ...`. Commits use `git commit --no-gpg-sign` (project convention; works in-sandbox).

---

## File Structure

| File | Build tag | Responsibility |
|---|---|---|
| `pkg/sign/signlink/storage.go` | neutral | `cacheStore` interface + `var store` handle + impl assertions |
| `pkg/sign/signlink/storage_mem.go` | neutral | `memStore` (pure-Go map store) + `newMemStore` |
| `pkg/sign/signlink/storage_mem_test.go` | neutral | `memStore` unit tests |
| `pkg/sign/signlink/storage_disk.go` | `!js` | `diskStore` + native `newCacheStore`; absorbs `FindCacheDir`/`GetUID` |
| `pkg/sign/signlink/storage_disk_test.go` | `!js` | `diskStore` round-trip test |
| `pkg/sign/signlink/storage_js.go` | `js` | browser `newCacheStore` → `memStore` |
| `pkg/sign/signlink/signlink_socket.go` | neutral | `resolveWSTarget` pure helper |
| `pkg/sign/signlink/signlink_socket_test.go` | neutral | `resolveWSTarget` unit tests |
| `pkg/sign/signlink/signlink_socket_native.go` | `!js` | `dialTCP` (real) + `ConfigureTransport` (no-op) |
| `pkg/sign/signlink/signlink_socket_js.go` | `js` | `dialTCP` (error) + `ConfigureTransport` (location-derive) |
| `pkg/sign/signlink/signlink.go` | neutral | **edit**: `Run()` + `OpenSocket` route through seam; drop moved code |
| `cmd/client/main.go` | neutral | **edit**: call `signlink.ConfigureTransport()` |
| `cmd/wasmserve/main.go` | neutral | static dev server with `application/wasm` MIME |
| `Makefile` | — | **edit**: `wasm` + `wasm-serve` targets |
| `README.md` | — | **edit**: browser build/serve/launch docs |

---

## Task 1: Storage interface + in-memory store

**Files:**
- Create: `pkg/sign/signlink/storage.go`
- Create: `pkg/sign/signlink/storage_mem.go`
- Test: `pkg/sign/signlink/storage_mem_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/sign/signlink/storage_mem_test.go`:

```go
package signlink

import (
	"bytes"
	"testing"
)

func TestMemStoreRoundTrip(t *testing.T) {
	s := newMemStore()

	if got := s.load("missing"); got != nil {
		t.Fatalf("miss should be nil, got %v", got)
	}

	s.save("config", []byte{1, 2, 3})
	if got := s.load("config"); !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("load: got %v, want [1 2 3]", got)
	}
}

func TestMemStoreCopySemantics(t *testing.T) {
	s := newMemStore()

	in := []byte{4, 5}
	s.save("x", in)
	in[0] = 9 // mutate caller's slice after save
	if got := s.load("x"); got[0] != 4 {
		t.Fatalf("store aliased the input slice: got %v", got)
	}

	out := s.load("x")
	out[0] = 7 // mutate returned slice
	if again := s.load("x"); again[0] != 4 {
		t.Fatalf("store aliased the returned slice: got %v", again)
	}
}

func TestMemStoreUIDAndDir(t *testing.T) {
	s := newMemStore()
	if s.uid() != browserUID {
		t.Fatalf("uid: got %d, want %d", s.uid(), browserUID)
	}
	if s.cacheDir() != "" {
		t.Fatalf("cacheDir: got %q, want empty", s.cacheDir())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/sign/signlink/ -run TestMemStore -v`
Expected: FAIL — compile error, `undefined: newMemStore` / `browserUID`.

- [ ] **Step 3: Write the interface**

Create `pkg/sign/signlink/storage.go`:

```go
package signlink

// cacheStore is signlink's persistence backend. The disk implementation
// (storage_disk.go, //go:build !js) preserves the Java file-store behavior;
// the in-memory implementation (storage_mem.go) backs the browser build.
//
// Methods are synchronous. They are called only from signlink.Run()'s own
// goroutine, so a future IndexedDB implementation (sub-project 2) may block
// awaiting a JS promise — safe because a blocked goroutine yields to the
// browser event loop under js/wasm.
type cacheStore interface {
	// load returns the bytes stored under name, or nil on a miss (mirrors the
	// current os.Stat-then-ReadFile behavior, which returns nil for an absent
	// file rather than an error).
	load(name string) []byte
	// save stores data under name. Best-effort: failures are logged, never
	// returned, matching the current os.WriteFile error handling.
	save(name string, data []byte)
	// uid returns the persistent client id (Java: GetUID). The browser
	// implementation returns a session-stable constant.
	uid() int
	// cacheDir returns the on-disk base used to build wave/MIDI scratch paths
	// in Run(). "" in the browser (no filesystem).
	cacheDir() string
}
```

(The `var store = newCacheStore()` handle is added in Task 2, Step 5, once the
build-tagged `newCacheStore` selectors exist — adding it now would not compile.)

- [ ] **Step 4: Write the in-memory store**

Create `pkg/sign/signlink/storage_mem.go`:

```go
package signlink

import "sync"

// browserUID is the session client id reported by the in-memory store. The
// browser has no persistent uid.dat; the TypeScript reference client
// (Client-TS/src/client/Client.ts:1729) likewise sends a fixed value and
// persists nothing, so a constant is parity, not a shortcut.
const browserUID = 1337

// memStore is a volatile, in-RAM cacheStore: a string-keyed blob map under a
// mutex. It has no syscall/js dependency, so it compiles and is unit-tested on
// native. Used by the browser build (storage_js.go); replaced by an
// IndexedDB-backed store in sub-project 2 behind this same interface.
type memStore struct {
	mu sync.Mutex
	m  map[string][]byte
}

func newMemStore() *memStore {
	return &memStore{m: make(map[string][]byte)}
}

func (s *memStore) load(name string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.m[name]
	if !ok {
		return nil
	}
	// Copy out so callers can't mutate stored bytes (the disk path decouples
	// by serializing to disk; this preserves that contract).
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp
}

func (s *memStore) save(name string, data []byte) {
	cp := make([]byte, len(data))
	copy(cp, data)
	s.mu.Lock()
	s.m[name] = cp
	s.mu.Unlock()
}

func (s *memStore) uid() int { return browserUID }

func (s *memStore) cacheDir() string { return "" }
```

NOTE: `storage.go`'s `var store = newCacheStore()` references `newCacheStore`, which does not exist until Task 2. To keep Task 1 self-contained and compiling, **temporarily** omit the `var store` line from `storage.go` in this task (add only the interface). Add the `var store` line in Task 2, Step 5, alongside the selectors. (Do not add a compile assertion for `memStore` yet either — add it in Task 2.)

So in this task, `storage.go` contains only the package clause, the `cacheStore` interface, and its doc comment — not the `var store` line.

- [ ] **Step 5: Run test to verify it passes**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/sign/signlink/ -run TestMemStore -v`
Expected: PASS (3 tests).

- [ ] **Step 6: Commit**

```bash
git add pkg/sign/signlink/storage.go pkg/sign/signlink/storage_mem.go pkg/sign/signlink/storage_mem_test.go
git commit --no-gpg-sign -m "feat(signlink): add cacheStore interface + in-memory store"
```

---

## Task 2: Disk store + route Run() through the seam

This task moves `FindCacheDir`/`GetUID` into the `!js` disk store, wires the build-tagged selectors, adds `var store`, and rewrites `Run()`'s three filesystem touchpoints to call `store`. Both native and js must compile at the end.

**Files:**
- Create: `pkg/sign/signlink/storage_disk.go` (`//go:build !js`)
- Create: `pkg/sign/signlink/storage_js.go` (`//go:build js`)
- Create: `pkg/sign/signlink/storage_disk_test.go` (`//go:build !js`)
- Modify: `pkg/sign/signlink/storage.go` (add `var store` + assertions)
- Modify: `pkg/sign/signlink/signlink.go` (remove `FindCacheDir`/`GetUID`; rewrite `Run()` slots; fix imports)

- [ ] **Step 1: Write the failing disk-store test**

Create `pkg/sign/signlink/storage_disk_test.go`:

```go
//go:build !js

package signlink

import (
	"bytes"
	"testing"
)

// diskStoreAt builds a diskStore bound to an explicit dir/id, bypassing
// FindCacheDir/GetUID so the test controls the location.
func diskStoreAt(dir string, id int) *diskStore {
	d := &diskStore{dir: dir, id: id}
	d.once.Do(func() {}) // mark initialized so ensure() won't probe the FS
	return d
}

func TestDiskStoreRoundTrip(t *testing.T) {
	d := diskStoreAt(t.TempDir(), 42)

	if got := d.load("missing"); got != nil {
		t.Fatalf("miss should be nil, got %v", got)
	}

	d.save("config", []byte{1, 2, 3})
	if got := d.load("config"); !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("load: got %v, want [1 2 3]", got)
	}

	if d.uid() != 42 {
		t.Fatalf("uid: got %d, want 42", d.uid())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/sign/signlink/ -run TestDiskStore -v`
Expected: FAIL — `undefined: diskStore`.

- [ ] **Step 3: Create the disk store, moving FindCacheDir/GetUID into it**

First, in `pkg/sign/signlink/signlink.go`, **delete** the `FindCacheDir` function (currently lines ~254-285) and the `GetUID` function (currently lines ~287-305). They move verbatim into the new file below.

Create `pkg/sign/signlink/storage_disk.go`:

```go
//go:build !js

package signlink

import (
	"encoding/binary"
	"errors"
	"log"
	"math/rand"
	"os"
	"path"
	"sync"
)

func newCacheStore() cacheStore { return &diskStore{} }

// diskStore is the native cacheStore: the original Java-parity file store under
// FindCacheDir()/.file_store_32. dir and id are resolved once, lazily, on first
// use — matching the historical timing where Run() called FindCacheDir/GetUID
// at startup (not at package init), so importing signlink without running it
// has no filesystem side effects.
type diskStore struct {
	once sync.Once
	dir  string
	id   int
}

func (d *diskStore) ensure() {
	d.once.Do(func() {
		d.dir = FindCacheDir()
		d.id = GetUID(d.dir)
	})
}

func (d *diskStore) cacheDir() string { d.ensure(); return d.dir }

func (d *diskStore) uid() int { d.ensure(); return d.id }

func (d *diskStore) load(name string) []byte {
	d.ensure()
	p := path.Join(d.dir, name)
	if _, err := os.Stat(p); err != nil {
		return nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		log.Printf("signlink: failed to read file %s: %v", p, err)
		return nil
	}
	return b
}

func (d *diskStore) save(name string, data []byte) {
	d.ensure()
	p := path.Join(d.dir, name)
	if err := os.WriteFile(p, data, 0644); err != nil {
		log.Printf("signlink: failed to write file %s: %v", p, err)
	}
}

// FindCacheDir and GetUID are moved verbatim from signlink.go. They remain
// exported because signlink_test.go:TestFindCacheDir calls them directly.
func FindCacheDir() string {
	var0 := []string{"c:/windows/", "c:/winnt/", "d:/windows/", "d:/winnt/", "e:/windows/", "e:/winnt/", "f:/windows/", "f:/winnt/", "c:/", "~/", "/tmp/", ""}
	var1 := ".file_store_32"
	for i := range len(var0) {
		var3 := var0[i]
		if len(var3) > 0 {
			if _, err := os.Stat(var3); err != nil {
				log.Printf("signlink: couldn't find cache at %s: %v", var3, err)
				continue
			}
		}
		var4 := path.Join(var3, var1)
		_, err := os.Stat(var4)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Printf("signlink: couldn't stat cache at %s: %v", var4, err)
				continue
			}
			err2 := os.Mkdir(var4, 0755)
			if err2 != nil {
				log.Printf("signlink: couldn't create cache at %s: %v", var4, err2)
				continue
			}
		}
		return path.Join(var3, var1, "/")
	}
	return ""
}

func GetUID(arg0 string) int {
	var1 := path.Join(arg0, "uid.dat")
	stat, err := os.Stat(var1)
	if err != nil || stat.Size() < 4 {
		bs := make([]byte, 4)
		binary.BigEndian.PutUint32(bs, uint32(rand.Float64()*9.9999999e7))
		os.WriteFile(var1, bs, 0644)
	}

	var5, err := os.ReadFile(var1)
	if err != nil {
		log.Println("signlink: couldn't read uid.dat")
		return 0
	}
	var6 := binary.BigEndian.Uint32(var5)
	return int(var6 + 1)
}
```

NOTE: copy the *exact current bodies* of `FindCacheDir`/`GetUID` from `signlink.go` before deleting them; the versions above reproduce the current code verbatim — verify they match `git show HEAD:pkg/sign/signlink/signlink.go` if in doubt.

- [ ] **Step 4: Create the browser selector**

Create `pkg/sign/signlink/storage_js.go`:

```go
//go:build js

package signlink

// newCacheStore returns the volatile in-memory store for the browser build.
// Durable IndexedDB-backed storage is sub-project 2, behind the same interface.
func newCacheStore() cacheStore { return newMemStore() }
```

- [ ] **Step 5: Add `var store` and compile assertions to storage.go**

In `pkg/sign/signlink/storage.go`, append after the interface:

```go
// store is the active backend, selected at build time by newCacheStore
// (storage_disk.go / storage_js.go), mirroring the profiling Start() split.
var store cacheStore = newCacheStore()

var _ cacheStore = (*memStore)(nil)
```

(The `*diskStore` assertion is implicit via `newCacheStore` on native; `*memStore` is asserted here because it's the build-neutral type.)

- [ ] **Step 6: Rewrite Run()'s three filesystem touchpoints**

In `pkg/sign/signlink/signlink.go`, in `Run()`:

Replace the startup lines (currently ~126-127):

```go
	var1 := FindCacheDir()
	uid := GetUID(var1)
```

with:

```go
	var1 := store.cacheDir()
	uid := store.uid()
```

Replace the load slot (currently ~178-188, the `case loadReq != "":` body up to the `mu.Lock()`):

```go
		case loadReq != "":
			var buf []byte
			p := path.Join(var1, loadReq)
			if _, err := os.Stat(p); err == nil {
				b, err := os.ReadFile(p)
				if err != nil {
					log.Printf("signlink: failed to read file %s: %v", p, err)
				} else {
					buf = b
				}
			}
			mu.Lock()
```

with:

```go
		case loadReq != "":
			buf := store.load(loadReq)
			mu.Lock()
```

Replace the save slot's write (currently ~194-199):

```go
		case saveReq != "":
			if saveBuf != nil {
				if err := os.WriteFile(path.Join(var1, saveReq), saveBuf[0:saveLen], 0644); err != nil {
					log.Printf("signlink: failed to write file %s: %v", path.Join(var1, saveReq), err)
				}
			}
```

with:

```go
		case saveReq != "":
			if saveBuf != nil {
				store.save(saveReq, saveBuf[0:saveLen])
			}
```

Leave the wave/MIDI path lines (`waveOut = path.Join(var1, saveReq)` / `midiOut = ...`) unchanged — `var1` is now `store.cacheDir()` ("" in browser, the real dir on native).

- [ ] **Step 7: Fix signlink.go imports**

`os`, `encoding/binary`, and `math/rand` are now unused in `signlink.go` (their only users moved to `storage_disk.go`). Remove those three lines from the `signlink.go` import block. Keep `errors` (still used at the `errors.New` in the socket/url path), `path` (wave/MIDI paths), `net`, `strconv`, `net/http`, `io`, `fmt`, `log`, `strings`, `sync`, `time`.

- [ ] **Step 8: Verify native build + tests**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... && TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/sign/signlink/...`
Expected: build OK; all signlink tests PASS (including existing `TestFindCacheDir`, `TestOpenSocket`, race/stress tests, and the new disk/mem tests).

- [ ] **Step 9: Verify the wasm build compiles**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "exit ${PIPESTATUS[0]}"`
Expected: `exit 0`. (`Run()` no longer references the moved functions; `storage_js.go` supplies `newCacheStore`.)

- [ ] **Step 10: Commit**

```bash
git add pkg/sign/signlink/storage.go pkg/sign/signlink/storage_disk.go pkg/sign/signlink/storage_js.go pkg/sign/signlink/storage_disk_test.go pkg/sign/signlink/signlink.go
git commit --no-gpg-sign -m "feat(signlink): route cache I/O through the storage seam"
```

---

## Task 3: Browser WebSocket target derivation + TCP guard

**Files:**
- Create: `pkg/sign/signlink/signlink_socket.go` (neutral — `resolveWSTarget`)
- Create: `pkg/sign/signlink/signlink_socket_test.go` (neutral)
- Create: `pkg/sign/signlink/signlink_socket_native.go` (`//go:build !js`)
- Create: `pkg/sign/signlink/signlink_socket_js.go` (`//go:build js`)
- Modify: `pkg/sign/signlink/signlink.go` (`OpenSocket` default branch → `dialTCP`)
- Modify: `cmd/client/main.go` (call `ConfigureTransport`)

- [ ] **Step 1: Write the failing resolver test**

Create `pkg/sign/signlink/signlink_socket_test.go`:

```go
package signlink

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

func TestResolveWSTarget(t *testing.T) {
	cases := []struct {
		name              string
		hostname, port    string
		protocol          string
		wantKind          clientextras.TransportKind
		wantHost          string
		wantPort          int
	}{
		{"http with explicit port", "localhost", "8080", "http:", clientextras.TransportWS, "localhost", 8080},
		{"https default port", "example.com", "", "https:", clientextras.TransportWSS, "example.com", 443},
		{"http default port", "example.com", "", "http:", clientextras.TransportWS, "example.com", 80},
		{"https explicit port", "10.0.0.1", "443", "https:", clientextras.TransportWSS, "10.0.0.1", 443},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, host, port := resolveWSTarget(c.hostname, c.port, c.protocol)
			if kind != c.wantKind || host != c.wantHost || port != c.wantPort {
				t.Fatalf("got (%v,%q,%d), want (%v,%q,%d)",
					kind, host, port, c.wantKind, c.wantHost, c.wantPort)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/sign/signlink/ -run TestResolveWSTarget -v`
Expected: FAIL — `undefined: resolveWSTarget`.

- [ ] **Step 3: Write the resolver**

Create `pkg/sign/signlink/signlink_socket.go`:

```go
package signlink

import (
	"strconv"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

// resolveWSTarget computes the WebSocket transport target from the browser's
// window.location fields. Pure and build-neutral so it is unit-tested on
// native. An empty portStr means the page is on the scheme's default port, so
// the connection targets 80 (ws) or 443 (wss) — matching Client-TS, which dials
// window.location.host back to the serving origin (ClientStream.ts).
func resolveWSTarget(hostname, portStr, protocol string) (clientextras.TransportKind, string, int) {
	secure := protocol == "https:"
	kind := clientextras.TransportWS
	if secure {
		kind = clientextras.TransportWSS
	}
	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		if secure {
			port = 443
		} else {
			port = 80
		}
	}
	return kind, hostname, port
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/sign/signlink/ -run TestResolveWSTarget -v`
Expected: PASS (4 subtests).

- [ ] **Step 5: Write the native socket file**

Create `pkg/sign/signlink/signlink_socket_native.go`:

```go
//go:build !js

package signlink

import (
	"net"
	"strconv"
	"time"
)

// dialTCP is the native raw-TCP dial used by OpenSocket's default (Java-parity)
// transport. The browser build replaces this with an error (signlink_socket_js.go).
func dialTCP(host string, port int, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), timeout)
}

// ConfigureTransport is a no-op on native: the transport/host are set by the
// command-line host-arg parsing in cmd/client/main.go.
func ConfigureTransport() {}
```

- [ ] **Step 6: Write the browser socket file**

Create `pkg/sign/signlink/signlink_socket_js.go`:

```go
//go:build js

package signlink

import (
	"errors"
	"net"
	"syscall/js"
	"time"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

// dialTCP cannot work in a browser (no raw sockets). OpenSocket should never
// reach it once ConfigureTransport has run, but if it does, fail loudly rather
// than hang.
func dialTCP(host string, port int, timeout time.Duration) (net.Conn, error) {
	return nil, errors.New("signlink: browser build requires a ws:// or wss:// origin (raw TCP is unavailable)")
}

// ConfigureTransport derives the WebSocket target from window.location and
// points the existing WS transport at the serving origin. Called once from
// cmd/client/main.go before any connection attempt. Connecting back to the
// origin (rather than a PortOffset+43594 TCP port) matches Client-TS and avoids
// mixed-content under HTTPS.
func ConfigureTransport() {
	loc := js.Global().Get("location")
	hostname := loc.Get("hostname").String()
	portStr := loc.Get("port").String()
	protocol := loc.Get("protocol").String()

	kind, host, port := resolveWSTarget(hostname, portStr, protocol)
	clientextras.Transport = kind
	clientextras.Host = host
	clientextras.WSPort = port
	clientextras.WSPath = "/"
}
```

- [ ] **Step 7: Point OpenSocket's default branch at dialTCP**

In `pkg/sign/signlink/signlink.go`, in `OpenSocket`, replace the default branch:

```go
	default:
		return net.DialTimeout("tcp", net.JoinHostPort(clientextras.Host, strconv.Itoa(port)), dialTimeout)
```

with:

```go
	default:
		return dialTCP(clientextras.Host, port, dialTimeout)
```

(`net` and `strconv` remain imported in `signlink.go` — both are still used elsewhere: DNS lookups / `net.Conn`, and several `strconv` calls.)

- [ ] **Step 8: Call ConfigureTransport from main.go**

In `cmd/client/main.go`, immediately before the `profiling.Start()` call (after the optional host-arg block that ends with `}`), insert:

```go
	// Browser builds derive the WebSocket target from window.location here;
	// no-op on native, where the transport comes from the host arg above.
	signlink.ConfigureTransport()

```

- [ ] **Step 9: Verify both builds + tests**

Run:
```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... && \
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/sign/signlink/... && \
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm exit ${PIPESTATUS[0]}"
```
Expected: native build OK; signlink tests PASS; `wasm exit 0`.

- [ ] **Step 10: Commit**

```bash
git add pkg/sign/signlink/signlink_socket.go pkg/sign/signlink/signlink_socket_test.go pkg/sign/signlink/signlink_socket_native.go pkg/sign/signlink/signlink_socket_js.go pkg/sign/signlink/signlink.go cmd/client/main.go
git commit --no-gpg-sign -m "feat(signlink): derive browser WS target from origin; guard TCP on wasm"
```

---

## Task 4: Build target, dev server, and docs

**Files:**
- Create: `cmd/wasmserve/main.go`
- Modify: `Makefile`
- Modify: `README.md`

- [ ] **Step 1: Write the static dev server**

Create `cmd/wasmserve/main.go`:

```go
// Command wasmserve serves the gogio js/wasm build for local browser testing.
// Go's net/http maps .wasm to application/wasm, which
// WebAssembly.instantiateStreaming requires (a plain file:// open or a server
// that returns application/octet-stream fails to stream-instantiate).
package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	dir := flag.String("dir", "gio/client", "directory to serve (gogio js output)")
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	log.Printf("wasmserve: serving %s on http://localhost%s", *dir, *addr)
	log.Printf("wasmserve: launch with e.g. http://localhost%s/?argv=10 0 highmem members", *addr)
	log.Fatal(http.ListenAndServe(*addr, http.FileServer(http.Dir(*dir))))
}
```

- [ ] **Step 2: Verify it builds**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./cmd/wasmserve`
Expected: builds, no output.

- [ ] **Step 3: Add Makefile targets**

In `Makefile`, add `wasm` and `wasm-serve` to the `.PHONY` line:

```make
.PHONY: help build run test test-race vet lint fmt check-fmt ci setup clean wasm wasm-serve
```

And add these targets after the `$(BIN):` target:

```make
# Browser build directory (gogio js output: index.html, main.wasm, wasm.js).
WASM_OUT := gio/client

wasm: ## Build the browser (js/wasm) client into gio/client/ via gogio
	go run gioui.org/cmd/gogio -target js -o $(WASM_OUT) $(CMD)

wasm-serve: ## Serve the browser build at http://localhost:8080 (run `make wasm` first)
	go run ./cmd/wasmserve -dir $(WASM_OUT)
```

- [ ] **Step 4: Verify the Makefile targets are wired**

Run: `make help | grep -E 'wasm'`
Expected: both `wasm` and `wasm-serve` appear in the help listing.

- [ ] **Step 5: Document the browser build in README**

In `README.md`, add a section (place it near the existing build/run instructions):

````markdown
## Browser (WebAssembly) build

The client can run in a browser via Gio's `js/wasm` target. The game data and
WebSocket server are expected to be served from the **same origin** as the page;
the client derives its server target from `window.location` automatically.

```bash
# 1. Build the wasm bundle into gio/client/ (needs gogio; pulled via go run).
make wasm

# 2. Serve it locally (maps .wasm to application/wasm, required for streaming).
make wasm-serve

# 3. Open the client, passing the non-host args via the ?argv= query parameter:
#    node-id port-offset lowmem|highmem free|members
#    http://localhost:8080/?argv=10 0 highmem members
```

Notes:
- The server **host/scheme are auto-derived** from the page origin (`ws://` over
  HTTP, `wss://` over HTTPS), so — unlike the desktop build — you do **not** pass
  a host argument.
- Storage is **in-memory only** in this build: the cache and client id do not
  survive a page reload (IndexedDB persistence is planned). 
- Audio is not yet wired for the browser build.
````

- [ ] **Step 6: Commit**

```bash
git add cmd/wasmserve/main.go Makefile README.md
git commit --no-gpg-sign -m "build(wasm): add gogio build target, dev server, and browser docs"
```

---

## Task 5: Full verification

- [ ] **Step 1: Native CI gate**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... && TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./... && TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go vet ./...`
Expected: all PASS, vet clean.

- [ ] **Step 2: WASM build**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "exit ${PIPESTATUS[0]}"`
Expected: `exit 0`.

- [ ] **Step 3: gofmt check**

Run: `make check-fmt`
Expected: no unformatted files.

- [ ] **Step 4: Manual browser smoke (human-run, outside the sandbox)**

This step is performed by the user on the host (the sandbox cannot open a
browser window). Document the result in the PR description.

1. `make wasm && make wasm-serve` (with a WebSocket-capable server reachable on
   the same origin/port, or just verify boot+render if no server is running).
2. Open `http://localhost:8080/?argv=10 0 highmem members`.
3. Confirm: the login screen renders; the browser console shows the WS dialing
   the origin (not a `127.0.0.1:43594` TCP attempt); no `ENOSYS`/cache error
   spam in the console.

- [ ] **Step 5: Final commit (if any formatting fixes were needed)**

```bash
git add -A
git commit --no-gpg-sign -m "chore(wasm): formatting and verification fixups"
```

(Skip if the tree is already clean.)

---

## Notes / deferred to later sub-projects

- **IndexedDB persistence** (sub-project 2): replace `memStore` with an
  IndexedDB-backed `cacheStore` behind this same interface; `Database.ts`
  (`Client-TS`) is the 1:1 reference.
- **Audio** (sub-project 3): SF2 soundfont loading without a real FS, wave-byte
  decoupling from `os.ReadFile`, and the autoplay-gesture `oto.Context.Resume()`.
- **WSPath**: defaulted to `/` here. If the game server expects a specific
  upgrade path, set it in `ConfigureTransport` / confirm against the server repo.
- **os.Exit / lifecycle** (sub-project 4): left as-is; acceptable on wasm.
```

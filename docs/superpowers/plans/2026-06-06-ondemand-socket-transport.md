# OnDemand Socket Transport (Java 274 parity) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the retired ≤254 HTTP `/ondemand.zip` shim in `pkg/jagex2/io/ondemand` with the genuine Java-274 socket OnDemand protocol so the client boots against the Engine-TS 274-GOSCAPE engine.

**Architecture:** The ondemand package keeps its frame-driven `Run()` pump (established WS1 decision) but gains the real wire protocol from `OnDemand.java @32f3062`: lazy socket open with handshake byte `15` on the world port, 4-byte requests, 6-byte response headers with 500-byte part reassembly, a 50-cycle resend walk, 750-cycle stale-connection teardown, and a 500-cycle in-game keepalive. The blocking dial+handshake moves to a one-shot goroutine (Java used a worker thread; our pump runs on the game-loop goroutine and must not stall — cf. the Wayland heartbeat-starvation lesson). Socket I/O reuses `pkg/jagex2/io/clientstream`, which already reproduces Java `InputStream.available()` semantics over a `net.Conn` (eager reader goroutine + ring buffer).

**Tech Stack:** Go 1.26, `net.Pipe` for protocol tests, existing `clientstream`/`datastruct`/`signlink` packages.

---

## Background (root cause, for context)

The P7 host smoke failed with `load error "ondemand"`. Traced root cause: Engine-TS 274-GOSCAPE dropped the `/ondemand.zip` HTTP route *and* its packer step (both exist on ≤254 branches), replacing them with the real Java-274 socket protocol (`src/engine/OnDemand.ts` + `OnDemandThread.ts`, routed off the world TCP port via handshake state). The Go port's WS1 seam still speaks the zip shim → `GET /ondemand.zip` 404s → `FailCount` trips the (correctly ported) 274 guard at `client.go:6294`.

**Java reference (always via `git show`, never the working tree):**

```bash
cd $HOME/Code/github.com/LostCityRS/Client-Java
git show 32f3062:src/main/java/jagex2/io/OnDemand.java
git show 32f3062:src/main/java/jagex2/client/Client.java   # load() wait loops at :5165-5250
```

Key Java line map (OnDemand.java @32f3062): `run()` 380–460 (inner pump 393–404, resend walk 406–425, packetCycle 426–437, message clear 438–441, keepalive 442–456, `od_ex` catch 457–459), `handleQueue` 463–493, `handlePending` 495–523, `handleExtra` 525–582, `read()` 583–670, `validate` 672–690, `send()` 692–733 (handshake 696–707).

## Decisions locked in (do not relitigate during execution)

1. **Frame-driven `Run()` stays.** Java's worker thread + `synchronized` are NOT restored. Everything except the connector goroutine and clientstream's internal goroutines runs on the game-loop goroutine.
2. **Async dial+handshake.** `send()` never blocks: it kicks a one-shot goroutine and polls a buffered channel. Deliberate, documented deviation from Java's synchronous open-on-worker-thread.
3. **`clientstream.ClientStream` is the stream type.** Java used raw socket streams; we need eager-buffer `Available()` semantics (memory: `feedback_porting_inputstream_available`) and clientstream is the project's existing solution. Its 30 s SO_TIMEOUT on blocking reads is an accepted bound (Java's ondemand socket had none and could hang its worker forever).
4. **Boot wait loops get `time.Sleep(20 * time.Millisecond)`.** Java polls each load() wait loop at `Thread.sleep(100L)` (Client.java:5175/5196/5220/5246 @32f3062) while the worker pumps at 20 ms. Go merges poller and worker into one loop, so it sleeps the **worker's** 20 ms cadence — the resend/keepalive/teardown counters (50/500/750 cycles) all assume ~20 ms per `Run()`. Without this, a tight boot loop spins `packetCycle` past 750 in microseconds and churns connections forever. In-game cadence needs nothing: `Loop()` already ticks at 20 ms.
5. **`cache` stays nil at the call site.** Java's `fileStreams` (dat/idx FileStream) was never ported; nil-cache behavior is identical to Java with `fileStreams[0] == null` (every gate verified). Porting FileStream is OUT OF SCOPE.
6. **Out of scope:** FileStream/dat-idx cache, JAGGRAB interplay (none — ondemand uses the world port), wasm browser smoke (transport seam handles ws:// automatically via `signlink.OpenSocket`), the `od_ex` catch-all (`catch (Exception)` has no Go analogue; error paths return explicitly).

## Standing project warnings (from memory / resume notes)

- Build/test prefix: `TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache`
- gopls/IDE diagnostics are systematically FALSE on this branch — trust only real `go build` / `go test`.
- Commits: `git commit --no-gpg-sign`, stage explicit paths only (masked dotfiles + untracked `audit-274/` at repo root).
- Every renamed/restructured site needs a `// Java: <name> (file:lines @32f3062)` comment.
- Branch: work directly on `rev-274`.

## File structure

| File | Change |
|---|---|
| `pkg/jagex2/io/ondemand/ondemand.go` | Transport swap: delete `Downloader`/`zip`/`downloadZip`/no-op `send`; add `App` seam, socket state fields, connector, real `send()`/`read()`, full `Run()` tail |
| `pkg/jagex2/io/ondemand/ondemand_internal_test.go` | Delete 2 zip-transport tests; add `net.Pipe` protocol harness + 10 socket tests |
| `pkg/jagex2/client/client.go` | `onDemandDownloader` → `onDemandApp`; `New()` call site; 4 boot-loop sleeps; comment refresh |

---

### Task 1: Transport seam + send() half

**Files:**
- Modify: `pkg/jagex2/io/ondemand/ondemand.go`
- Modify: `pkg/jagex2/io/ondemand/ondemand_internal_test.go`
- Modify: `pkg/jagex2/client/client.go` (call site + adapter; keeps the tree compiling)

- [ ] **Step 1: Delete the two zip-transport tests**

In `ondemand_internal_test.go`, delete `TestRun_BundleReadEndToEnd` (line ~261) and `TestRead_Archive3PromotedTo93` (line ~309) including their helper-local code (they construct zip bundles / fake Downloaders). Replacements land in Tasks 2–3. Also delete the now-unused `archive/zip` import if nothing else uses it.

- [ ] **Step 2: Write the protocol test harness + failing send tests**

Append to `ondemand_internal_test.go` (adjust the import block: add `errors`, `io`, `net`, `slices`, `sync`, `time`; keep existing imports):

```go
// ---- socket transport test harness ------------------------------------------

// fakeApp satisfies App over a pre-seeded queue of net.Pipe client ends.
// An empty queue makes OpenSocket fail, like an unreachable world server.
type fakeApp struct {
	mu     sync.Mutex
	conns  []net.Conn
	ingame bool
}

func (a *fakeApp) OpenSocket() (net.Conn, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.conns) == 0 {
		return nil, errors.New("fakeApp: no conn available")
	}
	c := a.conns[0]
	a.conns = a.conns[1:]
	return c, nil
}

func (a *fakeApp) InGame() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ingame
}

// odServer drives the server side of one ondemand connection: it validates
// the 15 handshake byte, replies with 8 zero bytes, then records every
// 4-byte request frame it receives (including keepalives).
type odServer struct {
	conn net.Conn
	mu   sync.Mutex
	reqs [][4]byte
}

// startODServer returns a running server and the client end of the pipe.
func startODServer(t *testing.T) (*odServer, net.Conn) {
	t.Helper()
	server, client := net.Pipe()
	t.Cleanup(func() {
		_ = server.Close()
		_ = client.Close()
	})
	s := &odServer{conn: server}
	go s.run()
	return s, client
}

func (s *odServer) run() {
	one := make([]byte, 1)
	if _, err := io.ReadFull(s.conn, one); err != nil || one[0] != 15 {
		return
	}
	if _, err := s.conn.Write(make([]byte, 8)); err != nil {
		return
	}
	for {
		var req [4]byte
		if _, err := io.ReadFull(s.conn, req[:]); err != nil {
			return
		}
		s.mu.Lock()
		s.reqs = append(s.reqs, req)
		s.mu.Unlock()
	}
}

func (s *odServer) requests() [][4]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.reqs)
}

// respond writes one response part: the 6-byte header followed by the chunk.
// size is the TOTAL file size; part selects the 500-byte window.
func (s *odServer) respond(t *testing.T, archive, file, size, part int, chunk []byte) {
	t.Helper()
	hdr := []byte{byte(archive), byte(file >> 8), byte(file), byte(size >> 8), byte(size), byte(part)}
	if _, err := s.conn.Write(hdr); err != nil {
		t.Fatalf("respond header: %v", err)
	}
	if len(chunk) > 0 {
		if _, err := s.conn.Write(chunk); err != nil {
			t.Fatalf("respond chunk: %v", err)
		}
	}
}

// waitFor polls cond (driving step each iteration) until it holds or 5s pass.
func waitFor(t *testing.T, step func(), cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("waitFor: condition not met within 5s")
		}
		step()
		time.Sleep(time.Millisecond)
	}
}

// connectOD drives od.send(probe) until the async dial+handshake completes
// and the stream is attached. The probe request's bytes reach the server
// (possibly more than once); tests must account for them or use a distinct
// file id for assertions.
func connectOD(t *testing.T, od *OnDemand, probe *OnDemandRequest) {
	t.Helper()
	waitFor(t, func() { od.send(probe) }, func() bool { return od.stream != nil })
}

// ---- send() ------------------------------------------------------------------

func TestSend_HandshakeAndUrgentRequestBytes(t *testing.T) {
	server, conn := startODServer(t)
	app := &fakeApp{conns: []net.Conn{conn}}
	od := New(buildMinimalVersionlist(1, 0), app, nil)

	r := newRequest()
	r.Archive = 0
	r.File = 1
	r.Urgent = true
	connectOD(t, od, r)

	// The server's run() already validated the 15 handshake byte (it records
	// nothing otherwise). Now the request frame: [archive, file>>8, file, 2].
	waitFor(t, func() { od.send(r) }, func() bool { return len(server.requests()) >= 1 })
	want := [4]byte{0, 0, 1, 2}
	if got := server.requests()[0]; got != want {
		t.Fatalf("urgent request bytes = %v, want %v", got, want)
	}
	if od.FailCount != -10000 {
		t.Fatalf("FailCount = %d, want -10000 after successful send", od.FailCount)
	}
}

func TestSend_PriorityByteNotUrgent(t *testing.T) {
	server, conn := startODServer(t)
	app := &fakeApp{conns: []net.Conn{conn}} // ingame=false → priority 1
	od := New(buildMinimalVersionlist(1, 0), app, nil)

	r := newRequest()
	r.Archive = 2
	r.File = 3
	r.Urgent = false
	connectOD(t, od, r)
	waitFor(t, func() { od.send(r) }, func() bool { return len(server.requests()) >= 1 })

	want := [4]byte{2, 0, 3, 1} // Java: !urgent && !app.ingame → buf[3] = 1
	if got := server.requests()[0]; got != want {
		t.Fatalf("pre-game request bytes = %v, want %v", got, want)
	}
}

func TestSend_DialFailureIncrementsFailCount(t *testing.T) {
	app := &fakeApp{} // no conns → OpenSocket errors
	od := New(buildMinimalVersionlist(1, 0), app, nil)

	r := newRequest()
	r.Archive = 0
	r.File = 1
	// First send kicks the connector; subsequent sends poll its failure.
	waitFor(t, func() { od.send(r) }, func() bool { return od.FailCount >= 1 })
	if od.stream != nil {
		t.Fatal("stream attached despite dial failure")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
cd $HOME/Code/github.com/zsrv/goscape-client
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go test ./pkg/jagex2/io/ondemand/ -run 'TestSend' -v
```

Expected: compile FAILURE (`App`, `od.stream`, `od.send` with new semantics undefined).

- [ ] **Step 4: Implement the seam + send() half in ondemand.go**

4a. Replace the package doc (current lines 1–13) with:

```go
// Package ondemand ports Java's jagex2.io.OnDemand.
// Types, versionlist parse, Validate, getters, the request/cycle/prefetch
// state machine, and the socket transport (send / read part-reassembly) all
// live here.
//
// Transport: the genuine Java 274 socket protocol (OnDemand.java @32f3062) —
// handshake byte 15 on the world port, 4-byte requests, 6-byte response
// headers with 500-byte part reassembly. The pre-274 "modernized"
// /ondemand.zip bundle shim (a Client-TS-era convention served by Engine-TS
// ≤254) was removed when Engine-TS 274 dropped the route in favour of the
// real protocol.
//
// Threading: Java runs OnDemand on a worker thread (startThread(this, 2),
// OnDemand.java:216) sleeping 20|50 ms between pump iterations; this port
// drives Run() once per game frame instead (established WS1 decision), so
// Java's synchronized blocks are dropped — all state lives on the game-loop
// goroutine. Two exceptions: clientstream's internal reader/writer
// goroutines (own only stream internals), and the one-shot dial+handshake
// connector goroutine (see send), which would otherwise stall the frame.
package ondemand
```

4b. Update the import block: remove `archive/zip`; add `net`, `time`, and `"github.com/zsrv/goscape-client/pkg/jagex2/io/clientstream"`.

4c. Update the `Cycle` field comment on `OnDemandRequest` (it goes live in Task 3):

```go
	// Java: nb.l — frames since this pending request was last (re)sent; the
	// Run() resend walk re-sends and zeroes it past 50 (OnDemand.java:406-425
	// @32f3062), and read() zeroes it on any response at-or-after this
	// request in the pending list (OnDemand.java:599-606).
	Cycle int
```

4d. Replace the `Downloader` interface (keep `Cache` and `Archive`) with:

```go
// App is the surface OnDemand needs from the client (Java: ub.q app —
// init() receives the Client itself; only these two members are read by
// the transport).
type App interface {
	// OpenSocket dials the world server for the ondemand service.
	// Java: app.openSocket(Client.portOff + 43594) (OnDemand.java:700
	// @32f3062) — portOff+43594 is the game port.
	OpenSocket() (net.Conn, error)
	// InGame mirrors Java app.ingame: read by send()'s priority byte and
	// the Run() keepalive gate.
	InGame() bool
}
```

4e. In the `OnDemand` struct: delete the `zip map[string][]byte` field and the `dl Downloader` field; change the `current` comment to `// Java: ub.I — the pending request read() is currently reassembling.`; replace the whole `FailCount` comment with:

```go
	// FailCount counts consecutive failed transport attempts; Client.load's
	// first two on-demand wait loops bail to showLoadError("ondemand") when
	// it exceeds 3. Java: failCount (OnDemand.java:105 @32f3062, NEW in 274)
	// — set to -10000 after a successful request write (OnDemand.java:721)
	// and incremented in send()'s IOException catch (:731), which here also
	// covers a failed dial or handshake.
	FailCount int
```

and add the socket-transport state block (after `FailCount`, before the seams):

```go
	// ---- socket transport state (Java: ub.E/F/G/J/K/L/N/O/P) ---------------

	// Java: ub.L — 500-byte wire scratch (request frames, response headers,
	// orphaned parts whose request is no longer pending)
	buf [500]byte
	// Java: ub.J — byte offset of the current part within current.Data
	partOffset int
	// Java: ub.K — bytes of the current part still unread (0 = expecting a
	// 6-byte header next)
	partAvailable int
	// Java: ub.N — frames the pending list has gone unanswered; > 750 tears
	// the connection down so send() redials
	packetCycle int
	// Java: ub.O — frames since the last request write; > 500 emits the
	// keepalive frame while in game
	noTimeoutCycle int
	// Java: ub.P — wall-clock ms of the last dial attempt; enforces the 4 s
	// redial backoff (274 tightened 5000→4000, OnDemand.java:697)
	socketOpenTime int64
	// stream wraps the ondemand socket. Java holds the raw Socket plus its
	// two streams (ub.E/F/G); the Go port reuses clientstream.ClientStream
	// because it reproduces InputStream.available()'s eager-buffering
	// semantics over a net.Conn (see that package's doc) with the same
	// read/readFully/write surface. nil = not connected (Java: socket==null).
	stream *clientstream.ClientStream
	// connecting is non-nil while the one-shot dial+handshake goroutine is
	// in flight; it delivers exactly one result, polled by send(). Java
	// performs the open synchronously on its worker thread
	// (OnDemand.java:696-707); this port is frame-driven on the game-loop
	// goroutine, so the blocking dial + 8-byte drain must not run inline.
	connecting chan connectResult
```

In the seams block, replace `dl Downloader` with `app App` (the `// Java: app.ingame used only by the socket heartbeat — not ported.` comment line is deleted; it is ported now).

4f. Change `New` (signature + doc):

```go
// New allocates an OnDemand, wires seams, and calls Unpack.
// Java: OnDemand constructor + init() (OnDemand.java:133-217 @32f3062).
// init()'s trailing startThread call is not ported — Run() is driven once
// per frame instead (see the package doc).
func New(versionlist Archive, app App, cache Cache) *OnDemand {
	od := &OnDemand{
		requests:   datastruct.NewLinkList2[*OnDemandRequest](),
		queue:      datastruct.NewLinkList[*OnDemandRequest](),
		missing:    datastruct.NewLinkList[*OnDemandRequest](),
		pending:    datastruct.NewLinkList[*OnDemandRequest](),
		completed:  datastruct.NewLinkList[*OnDemandRequest](),
		prefetches: datastruct.NewLinkList[*OnDemandRequest](),
		running:    true,
		app:        app,
		cache:      cache,
	}
	od.Unpack(versionlist)
	return od
}
```

4g. Replace the no-op `send`, the zip `read` body, and `downloadZip` (delete `downloadZip` entirely) with:

```go
// read consumes ondemand socket responses; the full part-reassembly port of
// Java read() (OnDemand.java:583-670 @32f3062) lands with the next commit.
func (od *OnDemand) read() {}

// connectResult carries the outcome of one async dial+handshake attempt.
type connectResult struct {
	stream *clientstream.ClientStream
	err    error
}

// openStream dials the world server and performs the ondemand handshake:
// write service byte 15, drain the 8 response bytes.
// Java: OnDemand.java:700-707 @32f3062 — synchronous on the worker thread
// there; here it runs on the connector goroutine (see the connecting field).
// Java ignores the drained bytes' values (including a -1 EOF), so only
// stream errors abort; clientstream's 30 s read bound turns the half-open
// hang Java could suffer into a counted failure.
func openStream(app App) connectResult {
	conn, err := app.OpenSocket()
	if err != nil {
		return connectResult{err: err}
	}
	cs := clientstream.NewClientStream(conn)
	hello := []byte{15}
	if err := cs.Write(hello, 1, 0); err != nil {
		cs.Close()
		return connectResult{err: err}
	}
	for range 8 {
		if _, err := cs.Read(); err != nil {
			cs.Close()
			return connectResult{err: err}
		}
	}
	return connectResult{stream: cs}
}

// closeStream tears down the ondemand connection. Java inlines this
// socket.close() + socket/in/out=null + partAvailable=0 sequence at all
// three failure sites (OnDemand.java:428-435 run, :661-668 read,
// :723-730 send).
func (od *OnDemand) closeStream() {
	if od.stream != nil {
		od.stream.Close()
		od.stream = nil
	}
	od.partAvailable = 0
}

// send transmits one 4-byte request frame, lazily (re)opening the ondemand
// socket. Java: send(OnDemandRequest) (OnDemand.java:692-733 @32f3062).
// Java's single IOException catch (close + failCount++) maps onto the
// per-call error paths below; the dial+handshake runs on a one-shot
// goroutine instead of inline (see the connecting field), during which
// send() returns early — the request stays pending and the Run() resend
// walk retries it once the connection lands.
//
// Note clientstream.Write is asynchronous (ring buffer + writer goroutine):
// a wire failure surfaces on a LATER Write call, exactly like Java's
// buffered OutputStream surfacing the IOException on a later flush.
func (od *OnDemand) send(r *OnDemandRequest) {
	if od.stream == nil {
		if od.connecting != nil {
			select {
			case res := <-od.connecting:
				od.connecting = nil
				if res.err != nil {
					od.FailCount++ // Java: failCount++ in the catch (OnDemand.java:731)
					return
				}
				od.stream = res.stream
				od.packetCycle = 0 // Java: OnDemand.java:707
			default:
				return // dial+handshake still in flight
			}
		} else {
			// Java: 4 s redial backoff (OnDemand.java:696-699).
			now := time.Now().UnixMilli() // Java: System.currentTimeMillis()
			if now-od.socketOpenTime < 4000 {
				return
			}
			od.socketOpenTime = now
			ch := make(chan connectResult, 1)
			od.connecting = ch
			app := od.app
			go func() { ch <- openStream(app) }()
			return
		}
	}

	// Java: OnDemand.java:709-719 — [archive, file>>8, file, priority].
	od.buf[0] = byte(r.Archive)
	od.buf[1] = byte(r.File >> 8)
	od.buf[2] = byte(r.File)
	if r.Urgent {
		od.buf[3] = 2
	} else if od.app.InGame() {
		od.buf[3] = 0
	} else {
		od.buf[3] = 1
	}
	if err := od.stream.Write(od.buf[:], 4, 0); err != nil {
		od.closeStream()
		od.FailCount++
		return
	}
	od.noTimeoutCycle = 0
	od.FailCount = -10000 // Java: OnDemand.java:720-721
}
```

- [ ] **Step 5: Update the client call site (tree must compile)**

In `pkg/jagex2/client/client.go`:

5a. Replace the `onDemandDownloader` type and its `Get` method (lines ~7325–7335) with:

```go
// onDemandApp adapts the client to ondemand.App.
// Java: OnDemand.app (ub.q) — init() receives the Client itself
// (OnDemand.java:214 @32f3062).
type onDemandApp struct{ c *Client }

// OpenSocket dials the world server for the ondemand service.
// Java: app.openSocket(Client.portOff + 43594) (OnDemand.java:700 @32f3062)
// — portOff+43594 is the game port; the Go client carries the full port in
// clientextras.WorldPort (from -world-server).
func (a onDemandApp) OpenSocket() (net.Conn, error) {
	return a.c.OpenSocket(clientextras.WorldPort)
}

// InGame mirrors Java app.ingame.
func (a onDemandApp) InGame() bool { return a.c.InGame }
```

5b. Change the constructor call (line ~6269) to:

```go
	c.OnDemand = ondemand.New(jagVersionList, onDemandApp{c}, nil)
```

5c. The comment block above the boot loops (lines ~6273–6277, beginning `// Boot on-demand request loops:`) still claims "Thread.sleep() calls are omitted"; leave it for now — Task 3 rewrites it together with the sleeps.

- [ ] **Step 6: Build and run the new tests**

```bash
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache go build ./... \
&& TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go test ./pkg/jagex2/io/ondemand/ -race -count=1 -v
```

Expected: build OK; `TestSend_*` PASS; all pre-existing ondemand tests PASS.

- [ ] **Step 7: Commit**

```bash
git add pkg/jagex2/io/ondemand/ondemand.go pkg/jagex2/io/ondemand/ondemand_internal_test.go pkg/jagex2/client/client.go
git commit --no-gpg-sign -m "feat(rev-274): ondemand socket transport — App seam + send half

Engine-TS 274 dropped the /ondemand.zip shim for the real Java-274 socket
protocol; this replaces the WS1 zip transport's seam and send() half
(handshake 15 on the world port, 4-byte request frames, failCount
accounting per OnDemand.java:692-733 @32f3062). read() is a stub until the
next commit. Dial+handshake runs on a one-shot connector goroutine so the
frame-driven Run() never stalls the game loop."
```

---

### Task 2: read() — header parse + 500-byte part reassembly

**Files:**
- Modify: `pkg/jagex2/io/ondemand/ondemand.go`
- Modify: `pkg/jagex2/io/ondemand/ondemand_internal_test.go`

- [ ] **Step 1: Write the failing read tests**

Append to `ondemand_internal_test.go` (add `"github.com/zsrv/goscape-client/pkg/sign/signlink"` to its imports for the rejection test's flag handling):

```go
// ---- read() ------------------------------------------------------------------

// pushPending crafts a request directly onto the pending list, as
// handlePending would after a cache miss.
func pushPending(od *OnDemand, archive, file int, urgent bool) *OnDemandRequest {
	r := newRequest()
	r.Archive = archive
	r.File = file
	r.Urgent = urgent
	od.pending.Push(r.node.Linkable)
	return r
}

// newConnectedOD returns an OnDemand with an attached stream and its server.
func newConnectedOD(t *testing.T) (*OnDemand, *odServer) {
	t.Helper()
	server, conn := startODServer(t)
	app := &fakeApp{conns: []net.Conn{conn}}
	od := New(buildMinimalVersionlist(1, 0), app, nil)
	probe := newRequest()
	probe.Archive = 0
	probe.File = 0
	connectOD(t, od, probe)
	return od, server
}

func TestRead_SinglePartCompletion(t *testing.T) {
	od, server := newConnectedOD(t)
	r := pushPending(od, 1, 7, true)

	payload := bytes.Repeat([]byte{0xAB}, 300)
	server.respond(t, 1, 7, 300, 0, payload)

	waitFor(t, func() { od.Run() }, func() bool { return od.completed.Head() != nil })
	if !bytes.Equal(r.Data, payload) {
		t.Fatalf("reassembled %d bytes, want 300 matching payload", len(r.Data))
	}
	if got := od.completed.Head().Value; got != r {
		t.Fatalf("completed head = %+v, want the pending request", got)
	}
}

func TestRead_MultiPartReassembly(t *testing.T) {
	od, server := newConnectedOD(t)
	r := pushPending(od, 0, 9, true)

	payload := make([]byte, 700)
	for i := range payload {
		payload[i] = byte(i * 31)
	}
	server.respond(t, 0, 9, 700, 0, payload[:500])
	server.respond(t, 0, 9, 700, 1, payload[500:])

	waitFor(t, func() { od.Run() }, func() bool { return od.completed.Head() != nil })
	if !bytes.Equal(r.Data, payload) {
		t.Fatal("multi-part reassembly mismatch")
	}
}

func TestRead_RejectionDeliversNilData(t *testing.T) {
	// The rejection path calls signlink.ReportErrorFunc, which defaults to
	// enabled (signlink.go:79) and would block forever on OpenURL's
	// cond.Wait — the signlink polling goroutine (StartPriv) is not running
	// in unit tests. Disable it for this test.
	old := signlink.ReportError
	signlink.ReportError = false
	t.Cleanup(func() { signlink.ReportError = old })

	od, server := newConnectedOD(t)
	r := pushPending(od, 2, 4, true)

	server.respond(t, 2, 4, 0, 0, nil) // size 0 = server rejection

	waitFor(t, func() { od.Run() }, func() bool { return od.completed.Head() != nil })
	if r.Data != nil {
		t.Fatalf("rejected request carries %d bytes, want nil", len(r.Data))
	}
}

func TestRead_MissingStartOfFileTearsDownStream(t *testing.T) {
	od, server := newConnectedOD(t)
	pushPending(od, 0, 5, true)

	// part 1 with no prior part 0 → Java throws IOException("missing start
	// of file") → its catch closes the socket.
	server.respond(t, 0, 5, 700, 1, bytes.Repeat([]byte{1}, 200))

	waitFor(t, func() { od.Run() }, func() bool { return od.stream == nil })
}

func TestRead_Archive3PromotedTo93(t *testing.T) {
	od, server := newConnectedOD(t)
	r := pushPending(od, 3, 6, false) // non-urgent map fetch

	server.respond(t, 3, 6, 100, 0, bytes.Repeat([]byte{7}, 100))

	waitFor(t, func() { od.Run() }, func() bool { return od.completed.Head() != nil })
	if r.Archive != 93 || !r.Urgent {
		t.Fatalf("archive=%d urgent=%v, want 93/true promotion", r.Archive, r.Urgent)
	}
}

func TestRead_OrphanResponseDrainsToScratch(t *testing.T) {
	od, server := newConnectedOD(t)

	// Response for a (archive, file) that is not pending: header parsed,
	// part drained into the scratch buffer, nothing completed, stream alive.
	server.respond(t, 1, 99, 50, 0, bytes.Repeat([]byte{3}, 50))

	// Then a real request still completes over the same connection.
	r := pushPending(od, 1, 7, true)
	payload := bytes.Repeat([]byte{0xCD}, 80)
	server.respond(t, 1, 7, 80, 0, payload)

	waitFor(t, func() { od.Run() }, func() bool { return od.completed.Head() != nil })
	if od.stream == nil {
		t.Fatal("stream torn down by orphan response")
	}
	if !bytes.Equal(r.Data, payload) {
		t.Fatal("real request corrupted by orphan response")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go test ./pkg/jagex2/io/ondemand/ -race -count=1 -run 'TestRead' -v
```

Expected: FAIL — `waitFor: condition not met` (read() is a stub), after the `if od.stream != nil` gate from Step 3 exists; before that, possibly a nil deref. Either failure mode is fine; what matters is they pass after Step 3.

- [ ] **Step 3: Implement read() and gate its call site**

3a. In `Run()`, change the inner-loop read call to Java's null-gated form:

```go
		od.handleExtras()
		// Java: if (in != null) read() (OnDemand.java:401-403)
		if od.stream != nil {
			od.read()
		}
```

3b. Replace the `read` stub with the full port (add `strconv` and `"github.com/zsrv/goscape-client/pkg/sign/signlink"` to the imports):

```go
// read consumes one response header and/or one ≤500-byte part from the
// ondemand socket, reassembling parts into current.Data and routing
// completions. Java: read() (OnDemand.java:583-670 @32f3062). Java wraps the
// body in one IOException catch (close + reset, :661-668); each Go error
// path calls closeStream instead. Both gates compare against the available
// count measured once at entry, exactly like Java's single
// in.available() snapshot.
func (od *OnDemand) read() {
	avail, _ := od.stream.Available() // clientstream.Available never errors

	if od.partAvailable == 0 && avail >= 6 {
		od.active = true

		// Java: 6-byte header [archive, file>>8, file, size>>8, size, part]
		// (OnDemand.java:588-597).
		if err := od.stream.ReadFully(od.buf[:], 0, 6); err != nil {
			od.closeStream()
			return
		}
		archive := int(od.buf[0]) & 0xFF
		file := (int(od.buf[1])&0xFF)<<8 + int(od.buf[2])&0xFF
		size := (int(od.buf[3])&0xFF)<<8 + int(od.buf[4])&0xFF
		part := int(od.buf[5]) & 0xFF

		// Find the matching pending request; every request from the match
		// onward gets its resend counter reset (Java: OnDemand.java:599-606).
		od.current = nil
		for n := od.pending.Head(); n != nil; n = od.pending.Next() {
			if n.Value.Archive == archive && n.Value.File == file {
				od.current = n.Value
			}
			if od.current != nil {
				n.Value.Cycle = 0
			}
		}

		if od.current != nil {
			od.packetCycle = 0
			if size == 0 {
				// Server rejected the request (Java: OnDemand.java:607-620).
				signlink.ReportErrorFunc("Rej: " + strconv.Itoa(archive) + "," + strconv.Itoa(file))
				od.current.Data = nil
				if od.current.Urgent {
					od.completed.Push(od.current.node.Linkable)
				} else {
					od.current.node.Unlink()
				}
				od.current = nil
			} else {
				if od.current.Data == nil && part == 0 {
					od.current.Data = make([]byte, size)
				}
				if od.current.Data == nil && part != 0 {
					// Java: throw new IOException("missing start of file")
					// (OnDemand.java:626) — lands in read()'s catch.
					od.closeStream()
					return
				}
			}
		}

		// Java: OnDemand.java:629-634 — runs whether or not a request
		// matched, so orphaned parts are measured for draining below.
		od.partOffset = part * 500
		od.partAvailable = 500
		if od.partAvailable > size-part*500 {
			od.partAvailable = size - part*500
		}
	}

	if od.partAvailable > 0 && avail >= od.partAvailable {
		od.active = true

		// Orphaned parts (no matching request) drain into the scratch
		// buffer (Java: OnDemand.java:637-642).
		dst := od.buf[:]
		off := 0
		if od.current != nil {
			dst = od.current.Data
			off = od.partOffset
		}
		if err := od.stream.ReadFully(dst, off, od.partAvailable); err != nil {
			od.closeStream()
			return
		}

		// Java: OnDemand.java:644-658 — completion when this part reaches
		// the end of the buffer.
		if od.partAvailable+od.partOffset >= len(dst) && od.current != nil {
			if od.cache != nil {
				od.cache.Write(od.current.Archive+1, od.current.File, dst)
			}
			// archive-3 → 93 promotion: a non-urgent map fetch becomes an
			// urgent archive-93 completion (Java: OnDemand.java:648-651).
			if !od.current.Urgent && od.current.Archive == 3 {
				od.current.Urgent = true
				od.current.Archive = 93
			}
			if od.current.Urgent {
				od.completed.Push(od.current.node.Linkable)
			} else {
				od.current.node.Unlink()
			}
		}
		od.partAvailable = 0
	}
}
```

Note: `od.current` deliberately survives between read() calls during a
multi-part file and is only reset when the next header is parsed — Java
behaves identically (current is field ub.I, nulled at :598, never at exit).

- [ ] **Step 4: Run tests to verify they pass**

```bash
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go test ./pkg/jagex2/io/ondemand/ -race -count=1 -v
```

Expected: all PASS (including Task 1's and the pre-existing tests).

- [ ] **Step 5: Commit**

```bash
git add pkg/jagex2/io/ondemand/ondemand.go pkg/jagex2/io/ondemand/ondemand_internal_test.go
git commit --no-gpg-sign -m "feat(rev-274): ondemand socket transport — read() part reassembly

Full port of OnDemand.read() @32f3062: 6-byte headers, 500-byte part
windows into current.Data, resend-counter resets along the pending list,
size-0 rejection (Rej: reporterror), missing-start teardown, orphan-part
scratch drain, and the archive-3→93 promotion (which the zip shim had
borrowed from Client-TS — now at its Java-native site)."
```

---

### Task 3: Run() tail — resend walk, packetCycle, keepalive — and boot pacing

**Files:**
- Modify: `pkg/jagex2/io/ondemand/ondemand.go`
- Modify: `pkg/jagex2/io/ondemand/ondemand_internal_test.go`
- Modify: `pkg/jagex2/client/client.go` (4 boot-loop sleeps + comment)

- [ ] **Step 1: Write the failing Run-tail tests**

Append to `ondemand_internal_test.go` (add `"compress/gzip"` usage is already imported):

```go
// ---- Run() tail --------------------------------------------------------------

func TestRun_ResendAfter50Cycles(t *testing.T) {
	od, server := newConnectedOD(t)
	pushPending(od, 0, 5, true)

	// 52 frames with no response → the resend walk re-sends once at >50.
	for range 52 {
		od.Run()
	}
	waitFor(t, func() {}, func() bool {
		return slices.Contains(server.requests(), [4]byte{0, 0, 5, 2})
	})
}

func TestRun_PacketCycleTearsDownAfter750(t *testing.T) {
	od, server := newConnectedOD(t)
	_ = server
	pushPending(od, 0, 5, true)

	for range 751 {
		od.Run()
	}
	if od.stream != nil {
		t.Fatal("stream still attached after 751 unanswered frames")
	}
}

func TestRun_KeepaliveAfter500IdleCyclesInGame(t *testing.T) {
	server, conn := startODServer(t)
	app := &fakeApp{conns: []net.Conn{conn}, ingame: true}
	od := New(buildMinimalVersionlist(1, 0), app, nil)
	probe := newRequest()
	probe.Archive = 0
	probe.File = 0
	connectOD(t, od, probe)

	// No pending requests → packetCycle stays 0, noTimeoutCycle climbs.
	for range 501 {
		od.Run()
	}
	waitFor(t, func() {}, func() bool {
		return slices.Contains(server.requests(), [4]byte{0, 0, 0, 10})
	})
}

func TestRun_MessageClearedWhenPendingEmpty(t *testing.T) {
	od, _ := newConnectedOD(t)
	od.message = "Loading extra files - 50%"
	od.Run()
	if od.message != "" {
		t.Fatalf("message = %q, want cleared with empty pending", od.message)
	}
}

// ---- end to end ---------------------------------------------------------------

func TestEndToEnd_RequestRunCycle(t *testing.T) {
	server, conn := startODServer(t)
	app := &fakeApp{conns: []net.Conn{conn}}
	od := New(buildMinimalVersionlist(1, 0), app, nil)

	// Wire payload = gzip(content) + 2-byte version trailer, as served and
	// cached; Cycle() strips the trailer and gunzips.
	content := []byte("the quick brown fox")
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	if _, err := zw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	wire := append(gz.Bytes(), 0, 1) // version trailer (unchecked off-cache)

	od.Request(0, 0)
	// Pump until the request frame reaches the server (connect + resend),
	// then answer it and pump to completion.
	waitFor(t, func() { od.Run() }, func() bool { return len(server.requests()) >= 1 })
	server.respond(t, 0, 0, len(wire), 0, wire)

	var got *OnDemandRequest
	waitFor(t, func() { od.Run() }, func() bool {
		got = od.Cycle()
		return got != nil
	})
	if got.Archive != 0 || got.File != 0 || !bytes.Equal(got.Data, content) {
		t.Fatalf("end-to-end: archive=%d file=%d data=%q", got.Archive, got.File, got.Data)
	}
	if od.Remaining() != 0 {
		t.Fatalf("Remaining = %d, want 0", od.Remaining())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go test ./pkg/jagex2/io/ondemand/ -race -count=1 -run 'TestRun_|TestEndToEnd' -v
```

Expected: `TestRun_MessageClearedWhenPendingEmpty` may already pass (the old
else-half is in place); the resend/teardown/keepalive/end-to-end tests FAIL
(no resend walk → request frame never reaches the wire).

- [ ] **Step 3: Implement the full Run() tail**

Replace everything in `Run()` after the inner `for i := 0; i < 100 && od.active; i++ { ... }` loop (i.e. the current `// Java: OnDemand.java:405-441 …` comment and the `if od.pending.Head() == nil { od.message = "" }` block) with:

```go
	// Resend walk (Java: OnDemand.java:406-425): urgent pending requests are
	// re-sent every 50 frames; only if none are urgent, all pending are.
	resent := false // Java: var3 — true while ANY pending request exists
	for n := od.pending.Head(); n != nil; n = od.pending.Next() {
		if n.Value.Urgent {
			resent = true
			n.Value.Cycle++
			if n.Value.Cycle > 50 {
				n.Value.Cycle = 0
				od.send(n.Value)
			}
		}
	}
	if !resent {
		for n := od.pending.Head(); n != nil; n = od.pending.Next() {
			resent = true
			n.Value.Cycle++
			if n.Value.Cycle > 50 {
				n.Value.Cycle = 0
				od.send(n.Value)
			}
		}
	}

	if resent {
		// Java: OnDemand.java:426-437 — tear down a connection that has
		// gone 750 frames without answering anything pending; send() then
		// redials past the 4 s backoff.
		od.packetCycle++
		if od.packetCycle > 750 {
			od.closeStream()
		}
	} else {
		// Java: OnDemand.java:438-441 — idle: reset staleness and clear the
		// "Loading extra files" status line.
		od.packetCycle = 0
		od.message = ""
	}

	// Keepalive (Java: OnDemand.java:442-456): every 500 idle frames while
	// in game, write the [0,0,0,10] no-op frame so the server keeps the
	// connection open. With a nil cache, Java's fileStreams[0]==null half of
	// the gate is always true.
	if od.app.InGame() && od.stream != nil && (od.topPriority > 0 || od.cache == nil) {
		od.noTimeoutCycle++
		if od.noTimeoutCycle > 500 {
			od.noTimeoutCycle = 0
			od.buf[0] = 0
			od.buf[1] = 0
			od.buf[2] = 0
			od.buf[3] = 10
			if err := od.stream.Write(od.buf[:], 4, 0); err != nil {
				od.packetCycle = 5000 // Java: forces the >750 teardown next frame
			}
		}
	}
```

Also update `Run`'s doc comment to:

```go
// Run pumps the request state machine once: drain incoming requests, promote
// misses to pending (sending each), service socket responses, re-send stale
// pending requests, and keep the connection alive. It is called once per
// game frame (≈20 ms); Java ran the same body on a worker thread sleeping
// 20|50 ms per iteration (OnDemand.java:380-460 @32f3062), so the 50/500/750
// cycle thresholds carry identical wall-clock meaning. Java's catch(Exception)
// → reporterror("od_ex") wrapper has no Go analogue — error paths return
// explicitly inside send()/read().
func (od *OnDemand) Run() {
```

- [ ] **Step 4: Add the four boot-loop sleeps in client.go**

4a. Replace the comment block at ~6273–6277 (`// Boot on-demand request loops: … a bare loop is correct and faithful.`) with:

```go
	// Boot on-demand request loops: MIDI, animations, flagged models, maps.
	// Java: Client.load (Client.java:5165-5250 @32f3062). Java polls each
	// wait loop at Thread.sleep(100) while the OnDemand worker thread pumps
	// at 20 ms; here Run() is driven inside OnDemandLoop() on this goroutine
	// (no worker), so each wait loop sleeps the WORKER's 20 ms cadence — the
	// socket transport's resend/keepalive/teardown counters (50/500/750
	// cycles) assume ~20 ms per Run(). A bare loop would spin packetCycle
	// past 750 in microseconds and churn connections forever.
```

4b. In each of the four wait loops (`for c.OnDemand.Remaining() > 0 {` at ~6288, ~6306, ~6327, ~6354), add as the FIRST statement of the loop body:

```go
		time.Sleep(20 * time.Millisecond) // Java: Thread.sleep(100L) poll (Client.java:5175/5196/5220/5246 @32f3062); 20 ms = the worker cadence, see above
```

(Keep each loop's existing body — progress MessageBox, `c.OnDemandLoop()`, FailCount guard — unchanged. `time` is already imported.)

- [ ] **Step 5: Run the full package tests + build**

```bash
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache go build ./... \
&& TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go test ./pkg/jagex2/io/ondemand/ -race -count=1 -v
```

Expected: build OK, all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/jagex2/io/ondemand/ondemand.go pkg/jagex2/io/ondemand/ondemand_internal_test.go pkg/jagex2/client/client.go
git commit --no-gpg-sign -m "feat(rev-274): ondemand socket transport — run-loop tail + boot pacing

Ports the rest of OnDemand.run() @32f3062: 50-cycle resend walk (urgent
first), 750-cycle stale-connection teardown, 500-cycle in-game keepalive
frame [0,0,0,10], and the idle message clear (now at its Java-native site).
The four Client.load wait loops gain 20 ms sleeps: Java paced these
counters from its worker thread; a bare Go loop would hit the 750-cycle
teardown in microseconds."
```

---

### Task 4: Documentation sweep + full gates

**Files:**
- Modify: `pkg/jagex2/io/ondemand/ondemand.go` (stale comments only)
- Modify: `pkg/jagex2/client/client.go` (stale comments only)

- [ ] **Step 1: Sweep stale shim references**

```bash
grep -rn "ondemand.zip\|modernized\|Downloader\|downloadZip\|WS1" pkg/jagex2/io/ondemand/ pkg/jagex2/client/client.go
```

For each hit, update or delete so no comment claims the zip transport exists. Known sites (verify the grep finds no others):
- `ondemand.go`: `active`/`cycle`/`importantCount` field comments cite "Client-TS" semantics — re-cite Java (`ub.s` active, `ub.Q` cycle, `ub.t/u` urgentCount/requestCount; note our `importantCount` keeps its established Go name with a `// Java: urgentCount` tag). `Cycle()` method doc keeps its Client-TS gunzip note (still true — we slice the trailer where Java gunzips over it). `read()`/`send()`/`handleQueue()`/`handlePending()`/`handleExtras()` docs: drop "Client-TS:" primary cites in favour of the Java lines given in Tasks 2–3 where not already done.
- `ondemand.go` `Request()` doc: the "synchronized blocks are dropped" sentence stays (still true; rationale now lives in the package doc).
- `client.go` `OnDemandLoop` doc (~9820): rewrite the ordering note: "The Java worker thread pumps I/O continuously and Client.onDemandLoop only drains cycle(); with no worker, Run() is pumped here first, then the drain — same data flow, one goroutine. Java: Client.onDemandLoop (Client.java:2248 @2e62978)."
- `client.go` ShowLoadError / boot-guard comments: no change needed (verify they don't mention the zip).

- [ ] **Step 2: Verify no stray references remain**

```bash
grep -rn "ondemand.zip" pkg/ cmd/ && echo "FAIL: stale refs" || echo "clean"
```

Expected: `clean`.

- [ ] **Step 3: Full gate run**

```bash
cd $HOME/Code/github.com/zsrv/goscape-client
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache go build ./... \
&& TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache go vet ./... \
&& gofmt -l pkg/ cmd/ \
&& TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache go test ./... -race -count=1 \
&& TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./... \
&& TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache GOLANGCI_LINT_CACHE=/tmp/claude-1000/lint-cache \
  go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run --max-issues-per-linter=0 --max-same-issues=0 ./...
```

Expected: every stage clean; `gofmt -l` prints nothing; lint reports 0 issues.

- [ ] **Step 4: Commit**

```bash
git add pkg/jagex2/io/ondemand/ondemand.go pkg/jagex2/client/client.go
git commit --no-gpg-sign -m "docs(rev-274): ondemand socket transport comment sweep

Retires every remaining reference to the /ondemand.zip shim and re-cites
the transport-adjacent comments against OnDemand.java @32f3062."
```

---

## Self-review checklist (run after writing code, before claiming done)

- [ ] Every Java transport behavior has a port: handshake(15)+8-drain ✓ Task 1, 4-byte frames + priority byte ✓ Task 1, failCount ±  ✓ Task 1, 4 s backoff ✓ Task 1, header parse + cycle-reset walk ✓ Task 2, rejection ✓ Task 2, missing-start ✓ Task 2, part reassembly + orphan drain ✓ Task 2, cache write + 3→93 ✓ Task 2, resend walk ✓ Task 3, packetCycle teardown ✓ Task 3, keepalive ✓ Task 3, message clear ✓ Task 3, boot pacing ✓ Task 3.
- [ ] `gofmt` clean, lint policy: new code fixed, faithful-port oddities get per-line `//nolint` + Java ref only if the linter actually fires.
- [ ] P7 smoke retry is the FINAL verification — the user runs the client on the host against the 274 engine; in-sandbox network checks prove nothing (netns isolation).

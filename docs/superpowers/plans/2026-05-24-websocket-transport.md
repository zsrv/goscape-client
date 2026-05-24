# WebSocket Transport Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an optional WebSocket transport alongside the existing TCP socket, selected by the `host` CLI argument's URL scheme (`ws://`/`wss://`), so the client can later run as a `js/wasm` browser build.

**Architecture:** The game already reads/writes through `ClientStream`, which wraps a generic `net.Conn`, and TCP is dialed in exactly one place — `signlink.OpenSocket`. We branch there: TCP keeps `net.DialTimeout`; `ws`/`wss` dials via `github.com/coder/websocket` and returns its `NetConn` adapter (a `net.Conn` over binary frames). `ClientStream`, `client.go`, and the game loop are untouched.

**Tech Stack:** Go 1.26, `github.com/coder/websocket` (first-class `js/wasm` support + `NetConn`), `net/url` for arg parsing.

**Design spec:** `docs/superpowers/specs/2026-05-24-websocket-transport-design.md`

---

## Environment / command prefix

All `go` commands run in the sandbox, which **denies writes to the default `$HOME/go` GOPATH**. Use a writable temp GOPATH consistently. Every command below is shown with this prefix:

```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache go <args>
```

> **Note:** Because GOPATH points at a fresh temp dir, the **first** `go` command may re-download all module dependencies (network to the Go module proxy required). If the proxy is unreachable in-sandbox, run the Task 1 `go get`/`go mod tidy` steps from a host terminal using the `! <command>` prompt prefix, then continue here. The critical rule: **every `go` command in this plan must use the same GOPATH**, or the newly added module won't resolve.

Commits use `git commit --no-gpg-sign` (works in-sandbox). Each commit message ends with the project's Co-Authored-By trailer.

---

## File Structure

| File | Responsibility |
|---|---|
| `go.mod` / `go.sum` | Declare the `github.com/coder/websocket` dependency. |
| `pkg/jagex2/client/clientextras/clientextras.go` (modify) | Hold transport config: `TransportKind` enum, `Transport`, `WSPort`, `WSPath`. |
| `cmd/client/hostarg.go` (create) | `parseHostArg` — pure parse of the host CLI arg into transport + bare host + ws port/path. |
| `cmd/client/hostarg_test.go` (create) | Table test for `parseHostArg`. |
| `cmd/client/main.go` (modify) | Call `parseHostArg`, populate config vars, update usage text. |
| `pkg/sign/signlink/signlink_ws.go` (create) | `buildWSURL` (pure URL assembly) + `openWebSocket` (dial + `NetConn`). |
| `pkg/sign/signlink/signlink_ws_test.go` (create) | `buildWSURL` table test + WS round-trip integration test. |
| `pkg/sign/signlink/signlink.go` (modify) | Branch `OpenSocket` on `clientextras.Transport`. |

---

## Task 1: Add the `coder/websocket` dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Fetch the module**

Run:
```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go get github.com/coder/websocket@latest
```
Expected: `go.mod` gains a `require github.com/coder/websocket vX.Y.Z` line (no error). If the proxy is unreachable, run this from a host terminal (`! go get …`) as noted above.

- [ ] **Step 2: Tidy**

Run:
```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go mod tidy
```
Expected: no error; `go.sum` updated.

- [ ] **Step 3: Verify the dependency is recorded**

Run: `grep coder/websocket go.mod`
Expected: a line like `github.com/coder/websocket v1.8.x` (exact version may differ).

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit --no-gpg-sign -m "$(cat <<'EOF'
build(net): add github.com/coder/websocket dependency

Used by the upcoming WebSocket transport. Chosen for its first-class
js/wasm support and NetConn adapter.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Transport config vars in `clientextras`

**Files:**
- Modify: `pkg/jagex2/client/clientextras/clientextras.go`

- [ ] **Step 1: Append the transport config**

Add to the end of `pkg/jagex2/client/clientextras/clientextras.go` (after the `Host` var):

```go
// Transport selects the game-server connection transport. It is set once at
// startup from the host CLI argument's URL scheme and read by
// signlink.OpenSocket. The WS path is a Go-original standalone extension (the
// original Java applet used raw sockets only); see
// docs/superpowers/specs/2026-05-24-websocket-transport-design.md.
type TransportKind int

const (
	TransportTCP TransportKind = iota // raw TCP socket (default; Java parity)
	TransportWS                       // WebSocket (ws://)
	TransportWSS                      // WebSocket over TLS (wss://)
)

var Transport TransportKind = TransportTCP

// WSPort is an explicit WebSocket port parsed from a ws[s]:// host argument.
// 0 means "use the default game port the dial site supplies (PortOffset+43594)".
var WSPort int

// WSPath is an explicit path parsed from a ws[s]:// host argument.
// "" means "/".
var WSPath string
```

- [ ] **Step 2: Verify it compiles**

Run:
```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go build ./pkg/jagex2/client/clientextras/...
```
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add pkg/jagex2/client/clientextras/clientextras.go
git commit --no-gpg-sign -m "$(cat <<'EOF'
feat(net): add transport config vars to clientextras

TransportKind enum (TCP/WS/WSS) plus WSPort/WSPath overrides, set at
startup and read by the dial path. Host stays a bare hostname.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: `buildWSURL` (pure URL assembly) + test

**Files:**
- Create: `pkg/sign/signlink/signlink_ws.go`
- Test: `pkg/sign/signlink/signlink_ws_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/sign/signlink/signlink_ws_test.go`:

```go
package signlink

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

func TestBuildWSURL(t *testing.T) {
	tests := []struct {
		name         string
		kind         clientextras.TransportKind
		host         string
		defaultPort  int
		overridePort int
		overridePath string
		want         string
	}{
		{"bare ws default port", clientextras.TransportWS, "gameserver", 43594, 0, "", "ws://gameserver:43594/"},
		{"port offset applied", clientextras.TransportWS, "gameserver", 43595, 0, "", "ws://gameserver:43595/"},
		{"override port", clientextras.TransportWS, "10.0.0.5", 43594, 8080, "", "ws://10.0.0.5:8080/"},
		{"wss with port and path", clientextras.TransportWSS, "play.example.com", 43594, 443, "/ws", "wss://play.example.com:443/ws"},
		{"path no override port", clientextras.TransportWS, "host", 43594, 0, "/path", "ws://host:43594/path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWSURL(tt.kind, tt.host, tt.defaultPort, tt.overridePort, tt.overridePath)
			if got != tt.want {
				t.Fatalf("buildWSURL = %q, want %q", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go test ./pkg/sign/signlink/ -run TestBuildWSURL
```
Expected: FAIL — build error `undefined: buildWSURL`.

- [ ] **Step 3: Write the minimal implementation**

Create `pkg/sign/signlink/signlink_ws.go`:

```go
package signlink

import (
	"net"
	"strconv"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

// buildWSURL assembles the WebSocket dial URL. overridePort is the explicit
// port from the host argument (0 -> use defaultPort); overridePath is the
// explicit path ("" -> "/"). Pure (no globals) so it is unit-tested directly.
func buildWSURL(kind clientextras.TransportKind, host string, defaultPort, overridePort int, overridePath string) string {
	scheme := "ws"
	if kind == clientextras.TransportWSS {
		scheme = "wss"
	}
	port := defaultPort
	if overridePort != 0 {
		port = overridePort
	}
	path := overridePath
	if path == "" {
		path = "/"
	}
	return scheme + "://" + net.JoinHostPort(host, strconv.Itoa(port)) + path
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:
```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go test ./pkg/sign/signlink/ -run TestBuildWSURL -v
```
Expected: PASS (all 5 subtests).

- [ ] **Step 5: Commit**

```bash
git add pkg/sign/signlink/signlink_ws.go pkg/sign/signlink/signlink_ws_test.go
git commit --no-gpg-sign -m "$(cat <<'EOF'
feat(net): add buildWSURL WebSocket URL assembler

Pure helper: applies the default game port when none is given and "/"
when no path is given; honors explicit port/path overrides.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: `openWebSocket` (dial + NetConn) + round-trip test

**Files:**
- Modify: `pkg/sign/signlink/signlink_ws.go`
- Test: `pkg/sign/signlink/signlink_ws_test.go`

- [ ] **Step 1: Write the failing integration test**

Append to `pkg/sign/signlink/signlink_ws_test.go`:

```go
import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/zsrv/goscape-client/pkg/jagex2/io/clientstream"
)

// TestOpenWebSocketRoundTrip dials an in-process echo server through
// openWebSocket, wraps the result in a ClientStream, and verifies bytes
// written are read back unchanged — proving the NetConn adapter and
// ClientStream interoperate over binary WebSocket frames.
func TestOpenWebSocketRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols: []string{"binary"},
		})
		if err != nil {
			return
		}
		defer c.CloseNow()
		nc := websocket.NetConn(r.Context(), c, websocket.MessageBinary)
		_, _ = io.Copy(nc, nc) // echo until the client closes
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("server port: %v", err)
	}

	// No explicit override: use the server's port as the default, root path.
	clientextras.WSPort = 0
	clientextras.WSPath = ""

	conn, err := openWebSocket(clientextras.TransportWS, u.Hostname(), port, 10*time.Second)
	if err != nil {
		t.Fatalf("openWebSocket: %v", err)
	}
	cs := clientstream.NewClientStream(conn)
	defer cs.Close()

	msg := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	// ClientStream.Write(buf, length, offset) — note the (buf, len, off) order.
	if err := cs.Write(msg, len(msg), 0); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := make([]byte, len(msg))
	if err := cs.ReadFully(got, 0, len(msg)); err != nil {
		t.Fatalf("readfully: %v", err)
	}
	if !bytes.Equal(got, msg) {
		t.Fatalf("round-trip = %v, want %v", got, msg)
	}
}
```

> Merge the two import blocks in the file into one if `gofmt`/`go vet` complains about duplicate `import` declarations; the test additions above list every new import needed (`bytes`, `io`, `net/http`, `net/http/httptest`, `net/url`, `strconv`, `time`, `github.com/coder/websocket`, the `clientstream` package). `clientstream` imports only stdlib, so there is no import cycle.

- [ ] **Step 2: Run the test to verify it fails**

Run:
```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go test ./pkg/sign/signlink/ -run TestOpenWebSocketRoundTrip
```
Expected: FAIL — build error `undefined: openWebSocket`.

- [ ] **Step 3: Write the minimal implementation**

Add to `pkg/sign/signlink/signlink_ws.go`. Update the import block to add `context`, `time`, and `github.com/coder/websocket`, and add the function:

```go
import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

// openWebSocket dials the game server over WebSockets and returns the
// connection adapted to net.Conn (so clientstream.ClientStream can wrap it
// unchanged). The handshake is bounded by `timeout` for parity with TCP's
// DialTimeout.
//
// Java: no equivalent — the original applet used raw sockets only. This is a
// Go-original standalone extension; see the design spec.
func openWebSocket(kind clientextras.TransportKind, host string, port int, timeout time.Duration) (net.Conn, error) {
	url := buildWSURL(kind, host, port, clientextras.WSPort, clientextras.WSPath)

	// CRITICAL: the dial-timeout context must NOT be the context passed to
	// NetConn. NetConn ties the connection's lifetime to its context, so
	// reusing dialCtx would tear the live connection down after `timeout`.
	// Cancel dialCtx once the handshake succeeds and give NetConn a background
	// context instead.
	dialCtx, cancel := context.WithTimeout(context.Background(), timeout)
	c, _, err := websocket.Dial(dialCtx, url, &websocket.DialOptions{
		Subprotocols: []string{"binary"},
	})
	cancel()
	if err != nil {
		return nil, err
	}

	// Server frames can be large (e.g. map data); the 32 KiB default read
	// limit would error on an oversized message. Raise it generously.
	c.SetReadLimit(1 << 20) // 1 MiB

	return websocket.NetConn(context.Background(), c, websocket.MessageBinary), nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:
```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go test ./pkg/sign/signlink/ -run 'TestBuildWSURL|TestOpenWebSocketRoundTrip' -v
```
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add pkg/sign/signlink/signlink_ws.go pkg/sign/signlink/signlink_ws_test.go
git commit --no-gpg-sign -m "$(cat <<'EOF'
feat(net): add openWebSocket dial returning a net.Conn

Dials via coder/websocket with the "binary" subprotocol and adapts the
connection with websocket.NetConn so ClientStream wraps it unchanged.
Handshake bounded by a 10s context; NetConn gets a background context.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Branch `signlink.OpenSocket` on the transport

**Files:**
- Modify: `pkg/sign/signlink/signlink.go` (the `OpenSocket` function near line 378)

- [ ] **Step 1: Replace the function body**

Find the existing function:

```go
func OpenSocket(port int) (net.Conn, error) {
	const dialTimeout = 10 * time.Second
	return net.DialTimeout("tcp", net.JoinHostPort(clientextras.Host, strconv.Itoa(port)), dialTimeout)
}
```

Replace it with (keep the existing docstring above the function; add the WS note shown):

```go
// Transport branch (Go-original extension): ws://wss:// hosts dial a
// WebSocket instead of a raw TCP socket, enabling a future js/wasm browser
// build. TCP remains the Java-parity default. See
// docs/superpowers/specs/2026-05-24-websocket-transport-design.md.
func OpenSocket(port int) (net.Conn, error) {
	const dialTimeout = 10 * time.Second
	switch clientextras.Transport {
	case clientextras.TransportWS, clientextras.TransportWSS:
		return openWebSocket(clientextras.Transport, clientextras.Host, port, dialTimeout)
	default:
		return net.DialTimeout("tcp", net.JoinHostPort(clientextras.Host, strconv.Itoa(port)), dialTimeout)
	}
}
```

- [ ] **Step 2: Verify the package builds and existing tests pass**

Run:
```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go test ./pkg/sign/signlink/...
```
Expected: PASS (all signlink tests, including the new WS tests).

- [ ] **Step 3: Commit**

```bash
git add pkg/sign/signlink/signlink.go
git commit --no-gpg-sign -m "$(cat <<'EOF'
feat(net): branch signlink.OpenSocket on transport (TCP/WS/WSS)

WS/WSS dials via openWebSocket; TCP remains the Java-parity default.
Single chokepoint change — ClientStream and client.go untouched.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: `parseHostArg` (pure) + test

**Files:**
- Create: `cmd/client/hostarg.go`
- Test: `cmd/client/hostarg_test.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/client/hostarg_test.go`:

```go
package main

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

func TestParseHostArg(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		wantKind clientextras.TransportKind
		wantHost string
		wantPort int
		wantPath string
		wantErr  bool
	}{
		{"bare host", "localhost", clientextras.TransportTCP, "localhost", 0, "", false},
		{"bare ip", "10.0.0.5", clientextras.TransportTCP, "10.0.0.5", 0, "", false},
		{"ws no port", "ws://gameserver", clientextras.TransportWS, "gameserver", 0, "", false},
		{"ws with port", "ws://10.0.0.5:8080", clientextras.TransportWS, "10.0.0.5", 8080, "", false},
		{"wss port and path", "wss://play.example.com:443/ws", clientextras.TransportWSS, "play.example.com", 443, "/ws", false},
		{"unsupported scheme", "http://example.com", 0, "", 0, "", true},
		{"empty hostname", "ws://", 0, "", 0, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, host, port, path, err := parseHostArg(tt.arg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseHostArg(%q) = nil error, want error", tt.arg)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseHostArg(%q) unexpected error: %v", tt.arg, err)
			}
			if kind != tt.wantKind || host != tt.wantHost || port != tt.wantPort || path != tt.wantPath {
				t.Fatalf("parseHostArg(%q) = (%v, %q, %d, %q), want (%v, %q, %d, %q)",
					tt.arg, kind, host, port, path, tt.wantKind, tt.wantHost, tt.wantPort, tt.wantPath)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go test ./cmd/client/ -run TestParseHostArg
```
Expected: FAIL — build error `undefined: parseHostArg`.

- [ ] **Step 3: Write the minimal implementation**

Create `cmd/client/hostarg.go`:

```go
package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

// parseHostArg interprets the optional host CLI argument. A bare host selects
// TCP (the default); a ws://… or wss://… URL selects a WebSocket transport and
// yields the bare hostname plus any explicit port (0 = use default) and path
// ("" = "/"). The returned host is always a bare hostname so it stays valid for
// clientextras.Host's TCP-dial, GetHost, and GetCodeBase consumers.
func parseHostArg(arg string) (tk clientextras.TransportKind, host string, wsPort int, wsPath string, err error) {
	if !strings.Contains(arg, "://") {
		return clientextras.TransportTCP, arg, 0, "", nil
	}
	u, perr := url.Parse(arg)
	if perr != nil {
		return 0, "", 0, "", fmt.Errorf("parse host %q: %w", arg, perr)
	}
	switch u.Scheme {
	case "ws":
		tk = clientextras.TransportWS
	case "wss":
		tk = clientextras.TransportWSS
	default:
		return 0, "", 0, "", fmt.Errorf("unsupported host scheme %q (use ws://, wss://, or a bare host)", u.Scheme)
	}
	host = u.Hostname()
	if host == "" {
		return 0, "", 0, "", fmt.Errorf("host %q has no hostname", arg)
	}
	if p := u.Port(); p != "" {
		wsPort, err = strconv.Atoi(p)
		if err != nil {
			return 0, "", 0, "", fmt.Errorf("invalid port in host %q: %w", arg, err)
		}
	}
	wsPath = u.EscapedPath()
	return tk, host, wsPort, wsPath, nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:
```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go test ./cmd/client/ -run TestParseHostArg -v
```
Expected: PASS (all 7 subtests).

- [ ] **Step 5: Commit**

```bash
git add cmd/client/hostarg.go cmd/client/hostarg_test.go
git commit --no-gpg-sign -m "$(cat <<'EOF'
feat(net): add parseHostArg for transport selection by URL scheme

ws://wss:// hosts select WebSockets and yield a bare hostname plus
port/path; bare hosts select TCP. Keeps clientextras.Host pristine.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Wire `parseHostArg` into `main.go`

**Files:**
- Modify: `cmd/client/main.go` (the `if len(os.Args) == 6` block at lines 53-60, and the usage strings)

- [ ] **Step 1: Replace the host-assignment block**

Find this block (currently around lines 53-60):

```go
	// Java main accepts exactly 4 args (deob/client.java:10599); the applet host
	// came from getCodeBase().getHost(). This optional 5th `host` arg is a
	// Go-original standalone extension (no browser codebase exists here) that
	// lets the operator point the binary at a non-localhost server. Behavioral
	// superset of Java's CLI, not a parity regression.
	if len(os.Args) == 6 {
		clientextras.Host = os.Args[5]
	}
```

Replace it with:

```go
	// Java main accepts exactly 4 args (deob/client.java:10599); the applet host
	// came from getCodeBase().getHost(). This optional 5th `host` arg is a
	// Go-original standalone extension (no browser codebase exists here) that
	// lets the operator point the binary at a non-localhost server. A ws:// or
	// wss:// scheme additionally selects the WebSocket transport (for a future
	// js/wasm build); a bare host keeps the TCP default. The parsed bare
	// hostname is stored in clientextras.Host so GetHost/GetCodeBase stay valid.
	if len(os.Args) == 6 {
		tk, host, wsPort, wsPath, err := parseHostArg(os.Args[5])
		if err != nil {
			fmt.Printf("invalid host: %v\n", err)
			os.Exit(1)
		}
		clientextras.Host = host
		clientextras.Transport = tk
		clientextras.WSPort = wsPort
		clientextras.WSPath = wsPath
	}
```

- [ ] **Step 2: Update the usage strings**

There are three usage `fmt.Println` calls. Update each `[host]` mention to indicate the scheme option. Change every occurrence of:

```go
	fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members], [host]")
```

to:

```go
	fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members], [host|ws://host[:port][/path]|wss://host[:port][/path]]")
```

And the one at line 41 (which omits `[host]`):

```go
		fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members]")
```

to:

```go
		fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members], [host|ws://host[:port][/path]|wss://host[:port][/path]]")
```

- [ ] **Step 3: Verify the whole module builds**

Run:
```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go build ./...
```
Expected: no output (success).

- [ ] **Step 4: Commit**

```bash
git add cmd/client/main.go
git commit --no-gpg-sign -m "$(cat <<'EOF'
feat(net): select transport from host arg in main

Parse the optional host arg via parseHostArg; ws://wss:// opts into the
WebSocket transport, bare host stays TCP. Usage text updated.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Full verification (native + wasm compile check)

**Files:** none (verification only)

- [ ] **Step 1: Build, vet, and test the whole module (native)**

Run:
```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go build ./... && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go vet ./... && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go test ./...
```
Expected: build/vet produce no output; tests report `ok` for each package (no FAIL).

- [ ] **Step 2: WASM compile check of the touched packages**

Run:
```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  GOOS=js GOARCH=wasm go build \
  ./pkg/sign/signlink/... ./pkg/jagex2/client/clientextras/... ./pkg/jagex2/io/clientstream/...
```
Expected: no output (success) — proves the WS transport path compiles for the browser. (A full `./...` wasm build is intentionally **out of scope**; other subsystems are not yet wasm-ready.)

- [ ] **Step 3: Sanity-check the CLI usage text**

Run:
```
cd $HOME/Code/github.com/zsrv/goscape-client && \
TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go GOCACHE=/tmp/claude-1000/go-cache \
  go run ./cmd/client 2>&1 | head -2
```
Expected: prints the release banner and a `Usage:` line containing `ws://host` (the arg-count check exits non-zero with usage text, which is the intended behavior here).

- [ ] **Step 4: No commit**

This task only verifies; nothing to commit. If any step fails, fix the offending task before proceeding.

---

## Manual verification (operator, outside the sandbox)

Tests cannot exercise a live game server. After the plan is implemented, the operator can confirm end-to-end behavior on the host:

- **TCP (unchanged):** `go run ./cmd/client 10 0 highmem members` — connects as before.
- **WebSocket:** point at a server exposing a `binary`-subprotocol WS endpoint, e.g. `go run ./cmd/client 10 0 highmem members ws://localhost` (dials `ws://localhost:43594/`) or `… wss://play.example.com:443/ws`.

---

## Self-Review notes (completed during planning)

- **Spec coverage:** dependency (T1), config vars (T2), `buildWSURL` (T3), `openWebSocket` + round-trip (T4), `OpenSocket` branch (T5), `parseHostArg` (T6), `main` wiring + usage (T7), native + wasm verification (T8). `ClientStream` unchanged is verified by T4/T5 passing without touching it.
- **Type consistency:** `TransportKind`/`Transport`/`WSPort`/`WSPath` (T2) are used identically in T3–T7; `buildWSURL` signature `(kind, host, defaultPort, overridePort, overridePath)` matches its call in T4; `parseHostArg` return tuple `(TransportKind, string, int, string, error)` matches its call in T7 and its test in T6.
- **Placeholder scan:** none — every code/command step contains concrete content.

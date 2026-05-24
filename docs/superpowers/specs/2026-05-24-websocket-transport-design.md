# WebSocket Transport (alongside TCP) — Design

**Date:** 2026-05-24
**Status:** Approved (pending implementation)
**Branch:** rev-225

## Goal

Let the client connect to the game server over **WebSockets** as an
alternative to its current raw **TCP** socket, selected at launch. The
motivation is a future `GOOS=js`/`GOARCH=wasm` browser build: browsers forbid
raw TCP, so WebSockets are the only way a wasm client can reach the server.

This task is the **transport layer only**. The full browser build (Gio's
WASM/WebGL backend, a browser-compatible cache store, the JS entrypoint and
HTML shell) is explicitly **out of scope** and tracked as separate future work.
The WebSocket code is, however, written so that it *compiles and runs* under
`js/wasm`.

## Non-goals

- Making the entire client build/run as wasm now.
- Reconnection/transport-failover logic beyond what already exists.
- Changing the RS2 application protocol in any way. WebSockets are a pure
  transport substitution; the byte stream on the wire is identical.

## Background: the existing seam

The whole game reads and writes through `ClientStream`
(`pkg/jagex2/io/clientstream/clientstream.go`), which wraps a generic
`net.Conn`. TCP is dialed in exactly **one** place:

```
client.LoginFunc (client.go:6434)
  └─ c.OpenSocket(PortOffset+43594)          // client.go:6907 → delegates to:
       └─ signlink.OpenSocket(port)          // signlink.go:378 → net.DialTimeout("tcp", …)
            └─ returns net.Conn
                 └─ clientstream.NewClientStream(conn)   // client.go:6440
```

Because `ClientStream` already depends only on the `net.Conn` interface, adding
WebSockets is fundamentally "produce a `net.Conn` from a WebSocket and return it
from the same chokepoint." `ClientStream`, `client.go`, and the game loop need
**no changes**.

The TypeScript client (`Client-TS/src/io/ClientStream.ts`) confirms the
on-the-wire contract: it dials `ws://host` / `wss://host` with the WebSocket
**subprotocol `"binary"`**, and its reader concatenates the bytes of every
binary frame into one continuous stream (message boundaries are irrelevant).
This is exactly the semantics of `coder/websocket`'s `NetConn` over
`MessageBinary`.

## Library

**`github.com/coder/websocket`** (formerly `nhooyr.io/websocket`). Chosen
because it is the only mature Go WebSocket library with first-class `js/wasm`
support (on wasm it transparently wraps the browser's native `WebSocket`) and it
ships `websocket.NetConn(ctx, c, msgType) net.Conn`, which adapts a WS
connection to a `net.Conn` byte stream. Zero non-stdlib dependencies.

Relevant API (verified against current docs):

- `websocket.Dial(ctx, url, *DialOptions) (*Conn, *http.Response, error)` —
  `DialOptions.Subprotocols` carries `["binary"]`. On wasm, `HTTPHeader` and
  `CompressionMode` are ignored and the returned `*http.Response` is `nil`
  (so the response value must not be dereferenced — bind it to `_`).
- `(*Conn).SetReadLimit(int64)` — default per-message read limit is 32 KiB; we
  raise it (see below).
- `websocket.NetConn(ctx, c, websocket.MessageBinary) net.Conn`.

New direct dependency added to `go.mod`.

## CLI: transport selection via host scheme

**No new positional argument.** The optional `host` argument (current position
5) is reused: its URL scheme selects the transport. This is fully backward
compatible — every existing invocation keeps working, and `Host`'s default
remains `127.0.0.1` over TCP.

| `host` argument                 | Transport     | Resulting endpoint                          |
|---------------------------------|---------------|---------------------------------------------|
| *(omitted)*                     | TCP (default) | `tcp` → `127.0.0.1:(PortOffset+43594)`      |
| `localhost` / `10.0.0.5`        | TCP           | `tcp` → `host:(PortOffset+43594)`           |
| `ws://gameserver`               | WS            | `ws://gameserver:(PortOffset+43594)/`       |
| `ws://10.0.0.5:8080`            | WS            | `ws://10.0.0.5:8080/`                       |
| `wss://play.example.com:443/ws` | WSS           | `wss://play.example.com:443/ws`             |

Rules:
- Scheme `ws://` → WebSocket (plaintext); `wss://` → WebSocket over TLS.
- No scheme → TCP, unchanged.
- If the URL omits a port, the default game port `PortOffset+43594` is used.
- If the URL carries a port and/or path, they are honored verbatim.

Argument count handling in `main.go` is unchanged (`len(os.Args)` 5 or 6).

### Keeping `clientextras.Host` pristine

`clientextras.Host` has three consumers, two of which require a **bare
hostname** (no scheme, no port, no path):

- `signlink.OpenSocket` — TCP dial, `net.JoinHostPort(Host, port)`.
- `client.GetHost()` (client.go:5055) — `strings.ToLower(Host)` for
  host-validation comparisons.
- `client.GetCodeBase()` (client.go:7309) — splices `Host` into
  `http://<Host>:<PortOffset+8888>` to fetch cache resources.

Therefore a `ws://…` argument must be **parsed**, not stored raw. The bare
hostname goes into `Host`; the WebSocket-only extras live in their own config
vars read solely by the WS dial path.

## Components & changes

### 1. `pkg/jagex2/client/clientextras/clientextras.go` — config vars

Add (next to the existing `Host`, `PortOffset`):

```go
// Transport selects the game-server connection transport. Set once at
// startup from the host CLI argument's URL scheme; read by signlink.OpenSocket.
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

### 2. `cmd/client/main.go` — parse the host argument

Replace the current raw assignment

```go
if len(os.Args) == 6 {
    clientextras.Host = os.Args[5]
}
```

with a call to a new **pure** helper, then assign the results to the config
vars:

```go
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

`parseHostArg` (also in `package main`, so it is unit-testable via
`cmd/client/main_test.go`):

```go
// parseHostArg interprets the optional host CLI argument. A bare host selects
// TCP (the default); a ws://… or wss://… URL selects a WebSocket transport and
// yields the bare hostname plus any explicit port (0 = default) and path
// ("" = "/"). The returned host is always a bare hostname so it stays valid for
// clientextras.Host's TCP/HTTP/host-validation consumers.
func parseHostArg(arg string) (tk clientextras.TransportKind, host string, wsPort int, wsPath string, err error)
```

Behavior:
- No `://` (or a scheme other than ws/wss): `TransportTCP`, `host = arg`,
  `wsPort = 0`, `wsPath = ""`. (A non-ws scheme is an error to avoid silently
  treating e.g. `http://x` as a hostname.)
- `ws://`/`wss://`: parse with `net/url`; `host = u.Hostname()`,
  `wsPort = atoi(u.Port())` (0 if absent), `wsPath = u.EscapedPath()`,
  `tk = TransportWS`/`TransportWSS`. Reject an empty hostname.

Update the usage strings to mention that `[host]` may be a bare host or a
`ws://`/`wss://` URL.

### 3. `pkg/sign/signlink/signlink.go` — branch `OpenSocket`

```go
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

The existing Java-parity docstring is retained; a note is added that the
WS/WSS branch is a **Go-original extension** (no Java equivalent), consistent
with how the optional `[host]` argument is already documented as a standalone
extension.

### 4. `pkg/sign/signlink/signlink_ws.go` — new file (WS dial)

Two pieces, the first pure and the second the actual dial:

```go
// buildWSURL assembles the dial URL. overridePort is the explicit port from the
// host argument (<= 0 → use defaultPort); overridePath is the explicit path
// ("" → "/"). Pure (no globals) so it is unit-tested directly with table inputs.
func buildWSURL(kind clientextras.TransportKind, host string, defaultPort, overridePort int, overridePath string) string {
    scheme := "ws"
    if kind == clientextras.TransportWSS {
        scheme = "wss"
    }
    port := defaultPort
    if overridePort > 0 { // non-positive (incl. the 0 sentinel) → keep defaultPort
        port = overridePort
    }
    path := overridePath
    if path == "" {
        path = "/"
    }
    return scheme + "://" + net.JoinHostPort(host, strconv.Itoa(port)) + path
}

func openWebSocket(kind clientextras.TransportKind, host string, port int, timeout time.Duration) (net.Conn, error) {
    url := buildWSURL(kind, host, port, clientextras.WSPort, clientextras.WSPath)

    // The dial/handshake is bounded by `timeout` (parity with TCP's
    // DialTimeout). CRITICAL: this timeout context must NOT be the context
    // handed to NetConn — NetConn ties the connection lifetime to its context,
    // so reusing the timeout ctx would tear the live connection down after
    // `timeout`. Cancel it once the handshake succeeds and give NetConn a
    // background context instead.
    dialCtx, cancel := context.WithTimeout(context.Background(), timeout)
    c, _, err := websocket.Dial(dialCtx, url, &websocket.DialOptions{
        Subprotocols: []string{"binary"},
    })
    cancel()
    if err != nil {
        return nil, err
    }

    // Server frames can be large (e.g. map data); the 32 KiB default would
    // error on an oversized message. Raise it generously.
    c.SetReadLimit(1 << 20) // 1 MiB

    return websocket.NetConn(context.Background(), c, websocket.MessageBinary), nil
}
```

Notes:
- `buildWSURL` takes the default port as a parameter (and `openWebSocket`
  passes it the override read from `clientextras.WSPort`), so the `+43594`
  constant continues to live solely at the existing `client.go:6434` call
  site — it is not duplicated.
- `NetConn` returns a `net.Conn` that is **not** a `*net.TCPConn`, so
  `NewClientStream`'s `SetNoDelay` type-assertion (clientstream.go:96) simply
  skips it — no change needed there.

### 5. `pkg/jagex2/io/clientstream/clientstream.go` — unchanged

Confirmed compatible as-is.

### 6. `go.mod` / `go.sum`

Add `github.com/coder/websocket`. (go.mod/go.sum are tracked.)

## Data flow (after change)

```
main.go: parseHostArg(os.Args[5]) ─▶ clientextras.{Host, Transport, WSPort, WSPath}
        │
   LoginFunc ─▶ c.OpenSocket(PortOffset+43594) ─▶ signlink.OpenSocket(port)
                                                        │ switch Transport
                          ┌─────────────────────────────┼───────────────────────┐
                       TCP │                          WS │ WSS                    │
        net.DialTimeout("tcp", Host:port)   openWebSocket(kind, Host, port, 10s)  │
                          │                  ├─ buildWSURL(...) → ws[s]://h:p/path │
                          │                  ├─ websocket.Dial(10s ctx, …,         │
                          │                  │     Subprotocols:["binary"])        │
                          │                  ├─ c.SetReadLimit(1<<20)              │
                          │                  └─ websocket.NetConn(bg, c, Binary)   │
                          └────────────────────────▶ net.Conn ◀───────────────────┘
                                                         │
                                       clientstream.NewClientStream(conn) ─▶ game loop
```

## Error handling

- Any dial/handshake failure (TCP or WS) returns an `error` up the existing
  path. `LoginFunc` already maps it to the on-screen
  `"Error connecting to server."` — no new UI.
- WS handshake is bounded by a 10 s context timeout (parity with TCP's
  `DialTimeout`).
- Invalid host argument (bad URL, empty hostname, unsupported scheme) → usage
  error + `os.Exit(1)` at startup, consistent with the other argument checks.
- Under `js/wasm` the TCP path's `net.Dial` fails at **runtime** (browsers have
  no raw TCP); this is expected — browser users pass a `ws://`/`wss://` host.
  The code still **compiles** for wasm.

## Testing

Native (`linux/amd64`) unit + integration tests:

1. **`buildWSURL` table test** (`signlink_ws_test.go`): every endpoint row from
   the CLI table — default port injection, override-port, override-path, and
   `ws` vs `wss` scheme — passed as direct function arguments (no globals).
2. **`parseHostArg` table test** (`cmd/client/main_test.go`): bare host → TCP;
   `ws://h` → WS with port 0/path ""; `wss://h:443/x` → WSS port 443 path `/x`;
   empty hostname → error; unsupported scheme (`http://…`) → error; omitted →
   defaults.
3. **WS round-trip integration test** (`signlink_ws_test.go`): start an
   in-process `coder/websocket` echo server (`httptest.Server` +
   `websocket.Accept`, accepting the `binary` subprotocol), dial it via
   `openWebSocket`, wrap the result in `clientstream.NewClientStream`, then
   write a byte slice and read it back via `ReadFully`, asserting byte-for-byte
   equality. (An explicit `Available()` assertion is intentionally omitted: it
   would require a poll/sleep to wait for the reader goroutine to buffer the
   inbound frame, introducing flakiness — and `ReadFully` already exercises the
   buffering path.) Proves the `NetConn` adapter and `ClientStream` interoperate.

WASM compile check (no run):

4. `GOOS=js GOARCH=wasm go build` of the touched packages
   (`./pkg/sign/signlink/...`, `./pkg/jagex2/client/clientextras/...`,
   `./pkg/jagex2/io/clientstream/...`) to prove the WS path builds for the
   browser. A full `./...` wasm build stays out of scope — other subsystems are
   not yet wasm-ready.

## Verification commands

```bash
# native (prefix per CLAUDE.local.md sandbox rules)
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go vet ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./...

# wasm compile check (touched packages only)
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache \
  GOOS=js GOARCH=wasm go build \
  ./pkg/sign/signlink/... ./pkg/jagex2/client/clientextras/... ./pkg/jagex2/io/clientstream/...
```

## Parity note

The WebSocket path has no Java equivalent (the original applet used raw
sockets). Like the existing optional `[host]` argument, it is a **Go-original
standalone extension** and is marked as such in code comments, preserving the
project's bug-for-bug parity discipline for the TCP path.

## File summary

| File | Change |
|---|---|
| `pkg/jagex2/client/clientextras/clientextras.go` | Add `TransportKind`, `Transport`, `WSPort`, `WSPath` |
| `cmd/client/main.go` | Add `parseHostArg`; populate the new config vars; update usage text |
| `cmd/client/main_test.go` | New — `parseHostArg` table test |
| `pkg/sign/signlink/signlink.go` | Branch `OpenSocket` on `Transport`; note Go-original WS extension |
| `pkg/sign/signlink/signlink_ws.go` | New — `buildWSURL` (pure) + `openWebSocket` |
| `pkg/sign/signlink/signlink_ws_test.go` | New — `buildWSURL` + round-trip tests |
| `pkg/jagex2/io/clientstream/clientstream.go` | Unchanged (verified compatible) |
| `go.mod` / `go.sum` | Add `github.com/coder/websocket` |

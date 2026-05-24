package signlink

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

// buildWSURL assembles the WebSocket dial URL. overridePort is the explicit
// port from the host argument (<= 0 -> use defaultPort); overridePath is the
// explicit path ("" -> "/"). Pure (no globals) so it is unit-tested directly.
func buildWSURL(kind clientextras.TransportKind, host string, defaultPort, overridePort int, overridePath string) string {
	scheme := "ws"
	if kind == clientextras.TransportWSS {
		scheme = "wss"
	}
	port := defaultPort
	if overridePort > 0 {
		port = overridePort
	}
	path := overridePath
	if path == "" {
		path = "/"
	}
	return scheme + "://" + net.JoinHostPort(host, strconv.Itoa(port)) + path
}

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

	// websocket.NetConn disables the per-message read limit internally
	// (SetReadLimit(-1)), so large server frames (e.g. map data) never trip the
	// 32 KiB default — no explicit limit is needed here. This matches the TS
	// client, which imposes no per-message cap either.
	return websocket.NetConn(context.Background(), c, websocket.MessageBinary), nil
}

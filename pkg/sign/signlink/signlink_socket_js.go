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

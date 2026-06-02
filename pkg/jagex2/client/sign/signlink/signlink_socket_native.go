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
// -world-server flag parsing in cmd/client/main.go.
func ConfigureTransport() {}

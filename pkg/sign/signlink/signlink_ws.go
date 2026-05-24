package signlink

import (
	"net"
	"strconv"

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

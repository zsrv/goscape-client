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

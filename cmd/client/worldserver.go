package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

// parseWorldServer interprets the -world-server flag, a [tcp|ws|wss]://host:port
// URL. The scheme and the port are both mandatory (a bare host is rejected). tcp
// selects the raw-socket transport (no path); ws/wss select the WebSocket
// transport and may carry a path ("" means the root "/"). The returned host is a
// bare hostname so it stays valid for clientextras.Host's TCP-dial, GetHost, and
// GetCodeBase consumers.
func parseWorldServer(arg string) (kind clientextras.TransportKind, host string, port int, path string, err error) {
	if !strings.Contains(arg, "://") {
		return 0, "", 0, "", fmt.Errorf("world-server %q must include a scheme (tcp://, ws://, or wss://)", arg)
	}
	u, perr := url.Parse(arg)
	if perr != nil {
		return 0, "", 0, "", fmt.Errorf("parse world-server %q: %w", arg, perr)
	}
	switch u.Scheme {
	case "tcp":
		kind = clientextras.TransportTCP
	case "ws":
		kind = clientextras.TransportWS
	case "wss":
		kind = clientextras.TransportWSS
	default:
		return 0, "", 0, "", fmt.Errorf("unsupported world-server scheme %q (use tcp://, ws://, or wss://)", u.Scheme)
	}
	host = u.Hostname()
	if host == "" {
		return 0, "", 0, "", fmt.Errorf("world-server %q has no host", arg)
	}
	p := u.Port()
	if p == "" {
		return 0, "", 0, "", fmt.Errorf("world-server %q must include an explicit port", arg)
	}
	port, err = strconv.Atoi(p)
	if err != nil {
		return 0, "", 0, "", fmt.Errorf("invalid world-server port in %q: %w", arg, err)
	}
	if port < 1 || port > 65535 {
		return 0, "", 0, "", fmt.Errorf("world-server port %d out of range (1-65535)", port)
	}
	path = u.EscapedPath()
	if kind == clientextras.TransportTCP {
		if path != "" && path != "/" {
			return 0, "", 0, "", fmt.Errorf("world-server %q: tcp:// does not take a path", arg)
		}
		path = ""
	}
	return kind, host, port, path, nil
}

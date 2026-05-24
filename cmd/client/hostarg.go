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
	// wsPort == 0 means "no explicit port; use the default game port". A literal
	// ws://host:0 collapses to the same sentinel — port 0 is not a valid game
	// port and the dial site would reject it anyway, so the ambiguity is benign.
	if p := u.Port(); p != "" {
		wsPort, err = strconv.Atoi(p)
		if err != nil {
			return 0, "", 0, "", fmt.Errorf("invalid port in host %q: %w", arg, err)
		}
	}
	wsPath = u.EscapedPath()
	return tk, host, wsPort, wsPath, nil
}

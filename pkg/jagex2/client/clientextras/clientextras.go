// Package clientextras contains variables from client that resulted in circular dependencies.
package clientextras

import "os"

var Field1307 [][]int = [][]int{{6798, 107, 10283, 16, 4797, 7744, 5799, 4634, 33697, 22433, 2983, 54193}, {8741, 12, 64030, 43162, 7735, 8404, 1701, 38430, 24094, 10153, 56621, 4783, 1341, 16578, 35003, 25239}, {25238, 8742, 12, 64030, 43162, 7735, 8404, 1701, 38430, 24094, 10153, 56621, 4783, 1341, 16578, 35003}, {4626, 11146, 6439, 12, 4758, 10270}, {4550, 4537, 5681, 5673, 5790, 6806, 8076, 4574}}

var Field1438 []int = []int{9104, 10275, 7595, 3610, 7975, 8526, 918, 38802, 24466, 10145, 58654, 5027, 1457, 16565, 34991, 25486}

var LoopCycle int

// Java: getHost() (deob/client.java:5508-5514) and the socket path
// getCodeBase().getHost() (deob/client.java:7244). With no signed applet and no
// frame, those resolve to the document-base/loopback host; "127.0.0.1" is the
// standalone default (matching the literal http://127.0.0.1:... at client.java:7624).
var Host = "127.0.0.1"

// Transport selects the game-server connection transport. It is set once at
// startup from the -world-server flag's URL scheme and read by
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

// WorldPort is the authoritative game-server port for every transport (TCP,
// ws, wss). On the native build it is parsed from the -world-server flag; the
// js/wasm build derives it from the page origin in signlink.ConfigureTransport.
// The 43594 default matches the Java client's base game port (the literal in
// openSocket(portOffset + 43594), deob/client.java:6786).
var WorldPort = 43594

// WSPath is an explicit path parsed from a ws[s]:// -world-server flag.
// "" means "/".
var WSPath string

// OndemandBaseURL is the scheme://host:port the native build fetches cache/data
// resources against — read by both signlink.OpenURL (signlink_url_native.go)
// and client.GetCodeBase (codebase_native.go). Set from the -ondemand-server
// flag; the default is http://127.0.0.1:8080. (Java's standalone build fetched
// cache data from a fixed URL at deob/client.java:7624 + portOffset; that literal
// is not mirrored — the Go default is configured directly.) The js/wasm build
// ignores this and derives the origin from window.location instead.
var OndemandBaseURL = "http://127.0.0.1:8080"

// ExitFunc terminates the process and defaults to os.Exit. GameShell.Shutdown
// and the standalone launch loop exit through it. Embedders that co-host other
// subsystems in the process (e.g. goscape-singleplayer's in-process server)
// replace it at startup — before the client runs — to perform a graceful
// shutdown of those subsystems first. Lives here rather than in pkg/jagex2/launch
// because package client (GameShell) must reach it without an import cycle.
var ExitFunc func(code int) = os.Exit

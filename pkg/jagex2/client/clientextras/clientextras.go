// Package clientextras contains variables from client that resulted in circular dependencies.
package clientextras

var Field1307 [][]int = [][]int{{6798, 107, 10283, 16, 4797, 7744, 5799, 4634, 33697, 22433, 2983, 54193}, {8741, 12, 64030, 43162, 7735, 8404, 1701, 38430, 24094, 10153, 56621, 4783, 1341, 16578, 35003, 25239}, {25238, 8742, 12, 64030, 43162, 7735, 8404, 1701, 38430, 24094, 10153, 56621, 4783, 1341, 16578, 35003}, {4626, 11146, 6439, 12, 4758, 10270}, {4550, 4537, 5681, 5673, 5790, 6806, 8076, 4574}}

var Field1438 []int = []int{9104, 10275, 7595, 3610, 7975, 8526, 918, 38802, 24466, 10145, 58654, 5027, 1457, 16565, 34991, 25486}

var LoopCycle int

// Java: getHost() (deob/client.java:5508-5514) and the socket path
// getCodeBase().getHost() (deob/client.java:7244). With no signed applet and no
// frame, those resolve to the document-base/loopback host; "127.0.0.1" is the
// standalone default (matching the literal http://127.0.0.1:... at client.java:7624).
var Host = "127.0.0.1"

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
// 0 means "use the default game port the dial site supplies (43594)".
var WSPort int

// WSPath is an explicit path parsed from a ws[s]:// host argument.
// "" means "/".
var WSPath string

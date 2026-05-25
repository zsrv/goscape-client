package signlink

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
	"github.com/zsrv/goscape-client/pkg/jagex2/io/clientstream"
)

// TestOpenWebSocketRoundTrip dials an in-process echo server through
// openWebSocket, wraps the result in a ClientStream, and verifies bytes
// written are read back unchanged — proving the NetConn adapter and
// ClientStream interoperate over binary WebSocket frames.
func TestOpenWebSocketRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols: []string{"binary"},
		})
		if err != nil {
			return
		}
		defer func() { _ = c.CloseNow() }()
		nc := websocket.NetConn(r.Context(), c, websocket.MessageBinary)
		_, _ = io.Copy(nc, nc) // echo until the client closes
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("server port: %v", err)
	}

	// No explicit override: use the server's port as the default, root path.
	clientextras.WSPort = 0
	clientextras.WSPath = ""

	conn, err := openWebSocket(clientextras.TransportWS, u.Hostname(), port, 10*time.Second)
	if err != nil {
		t.Fatalf("openWebSocket: %v", err)
	}
	cs := clientstream.NewClientStream(conn)
	defer cs.Close()

	msg := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	// ClientStream.Write(buf, length, offset) — note the (buf, len, off) order.
	if err := cs.Write(msg, len(msg), 0); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := make([]byte, len(msg))
	if err := cs.ReadFully(got, 0, len(msg)); err != nil {
		t.Fatalf("readfully: %v", err)
	}
	if !bytes.Equal(got, msg) {
		t.Fatalf("round-trip = %v, want %v", got, msg)
	}
}

func TestBuildWSURL(t *testing.T) {
	tests := []struct {
		name         string
		kind         clientextras.TransportKind
		host         string
		defaultPort  int
		overridePort int
		overridePath string
		want         string
	}{
		{"bare ws default port", clientextras.TransportWS, "gameserver", 43594, 0, "", "ws://gameserver:43594/"},
		{"caller-supplied default port", clientextras.TransportWS, "gameserver", 43595, 0, "", "ws://gameserver:43595/"},
		{"non-positive override falls back to default", clientextras.TransportWS, "gameserver", 43594, -1, "", "ws://gameserver:43594/"},
		{"override port", clientextras.TransportWS, "10.0.0.5", 43594, 8080, "", "ws://10.0.0.5:8080/"},
		{"wss with port and path", clientextras.TransportWSS, "play.example.com", 43594, 443, "/ws", "wss://play.example.com:443/ws"},
		{"path no override port", clientextras.TransportWS, "host", 43594, 0, "/path", "ws://host:43594/path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWSURL(tt.kind, tt.host, tt.defaultPort, tt.overridePort, tt.overridePath)
			if got != tt.want {
				t.Fatalf("buildWSURL = %q, want %q", got, tt.want)
			}
		})
	}
}

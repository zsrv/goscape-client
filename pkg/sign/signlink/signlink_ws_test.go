package signlink

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

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

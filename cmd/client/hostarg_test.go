package main

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

func TestParseHostArg(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		wantKind clientextras.TransportKind
		wantHost string
		wantPort int
		wantPath string
		wantErr  bool
	}{
		{"bare host", "localhost", clientextras.TransportTCP, "localhost", 0, "", false},
		{"bare ip", "10.0.0.5", clientextras.TransportTCP, "10.0.0.5", 0, "", false},
		{"ws no port", "ws://gameserver", clientextras.TransportWS, "gameserver", 0, "", false},
		{"ws with port", "ws://10.0.0.5:8080", clientextras.TransportWS, "10.0.0.5", 8080, "", false},
		{"wss port and path", "wss://play.example.com:443/ws", clientextras.TransportWSS, "play.example.com", 443, "/ws", false},
		{"ws trailing slash", "ws://gameserver/", clientextras.TransportWS, "gameserver", 0, "/", false},
		{"ws ipv6 with port", "ws://[::1]:8080", clientextras.TransportWS, "::1", 8080, "", false},
		{"unsupported scheme", "http://example.com", 0, "", 0, "", true},
		{"empty hostname", "ws://", 0, "", 0, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, host, port, path, err := parseHostArg(tt.arg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseHostArg(%q) = nil error, want error", tt.arg)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseHostArg(%q) unexpected error: %v", tt.arg, err)
			}
			if kind != tt.wantKind || host != tt.wantHost || port != tt.wantPort || path != tt.wantPath {
				t.Fatalf("parseHostArg(%q) = (%v, %q, %d, %q), want (%v, %q, %d, %q)",
					tt.arg, kind, host, port, path, tt.wantKind, tt.wantHost, tt.wantPort, tt.wantPath)
			}
		})
	}
}

package main

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

func TestParseWorldServer(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		wantKind clientextras.TransportKind
		wantHost string
		wantPort int
		wantPath string
		wantErr  bool
	}{
		{"tcp localhost", "tcp://127.0.0.1:43594", clientextras.TransportTCP, "127.0.0.1", 43594, "", false},
		{"tcp host", "tcp://gs.example.com:40000", clientextras.TransportTCP, "gs.example.com", 40000, "", false},
		{"tcp trailing slash ok", "tcp://gs.example.com:40000/", clientextras.TransportTCP, "gs.example.com", 40000, "", false},
		{"ws no path", "ws://gameserver:43594", clientextras.TransportWS, "gameserver", 43594, "", false},
		{"wss port and path", "wss://play.example.com:443/ws", clientextras.TransportWSS, "play.example.com", 443, "/ws", false},
		{"ws trailing slash", "ws://gs:43594/", clientextras.TransportWS, "gs", 43594, "/", false},
		{"ws ipv6", "ws://[::1]:8080", clientextras.TransportWS, "::1", 8080, "", false},
		{"missing scheme", "localhost", 0, "", 0, "", true},
		{"missing scheme with port", "localhost:43594", 0, "", 0, "", true},
		{"missing port tcp", "tcp://gs.example.com", 0, "", 0, "", true},
		{"missing port wss", "wss://host", 0, "", 0, "", true},
		{"unsupported scheme", "http://example.com:80", 0, "", 0, "", true},
		{"tcp with path", "tcp://gs.example.com:40000/path", 0, "", 0, "", true},
		{"empty host", "ws://:43594", 0, "", 0, "", true},
		{"bad port", "tcp://gs:notaport", 0, "", 0, "", true},
		{"port out of range", "tcp://gs:70000", 0, "", 0, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, host, port, path, err := parseWorldServer(tt.arg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseWorldServer(%q) = nil error, want error", tt.arg)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseWorldServer(%q) unexpected error: %v", tt.arg, err)
			}
			if kind != tt.wantKind || host != tt.wantHost || port != tt.wantPort || path != tt.wantPath {
				t.Fatalf("parseWorldServer(%q) = (%v, %q, %d, %q), want (%v, %q, %d, %q)",
					tt.arg, kind, host, port, path, tt.wantKind, tt.wantHost, tt.wantPort, tt.wantPath)
			}
		})
	}
}

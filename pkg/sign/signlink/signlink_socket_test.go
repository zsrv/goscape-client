package signlink

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

func TestResolveWSTarget(t *testing.T) {
	cases := []struct {
		name           string
		hostname, port string
		protocol       string
		wantKind       clientextras.TransportKind
		wantHost       string
		wantPort       int
	}{
		{"http with explicit port", "localhost", "8080", "http:", clientextras.TransportWS, "localhost", 8080},
		{"https default port", "example.com", "", "https:", clientextras.TransportWSS, "example.com", 443},
		{"http default port", "example.com", "", "http:", clientextras.TransportWS, "example.com", 80},
		{"https explicit port", "10.0.0.1", "443", "https:", clientextras.TransportWSS, "10.0.0.1", 443},
		// A non-numeric port (Atoi returns 0) must fall through to the scheme
		// default — this also covers the "<undefined>" string syscall/js emits
		// for a missing location.port, which native tooling can't exercise.
		{"non-numeric port falls back to default", "example.com", "abc", "http:", clientextras.TransportWS, "example.com", 80},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, host, port := resolveWSTarget(c.hostname, c.port, c.protocol)
			if kind != c.wantKind || host != c.wantHost || port != c.wantPort {
				t.Fatalf("got (%v,%q,%d), want (%v,%q,%d)",
					kind, host, port, c.wantKind, c.wantHost, c.wantPort)
			}
		})
	}
}

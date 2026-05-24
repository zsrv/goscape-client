package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServesFromBundle(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"index.html", "main.wasm", "wasm.js"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		path string
		want bool
	}{
		{"/", true},          // index.html
		{"/main.wasm", true}, // bundle asset
		{"/wasm.js", true},   // bundle asset
		// Cache-data requests: not present in the bundle dir, must be proxied.
		{"/crc53108508", false},
		{"/title12345", false},
		{"/scape_main_1.mid", false},
		{"/worldmap.jag", false},
		// Path-traversal attempts collapse under the bundle dir and don't exist.
		{"/../main.go", false},
	}
	for _, c := range cases {
		if got := servesFromBundle(dir, c.path); got != c.want {
			t.Errorf("servesFromBundle(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsWebSocketUpgrade(t *testing.T) {
	cases := []struct {
		name                string
		upgrade, connection string
		want                bool
	}{
		{"ws upgrade", "websocket", "Upgrade", true},
		{"ws upgrade with keep-alive", "websocket", "keep-alive, Upgrade", true},
		{"case-insensitive", "WebSocket", "upgrade", true},
		{"plain GET", "", "", false},
		{"upgrade but not websocket", "h2c", "Upgrade", false},
		{"websocket header but no connection upgrade", "websocket", "keep-alive", false},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		if c.upgrade != "" {
			r.Header.Set("Upgrade", c.upgrade)
		}
		if c.connection != "" {
			r.Header.Set("Connection", c.connection)
		}
		if got := isWebSocketUpgrade(r); got != c.want {
			t.Errorf("%s: isWebSocketUpgrade = %v, want %v", c.name, got, c.want)
		}
	}
}

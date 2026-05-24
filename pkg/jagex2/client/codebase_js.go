//go:build js

package client

import "syscall/js"

// codeBaseURL returns the page's own origin (e.g. "http://localhost:8080") so
// the browser fetches cache data (crc, jag archives, MIDI) same-origin — no
// CORS. This mirrors the Java applet's getCodeBase() document base and the
// Client-TS relative-path fetches, and pairs with signlink.ConfigureTransport,
// which derives the WebSocket target from the same window.location. See
// GetCodeBase for the platform split.
func codeBaseURL() string {
	return js.Global().Get("location").Get("origin").String()
}

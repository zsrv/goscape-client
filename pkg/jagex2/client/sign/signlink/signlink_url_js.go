//go:build js

package signlink

import "syscall/js"

// urlBase returns the page's own origin so signlink.OpenURL fetches cache
// resources (the SF2 soundfont, reporterror) same-origin — no CORS. Mirrors
// client.GetCodeBase (codebase_js.go) and signlink.ConfigureTransport, which
// derive from the same window.location. The serving origin (e.g. wasmserve)
// proxies these to the data backend.
func urlBase() string {
	return js.Global().Get("location").Get("origin").String()
}

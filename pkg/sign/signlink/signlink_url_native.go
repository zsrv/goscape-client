//go:build !js

package signlink

// dataServerURL is the scheme://host[:port] that signlink.OpenURL fetches
// against on the native standalone build. It mirrors Java's literal
// http://127.0.0.1:8888 (deob/client.java:7624). The Java port offset
// (portOffset+8888) is intentionally not ported — see cmd/client/main.go for
// the rationale. This is a var rather than a const only so tests can redirect
// it at an httptest server; production never reassigns it.
var dataServerURL = "http://127.0.0.1:8888"

// urlBase is the scheme://host[:port] that signlink.OpenURL fetches against.
// See signlink_url_js.go for the browser origin-derived variant.
func urlBase() string {
	return dataServerURL
}

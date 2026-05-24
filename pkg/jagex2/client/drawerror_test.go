package client

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
)

// TestDrawError_NilFontDoesNotPanic reproduces the segfault seen when a host is
// specified on the command line: Client.Load flags an error (e.g. ErrorHost
// when the host is not in the allowlist) and returns BEFORE FontBold12 is
// loaded, so DrawError dereferenced a nil *PixFont. Java's drawError used
// always-available AWT system fonts; the Go port reuses the cache-loaded
// FontBold12, which is nil on these early-error paths. DrawError must degrade
// to background-only rather than crash.
//
// All three error blocks (ErrorLoading, ErrorHost, ErrorStarted) draw text and
// so must each tolerate a nil font.
func TestDrawError_NilFontDoesNotPanic(t *testing.T) {
	cases := []struct {
		name  string
		apply func(c *Client)
	}{
		{"ErrorHost", func(c *Client) { c.ErrorHost = true }},
		{"ErrorLoading", func(c *Client) { c.ErrorLoading = true }},
		{"ErrorStarted", func(c *Client) { c.ErrorStarted = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(pix2d.Reset)
			c := &Client{}
			c.ScreenWidth = 789
			c.ScreenHeight = 532
			tc.apply(c)
			// c.FontBold12 is intentionally nil (fonts not yet loaded).
			c.DrawError() // must not panic
		})
	}
}

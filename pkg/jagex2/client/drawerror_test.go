package client

import (
	"slices"
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/platform/platformtest"
)

// TestDrawError_RendersWithoutCacheFont covers the crash and the blank-screen
// bug seen when a host is specified: Client.Load flags an error (e.g. ErrorHost
// when the host is not in the allowlist) and returns BEFORE the cache fonts
// load, so the original DrawError dereferenced a nil *PixFont (SIGSEGV), and
// the first fix made it render nothing. DrawError now draws through the boot
// font (basicfont.Face7x13), which is always available — so each error block
// must (a) not panic with a nil B12 and (b) actually paint its text into
// the overlay.
func TestDrawError_RendersWithoutCacheFont(t *testing.T) {
	cases := []struct {
		name      string
		apply     func(c *Client)
		wantColor int // a text color the block paints (vs the 0x000000 background)
	}{
		{"ErrorHost", func(c *Client) { c.ErrorHost = true }, 0xFFFFFF},
		{"ErrorLoading", func(c *Client) { c.ErrorLoading = true }, 0xFFFF00},
		{"ErrorStarted", func(c *Client) { c.ErrorStarted = true }, 0xFFFF00},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(pix2d.Reset)
			defer platformtest.Install()()
			c := &Client{}
			c.ScreenWidth = 789
			c.ScreenHeight = 532
			tc.apply(c)
			// c.B12 is intentionally nil (cache fonts not yet loaded).
			c.DrawError() // must not panic
			if !slices.Contains(c.OverlayPixMap.Data, tc.wantColor) {
				t.Errorf("no %#06x text pixels in overlay; error message did not render", tc.wantColor)
			}
		})
	}
}

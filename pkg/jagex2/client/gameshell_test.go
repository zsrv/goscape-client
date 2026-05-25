package client

import (
	"slices"
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/platform/platformtest"
)

// Regression: the host-shell refactor (036edcb) removed InitApplication, which
// used to set ScreenWidth/Height from the window size. RunShell must
// re-establish them from the active backend BEFORE the first DrawProgress, or
// ensureOverlay allocates a 0x0 PixMap whose empty pixel slice crashes the
// native gl.Ptr (panic: reflect: slice index out of range). This test does NOT
// hand-set the dimensions — it exercises the production init path.
func TestInitScreenSize_PopulatesNonEmptyOverlayFromBackend(t *testing.T) {
	t.Cleanup(pix2d.Reset)
	defer platformtest.Install()() // fake backend Size() = 789x532

	c := &Client{} // ScreenWidth/Height intentionally left zero

	c.initScreenSize()

	if c.ScreenWidth != 789 || c.ScreenHeight != 532 {
		t.Fatalf("initScreenSize set (%d,%d); want (789,532) from backend",
			c.ScreenWidth, c.ScreenHeight)
	}

	// With dims set, the overlay must allocate non-empty (the crash precondition).
	c.Refresh = true
	c.DrawProgressGameShell("Loading...", 0)
	if c.OverlayPixMap == nil || len(c.OverlayPixMap.Data) == 0 {
		t.Fatalf("overlay empty after boot init; native gl.Ptr would panic")
	}
	if c.OverlayPixMap.Width != 789 || c.OverlayPixMap.Height != 532 {
		t.Errorf("overlay size = (%d,%d); want (789,532)",
			c.OverlayPixMap.Width, c.OverlayPixMap.Height)
	}
}

func TestDrawProgressGameShell_ClearsRefreshAndPopulatesOverlay(t *testing.T) {
	t.Cleanup(pix2d.Reset)
	defer platformtest.Install()()

	c := &Client{}
	c.ScreenWidth = 789
	c.ScreenHeight = 532
	c.Refresh = true

	c.DrawProgressGameShell("Connecting to fileserver", 25)

	if c.Refresh {
		t.Errorf("Refresh = true after DrawProgressGameShell; want false")
	}
	if c.OverlayPixMap == nil {
		t.Fatalf("OverlayPixMap nil after DrawProgressGameShell")
	}
	if c.OverlayPixMap.Width != 789 || c.OverlayPixMap.Height != 532 {
		t.Errorf("OverlayPixMap size = (%d,%d); want (789,532)",
			c.OverlayPixMap.Width, c.OverlayPixMap.Height)
	}
	// Verify at least one red pixel from the bar fill exists somewhere
	// in the overlay buffer (indicating draw calls actually fired).
	if !slices.Contains(c.OverlayPixMap.Data, 0x8C1111) {
		t.Errorf("no 0x8C1111 (bar red) pixels found in overlay; draw did not fire")
	}
}

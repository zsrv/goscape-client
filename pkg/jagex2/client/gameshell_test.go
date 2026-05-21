package client

import "testing"

func TestDrawProgressGameShell_ClearsRefreshAndPopulatesOverlay(t *testing.T) {
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
	foundRed := false
	for _, px := range c.OverlayPixMap.Data {
		if px == 0x8C1111 {
			foundRed = true
			break
		}
	}
	if !foundRed {
		t.Errorf("no 0x8C1111 (bar red) pixels found in overlay; draw did not fire")
	}
}

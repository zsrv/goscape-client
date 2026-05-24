package errorfont

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pixmap"
)

// TestDrawString_RendersColoredPixels verifies the Go-bold face actually paints
// glyph pixels into the PixMap: a solid-interior pixel of a bold glyph lands at
// full coverage, so the exact text color appears over the (zeroed/black) buffer.
func TestDrawString_RendersColoredPixels(t *testing.T) {
	p := pixmap.NewPixMap(240, 40)
	DrawString(p, 5, 28, 0xFFFFFF, "Error")

	var white int
	for _, px := range p.Data {
		if px == 0xFFFFFF {
			white++
		}
	}
	if white == 0 {
		t.Fatal("no full-coverage white text pixels rendered")
	}
}

// TestDrawString_EmptyIsNoop confirms an empty string leaves the buffer
// untouched (no allocation, no writes).
func TestDrawString_EmptyIsNoop(t *testing.T) {
	p := pixmap.NewPixMap(20, 20)
	DrawString(p, 0, 10, 0xFFFFFF, "")
	for i, px := range p.Data {
		if px != 0 {
			t.Fatalf("empty string wrote pixel %#06x at %d", px, i)
		}
	}
}

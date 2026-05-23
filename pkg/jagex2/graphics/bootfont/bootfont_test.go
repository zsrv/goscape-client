package bootfont

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pixmap"
)

func TestHeight(t *testing.T) {
	if got := Height(); got != 13 {
		t.Fatalf("Height() = %d, want 13", got)
	}
}

func TestStringWidthEmpty(t *testing.T) {
	if got := StringWidth(""); got != 0 {
		t.Fatalf("StringWidth(\"\") = %d, want 0", got)
	}
}

func TestStringWidthASCII(t *testing.T) {
	// basicfont.Face7x13 advances 7 pixels per glyph.
	if got := StringWidth("hello"); got != 35 {
		t.Fatalf("StringWidth(\"hello\") = %d, want 35", got)
	}
}

func TestDrawStringWritesPixels(t *testing.T) {
	p := pixmap.NewPixMap(200, 50)
	// Pre-fill with sentinel so we can detect any writes from DrawString.
	for i := range p.Data {
		p.Data[i] = 0x0000FF // blue
	}
	DrawString(p, 10, 20, 0xFFFFFF, "A")
	// Scan the bounding box for the "A" glyph. With Face7x13:
	// Advance=7, Width=6, Height=13, Ascent=11.
	// At dot=(10,20), glyph occupies x in [10..15], y in [20-11..20+2]=[9..22].
	written := 0
	for y := 9; y <= 22; y++ {
		for x := 10; x <= 15; x++ {
			if p.Data[y*p.Width+x] == 0xFFFFFF {
				written++
			}
		}
	}
	if written == 0 {
		t.Fatalf("DrawString wrote no white pixels in glyph bounding box; expected at least one")
	}
	// Sanity check the upper bound too: a bug that flood-filled the
	// bounding box with hexColor would also satisfy written > 0. A real
	// "A" glyph from basicfont.Face7x13 fills a small fraction of the
	// 6x14 scan area.
	const boxArea = (15 - 10 + 1) * (22 - 9 + 1)
	if written >= boxArea {
		t.Fatalf("DrawString wrote all %d pixels in glyph bounding box; expected a partial glyph, not a fill", written)
	}
}

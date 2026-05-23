package pixmap

import (
	"bytes"
	"image"
	"testing"
)

// TestConvertPixmapPixelsProducesRGBA pins down the format the converter
// must emit. Gio's paint.NewImageOp fast-paths *image.RGBA (used as-is)
// but force-converts every other type via a per-frame draw.Draw — which
// the baseline profile (profiles/20260523T122349Z) showed costing ~25%
// of CPU and ~48% of all allocations. Because the PixMap is always fully
// opaque (Java DirectColorModel, no alpha mask), premultiplied RGBA and
// straight NRGBA are byte-identical, so emitting RGBA is both faster and
// pixel-equivalent. This test locks in the type and the exact bytes.
func TestConvertPixmapPixelsProducesRGBA(t *testing.T) {
	width, height := 2, 1
	// Distinct channels per pixel so a channel-order regression is caught.
	pixels := []int{0x00FF8040, 0x00010203}

	img := convertPixmapPixels(width, height, pixels)

	rgba, ok := any(img).(*image.RGBA)
	if !ok {
		t.Fatalf("convertPixmapPixels returned %T, want *image.RGBA "+
			"(Gio fast-paths only *image.RGBA; other types force a "+
			"per-frame draw.Draw conversion — see baseline profile)", img)
	}

	want := []uint8{
		0xFF, 0x80, 0x40, 0xFF, // pixel 0: R, G, B, A(opaque)
		0x01, 0x02, 0x03, 0xFF, // pixel 1
	}
	if !bytes.Equal(rgba.Pix, want) {
		t.Errorf("Pix = %v, want %v", rgba.Pix, want)
	}
}

// TestWritePixmapPixelsFillsRGBA verifies the in-place fill writes the
// same opaque [R,G,B,0xFF] bytes the allocating converter produced.
func TestWritePixmapPixelsFillsRGBA(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 2, 1))
	pixels := []int{0x00FF8040, 0x00010203}

	writePixmapPixels(dst, pixels)

	want := []uint8{
		0xFF, 0x80, 0x40, 0xFF, // pixel 0: R, G, B, A(opaque)
		0x01, 0x02, 0x03, 0xFF, // pixel 1
	}
	if !bytes.Equal(dst.Pix, want) {
		t.Errorf("Pix = %v, want %v", dst.Pix, want)
	}
}

// TestWritePixmapPixelsReusesBuffer documents the reuse contract: a second
// fill overwrites the first's content and does NOT reallocate the backing
// array (the whole point of the optimization).
func TestWritePixmapPixelsReusesBuffer(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 2, 1))
	before := dst.Pix

	writePixmapPixels(dst, []int{0x00112233, 0x00445566})
	writePixmapPixels(dst, []int{0x00AABBCC, 0x00DDEEFF})

	want := []uint8{
		0xAA, 0xBB, 0xCC, 0xFF,
		0xDD, 0xEE, 0xFF, 0xFF,
	}
	if !bytes.Equal(dst.Pix, want) {
		t.Errorf("after reuse Pix = %v, want %v", dst.Pix, want)
	}
	if &dst.Pix[0] != &before[0] {
		t.Error("writePixmapPixels reallocated the backing array; expected in-place reuse")
	}
}

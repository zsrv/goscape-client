package pixmap

import (
	"bytes"
	"image"
	"testing"

	"gioui.org/op"
)

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

// TestNewPixMapAllocatesImageBuffer verifies the reusable upload buffer is
// created at construction time, sized to the PixMap, so Draw never allocates.
func TestNewPixMapAllocatesImageBuffer(t *testing.T) {
	p := NewPixMap(4, 3)

	if p.imgBuf == nil {
		t.Fatal("NewPixMap did not allocate imgBuf")
	}
	if b := p.imgBuf.Bounds(); b.Dx() != 4 || b.Dy() != 3 {
		t.Errorf("imgBuf bounds = %v, want 4x3", b)
	}
}

func TestHashPixelsDetectsChange(t *testing.T) {
	a := []int{1, 2, 3, 0xFFFFFF}
	if hashPixels(a) != hashPixels([]int{1, 2, 3, 0xFFFFFF}) {
		t.Fatal("identical data hashed differently")
	}
	if hashPixels(a) == hashPixels([]int{1, 2, 3, 0xFFFFFE}) {
		t.Fatal("different data hashed identically")
	}
}

func TestPixMapUploadsOnlyOnChange(t *testing.T) {
	p := NewPixMap(4, 4)
	var ops op.Ops

	p.Draw(&ops, 0, 0) // first draw must upload
	g1 := p.imageOp.Generation()
	if g1 == 0 {
		t.Fatalf("first Draw should bump generation, got %d", g1)
	}

	ops.Reset()
	p.Draw(&ops, 0, 0) // unchanged -> no re-upload
	if g2 := p.imageOp.Generation(); g2 != g1 {
		t.Fatalf("unchanged Draw re-uploaded: %d -> %d", g1, g2)
	}

	p.Data[5] = 0x123456 // change a pixel
	ops.Reset()
	p.Draw(&ops, 0, 0) // changed -> re-upload
	if g3 := p.imageOp.Generation(); g3 == g1 {
		t.Fatalf("changed Draw did not re-upload (gen stayed %d)", g3)
	}
}

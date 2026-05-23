package pix32

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
)

// TestDrawRotatedMaskedRecoversOnOutOfBounds locks in the parity fix that
// restores Java's `catch (Exception){}` swallow around drawRotatedMasked
// (Pix32.java:442-468). With a valid destination but a zoom large enough to
// drive the rotated *source* index negative (-12 here), the read of p.Pixels
// would panic with index-out-of-range; Java tolerates this and skips the draw.
// The method's deferred recover must absorb the panic — if it is removed, this
// test crashes the goroutine and fails.
func TestDrawRotatedMaskedRecoversOnOutOfBounds(t *testing.T) {
	pix2d.Reset()
	t.Cleanup(pix2d.Reset)
	pix2d.Bind(4, make([]int, 16), 4)

	p := NewPix321(2, 2) // Pixels len 4, Wi 2

	// angle 0, zoom 1000: srcX = srcY = -256000, so the source index
	// (srcX>>16)+(srcY>>16)*Wi == (-4)+(-4)*2 == -12 (out of bounds).
	p.DrawRotatedMasked(0, 2, []int{0}, 1, 0, 1000, 0, 0, 0, []int{1})

	// Reaching this point means the out-of-bounds access was recovered.
}

// TestCropRecoversOnOutOfBounds covers the sibling guard in Crop
// (Pix32.java:302-353, "error in sprite clipping routine"). A zero scale
// denominator forces a divide-by-zero panic that Java swallows.
func TestCropRecoversOnOutOfBounds(t *testing.T) {
	pix2d.Reset()
	t.Cleanup(pix2d.Reset)
	pix2d.Bind(4, make([]int, 16), 4)

	p := NewPix321(2, 2)

	// arg2 == 0 makes the `(var6 << 16) / arg2` divisions panic (integer
	// divide by zero); the deferred recover must absorb it.
	p.Crop(4, 0, 0, 0)

	// Reaching this point means the panic was recovered.
}

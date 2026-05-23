package pixmap

import (
	"image"
	"testing"
)

// benchPixels builds a full client-window worth of varied opaque pixels.
const benchW, benchH = 532, 789

func benchPixels() []int {
	p := make([]int, benchW*benchH)
	for i := range p {
		// Spread values across all three channels so the conversion
		// can't be specialised by a constant-folding optimiser.
		p[i] = (i * 2654435761) & 0x00FFFFFF
	}
	return p
}

func BenchmarkWritePixmapPixels(b *testing.B) {
	pixels := benchPixels()
	// Allocate the destination ONCE outside the loop: the steady-state
	// per-frame cost is the fill alone, which must be 0 allocs/op.
	dst := image.NewRGBA(image.Rect(0, 0, benchW, benchH))
	for b.Loop() {
		writePixmapPixels(dst, pixels)
	}
}

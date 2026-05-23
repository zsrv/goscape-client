package pixmap

import "testing"

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

func BenchmarkConvertPixmapPixels(b *testing.B) {
	pixels := benchPixels()
	for b.Loop() {
		_ = convertPixmapPixels(benchW, benchH, pixels)
	}
}

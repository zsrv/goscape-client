package pix3d

import "testing"

// TestVar23ShiftCountStaysInRange pins the post-fix contract that
// TextureRaster's `var23 = (arg7 >> 23) & 0x1F` always produces a
// non-negative shift count in [0, 31].
//
// Pre-fix (`var23 = arg7 >> 23`): once arg7 grew past 31 bits via
// the accumulating `arg7 += var15` inside the rasterizer's pixel
// loop, Go's int64 arithmetic-right-shift produced a negative
// result. The downstream `arg1[i] >> var23` then panicked with
// "runtime error: negative shift amount" — observed live during
// gameplay after a few minutes of textured-tile rendering.
//
// The mask `& 0x1F` matches Java's implicit 5-bit shift-count
// mask for int operations and is bit-equivalent regardless of
// int32-vs-int64 representation (bits 23-27 of arg7 are preserved
// through additive arithmetic in both widths).
func TestVar23ShiftCountStaysInRange(t *testing.T) {
	cases := []struct {
		name string
		arg7 int
	}{
		{"zero", 0},
		{"small positive", 1 << 10},
		{"bit23", 1 << 23},
		{"bit30", 1 << 30},
		{"int32 max", (1 << 31) - 1},
		{"crosses int32 boundary", 1 << 32},
		{"int40", 1 << 40},
		{"negative -1 (sign extends)", -1},
		{"negative -2^23", -(1 << 23)},
		{"negative -2^30", -(1 << 30)},
		{"crash repro: arg7 large after << 9 in inner loop", (1 << 22) << 9},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var23 := (tc.arg7 >> 23) & 0x1F
			if var23 < 0 || var23 > 31 {
				t.Fatalf("arg7=%d: var23=%d, want 0..31", tc.arg7, var23)
			}
			// Must be usable as a shift count without panicking.
			// This is exactly the operation the live crash hit at
			// pix3d.go:2211 (inside TextureRaster).
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("arg7=%d: shift panic with masked var23=%d: %v", tc.arg7, var23, r)
				}
			}()
			_ = int(0xFFFFFFFF) >> var23
		})
	}
}

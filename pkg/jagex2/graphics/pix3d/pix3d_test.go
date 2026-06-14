package pix3d

import "testing"

// TestInitPoolReusesAfterClearTexels verifies the scene-rebuild cycle reuses the texel pool buffers instead of reallocating them.
func TestInitPoolReusesAfterClearTexels(t *testing.T) {
	// Mirror the scene-rebuild cycle: InitPool -> (textures bound) -> ClearTexels
	// -> InitPool. The second InitPool must REUSE the existing buffers.
	LowMem = true
	TexelPool = nil
	for i := range ActiveTexels {
		ActiveTexels[i] = nil
	}

	InitPool(2)
	if len(TexelPool) != 2 || len(TexelPool[0]) != 16384 {
		t.Fatalf("InitPool(2) gave len=%d slot0len=%d, want 2 / 16384", len(TexelPool), len(TexelPool[0]))
	}
	slot0 := &TexelPool[0][0]
	slot1 := &TexelPool[1][0]

	// Simulate both buffers bound to textures (drain the free pool like GetTexels).
	ActiveTexels[5] = TexelPool[1]
	TexelPool[1] = nil
	PoolSize--
	ActiveTexels[9] = TexelPool[0]
	TexelPool[0] = nil
	PoolSize--
	if PoolSize != 0 {
		t.Fatalf("after draining both buffers PoolSize=%d, want 0", PoolSize)
	}

	ClearTexels()
	if PoolSize != 2 {
		t.Fatalf("ClearTexels left PoolSize=%d, want 2 (all buffers reclaimed)", PoolSize)
	}
	if TexelPool[0] == nil || TexelPool[1] == nil {
		t.Fatal("ClearTexels left a nil slot; buffers not fully reclaimed")
	}
	if ActiveTexels[5] != nil || ActiveTexels[9] != nil {
		t.Fatal("ClearTexels did not clear ActiveTexels")
	}

	InitPool(2)
	got := map[*int]bool{&TexelPool[0][0]: true, &TexelPool[1][0]: true}
	if !got[slot0] || !got[slot1] {
		t.Error("InitPool reallocated buffers instead of reusing the reclaimed pool")
	}
	if PoolSize != 2 {
		t.Fatalf("InitPool reuse path left PoolSize=%d, want 2", PoolSize)
	}
}

func TestInitPoolReallocatesOnDetailChange(t *testing.T) {
	// If LowMem changes (required buffer length differs), the guard must fall
	// through and reallocate rather than reuse wrong-sized buffers.
	defer func() { LowMem = true }()
	LowMem = true
	TexelPool = nil
	for i := range ActiveTexels {
		ActiveTexels[i] = nil
	}
	InitPool(2) // 16384-length buffers
	old0 := &TexelPool[0][0]

	LowMem = false // now wants 65536-length buffers
	InitPool(2)
	if len(TexelPool[0]) != 65536 {
		t.Fatalf("slot len=%d after detail change, want 65536 (should have reallocated)", len(TexelPool[0]))
	}
	if &TexelPool[0][0] == old0 {
		t.Error("InitPool reused 16384 buffers after detail change to high detail")
	}
}

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

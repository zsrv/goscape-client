package flotype

import "testing"

// TestSetColourNonZeroHSL is a regression test for the f07171a fix.
// The previous Go port did integer division on `((arg1>>16)&0xFF) / 256.0`
// (Java cast its operand to double before the divide), so every byte value
// 0..255 went to 0 — every floor surface ended up Hue/Sat/Light = 0 and
// rendered as black. This test pins that pure red (0xFF0000) produces a
// non-zero Saturation (it's pure-saturated red, by definition).
func TestSetColourNonZeroHSL(t *testing.T) {
	f := NewFloType()
	f.SetColour(0xFF0000) // pure red

	// Hue for red is at the 0/256 boundary (var13/6.0 → either ~0 or ~256).
	// Skip pinning Hue since the boundary case is fragile. Saturation and
	// Lightness, however, MUST be non-zero for fully-saturated red.
	if f.Saturation == 0 {
		t.Error("Saturation = 0 for pure red; expected non-zero (integer-div regression)")
	}
	if f.Lightness == 0 {
		t.Error("Lightness = 0 for pure red; expected non-zero")
	}
	// Java math for r=255/256, g=0, b=0:
	//   min = 0, max = 255/256
	//   lightness = (min+max)/2 ≈ 0.498
	//   saturation = (max-min)/(max+min) = 1.0  (since 2*lightness < 1.0)
	//   → Saturation = int(1.0 * 256) = 256, clamped to 0xFF = 255
	//   → Lightness = int(0.498 * 256) ≈ 127
	if f.Saturation != 0xFF {
		t.Errorf("Saturation = %d, want 0xFF (255)", f.Saturation)
	}
	if f.Lightness != 127 {
		t.Errorf("Lightness = %d, want 127", f.Lightness)
	}
}

// TestSetColourMidGray verifies that an unsaturated input (gray) gives
// Saturation = 0 but a non-zero Lightness — the integer-div regression
// would have given Lightness = 0 too.
func TestSetColourMidGray(t *testing.T) {
	f := NewFloType()
	f.SetColour(0x808080) // mid-gray

	if f.Saturation != 0 {
		t.Errorf("Saturation = %d, want 0 (achromatic input)", f.Saturation)
	}
	if f.Lightness == 0 {
		t.Error("Lightness = 0 for mid-gray; expected non-zero (integer-div regression)")
	}
	// Lightness for r=g=b=0x80 → (0x80/256 + 0x80/256)/2 = 0.5; 0.5*256 = 128.
	if f.Lightness != 128 {
		t.Errorf("Lightness = %d, want 128", f.Lightness)
	}
}

// TestSetColourBlack pins the zero-input degenerate case: all HSL fields
// stay at 0, no NaN propagation from divide-by-zero.
func TestSetColourBlack(t *testing.T) {
	f := NewFloType()
	f.SetColour(0x000000)

	if f.Hue != 0 || f.Saturation != 0 || f.Lightness != 0 {
		t.Errorf("black input gave Hue=%d Saturation=%d Lightness=%d, want all 0",
			f.Hue, f.Saturation, f.Lightness)
	}
}

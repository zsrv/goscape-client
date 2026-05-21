package io

import "testing"

// TestGBitOperatorPrecedence — regression for a Java→Go translation gap in
// the bit reader. The Java reference is:
//
//	var5 += this.data[var3] >> var4 - arg1 & BITMASK[arg1];
//
// Java precedence makes shift tighter than `&`, so the mask is applied to
// the shifted value: `(data >> shift) & mask`. A previous Go port wrapped
// `(remainingBits - n) & Bitmask[n]` in parens, masking the SHIFT COUNT
// instead — high bits of the byte leaked through, producing garbage values
// (e.g. 63 instead of 3 when reading the top 2 bits of a 0xFF byte). The
// bug surfaced post-login as a "Too many players" panic because the player
// info bitstream decoded wrong.
func TestGBitOperatorPrecedence(t *testing.T) {
	cases := []struct {
		name   string
		data   []byte
		bitPos int
		n      int
		want   int
	}{
		// Top 2 bits of 0xFF should be 3. Buggy port returned 63.
		{"top2-of-FF", []byte{0xFF, 0x00}, 0, 2, 3},
		// Top 1 bit of 0x80 should be 1. Buggy returned 0x40 (64).
		{"top1-of-80", []byte{0x80, 0x00}, 0, 1, 1},
		// Cross-byte read landing in the n<remainingBits else branch.
		// After 21 bits, bitPos=21 → bytePos=2, remainingBits=3. Read 8.
		// First grab: low 3 bits of byte 2 (0x07 → 0b111). Second: top 5 bits
		// of byte 3 (0xA5 = 10100101 → top 5 = 10100 = 20). Final: (7<<5)|20 = 244.
		{"cross-byte-244", []byte{0xFF, 0xFF, 0x07, 0xA5, 0x00}, 21, 8, 244},
		// Sanity: reading a full byte at byte-boundary uses the
		// equal-branch and should work in both buggy and fixed versions.
		{"full-byte-aligned", []byte{0x42, 0x00}, 0, 8, 0x42},
		// Sanity: zeros stay zero regardless of shift logic.
		{"zeros", []byte{0, 0, 0, 0}, 5, 7, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPacket(tc.data)
			p.BitPos = tc.bitPos
			got := p.GBit(tc.n)
			if got != tc.want {
				t.Errorf("GBit(%d) at bitPos=%d on %x: got %d, want %d",
					tc.n, tc.bitPos, tc.data, got, tc.want)
			}
		})
	}
}

// TestGBitAdvancesBitPos verifies the cursor moves forward exactly n bits
// regardless of byte alignment.
func TestGBitAdvancesBitPos(t *testing.T) {
	p := NewPacket([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	p.AccessBits()
	_ = p.GBit(3)
	_ = p.GBit(5)
	_ = p.GBit(1)
	_ = p.GBit(2)
	want := 3 + 5 + 1 + 2
	if p.BitPos != want {
		t.Fatalf("BitPos = %d, want %d", p.BitPos, want)
	}
}

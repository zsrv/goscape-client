package io

import (
	"testing"
	"unicode/utf8"
)

// TestGJStrLatin1ToUTF8 — Java's `gjstr` decodes the wire bytes with the JVM
// default charset, which on the client is effectively Latin-1: byte 0xA3 →
// U+00A3 ('£'). The Go port previously sliced the raw bytes into a string,
// producing invalid UTF-8 for any byte >= 0x80. Downstream pixfont
// byte-indexing happened to "work" against that invalid UTF-8, but the same
// font code applied to a Go literal like "£" (valid UTF-8 0xC2 0xA3) saw
// different bytes and failed. The fix transcodes Latin-1 → UTF-8 on read so
// all callers see a valid UTF-8 string.
func TestGJStrLatin1ToUTF8(t *testing.T) {
	// Wire bytes: "5gp £" then 0x0A terminator.
	wire := []byte{'5', 'g', 'p', ' ', 0xA3, 0x0A, 0x00}
	p := NewPacket(wire)
	got := p.GJStr()
	want := "5gp £"
	if got != want {
		t.Fatalf("GJStr = %q (%x), want %q (%x)", got, []byte(got), want, []byte(want))
	}
	if !utf8.ValidString(got) {
		t.Fatalf("GJStr returned invalid UTF-8: %x", []byte(got))
	}
	if p.Pos != 6 {
		t.Fatalf("Pos = %d, want 6 (consumed bytes up to and including \\n)", p.Pos)
	}
}

// TestPJStrUTF8ToLatin1 — Java's `pjstr` calls getBytes(0, length, dst, pos)
// which writes the low byte of each UTF-16 code unit. We iterate runes and
// truncate each to a byte: rune '£' (U+00A3) → 0xA3. Pure ASCII passes through
// unchanged.
func TestPJStrUTF8ToLatin1(t *testing.T) {
	buf := make([]byte, 32)
	p := NewPacket(buf)
	p.PJStr("5gp £")
	wantBytes := []byte{'5', 'g', 'p', ' ', 0xA3, 0x0A}
	got := buf[:p.Pos]
	if len(got) != len(wantBytes) {
		t.Fatalf("PJStr wrote %d bytes, want %d (%x)", len(got), len(wantBytes), got)
	}
	for i, b := range wantBytes {
		if got[i] != b {
			t.Fatalf("PJStr byte %d = 0x%02X, want 0x%02X (full: %x)", i, got[i], b, got)
		}
	}
}

// TestPacketRoundTripLatin1 — encode + decode through PJStr/GJStr returns the
// original Go string for any Latin-1-bounded input.
func TestPacketRoundTripLatin1(t *testing.T) {
	cases := []string{"", "hello", "5gp £", "£££", "Foo Bar 123"}
	for _, want := range cases {
		t.Run(want, func(t *testing.T) {
			buf := make([]byte, 128)
			out := NewPacket(buf)
			out.PJStr(want)
			in := NewPacket(buf)
			got := in.GJStr()
			if got != want {
				t.Fatalf("round-trip %q → %q", want, got)
			}
		})
	}
}

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

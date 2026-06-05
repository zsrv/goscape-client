package io

import (
	"math/big"
	"testing"
	"unicode/utf8"
)

// TestGStrLatin1ToUTF8 — Java's `gstr` (254 name; was `gjstr`) decodes the
// wire bytes with the JVM
// default charset, which on the client is effectively Latin-1: byte 0xA3 →
// U+00A3 ('£'). The Go port previously sliced the raw bytes into a string,
// producing invalid UTF-8 for any byte >= 0x80. Downstream pixfont
// byte-indexing happened to "work" against that invalid UTF-8, but the same
// font code applied to a Go literal like "£" (valid UTF-8 0xC2 0xA3) saw
// different bytes and failed. The fix transcodes Latin-1 → UTF-8 on read so
// all callers see a valid UTF-8 string.
func TestGStrLatin1ToUTF8(t *testing.T) {
	// Wire bytes: "5gp £" then 0x0A terminator.
	wire := []byte{'5', 'g', 'p', ' ', 0xA3, 0x0A, 0x00}
	p := NewPacket(wire)
	got := p.GStr()
	want := "5gp £"
	if got != want {
		t.Fatalf("GStr = %q (%x), want %q (%x)", got, []byte(got), want, []byte(want))
	}
	if !utf8.ValidString(got) {
		t.Fatalf("GStr returned invalid UTF-8: %x", []byte(got))
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

// TestPacketRoundTripLatin1 — encode + decode through PJStr/GStr returns the
// original Go string for any Latin-1-bounded input.
func TestPacketRoundTripLatin1(t *testing.T) {
	cases := []string{"", "hello", "5gp £", "£££", "Foo Bar 123"}
	for _, want := range cases {
		t.Run(want, func(t *testing.T) {
			buf := make([]byte, 128)
			out := NewPacket(buf)
			out.PJStr(want)
			in := NewPacket(buf)
			got := in.GStr()
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

// TestJavaBytesFromBigInt_MSBSetPositive verifies that the emit helper
// prepends a 0x00 sign byte when a positive integer's magnitude byte 0 has
// the high bit set, mirroring java.math.BigInteger.toByteArray(). Without
// the sign byte, the Java server's `new BigInteger(byte[])` would re-parse
// the value as negative (two's complement). This was the C12 / 1fb2791 bug.
func TestJavaBytesFromBigInt_MSBSetPositive(t *testing.T) {
	cases := []struct {
		name string
		// value in hex; magnitude byte 0 has the high bit set when expected.
		hex  string
		want []byte
	}{
		{"0x80", "80", []byte{0x00, 0x80}},
		{"0xFF", "FF", []byte{0x00, 0xFF}},
		{"0xC0FE", "C0FE", []byte{0x00, 0xC0, 0xFE}},
		// Magnitude byte 0 clear → no sign byte.
		{"0x7F", "7F", []byte{0x7F}},
		{"0x01", "01", []byte{0x01}},
		// Zero is a special case — Java returns [0x00].
		{"zero", "00", []byte{0x00}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n, ok := new(big.Int).SetString(tc.hex, 16)
			if !ok {
				t.Fatalf("bad hex %q", tc.hex)
			}
			got := javaBytesFromBigInt(n)
			if !equalBytes(got, tc.want) {
				t.Errorf("javaBytesFromBigInt(0x%s) = %#x, want %#x", tc.hex, got, tc.want)
			}
		})
	}
}

// TestJavaBigIntFromBytes_TwosComplement verifies that the parse helper
// treats byte 0's high bit as a two's-complement sign, matching
// `new java.math.BigInteger(byte[])`. Go's big.Int.SetBytes is unsigned.
func TestJavaBigIntFromBytes_TwosComplement(t *testing.T) {
	cases := []struct {
		name string
		b    []byte
		want string // hex; "-" prefix for negative
	}{
		// Positive (byte 0 < 0x80): same as SetBytes.
		{"0x7F", []byte{0x7F}, "7F"},
		// Negative (byte 0 >= 0x80): two's-complement of magnitude.
		// 0xFF as a single byte is -1.
		{"0xFF=-1", []byte{0xFF}, "-1"},
		// 0x80 as a single byte is -128.
		{"0x80=-128", []byte{0x80}, "-80"},
		// 0xFF 0xFE = -2.
		{"0xFFFE=-2", []byte{0xFF, 0xFE}, "-2"},
		// 0x00 0xFF — leading zero forces positive parse.
		{"0x00FF=+255", []byte{0x00, 0xFF}, "FF"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := javaBigIntFromBytes(tc.b)
			want, ok := new(big.Int).SetString(tc.want, 16)
			if !ok {
				t.Fatalf("bad hex %q", tc.want)
			}
			if got.Cmp(want) != 0 {
				t.Errorf("javaBigIntFromBytes(%#x) = %s, want %s",
					tc.b, got.Text(16), tc.want)
			}
		})
	}
}

// TestJavaBigIntRoundTrip_Positive verifies parse-then-emit for the
// positive-integer encoding produced by Java BigInteger.toByteArray().
// The negative-side roundtrip is documented as unreachable through RSAEnc
// (modPow output is always non-negative) so we don't exercise it here.
func TestJavaBigIntRoundTrip_Positive(t *testing.T) {
	cases := [][]byte{
		{0x00},                   // zero
		{0x01},                   // positive small
		{0x7F},                   // largest positive single byte (no sign byte)
		{0x00, 0x80},             // smallest two-byte positive with sign byte
		{0x00, 0xFF},             // 255 with sign byte
		{0x00, 0xFF, 0xFE, 0xFD}, // larger positive with sign byte
		{0x01, 0x02, 0x03, 0x04}, // positive without sign byte
	}
	for _, b := range cases {
		t.Run(string(b), func(t *testing.T) {
			n := javaBigIntFromBytes(b)
			got := javaBytesFromBigInt(n)
			if !equalBytes(got, b) {
				t.Errorf("round trip: in=%#x out=%#x", b, got)
			}
		})
	}
}

// TestRSAEncCiphertextPayload exercises the full RSAEnc emit path: pack
// plaintext via P-ops, call RSAEnc with a known modulus/exponent, and
// verify the emitted length byte + payload mirror Java's
// `out.p1(b.length); out.pdata(b, 0, b.length)` for a ciphertext whose
// magnitude byte 0 has the high bit set.
//
// This was the 1fb2791 bug: Go's big.Int.Bytes() omits the leading 0x00
// sign byte that Java BigInteger.toByteArray() inserts on MSB-set values,
// so the Java server would re-parse the unsigned form as a different
// (sign-extended-negative) BigInteger when decrypting.
func TestRSAEncCiphertextPayload(t *testing.T) {
	// Trivial RSA: mod = 2^256, exp = 1. modPow(x, 1, 2^256) returns the
	// plaintext untouched (so we control the ciphertext byte 0 directly
	// without doing real crypto), letting us drive the emit path under
	// known conditions. The bug under test is the wire format, not the math.
	mod := new(big.Int).Lsh(big.NewInt(1), 256)
	exp := big.NewInt(1)

	// Plaintext bytes [0x00, 0x80, 0x01, 0x02, 0x03, 0x04, 0x05]. Java's
	// `new BigInteger(byte[])` parses this as positive (leading 0x00 forces
	// positive sign), then modPow(.,1,2^256) returns the same magnitude.
	// The magnitude's first nonzero byte is 0x80, so Java's toByteArray()
	// prepends a 0x00 sign byte → wire len = 7.
	plaintextBytes := []byte{0x00, 0x80, 0x01, 0x02, 0x03, 0x04, 0x05}

	p := NewPacket(make([]byte, 128))
	for _, b := range plaintextBytes {
		p.P1(int(b))
	}
	p.RSAEnc(mod, exp)

	// Wire format: [len:1][data:len]. Expect 7-byte payload: sign byte
	// then the 6 magnitude bytes [0x80, 0x01, ... 0x05].
	if p.Data[0] != 7 {
		t.Errorf("length byte = %d, want 7 (sign byte + 6 magnitude bytes)", p.Data[0])
	}
	if p.Data[1] != 0x00 {
		t.Errorf("data[0] = %#x, want 0x00 (Java sign byte)", p.Data[1])
	}
	wantMag := []byte{0x80, 0x01, 0x02, 0x03, 0x04, 0x05}
	for i, want := range wantMag {
		if got := p.Data[2+i]; got != want {
			t.Errorf("data[%d] = %#x, want %#x", 1+i+1, got, want)
		}
	}
}

// TestRSAEncCiphertextPayload_MSBClear is the negative case: no sign byte
// is added when the magnitude's byte 0 already has the high bit clear.
func TestRSAEncCiphertextPayload_MSBClear(t *testing.T) {
	mod := new(big.Int).Lsh(big.NewInt(1), 256)
	exp := big.NewInt(1)

	// Positive plaintext (byte 0 < 0x80) — parses positive, modPow gives
	// itself back, magnitude byte 0 = 0x7F so no sign byte needed.
	plaintextBytes := []byte{0x7F, 0x01, 0x02}

	p := NewPacket(make([]byte, 128))
	for _, b := range plaintextBytes {
		p.P1(int(b))
	}
	p.RSAEnc(mod, exp)

	if p.Data[0] != 3 {
		t.Errorf("length byte = %d, want 3 (no sign byte needed)", p.Data[0])
	}
	if p.Data[1] != 0x7F {
		t.Errorf("data[0] = %#x, want 0x7F", p.Data[1])
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

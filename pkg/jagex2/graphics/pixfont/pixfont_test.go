package pixfont

import "testing"

// TestStringWidthLatin1 exercises the rune-iteration fix in StringWidth: a
// string containing '£' (UTF-8 0xC2 0xA3) must hit the '£' glyph slot once,
// not "two unrelated bytes". With a synthetic DrawWidth table where each
// table index returns its own ordinal, we can verify the per-character width
// sum matches the rune-by-rune expectation rather than the byte-by-byte one.
//
// The bug this guards against: byte-indexing `str[i]` against CHAR_LOOKUP
// would see UTF-8 continuation bytes (0xC2, 0xA3) instead of the codepoint
// 0xA3, producing two distinct lookups summed together — and would also
// PANIC if the underlying CHAR_LOOKUP/DrawWidth arrays were not sized 256.
func TestStringWidthLatin1(t *testing.T) {
	// Save and restore the package-level CHAR_LOOKUP so we can install a
	// deterministic table for the test.
	saved := make([]int, 256)
	copy(saved, CHAR_LOOKUP)
	t.Cleanup(func() { copy(CHAR_LOOKUP, saved) })
	for i := range CHAR_LOOKUP {
		CHAR_LOOKUP[i] = i // identity, just so it stays in range
	}

	p := &PixFont{
		DrawWidth: make([]int, 256),
	}
	// Every codepoint claims width = codepoint value. So a string's total
	// width equals the sum of its codepoints.
	for i := range p.DrawWidth {
		p.DrawWidth[i] = i
	}

	cases := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"ascii", "ab", int('a') + int('b')},
		// "£" is codepoint 0xA3 (163). Byte-indexing UTF-8 would give
		// DrawWidth[0xC2] + DrawWidth[0xA3] = 0xC2 + 0xA3 = 0x165 instead
		// of the correct 0xA3.
		{"pound-only", "£", 0xA3},
		{"mixed", "5gp £", int('5') + int('g') + int('p') + int(' ') + 0xA3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := p.StringWidth(tc.in)
			if got != tc.want {
				t.Fatalf("StringWidth(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// TestStringWidthTagSkip confirms the `@xxx@` tag skip still works after the
// rune conversion — five runes ('@' + 3 tag chars + '@') must be consumed
// without contributing width, regardless of whether the tag interior is
// ASCII.
func TestStringWidthTagSkip(t *testing.T) {
	saved := make([]int, 256)
	copy(saved, CHAR_LOOKUP)
	t.Cleanup(func() { copy(CHAR_LOOKUP, saved) })
	for i := range CHAR_LOOKUP {
		CHAR_LOOKUP[i] = i
	}

	p := &PixFont{DrawWidth: make([]int, 256)}
	for i := range p.DrawWidth {
		p.DrawWidth[i] = 1 // every char contributes 1
	}

	// "@red@ab" → tag (5 runes, contributes 0) + "ab" (2 runes, contributes 2).
	if got := p.StringWidth("@red@ab"); got != 2 {
		t.Fatalf("@red@ab width = %d, want 2", got)
	}
	// Bare "@" (no closing) — counts as a normal char (width 1).
	if got := p.StringWidth("@"); got != 1 {
		t.Fatalf("@ width = %d, want 1", got)
	}
}

package pixfont

import (
	"testing"
)

// TestStringWidthLatin1 exercises the rune-iteration behavior of StringWidth:
// a string containing '£' (UTF-8 0xC2 0xA3) must hit the '£' advance slot
// once, not "two unrelated bytes". With a synthetic CharAdvance table where
// each index returns its own ordinal, the per-character width sum must match
// the rune-by-rune expectation rather than the byte-by-byte one.
//
// The bug this guards against: byte-indexing `str[i]` would see UTF-8
// continuation bytes (0xC2, 0xA3) instead of the codepoint 0xA3, producing
// two distinct lookups summed together. (274 indexes CharAdvance directly by
// char code — the 254-era CHAR_LOOKUP/DrawWidth indirection is gone.)
func TestStringWidthLatin1(t *testing.T) {
	p := &PixFont{
		CharAdvance: make([]int, 256),
	}
	// Every codepoint claims width = codepoint value. So a string's total
	// width equals the sum of its codepoints.
	for i := range p.CharAdvance {
		p.CharAdvance[i] = i
	}

	cases := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"ascii", "ab", int('a') + int('b')},
		// "£" is codepoint 0xA3 (163). Byte-indexing UTF-8 would give
		// CharAdvance[0xC2] + CharAdvance[0xA3] = 0xC2 + 0xA3 = 0x165
		// instead of the correct 0xA3.
		{"pound-only", "£", 0xA3},
		// Space still contributes its advance in stringWid (only the draw
		// methods use ' ' as a plot-skip sentinel; the advance always adds).
		{"mixed", "5gp £", int('5') + int('g') + int('p') + int(' ') + 0xA3},
		// Out-of-Latin-1 runes clamp to ' ' (charIndex guard) — Java 274
		// would AIOOBE here; the Go port substitutes the space advance.
		{"out-of-latin1", "€", int(' ')},
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

// TestStringWidthTagSkip confirms the `@xxx@` tag skip works with the rune
// conversion — five runes ('@' + 3 tag chars + '@') must be consumed without
// contributing width, regardless of whether the tag interior is ASCII.
func TestStringWidthTagSkip(t *testing.T) {
	p := &PixFont{CharAdvance: make([]int, 256)}
	for i := range p.CharAdvance {
		p.CharAdvance[i] = 1 // every char contributes 1
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

// TestJavaRandomMatchesJavaUtilRandom pins the javaRandom LCG against
// java.util.Random's documented stream: nextInt() values for setSeed(0) and
// setSeed(42). The DrawStringTooltip jitter (Java drawStringAntiMacro,
// PixFont.java:184-212 @32f3062) depends on reproducing these sequences
// exactly.
func TestJavaRandomMatchesJavaUtilRandom(t *testing.T) {
	cases := []struct {
		seed int64
		want []int
	}{
		{0, []int{-1155484576, -723955400, 1033096058, -1690734402, -1557280266}},
		{42, []int{-1170105035, 234785527, -1360544799, 205897768, 1325939940}},
	}
	var r javaRandom
	for _, tc := range cases {
		r.SetSeed(tc.seed)
		for i, want := range tc.want {
			if got := r.NextInt(); got != want {
				t.Fatalf("seed %d draw %d: NextInt() = %d, want %d", tc.seed, i, got, want)
			}
		}
	}
}

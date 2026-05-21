package bootfont

import "testing"

func TestHeight(t *testing.T) {
	if got := Height(); got != 13 {
		t.Fatalf("Height() = %d, want 13", got)
	}
}

func TestStringWidthEmpty(t *testing.T) {
	if got := StringWidth(""); got != 0 {
		t.Fatalf("StringWidth(\"\") = %d, want 0", got)
	}
}

func TestStringWidthASCII(t *testing.T) {
	// basicfont.Face7x13 advances 7 pixels per glyph.
	if got := StringWidth("hello"); got != 35 {
		t.Fatalf("StringWidth(\"hello\") = %d, want 35", got)
	}
}

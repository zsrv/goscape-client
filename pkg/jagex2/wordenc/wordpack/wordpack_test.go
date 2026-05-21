package wordpack

import (
	"strings"
	"testing"

	"goscape-client/pkg/jagex2/io"
)

// TestPackUnpackPound verifies that '£' (U+00A3, multi-byte UTF-8) round-trips.
// Regression test: an earlier Go port indexed the input by bytes, splitting '£'
// into 0xC2 0xA3 and encoding it as two TABLE-miss entries (spaces).
func TestPackUnpackPound(t *testing.T) {
	in := "hello £5"
	buf := io.NewPacket(make([]byte, 64))
	Pack(buf, in)
	packedLen := buf.Pos
	buf.Pos = 0
	got := Unpack(buf, packedLen)
	// Unpack title-cases the first letter (Java behavior preserved).
	want := "Hello £5"
	if got != want {
		t.Fatalf("round-trip mismatch:\n got=%q\nwant=%q", got, want)
	}
}

func TestPackUnpackASCII(t *testing.T) {
	in := "hello world"
	buf := io.NewPacket(make([]byte, 64))
	Pack(buf, in)
	packedLen := buf.Pos
	buf.Pos = 0
	got := Unpack(buf, packedLen)
	want := "Hello world"
	if got != want {
		t.Fatalf("round-trip mismatch:\n got=%q\nwant=%q", got, want)
	}
}

// TestPackTruncatesAt80 verifies that Pack mirrors Java's
// `arg2.length() > 80 ? arg2.substring(0, 80)` truncation. Java's `length()`
// counts UTF-16 code units; the Go port uses `[]rune` (Unicode codepoints),
// which agrees with UTF-16 for the BMP-only inputs this codec supports.
func TestPackTruncatesAt80(t *testing.T) {
	// Java semantics: any 81+ "char" input is truncated to the first 80 chars
	// before encoding. Use 'a' (a TABLE entry, packed as a single nibble) so the
	// packed length is exactly ceil(80/2) = 40 bytes regardless of where the
	// truncation cuts.
	in80 := strings.Repeat("a", 80)
	in100 := strings.Repeat("a", 100)

	buf80 := io.NewPacket(make([]byte, 128))
	Pack(buf80, in80)

	buf100 := io.NewPacket(make([]byte, 128))
	Pack(buf100, in100)

	if buf80.Pos != buf100.Pos {
		t.Fatalf("80-char and 100-char inputs should pack to the same length after truncation: 80=%d 100=%d", buf80.Pos, buf100.Pos)
	}
	for i := range buf80.Pos {
		if buf80.Data[i] != buf100.Data[i] {
			t.Fatalf("packed bytes differ at index %d: 80=%#x 100=%#x", i, buf80.Data[i], buf100.Data[i])
		}
	}

	// Round-trip the truncated form to confirm 80 chars survive.
	buf100.Pos = 0
	got := Unpack(buf100, 40)
	if len(got) != 80 {
		t.Fatalf("expected 80-char unpack result, got %d chars: %q", len(got), got)
	}
}

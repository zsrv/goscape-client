package wordpack

import (
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

package bzip2

import (
	"encoding/hex"
	"testing"
)

// TestRead_HelloWorld decompresses a tiny bzip2 stream and checks the bytes.
// The payload below was produced by `printf "hello bzip2" | bzip2 -c -1`.
// The Jagex variant of bzip2.Read expects the input WITHOUT the four-byte
// "BZh1" magic, so we strip it before invoking Read (matching jagfile.go,
// which always passes nextIn past the size header).
func TestRead_HelloWorld(t *testing.T) {
	full, err := hex.DecodeString(
		"425a6831314159265359555a44f70000021980400010001264c01020" +
			"00220069ea100305d3b62183c5dc914e14241556913dc0",
	)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	want := []byte("hello bzip2")

	// Strip the 4-byte BZh1 magic; Jagex bzip2 streams omit it on the wire.
	stream := full[4:]
	out := make([]byte, len(want))
	n := Read(out, len(want), stream, len(stream), 0)

	if n != len(want) {
		t.Fatalf("Read returned %d, want %d", n, len(want))
	}
	if string(out) != string(want) {
		t.Fatalf("Read output = %q, want %q", out, want)
	}
}

// TestRead_RepeatedRuns covers a payload with run-length encoding (long runs
// exercise the BZ_GET_FAST_C state machine in finish()), guarding against
// regressions in the byte<->int handling that was the subject of the
// verification sweep.
func TestRead_RepeatedRuns(t *testing.T) {
	// `printf "aaaaaaaaaabbbbbbbbbbcccccccccc" | bzip2 -c -1`
	full, err := hex.DecodeString(
		"425a683131415926535921f825550000034100010038002000223c9a8334d2d2" +
			"5861f17724538509021f825550",
	)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	want := []byte("aaaaaaaaaabbbbbbbbbbcccccccccc")
	stream := full[4:]
	out := make([]byte, len(want))
	n := Read(out, len(want), stream, len(stream), 0)

	if n != len(want) {
		t.Fatalf("Read returned %d, want %d", n, len(want))
	}
	if string(out) != string(want) {
		t.Fatalf("Read output = %q, want %q", out, want)
	}
}

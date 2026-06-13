package client

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestOpenFileStreamCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := openFileStreamCache(dir)
	if c == nil {
		t.Fatal("expected non-nil cache for a writable dir")
	}
	// 900 bytes spans two 512-byte blocks; archive index 2 is an on-demand archive.
	data := bytes.Repeat([]byte{0x5A}, 900)
	c.Write(2, 8, data)
	if got := c.Read(2, 8); !bytes.Equal(got, data) {
		t.Fatalf("round-trip via cache failed (got %d bytes)", len(got))
	}
}

func TestOpenFileStreamCacheUnusableDirIsNilInterface(t *testing.T) {
	// A path whose parent does not exist cannot be opened; the result must be a
	// true-nil interface so OnDemand's `cache == nil` gate behaves like Java's
	// fileStreams[0] == null. Guards against the typed-nil interface trap.
	c := openFileStreamCache(filepath.Join(t.TempDir(), "missing-parent", "deeper"))
	if c != nil {
		t.Fatalf("expected nil interface for an unusable dir, got %#v", c)
	}
}

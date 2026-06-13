package io

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// openRW opens (creating if needed) a read/write file in dir, registering it for
// cleanup. It mirrors Java's `new RandomAccessFile(path, "rw")`.
func openRW(t *testing.T, dir, name string) *os.File {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// newTestStream builds a FileStream over a fresh dat+idx pair with the given
// archive tag and the standard 500000 max file size.
func newTestStream(t *testing.T, archive int) *FileStream {
	t.Helper()
	dir := t.TempDir()
	return NewFileStream(openRW(t, dir, "cache.dat"), openRW(t, dir, "cache.idx"), archive, 500000)
}

func TestFileStreamRoundTripSmall(t *testing.T) {
	fs := newTestStream(t, 1)
	data := []byte("hello world, on-demand cache entry")
	if !fs.WriteToFile(len(data), 5, data) {
		t.Fatal("WriteToFile returned false")
	}
	got := fs.ReadFromFile(5)
	if !bytes.Equal(got, data) {
		t.Fatalf("round-trip mismatch:\n got %q\nwant %q", got, data)
	}
}

func TestFileStreamRoundTripMultiBlock(t *testing.T) {
	fs := newTestStream(t, 1)
	// 1300 bytes spans three 512-byte data blocks (512 + 512 + 276).
	data := make([]byte, 1300)
	for i := range data {
		data[i] = byte(i*7 + 3)
	}
	if !fs.WriteToFile(len(data), 2, data) {
		t.Fatal("WriteToFile returned false")
	}
	got := fs.ReadFromFile(2)
	if !bytes.Equal(got, data) {
		t.Fatalf("multi-block round-trip mismatch: got %d bytes, want %d", len(got), len(data))
	}
}

func TestFileStreamBlockBoundaryExact(t *testing.T) {
	fs := newTestStream(t, 1)
	for _, size := range []int{512, 1024} {
		data := bytes.Repeat([]byte{byte(size)}, size)
		if !fs.WriteToFile(len(data), 1, data) {
			t.Fatalf("size %d: WriteToFile returned false", size)
		}
		if got := fs.ReadFromFile(1); !bytes.Equal(got, data) {
			t.Fatalf("size %d: round-trip mismatch (got %d bytes)", size, len(got))
		}
	}
}

func TestFileStreamReadMissingReturnsNil(t *testing.T) {
	fs := newTestStream(t, 1)
	if got := fs.ReadFromFile(9); got != nil {
		t.Fatalf("expected nil for never-written file, got %d bytes", len(got))
	}
}

func TestFileStreamOverwriteSameSize(t *testing.T) {
	fs := newTestStream(t, 1)
	a := bytes.Repeat([]byte{0xAA}, 600)
	b := bytes.Repeat([]byte{0xBB}, 600)
	fs.WriteToFile(len(a), 3, a)
	if !fs.WriteToFile(len(b), 3, b) {
		t.Fatal("overwrite WriteToFile returned false")
	}
	if got := fs.ReadFromFile(3); !bytes.Equal(got, b) {
		t.Fatal("overwrite (same size) did not take effect")
	}
}

func TestFileStreamOverwriteGrowAndShrink(t *testing.T) {
	fs := newTestStream(t, 1)
	small := bytes.Repeat([]byte{0x01}, 600)
	large := bytes.Repeat([]byte{0x02}, 1300)

	// grow: 1 block -> 3 blocks (chain must extend by appending)
	fs.WriteToFile(len(small), 4, small)
	if !fs.WriteToFile(len(large), 4, large) {
		t.Fatal("grow overwrite returned false")
	}
	if got := fs.ReadFromFile(4); !bytes.Equal(got, large) {
		t.Fatalf("grow overwrite mismatch (got %d bytes)", len(got))
	}

	// shrink: 3 blocks -> 1 block (chain must terminate early)
	if !fs.WriteToFile(len(small), 4, small) {
		t.Fatal("shrink overwrite returned false")
	}
	if got := fs.ReadFromFile(4); !bytes.Equal(got, small) {
		t.Fatalf("shrink overwrite mismatch (got %d bytes)", len(got))
	}
}

func TestFileStreamInterleavedFiles(t *testing.T) {
	fs := newTestStream(t, 1)
	f1 := bytes.Repeat([]byte{0x11}, 700)
	f2 := bytes.Repeat([]byte{0x22}, 700)
	fs.WriteToFile(len(f1), 10, f1)
	fs.WriteToFile(len(f2), 11, f2)
	if got := fs.ReadFromFile(10); !bytes.Equal(got, f1) {
		t.Fatal("file 10 corrupted by interleaving")
	}
	if got := fs.ReadFromFile(11); !bytes.Equal(got, f2) {
		t.Fatal("file 11 corrupted by interleaving")
	}
}

func TestFileStreamSharedDatDistinctArchives(t *testing.T) {
	// Two archives share one dat (different idx), exactly as Java's five
	// fileStreams share signlink.cache_dat. The per-block archive tag must keep
	// same-numbered files in different archives from colliding.
	dir := t.TempDir()
	dat := openRW(t, dir, "main_file_cache.dat")
	fs1 := NewFileStream(dat, openRW(t, dir, "main_file_cache.idx1"), 1, 500000)
	fs2 := NewFileStream(dat, openRW(t, dir, "main_file_cache.idx2"), 2, 500000)

	d1 := bytes.Repeat([]byte{0x33}, 800)
	d2 := bytes.Repeat([]byte{0x44}, 800)
	fs1.WriteToFile(len(d1), 7, d1)
	fs2.WriteToFile(len(d2), 7, d2)

	if got := fs1.ReadFromFile(7); !bytes.Equal(got, d1) {
		t.Fatal("archive 1 file 7 mismatch")
	}
	if got := fs2.ReadFromFile(7); !bytes.Equal(got, d2) {
		t.Fatal("archive 2 file 7 mismatch")
	}
}

func TestFileStreamCacheReadWrite(t *testing.T) {
	dir := t.TempDir()
	dat := openRW(t, dir, "main_file_cache.dat")
	var idx [5]*os.File
	for i := range 5 {
		idx[i] = openRW(t, dir, "main_file_cache.idx"+strconv.Itoa(i))
	}
	c := NewFileStreamCache(dat, idx)

	data := bytes.Repeat([]byte{0x42}, 1000)
	c.Write(2, 15, data)

	if got := c.Read(2, 15); !bytes.Equal(got, data) {
		t.Fatalf("cache round-trip mismatch (got %d bytes)", len(got))
	}
	// A different archive index must not see archive 2's data.
	if got := c.Read(3, 15); got != nil {
		t.Fatalf("archive 3 should miss, got %d bytes", len(got))
	}
}

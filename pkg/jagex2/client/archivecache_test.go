//go:build !js

package client

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestArchiveCacheStoresAtIndex0 verifies that the native startup-archive cache
// helper persists archives into FileStore index 0 by file-id (the Java
// fileStreams[0].writeToFile/readFromFile path), so main_file_cache.idx0 is
// populated rather than left at zero bytes.
func TestArchiveCacheStoresAtIndex0(t *testing.T) {
	dir := t.TempDir()
	c := &Client{Cache: openFileStreamCache(dir)}

	data := bytes.Repeat([]byte{0xAB}, 300)
	c.archiveCacheSave(2, data) // config = file id 2

	if got := c.archiveCacheLoad(2); !bytes.Equal(got, data) {
		t.Fatalf("archive round-trip failed: got %d bytes, want %d", len(got), len(data))
	}

	// The whole point of the fix: index 0 is no longer zero bytes on disk.
	fi, err := os.Stat(filepath.Join(dir, "main_file_cache.idx0"))
	if err != nil {
		t.Fatalf("stat idx0: %v", err)
	}
	if fi.Size() == 0 {
		t.Fatal("main_file_cache.idx0 is empty; archive was not stored at index 0")
	}
}

// TestArchiveCacheNilCacheNoPanic verifies the Java-faithful behavior when the
// FileStore could not be opened (signlink.cache_dat == null): caching is skipped
// silently, with no panic and a nil load result.
func TestArchiveCacheNilCacheNoPanic(t *testing.T) {
	c := &Client{Cache: nil}
	c.archiveCacheSave(2, []byte{1, 2, 3}) // must not panic
	if got := c.archiveCacheLoad(2); got != nil {
		t.Fatalf("expected nil from a nil cache, got %d bytes", len(got))
	}
}

//go:build !js

package signlink

import (
	"bytes"
	"testing"
)

// diskStoreAt builds a diskStore bound to an explicit dir/id, bypassing
// FindCacheDir/GetUID so the test controls the location.
func diskStoreAt(dir string, id int) *diskStore {
	d := &diskStore{dir: dir, id: id}
	d.once.Do(func() {}) // mark initialized so ensure() won't probe the FS
	return d
}

func TestDiskStoreRoundTrip(t *testing.T) {
	d := diskStoreAt(t.TempDir(), 42)

	if got := d.load("missing"); got != nil {
		t.Fatalf("miss should be nil, got %v", got)
	}

	d.save("config", []byte{1, 2, 3})
	if got := d.load("config"); !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("load: got %v, want [1 2 3]", got)
	}

	if d.uid() != 42 {
		t.Fatalf("uid: got %d, want 42", d.uid())
	}
}

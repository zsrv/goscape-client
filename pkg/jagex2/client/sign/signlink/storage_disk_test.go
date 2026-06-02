//go:build !js

package signlink

import (
	"bytes"
	"os"
	"path/filepath"
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

// TestGetUIDShortFileDoesNotPanic reproduces the parity bug where a short or
// corrupt uid.dat (fewer than 4 bytes) that cannot be rewritten crashed the
// client: binary.BigEndian.Uint32 panicked on the under-length slice. Java's
// getuid reads via DataInputStream.readInt(), whose EOFException is caught and
// returns 0 (sign/signlink.java:213-220), so GetUID must return 0 here too.
func TestGetUIDShortFileDoesNotPanic(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses file permissions; the rewrite would succeed and mask the crash path")
	}
	dir := t.TempDir()
	// Write a 2-byte uid.dat read-only so GetUID's rewrite (os.WriteFile O_TRUNC)
	// fails for the owner and it falls through to ReadFile returning 2 bytes —
	// exactly the crash condition.
	if err := os.WriteFile(filepath.Join(dir, "uid.dat"), []byte{0x01, 0x02}, 0o400); err != nil {
		t.Fatal(err)
	}
	if got := GetUID(dir); got != 0 {
		t.Fatalf("GetUID on short uid.dat: got %d, want 0", got)
	}
}

package pix8

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// nameHash reproduces JagFile.read's entry-name hash (upper-cased, *61).
func nameHash(name string) int {
	hash := int32(0)
	for _, ch := range name {
		if ch >= 'a' && ch <= 'z' {
			ch -= 32
		}
		hash = hash*61 + ch - 32
	}
	return int(hash)
}

// buildSpriteArchive returns a JagFile holding "<name>.dat" and "index.dat"
// verbatim (Unpacked=true skips decompression, as in iftype's test helper).
func buildSpriteArchive(t *testing.T, name string, dat, idx []byte) *io.Jagfile {
	t.Helper()
	buf := append(append([]byte{}, dat...), idx...)
	return &io.Jagfile{
		Buffer:           buf,
		Unpacked:         true,
		FileCount:        2,
		FileHash:         []int{nameHash(name + ".dat"), nameHash("index.dat")},
		FileUnpackedSize: []int{len(dat), len(idx)},
		FilePackedSize:   []int{len(dat), len(idx)},
		FileOffset:       []int{0, len(dat)},
	}
}

// twoSpriteArchive builds a media archive holding exactly two 2x2 sprites,
// followed by seven trailing bytes that decode as a bogus third entry with
// Wi=Hi=8192 and pixelOrder=99.
//
// This is the shape that breaks Client.Load's sprite loops: they run to a
// fixed bound (100 for mapscene) and rely on the read loop throwing once the
// archive is exhausted. A pixelOrder outside {0,1} skips both read branches,
// so nothing throws and a garbage-sized array is allocated and retained.
func twoSpriteArchive(t *testing.T) *io.Jagfile {
	t.Helper()
	// dat: 2-byte index offset (0), then 4 pixels per sprite.
	dat := []byte{0, 0, 1, 2, 3, 4, 5, 6, 7, 8}
	idx := []byte{
		0, 2, // OWi = 2
		0, 2, // OHi = 2
		1,                   // palCount = 1 (no palette entries follow)
		0, 0, 0, 2, 0, 2, 0, // sprite 0: XOf,YOf,Wi=2,Hi=2,order=0
		0, 0, 0, 2, 0, 2, 0, // sprite 1: XOf,YOf,Wi=2,Hi=2,order=0
		0, 0, 0x20, 0, 0x20, 0, 99, // trailing garbage: Wi=8192,Hi=8192,order=99
	}
	return buildSpriteArchive(t, "test", dat, idx)
}

// TestNewPix8ReadsRealSprites is the guard against over-correcting: the fix for
// the overshoot must not reject legitimate sprites, including the final one,
// whose pixel count exactly consumes the remaining .dat bytes.
func TestNewPix8ReadsRealSprites(t *testing.T) {
	jag := twoSpriteArchive(t)
	for _, tc := range []struct {
		sprite int
		want   []byte
	}{
		{0, []byte{1, 2, 3, 4}},
		{1, []byte{5, 6, 7, 8}},
	} {
		p := NewPix8(jag, "test", tc.sprite)
		if p.Wi != 2 || p.Hi != 2 {
			t.Errorf("sprite %d: got %dx%d, want 2x2", tc.sprite, p.Wi, p.Hi)
		}
		if string(p.Pixels) != string(tc.want) {
			t.Errorf("sprite %d: pixels = %v, want %v", tc.sprite, p.Pixels, tc.want)
		}
	}
}

// TestNewPix8PanicsPastEndOfArchive pins the fix for the ~2.6GB startup
// reservation (docs/superpowers/specs/2026-08-31-python-scripted-bot-design.md
// section 9).
//
// Requesting a sprite past the archive's real count yields garbage dimensions.
// Java tolerated this because the subsequent read loop threw promptly, and
// Client.Load's catch(Exception) ended the loop. When pixelOrder is garbage
// both read branches are skipped, nothing throws, and NewPix8 returns a Pix8
// owning a 64MB slice -- so the caller's loop keeps going and keeps allocating.
//
// NewPix8 must panic instead, so RecoverPanic ends the loop exactly as Java's
// catch did.
func TestNewPix8PanicsPastEndOfArchive(t *testing.T) {
	jag := twoSpriteArchive(t)

	defer func() {
		if recover() == nil {
			t.Fatal("NewPix8 past end of archive returned normally; " +
				"want panic so Client.Load's RecoverPanic ends the sprite loop")
		}
	}()

	p := NewPix8(jag, "test", 2)
	t.Fatalf("unreachable: allocated %d bytes for a %dx%d phantom sprite",
		len(p.Pixels), p.Wi, p.Hi)
}

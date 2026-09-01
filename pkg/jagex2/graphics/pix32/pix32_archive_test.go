package pix32

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

// twoSpriteArchive builds a media archive holding exactly two 2x2 sprites,
// followed by seven trailing bytes that decode as a bogus third entry with
// Wi=Hi=8192 and pixelOrder=99 — the shape that breaks Client.Load's fixed-
// bound sprite loops. See the sibling test in package pix8; Pix32 is worse
// because its Pixels are []int (8 bytes each), not []byte.
func twoSpriteArchive(t *testing.T) *io.Jagfile {
	t.Helper()
	dat := []byte{0, 0, 1, 1, 1, 1, 1, 1, 1, 1}
	idx := []byte{
		0, 2, // OWi = 2
		0, 2, // OHi = 2
		2, 0, 0, 1, // palCount = 2, one 3-byte palette entry (colour 1)
		0, 0, 0, 2, 0, 2, 0, // sprite 0: Wi=2, Hi=2, order=0
		0, 0, 0, 2, 0, 2, 0, // sprite 1: Wi=2, Hi=2, order=0
		0, 0, 0x20, 0, 0x20, 0, 99, // trailing garbage: Wi=8192, Hi=8192, order=99
	}
	buf := append(append([]byte{}, dat...), idx...)
	return &io.Jagfile{
		Buffer:           buf,
		Unpacked:         true,
		FileCount:        2,
		FileHash:         []int{nameHash("test.dat"), nameHash("index.dat")},
		FileUnpackedSize: []int{len(dat), len(idx)},
		FilePackedSize:   []int{len(dat), len(idx)},
		FileOffset:       []int{0, len(dat)},
	}
}

// TestNewPix323ReadsRealSprites guards against over-correcting: the fix must
// still accept every genuine sprite, including the last one, whose pixel count
// exactly consumes the remaining .dat bytes.
func TestNewPix323ReadsRealSprites(t *testing.T) {
	jag := twoSpriteArchive(t)
	for _, sprite := range []int{0, 1} {
		p := NewPix323(jag, "test", sprite)
		if p.Wi != 2 || p.Hi != 2 {
			t.Errorf("sprite %d: got %dx%d, want 2x2", sprite, p.Wi, p.Hi)
		}
		if len(p.Pixels) != 4 {
			t.Errorf("sprite %d: len(Pixels) = %d, want 4", sprite, len(p.Pixels))
		}
	}
}

// TestNewPix323PanicsPastEndOfArchive pins the same fix as the pix8 sibling.
// A pixelOrder outside {0,1} falls through the switch without reading, so
// nothing throws and a garbage-sized []int is retained by the caller, whose
// fixed-bound loop then advances and allocates again.
func TestNewPix323PanicsPastEndOfArchive(t *testing.T) {
	jag := twoSpriteArchive(t)

	defer func() {
		if recover() == nil {
			t.Fatal("NewPix323 past end of archive returned normally; " +
				"want panic so Client.Load's RecoverPanic ends the sprite loop")
		}
	}()

	p := NewPix323(jag, "test", 2)
	t.Fatalf("unreachable: allocated %d ints (%d bytes) for a %dx%d phantom sprite",
		len(p.Pixels), len(p.Pixels)*8, p.Wi, p.Hi)
}

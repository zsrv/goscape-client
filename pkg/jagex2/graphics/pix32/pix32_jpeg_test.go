package pix32

import (
	"os"
	"testing"
)

// TestNewPix322_DecodesJPEG guards the `import _ "image/jpeg"` decoder
// registration in pix32.go. Without it, image.Decode returns "image: unknown
// format" on the title-screen JPEG (title.dat), so NewPix322 hits the "Error
// converting jpg" path and the title background renders black.
//
// The fixture is read as raw bytes and this file deliberately does NOT import
// image/jpeg — otherwise the decoder would register in the test binary and mask
// a missing production import.
func TestNewPix322_DecodesJPEG(t *testing.T) {
	data, err := os.ReadFile("testdata/tiny.jpg")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	p := NewPix322(data)

	if p.Wi != 4 || p.Hi != 3 {
		t.Fatalf("decoded size = (%d,%d); want (4,3) — is the JPEG decoder registered?", p.Wi, p.Hi)
	}
	if len(p.Pixels) != 12 {
		t.Fatalf("Pixels len = %d; want 12", len(p.Pixels))
	}
	// The fixture is a solid non-black fill, so a decoded image has non-zero pixels.
	for _, px := range p.Pixels {
		if px != 0 {
			return
		}
	}
	t.Errorf("all decoded pixels are zero; JPEG was not actually decoded")
}

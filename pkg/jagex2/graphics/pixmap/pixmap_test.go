package pixmap

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
)

type fakeTex struct{ id int }

type fakeBackend struct {
	uploads int
	blits   int
	nextID  int
}

func (f *fakeBackend) PollEvents() []platform.Event { return nil }
func (f *fakeBackend) ShouldClose() bool            { return false }
func (f *fakeBackend) Size() (int, int)             { return 532, 789 }
func (f *fakeBackend) NewTexture(w, h int) platform.Texture {
	f.nextID++
	return &fakeTex{id: f.nextID}
}
func (f *fakeBackend) UploadTexture(t platform.Texture, rgba []byte) { f.uploads++ }
func (f *fakeBackend) BeginFrame()                                   {}
func (f *fakeBackend) Blit(t platform.Texture, x, y int)             { f.blits++ }
func (f *fakeBackend) EndFrame()                                     {}
func (f *fakeBackend) Destroy()                                      {}

func TestPixMapUploadsOnlyOnChange(t *testing.T) {
	fb := &fakeBackend{}
	platform.Active = fb
	p := NewPixMap(4, 4)

	// NewPixMap allocates the texture but does not upload; the first Draw is
	// what performs the cold-start upload.
	p.Draw(0, 0)
	if fb.uploads != 1 {
		t.Fatalf("first Draw should upload once, got %d", fb.uploads)
	}
	// Unchanged content: the second identical Draw must not re-upload.
	p.Draw(0, 0)
	if fb.uploads != 1 {
		t.Fatalf("unchanged Draw re-uploaded: got %d, want 1", fb.uploads)
	}
	uploadsAfterStable := fb.uploads

	blitsBefore := fb.blits
	p.Data[0] = 0x123456
	p.Draw(0, 0) // changed -> exactly one upload
	if fb.uploads != uploadsAfterStable+1 {
		t.Fatalf("changed Draw should upload once: %d -> %d", uploadsAfterStable, fb.uploads)
	}
	if fb.blits != blitsBefore+1 {
		t.Fatalf("every Draw must blit once: %d -> %d", blitsBefore, fb.blits)
	}
}

func TestHashPixelsDetectsChange(t *testing.T) {
	a := []int{1, 2, 3}
	if hashPixels(a) != hashPixels([]int{1, 2, 3}) {
		t.Fatal("equal data hashed differently")
	}
	if hashPixels(a) == hashPixels([]int{1, 2, 4}) {
		t.Fatal("changed data hashed the same")
	}
}

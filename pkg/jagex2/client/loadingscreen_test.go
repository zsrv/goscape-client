package client

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pixmap"
	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
)

// countingBackend records Blit calls. NewTexture returns distinct handles so
// pixmap.NewPixMap works without a GPU.
type countingBackend struct {
	blits  int
	nextID int
}

func (b *countingBackend) PollEvents() []platform.Event           { return nil }
func (b *countingBackend) ShouldClose() bool                      { return false }
func (b *countingBackend) Size() (int, int)                       { return 789, 532 }
func (b *countingBackend) NewTexture(w, h int) platform.Texture   { b.nextID++; return b.nextID }
func (b *countingBackend) UploadTexture(platform.Texture, []byte) {}
func (b *countingBackend) BeginFrame()                            {}
func (b *countingBackend) Blit(platform.Texture, int, int)        { b.blits++ }
func (b *countingBackend) EndFrame()                              {}
func (b *countingBackend) Destroy()                               {}

// TestPresentLoadingMessage_CompositesSurroundNotJustViewport guards the fix for
// the one-frame black flicker during area loading. The GL backend clears the
// framebuffer every BeginFrame, so the old present(AreaViewport.Draw) blacked
// out the surrounding UI; presentLoadingMessage must re-blit the whole retained
// screen. With a viewport plus two surround areas allocated, a correct present
// fires 3 blits; the buggy viewport-only present would fire 1.
func TestPresentLoadingMessage_CompositesSurroundNotJustViewport(t *testing.T) {
	t.Cleanup(pix2d.Reset)
	be := &countingBackend{}
	prev := platform.Active
	platform.Active = be
	t.Cleanup(func() { platform.Active = prev })

	c := &Client{}
	c.AreaViewport = pixmap.NewPixMap(512, 334)
	c.AreaSidebar = pixmap.NewPixMap(190, 261)
	c.AreaChatback = pixmap.NewPixMap(479, 96)

	be.blits = 0 // NewPixMap issues no Blit; reset to be explicit
	c.presentLoadingMessage()

	if be.blits != 3 {
		t.Errorf("presentLoadingMessage blitted %d areas; want 3 (viewport + sidebar + chatback) — is the surround composited?", be.blits)
	}
}

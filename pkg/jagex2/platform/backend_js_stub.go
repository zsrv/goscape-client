//go:build js

package platform

// jsStub is a compile-only placeholder so GOOS=js builds during Plan 1. Plan 2
// replaces this file with the real syscall/js + WebGL backend.
type jsStub struct{}

func newJSBackend(width, height int, title string) Backend {
	panic("platform(js): WebGL backend not yet implemented (Plan 2)")
}

func (jsStub) PollEvents() []Event               { panic("stub") }
func (jsStub) ShouldClose() bool                 { panic("stub") }
func (jsStub) Size() (int, int)                  { panic("stub") }
func (jsStub) NewTexture(w, h int) Texture       { panic("stub") }
func (jsStub) UploadTexture(t Texture, b []byte) { panic("stub") }
func (jsStub) BeginFrame()                       { panic("stub") }
func (jsStub) Blit(t Texture, x, y int)          { panic("stub") }
func (jsStub) EndFrame()                         { panic("stub") }
func (jsStub) Destroy()                          { panic("stub") }

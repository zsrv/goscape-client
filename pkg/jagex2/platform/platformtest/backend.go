// Package platformtest provides a no-op platform.Backend for unit tests that
// construct a pixmap.PixMap (which calls platform.Active.NewTexture) without a
// real GPU context. Install it at the top of such a test and defer the restore.
package platformtest

import "github.com/zsrv/goscape-client/pkg/jagex2/platform"

type fakeTexture struct{ id int }

// fakeBackend is a no-op Backend: NewTexture returns distinct non-nil handles;
// uploads/blits/frames are no-ops.
type fakeBackend struct{ nextID int }

func (f *fakeBackend) PollEvents() []platform.Event { return nil }
func (f *fakeBackend) ShouldClose() bool            { return false }
func (f *fakeBackend) Size() (int, int)             { return 789, 532 }
func (f *fakeBackend) NewTexture(w, h int) platform.Texture {
	f.nextID++
	return &fakeTexture{id: f.nextID}
}
func (f *fakeBackend) UploadTexture(platform.Texture, []byte) {}
func (f *fakeBackend) BeginFrame()                            {}
func (f *fakeBackend) Blit(platform.Texture, int, int)        {}
func (f *fakeBackend) EndFrame()                              {}
func (f *fakeBackend) Destroy()                               {}

// Install sets a no-op backend as platform.Active and returns a function that
// restores the previous value. Usage in a test:
//
//	defer platformtest.Install()()
//
// or:
//
//	restore := platformtest.Install()
//	defer restore()
func Install() (restore func()) {
	prev := platform.Active
	platform.Active = &fakeBackend{}
	return func() { platform.Active = prev }
}

package client

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
)

// testFakeTex is an opaque texture handle used in unit tests.
type testFakeTex struct{ id int }

// testFakeBackend is a no-op platform.Backend for unit tests that need
// pixmap.NewPixMap without an actual GPU context (e.g. DrawError,
// DrawProgressGameShell). It satisfies the full Backend interface.
type testFakeBackend struct{ nextID int }

func (f *testFakeBackend) PollEvents() []platform.Event { return nil }
func (f *testFakeBackend) ShouldClose() bool            { return false }
func (f *testFakeBackend) Size() (int, int)             { return 789, 532 }
func (f *testFakeBackend) NewTexture(w, h int) platform.Texture {
	f.nextID++
	return &testFakeTex{id: f.nextID}
}
func (f *testFakeBackend) UploadTexture(_ platform.Texture, _ []byte) {}
func (f *testFakeBackend) BeginFrame()                                {}
func (f *testFakeBackend) Blit(_ platform.Texture, _, _ int)          {}
func (f *testFakeBackend) EndFrame()                                  {}
func (f *testFakeBackend) Destroy()                                   {}

// setupTestBackend installs a fake platform backend for the test and restores
// the original (nil) value on cleanup. Call at the top of any test that uses
// pixmap.NewPixMap (e.g. via ensureOverlay).
func setupTestBackend(t *testing.T) {
	t.Helper()
	prev := platform.Active
	platform.Active = &testFakeBackend{}
	t.Cleanup(func() { platform.Active = prev })
}

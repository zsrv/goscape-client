// Package platform is the toolkit-neutral host-shell seam: a window/loop/
// present backend and neutral input events, with build-tagged implementations
// (GLFW+go-gl native; syscall/js+WebGL browser). It replaces Gio. See
// docs/superpowers/specs/2026-05-25-hand-rolled-host-shell-design.md.
package platform

// Texture is an opaque, backend-owned GPU texture handle.
type Texture any

// Backend is the host shell. Exactly one is Active for the process lifetime.
// Drawing is immediate-mode (Blit one textured quad per PixMap, mirroring Java
// Graphics.drawImage / TS putImageData); there is no retained op list.
type Backend interface {
	// PollEvents drains input accumulated since the last call.
	PollEvents() []Event
	// ShouldClose reports whether the user asked to close the window.
	ShouldClose() bool
	// Size returns the drawable size in pixels.
	Size() (w, h int)

	// NewTexture allocates a w*h RGBA texture (nearest-neighbor).
	NewTexture(w, h int) Texture
	// UploadTexture re-uploads rgba into t in place (glTexSubImage2D). The
	// caller gates this on a pixel-change check; it is never called per frame
	// for unchanged content.
	UploadTexture(t Texture, rgba []byte)

	// BeginFrame binds the default framebuffer and sets the viewport.
	BeginFrame()
	// Blit draws t as a quad with its top-left at pixel (x, y).
	Blit(t Texture, x, y int)
	// EndFrame presents (SwapBuffers native; implicit flush on yield in wasm).
	EndFrame()

	// Destroy releases the window and GL resources.
	Destroy()
}

// Active is the process-wide current backend. Set once at startup (see Main)
// before the game loop or any PixMap is created.
var Active Backend

// Mod is a neutral modifier-key bitset.
type Mod uint8

const (
	ModShift Mod = 1 << iota
	ModCtrl
	ModAlt
	ModSuper
	ModCommand
)

// Has reports whether all bits of m2 are set in m.
func (m Mod) Has(m2 Mod) bool { return m&m2 == m2 }

// Key is a neutral physical-key identifier. KeyRune means "a printable letter
// or digit"; the actual character is carried in KeyPress.Rune. Backends map
// their native key ids to these; the AWT-code and keyChar mapping
// (awtFor/charFor in package client) is shared, not per-backend.
type Key int

const (
	KeyNone Key = iota
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyReturn
	KeyEnter
	KeyEscape
	KeyHome
	KeyEnd
	KeyBackspace
	KeyDelete
	KeyPageUp
	KeyPageDown
	KeyTab
	KeySpace
	KeyCtrl
	KeyShift
	KeyAlt
	KeySuper
	KeyCommand
	KeyBack
	KeyRune // a letter/digit; see KeyPress.Rune
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
)

// Event is a neutral input event produced by a Backend and consumed by the
// game's input translation (package client). No Gio types appear here.
type Event interface{ isEvent() }

// MouseMove reports the cursor position (also used for drags).
type MouseMove struct{ X, Y int }

// MouseButton reports a press/release. Button: 1=left, 2=right.
type MouseButton struct {
	X, Y, Button int
	Pressed      bool
}

// MouseCross reports the cursor entering (true) or leaving (false) the window.
type MouseCross struct{ Entered bool }

// KeyPress reports a non-text key press/release. For KeyRune, Rune holds the
// physical letter/digit ('A'..'Z', '0'..'9') and Mods the modifier state, so
// charFor can resolve the typed character; for named keys Rune is 0.
type KeyPress struct {
	Key  Key
	Rune rune
	Mods Mod
	Down bool
}

// CharInput reports one layout/shift/IME-resolved typed character (the text
// channel, equivalent to Gio key.EditEvent / Java getKeyChar).
type CharInput struct{ Rune rune }

// FocusChange reports the window gaining (true) or losing (false) focus.
type FocusChange struct{ Gained bool }

func (MouseMove) isEvent()   {}
func (MouseButton) isEvent() {}
func (MouseCross) isEvent()  {}
func (KeyPress) isEvent()    {}
func (CharInput) isEvent()   {}
func (FocusChange) isEvent() {}

package client

// Decision (PORTING.md §4.1): ViewBox is NOT being literally ported.
//
// Java ViewBox extends java.awt.Frame and exists only to:
//   1. Create the top-level OS window with title "Jagex", non-resizable,
//      sized to (screenWidth + 8, screenHeight + 28) to account for AWT's
//      window-chrome insets.
//   2. Override getGraphics() to translate by (4, 24), so callers can draw
//      in client-area coordinates without thinking about the chrome inset.
//   3. Forward update(Graphics) / paint(Graphics) to GameShell.
//
// Every one of those responsibilities is already handled by Gio's app.Window
// in gameshell.go (InitApplication creates the window with the correct Title
// and fixed Size/MinSize/MaxSize, and Gio's coordinate system is already
// content-area-relative, so the (4, 24) AWT-inset translation is a no-op).
// A literal port would just be a thin wrapper around app.Window contributing
// nothing the gameshell.go path doesn't already provide.
//
// We therefore keep ViewBox as a stub solely to preserve the Client.Frame
// field as a "has a window been initialised yet?" sentinel. The only
// remaining caller is client.go:2235, which uses `c.Frame != nil` to gate
// the "::clientdrop" debug command — the same nil/non-nil check works
// against any zero-cost sentinel value.
//
// TODO: Gio replacement. The cleaner long-term shape is to delete ViewBox
// entirely and replace `c.Frame *ViewBox` with `c.Window *app.Window`
// (assigned in InitApplication / gameshell.go where we currently create the
// window inline). Then `c.Frame != nil` becomes `c.Window != nil`, and we
// can drop this file. Skipped here to keep the change small and avoid
// touching the client struct layout.
//
// Java source: jagex2/client/ViewBox.java
// Go callers:
//   - pkg/jagex2/client/client.go (Client.Frame field, line ~123)
//   - pkg/jagex2/client/client.go (c.Frame != nil check, line ~2235)

// ViewBox is an intentional stub. See file-header comment for the decision
// rationale; do not flesh this out as a literal AWT port.
type ViewBox struct {
	Shell *GameShell
}

// NewViewBox returns a sentinel ViewBox. The Java constructor opens the OS
// window via java.awt.Frame; the Go side already does that work in
// gameshell.go (Client.InitApplication creates the app.Window directly).
//
// Java signature: ViewBox(int screenHeight, GameShell shell, int screenWidth).
func NewViewBox(arg0 int, arg2 *GameShell, arg3 int) *ViewBox {
	var v ViewBox
	v.Shell = arg2
	return &v
}

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
// We therefore keep ViewBox only as the Go type for the Client.Frame field,
// which is the structural 1:1 mapping of Java's GameShell.frame (a ViewBox
// reference). NewViewBox is never called, so c.Frame is always nil — and that
// is the correct, consistent choice for this port: Go always behaves as Java's
// "frame == null" (applet/embedded) case. GetHost already returns the
// configured host unconditionally instead of Java's standalone "runescape.com"
// branch, so the field's only live consumer is the "::clientdrop" debug gate,
// where `c.Frame != nil` is always false. That leaves the command gated purely
// on a 192.168.1.x LAN host — a deliberate, harmless deviation from Java's
// standalone path (where frame != null would also allow it).
//
// Deferred cleanup (intentionally NOT done — PORTING.md §2 rule 4, "don't
// refactor opportunistically"): the tidier long-term shape is to delete this
// file, drop the Client.Frame *ViewBox field, and reduce the clientdrop gate
// to the host check alone (behavior-preserving, since c.Frame is always nil).
// That edits the Client struct layout and erases the Java `super.frame`
// mapping, so it is left for a dedicated pass rather than folded into
// unrelated work.
//
// Java source: jagex2/client/ViewBox.java
// Go callers:
//   - pkg/jagex2/client/client.go (Client.Frame field, ~line 134)
//   - pkg/jagex2/client/client.go (c.Frame != nil check, ~line 2266)

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

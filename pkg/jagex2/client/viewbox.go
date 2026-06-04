package client

// Decision (PORTING.md §4.1): ViewBox is NOT being literally ported.
//
// Java 245.2 ViewBox (ViewBox.java @176a85f) extends javax.swing.JFrame
// (244: java.awt.Frame) and exists only to:
//   1. Create the top-level OS window, non-resizable, titled
//      "RS2 user client - release #" + signlink.clientversion
//      (ViewBox.java:17; 244 titled it "Jagex").
//   2. Add the GameShell as the BorderLayout CENTER child and pack()
//      (ViewBox.java:20-24), so the content area is exactly the shell's
//      preferred size. 245.2 deleted 244's AWT insets machinery upstream:
//      the (+8, +28) chrome-compensating resize, the (4, 24) getGraphics()
//      translation, and the frame.insets subtraction in GameShell's
//      mousePressed/mouseDragged/mouseMoved (GameShell.java:311/381/394
//      @176a85f) are all gone — events now arrive canvas-relative in Java
//      too.
//   3. Forward update(Graphics) / paint(Graphics) to GameShell.
//
// Every one of those responsibilities is already handled by the `platform`
// windowing seam (native: GLFW + go-gl; browser: syscall/js + WebGL): the OS
// window is created with the correct title and fixed size during boot
// (cmd/client/main.go passes the 245.2 title to platform.Main), and both
// backends' coordinate systems were always content-area-relative (GLFW
// cursor callbacks report content-area coords; the browser backend uses
// offsetX/offsetY) — so 245.2's AWT→Swing rework converges the Java shape
// toward what the Go port already does. The same goes for the rest of the
// 245.2 windowing delta: setPreferredSize in initApplication
// (GameShell.java:106) is subsumed by glfw.CreateWindow sizing the content
// area directly, and the initApplet(height, width)→(width, height)
// signature/caller flip (GameShell.java:118, Client.java:1326) touches only
// the applet entry path, which is not ported (the Go client always launches
// standalone). A literal port would just be a thin wrapper contributing
// nothing the platform seam doesn't already provide.
//
// IMPORTANT — which Java case the Go port emulates: the Go client always
// launches STANDALONE (the 4-arg main path). In Java standalone,
// initApplication does `frame = new ViewBox(...)` (GameShell.java:110
// @176a85f), so `super.frame != null` is TRUE; only the applet path
// (initApplet) leaves frame null. NewViewBox is never called here, so c.Frame
// is always nil — which means the Go port must reproduce Java's STANDALONE
// (frame != null) behavior EXPLICITLY at each site rather than let a
// `c.Frame != nil` test silently select Java's applet (frame == null) branch:
//   - GetHost (client.go) returns the configured host — Java standalone
//     getHost(), not the applet document-base host.
//   - GetCodeBase (client.go) returns the configured cache-server base URL
//     (clientextras.OndemandBaseURL; default http://127.0.0.1:8888) — Java
//     standalone getCodeBase(), not the applet doc-base URL. (The Java port
//     offset is intentionally not ported; see cmd/client/main.go.)
//   - The "::clientdrop" debug gate (client.go) always reconnects — Java
//     standalone, where `super.frame != null` is always true. It no longer
//     consults c.Frame (deob/client.java:2838); 245.2 removed the frame
//     consult upstream too — `::clientdrop` calls tryReconnect()
//     unconditionally (Client.java:4696 @176a85f), converging with this
//     mapping.
//
// As a result c.Frame now has NO live consumer; it is retained only as the
// structural 1:1 mapping of Java's GameShell.frame.
//
// Deferred cleanup (intentionally NOT done — PORTING.md §2 rule 4, "don't
// refactor opportunistically"): the tidier long-term shape is to delete this
// file and drop the now-vestigial Client.Frame *ViewBox field. That edits the
// Client struct layout and erases the Java `super.frame` mapping, so it is left
// for a dedicated pass rather than folded into unrelated work.
//
// Java source: jagex2/client/ViewBox.java
// Go callers: none live — Client.Frame (client.go ~line 136) is vestigial.

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

package client

import (
	"os"
	"time"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/bootfont"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
)

type GameShell struct {
}

// Shutdown ports GameShell.shutdown() (GameShell.java:228 @32f3062).
// 274 drops 254's dead boolean deob arg (this port never carried it) and
// inverts the frame guard into an early `if (frame == null) return` before
// the sleep+exit (state=-2 and unload stay unconditional, as in 254):
// frame is non-null exactly when running standalone (set by
// initApplication) and null in applet mode, where the browser owns the
// process. This host-shell port always runs standalone — native and wasm
// both own the program lifecycle via platform.Main — so the guard folds to
// constant-false and the sleep+exit below remain unconditional.
func (c *Client) Shutdown() {
	c.State = -2
	c.Unload()
	time.Sleep(1 * time.Second)
	// os.Exit halts the Go program cleanly on wasm too (handled by the
	// wasm_exec.js exit callback). Reviewed for the browser and intentionally
	// unchanged: DestroyEvent only fires on tab/canvas teardown, when the page
	// is going away regardless, so the best-effort Unload above is sufficient.
	os.Exit(0)
}

func (c *Client) SetFrameRate(arg1 int) {
	c.DelTime = 1000 / arg1
}

// PollKey pops the next queued key code from the ring buffer, or returns -1
// when the queue is empty. Java: pollKey() at GameShell.java:459-466.
func (c *Client) PollKey() int {
	var2 := -1
	if c.KeyQueueWritePos != c.KeyQueueReadPos {
		var2 = c.KeyQueue[c.KeyQueueReadPos]
		c.KeyQueueReadPos = (c.KeyQueueReadPos + 1) & 0x7F
	}
	return var2
}

// MessageBoxGameShell renders the boot-time loading bar used before
// c.Title has been downloaded (i.e. when Client.MessageBox falls
// through to here because no title archive yet). Java:
// GameShell.messageBox(String, int) at GameShell.java:553-584 @32f3062
// (274 renames 254's drawProgress; body unchanged — its sHei/sWid
// fields are the trap-#4 INVERTED names where sHei holds the width
// value and sWid the height; Go keeps ScreenWidth/ScreenHeight in
// their true roles).
//
// Java painted directly to the AWT base component's Graphics; the Go
// port paints into a shared overlay PixMap (via ensureOverlay) using
// pix2d for rectangles and bootfont (basicfont.Face7x13) for the
// message text, then composites via OverlayPixMap.Draw. The Helvetica
// BOLD 13 in the Java source is substituted with bootfont's monospace
// 7x13 — pixfont's RuneScape font isn't available at this phase.
func (c *Client) MessageBoxGameShell(message string, percent int) {
	c.ensureOverlay()
	c.OverlayPixMap.Bind()

	if c.Refresh {
		pix2d.FillRect(0, 0, 0x000000, c.ScreenWidth, c.ScreenHeight)
		c.Refresh = false
	}

	pix2d.DrawRect(c.ScreenWidth/2-152, 0x8C1111, 34, c.ScreenHeight/2-18, 304)
	pix2d.FillRect(c.ScreenHeight/2-16, c.ScreenWidth/2-150, 0x8C1111, percent*3, 30)
	pix2d.FillRect(c.ScreenHeight/2-16, c.ScreenWidth/2-150+percent*3, 0x000000, 300-percent*3, 30)

	textX := (c.ScreenWidth - bootfont.StringWidth(message)) / 2
	textY := c.ScreenHeight/2 - 18 + 22
	bootfont.DrawString(c.OverlayPixMap, textX, textY, 0xFFFFFF, message)

	c.present(func() { c.OverlayPixMap.Draw(0, 0) })
}

// present issues one full frame: BeginFrame, the supplied draw, EndFrame. Used
// by out-of-band repaints (loading/connection messages) that must show
// immediately, before a blocking operation; called from the tick/prologue
// phase before BeginFrame, never from within a BeginFrame/EndFrame pair.
// Runs on the loop goroutine.
func (c *Client) present(draw func()) {
	platform.Active.BeginFrame()
	draw()
	platform.Active.EndFrame()
}

// dispatchInputEvent routes one neutral platform event to the matching handler.
func (c *Client) dispatchInputEvent(ev platform.Event) {
	switch e := ev.(type) {
	case platform.MouseMove:
		c.handleMouseMove(e)
	case platform.MouseButton:
		c.handleMouseButton(e)
	case platform.MouseCross:
		c.handleMouseCross(e)
	case platform.KeyPress:
		c.handleKey(e)
	case platform.CharInput:
		c.handleCharInput(e)
	case platform.FocusChange:
		c.handleFocus(e)
	}
}

// handleFocus mirrors Java's focusGained / focusLost on GameShell
// (GameShell.java:476-488 @32f3062). Java sets `fullredraw = true` and
// calls `refresh()` on focus gained; focus lost zeroes every keyHeld slot.
// The neutral FocusChange event is delivered by the platform backend when
// the window's focus state changes.
func (c *Client) handleFocus(e platform.FocusChange) {
	// Java: focus = true/false (GameShell.java:477,484 @32f3062) — NEW in
	// 254 (as hasFocus); read by gameLoop's EVENT_APPLET_FOCUS telemetry.
	c.HasFocus = e.Gained
	if e.Gained {
		c.Refresh = true
		// Java: this.refresh() (GameShell.java:479) dispatches to the Client
		// override, which forces the full frame rebuild (audit gameshell-07)
		c.RefreshFunc()
		return
	}
	// Java: GameShell.java:485-487 — NEW in 274 (stuck-key fix): a key held
	// across the focus loss never delivers its release event, so its
	// keyHeld slot would otherwise read as held forever.
	for i := range 128 {
		c.KeyHeld[i] = 0
	}
}

// handleCharInput is the typed-text channel (Java getKeyChar). One resolved
// character per event; control chars < 30 are dropped (matching Java).
//
// Java: keyPressed zeroes any keyChar < 30 then pushes only var3 > 4
// (GameShell.java:363-420 @32f3062), so a bare control char in [5,29] is dropped;
// only the explicit sentinel overrides (5/8/9/10/1000+) survive, and
// those arrive via KeyPress (handleKey), not CharInput text. Skip < 30
// here to match Java's drop — CharInput.Rune only carries printable
// runes (>= 32), so this never discards a needed character.
func (c *Client) handleCharInput(e platform.CharInput) {
	c.IdleCycles = 0
	var3 := int(e.Rune)
	if var3 < 30 {
		return
	}
	c.KeyQueue[c.KeyQueueWritePos] = var3
	c.KeyQueueWritePos = (c.KeyQueueWritePos + 1) & 0x7F
}

// handleMouseButton maps a platform press/release event onto the same mouse*
// fields Java's mousePressed/mouseReleased use.
// Java reference: GameShell.java:294-320 @32f3062. 274 also drops 254's
// NoSuchMethodError/isMetaDown fallback try/catch around the button test —
// a no-op here: this port always branched on the platform button id
// directly and never carried the AWT-reflection fallback.
//
// 245.2 (GameShell.java:311-374 @176a85f) drops the `x -= frame.insets.left /
// y -= frame.insets.top` adjustment from mousePressed — a no-op here: the
// platform backends were always content-area-relative (GLFW cursor callbacks
// report content-area coords; the browser backend uses offsetX/offsetY), so
// this port never carried the subtraction. 245.2 also renames the
// lastMouseClick* press-side latch fields to nextMouseClick*; this port maps
// that double-buffer away entirely (events are polled on the loop goroutine,
// not delivered by an async AWT event thread — see RunShell), so the rename
// has no Go counterpart.
func (c *Client) handleMouseButton(e platform.MouseButton) {
	c.IdleCycles = 0
	if e.Pressed {
		c.MouseClickX = e.X
		c.MouseClickY = e.Y
		// Java: nextMouseClickTime = System.currentTimeMillis()
		// (GameShell.java:304 @32f3062), latched into mouseClickTime once per
		// loop tick — the double-buffer is mapped away here (see above).
		c.MouseClickTime = time.Now().UnixMilli()
		// Java distinguishes the right button via getButton() == BUTTON3;
		// Button 2 maps to AWT's right-click (mouseButton == 2).
		if e.Button == 2 {
			c.MouseClickButton = 2
			c.MouseButton = 2
		} else {
			c.MouseClickButton = 1
			c.MouseButton = 1
		}
		return
	}
	// Java: mouseReleased (GameShell.java:316-320 @32f3062) — idleTimer is
	// reset at the top of this function for both arms.
	c.MouseButton = 0
}

// handleMouseMove maps a platform move/drag event onto mouseX/Y. Java's
// mouseDragged and mouseMoved are identical at this rev
// (GameShell.java:337-361 @32f3062): both reset idleTimer and set mouseX/Y.
//
// 245.2 (GameShell.java:381-407 @176a85f) drops the frame.insets
// subtraction from both handlers — a no-op here; the platform backends
// were always content-area-relative (see handleMouseButton).
func (c *Client) handleMouseMove(e platform.MouseMove) {
	c.IdleCycles = 0
	c.MouseX = e.X
	c.MouseY = e.Y
}

// handleMouseCross maps a platform enter/leave event to the mouse-state
// resets. Java: mouseEntered/mouseExited on GameShell
// (GameShell.java:326-334 @32f3062).
func (c *Client) handleMouseCross(e platform.MouseCross) {
	if e.Entered {
		// Java: mouseEntered is empty in 274 (254's only content was the
		// InputTracking record).
		return
	}
	// Java: mouseExited resets idleTimer/mouseX/mouseY
	// (GameShell.java:331-333) — without them the client keeps hovering at
	// the last in-window position after the cursor leaves.
	c.IdleCycles = 0
	c.MouseX = -1
	c.MouseY = -1
}

// handleKey ports keyPressed/keyReleased (GameShell.java:363-460 @32f3062).
// Java's pipeline: getKeyCode (var2) → translate to a Java code (var3) via a
// series of `if (var2 == N) var3 = ...` overrides; chars below 30 are zeroed;
// the final var3 drives keyHeld and the keyQueue.
//
// The platform layer reports physical platform.Key identifiers, so we
// synthesize AWT keycodes via awtFor, then apply Java's exact override
// sequence, preserving the bug-for-bug translation principle.
func (c *Client) handleKey(e platform.KeyPress) {
	c.IdleCycles = 0

	var2 := awtFor(e.Key) // AWT keyCode (Java: arg0.getKeyCode())
	var3 := charFor(e)    // initial keyChar (Java: arg0.getKeyChar())

	// Java: `if (var3 < 30) var3 = 0;` strips control characters except the
	// few it explicitly overrides below.
	if var3 < 30 {
		var3 = 0
	}
	if var2 == 37 {
		var3 = 1
	}
	if var2 == 39 {
		var3 = 2
	}
	if var2 == 38 {
		var3 = 3
	}
	if var2 == 40 {
		var3 = 4
	}
	if var2 == 17 {
		var3 = 5
	}
	if var2 == 8 {
		var3 = 8
	}
	if var2 == 127 {
		var3 = 8
	}
	if var2 == 9 {
		var3 = 9
	}
	if var2 == 10 {
		var3 = 10
	}

	// Java: GameShell.java:363-420 @32f3062 (keyPressed) applies the F-key,
	// Home, End, PgUp, PgDown overrides AFTER the var2 == 10 line;
	// GameShell.java:422-460 (keyReleased) deliberately stops at
	// var2 == 10 and does NOT apply those overrides — so they must stay
	// inside the press-only branch here.
	if e.Down {
		if var2 >= 112 && var2 <= 123 {
			var3 = var2 + 1008 - 112
		}
		if var2 == 36 {
			var3 = 1000
		}
		if var2 == 35 {
			var3 = 1001
		}
		if var2 == 33 {
			var3 = 1002
		}
		if var2 == 34 {
			var3 = 1003
		}
		// KeyHeld records held-state for any in-range key, including
		// text characters — non-text "letter held" checks elsewhere in
		// the game still need this (e.g. while the chat input cursor
		// is active).
		if var3 > 0 && var3 < 128 {
			c.KeyHeld[var3] = 1
		}
		// Java (GameShell.java keyPressed): the KeyQueue push is gated by
		// `var3 > 4`. Printable text arrives via the separate CharInput path in
		// this port, so the only values that reach handleKey and satisfy
		// `var3 > 4` are the AWT-style sentinels for non-text keys (Ctrl=5,
		// Backspace=8, Tab=9, Enter=10, Home/End/PgUp/PgDown=1000..1003,
		// F1..F12=1008..1019); arrow keys map to var3 1..4 and are correctly
		// excluded here.
		isSentinel := var3 == 5 || var3 == 8 || var3 == 9 || var3 == 10 || var3 >= 1000
		if isSentinel {
			c.KeyQueue[c.KeyQueueWritePos] = var3
			c.KeyQueueWritePos = (c.KeyQueueWritePos + 1) & 0x7F
		}
		return
	}

	// key release — note: no F-key / Home / End / PgUp / PgDown overrides
	// here, matching Java's keyReleased.
	if var3 > 0 && var3 < 128 {
		c.KeyHeld[var3] = 0
	}
}

// awtFor maps a neutral platform.Key back to the AWT keyCode the Java client
// expected. Returns 0 for any key without a Java override — the call sites
// only branch on the specific codes listed below, so 0 is a safe sentinel.
func awtFor(k platform.Key) int {
	switch k {
	case platform.KeyLeft:
		return 37
	case platform.KeyRight:
		return 39
	case platform.KeyUp:
		return 38
	case platform.KeyDown:
		return 40
	case platform.KeyCtrl:
		return 17
	case platform.KeyBackspace:
		return 8
	case platform.KeyDelete:
		return 127
	case platform.KeyTab:
		return 9
	case platform.KeyReturn, platform.KeyEnter:
		return 10
	case platform.KeyHome:
		return 36
	case platform.KeyEnd:
		return 35
	case platform.KeyPageUp:
		return 33
	case platform.KeyPageDown:
		return 34
	case platform.KeyF1:
		return 112
	case platform.KeyF2:
		return 113
	case platform.KeyF3:
		return 114
	case platform.KeyF4:
		return 115
	case platform.KeyF5:
		return 116
	case platform.KeyF6:
		return 117
	case platform.KeyF7:
		return 118
	case platform.KeyF8:
		return 119
	case platform.KeyF9:
		return 120
	case platform.KeyF10:
		return 121
	case platform.KeyF11:
		return 122
	case platform.KeyF12:
		return 123
	}
	return 0
}

// charFor synthesizes Java's keyChar from a neutral KeyPress. AWT delivers
// the typed character directly (case-shifted by Shift); the platform layer
// reports only the rune ('A'..'Z', '0'..'9') plus a Modifiers bitset for
// KeyRune, so we recreate the shift case-folding here. For named keys
// (non-KeyRune), returns 0; the override sequence in handleKey then sets
// var3 to the proper sentinel (1=←, 2=→, 8=BS, …).
//
// shiftedChar maps US-keyboard physical-key characters to their
// Shift-modified equivalents. AWT's KeyEvent.getKeyChar() (which
// the Java client consumes) reports the typed character directly;
// the platform layer reports only the physical key plus the modifier
// bitmask, so the application has to resolve Shift itself. Keys not
// in this table (letters, the function keys, arrows, etc.) are
// handled separately by charFor. Layout-dependent — covers US
// QWERTY only; other layouts (UK, AZERTY, Dvorak) would map keys
// differently and need their own table or an IME path.
var shiftedChar = map[rune]rune{
	'1': '!', '2': '@', '3': '#', '4': '$', '5': '%',
	'6': '^', '7': '&', '8': '*', '9': '(', '0': ')',
	'-': '_', '=': '+', '[': '{', ']': '}', '\\': '|',
	';': ':', '\'': '"', ',': '<', '.': '>', '/': '?',
	'`': '~',
}

func charFor(e platform.KeyPress) int {
	if e.Key == platform.KeyEscape {
		// Java: AWT getKeyChar() for VK_ESCAPE is the control char 27 — not
		// CHAR_UNDEFINED — so handleKey's `var3 < 30` zeroes it: Escape pushes
		// nothing to KeyQueue, matching Java keyPressed
		// (audit client-shell-01).
		return 27
	}
	if e.Key != platform.KeyRune {
		// Java: arg0.getKeyChar() returns KeyEvent.CHAR_UNDEFINED ('￿' = 65535)
		// for keys with no character (Shift/Alt/Caps/etc). handleKey's `var3 < 30`
		// leaves 65535 intact and `var3 > 4` then records it in keyQueue
		// exactly as Java does (arrow keys are separately remapped to
		// 1..4 by the var2 == 37/39/38/40 overrides, so their 65535 here is moot).
		return 65535
	}
	r := e.Rune
	shift := e.Mods.Has(platform.ModShift)
	switch {
	case r >= 'A' && r <= 'Z':
		if !shift {
			r = r - 'A' + 'a'
		}
		return int(r)
	case r >= '0' && r <= '9':
		if shift {
			if mapped, ok := shiftedChar[r]; ok {
				return int(mapped)
			}
		}
		return int(r)
	}
	if shift {
		if mapped, ok := shiftedChar[r]; ok {
			return int(mapped)
		}
	}
	return int(r)
}

// RunShell is the single game loop: poll input, run catch-up logic ticks, draw,
// present, sleep. Faithful to Java GameShell.run() / the TS client (no
// requestAnimationFrame). Runs on the loop goroutine established by platform.Main.
// initScreenSize sets the client's screen dimensions from the active platform
// backend's window size. The host-shell refactor's RunShell replaces the old
// InitApplication, which formerly set ScreenWidth/Height; this restores that
// step. Must run before the first MessageBox: ensureOverlay sizes the overlay
// PixMap from these, and a zero size yields an empty pixel buffer that crashes
// the native gl.Ptr upload (panic: reflect: slice index out of range).
func (c *Client) initScreenSize() {
	c.ScreenWidth, c.ScreenHeight = platform.Active.Size()
}

func (c *Client) RunShell() {
	c.initScreenSize()
	c.MessageBox("Loading...", 0)
	c.Load()

	var3 := 0
	var4 := 256
	var5 := 1
	var6 := 0
	for i := range 10 {
		c.OTim[i] = time.Now().UnixMilli()
	}
	// Java: GameShell.java:136 — `long var1 = System.currentTimeMillis()`.
	// Value is unconditionally reassigned before any read (line 152 below),
	// so the initial value is observationally irrelevant; aligned with Java
	// for literal-port hygiene. 245.2 keeps this exact pre-loop hoist
	// (GameShell.java:150 @176a85f, `long ntime`); 244's deob had declared
	// it inside the loop instead — already matching, nothing to re-port.
	var1 := time.Now().UnixMilli() //nolint:ineffassign,staticcheck // Java: faithful pre-loop init; reassigned before any read (see comment above)
	for c.State >= 0 && !platform.Active.ShouldClose() {
		if c.State > 0 {
			c.State--
			if c.State == 0 {
				c.Shutdown()
				return
			}
		}
		for _, ev := range platform.Active.PollEvents() {
			c.dispatchInputEvent(ev)
		}

		var8 := var4
		var9 := var5
		var4 = 300
		var5 = 1
		var1 = time.Now().UnixMilli()
		if c.OTim[var3] == 0 {
			var4 = var8
			var5 = var9
		} else if var1 > c.OTim[var3] {
			var4 = int(int64(c.DelTime*2560) / (var1 - c.OTim[var3]))
		}
		var4 = max(var4, 25)
		// Independent `if` (not else-if): matches 245.2 (GameShell.java:178
		// @176a85f). 244's deob chained it as `else if` — behaviorally
		// identical (the 25-clamped value can never exceed 256).
		if var4 > 256 {
			var4 = 256
			var5 = int(int64(c.DelTime) - (var1-c.OTim[var3])/10)
		}
		// Java: GameShell.java:189-191 — clamp the sleep delta to deltime
		// (only exceedable under backward clock skew).
		if var5 > c.DelTime {
			var5 = c.DelTime
		}
		c.OTim[var3] = var1
		var3 = (var3 + 1) % 10
		if var5 > 1 {
			for i := range 10 {
				if c.OTim[i] != 0 {
					c.OTim[i] += int64(var5)
				}
			}
		}
		var5 = max(var5, c.MinDel)
		time.Sleep(time.Duration(var5) * time.Millisecond)
		for var6 < 256 {
			c.Loop() // Java: this.loop() (GameShell.java:205 @2e62978) — 254 rename of update()
			c.MouseClickButton = 0
			c.KeyQueueReadPos = c.KeyQueueWritePos
			var6 += var4
		}
		var6 &= 0xFF
		// Java: GameShell.java:186-188 computed `this.fps = var4 * 1000 /
		// (this.delTime * 256)` here. fps was never read; dropped per the
		// deob-artifact exclusion policy.

		platform.Active.BeginFrame()
		c.Draw()
		platform.Active.EndFrame()
	}
	// Java: GameShell start()/stop()/destroy() and windowClosing()→destroy()
	// drive applet pause/resume/teardown via the state field. Intentionally
	// not ported: the standalone host shell collapses them to this single
	// exit — ShouldClose() replaces windowClosing→destroy→state=-1
	// (audit gameshell-05).
	if c.State == -1 || platform.Active.ShouldClose() {
		c.Shutdown()
	}
}

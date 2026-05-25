package client

import (
	"os"
	"time"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/inputtracking"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/bootfont"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
)

type GameShell struct {
}

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

// DrawProgressGameShell renders the boot-time loading bar used before
// c.JagTitle has been downloaded (i.e. when Client.DrawProgress falls
// through to here because no title archive yet). Java:
// GameShell.drawProgress(String, int) at GameShell.java:529-560.
//
// Java painted directly to the AWT base component's Graphics; the Go
// port paints into a shared overlay PixMap (via ensureOverlay) using
// pix2d for rectangles and bootfont (basicfont.Face7x13) for the
// message text, then composites via OverlayPixMap.Draw. The Helvetica
// BOLD 13 in the Java source is substituted with bootfont's monospace
// 7x13 — pixfont's RuneScape font isn't available at this phase.
func (c *Client) DrawProgressGameShell(message string, percent int) {
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

// handleFocus mirrors Java's focusGained / focusLost on
// GameShell (GameShell.java:444-456). Java sets `refresh = true`
// and calls `refresh()` on focus gained, then forwards both
// events to InputTracking. The neutral FocusChange event is
// delivered by the platform backend when the window's focus state changes.
func (c *Client) handleFocus(e platform.FocusChange) {
	if e.Gained {
		c.Refresh = true
		if inputtracking.Enabled {
			inputtracking.FocusGained()
		}
		return
	}
	if inputtracking.Enabled {
		inputtracking.FocusLost()
	}
}

// handleCharInput is the typed-text channel (Java getKeyChar). One resolved
// character per event; control chars < 30 are dropped (matching Java).
//
// Java: keyPressed zeroes any keyChar < 30 then pushes only var3 > 4
// (GameShell.java:342-396), so a bare control char in [5,29] is dropped;
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
	if inputtracking.Enabled {
		inputtracking.KeyPressed(var3)
	}
}

// handleMouseButton maps a platform press/release event onto the same mouse*
// fields and InputTracking calls Java's mousePressed/mouseReleased used.
// Java reference: GameShell.java:263-300.
func (c *Client) handleMouseButton(e platform.MouseButton) {
	c.IdleCycles = 0
	if e.Pressed {
		c.MouseClickX = e.X
		c.MouseClickY = e.Y
		// Java distinguishes the right ("meta") button via isMetaDown();
		// Button 2 maps to AWT's right-click (mouseButton == 2).
		if e.Button == 2 {
			c.MouseClickButton = 2
			c.MouseButton = 2
			if inputtracking.Enabled {
				inputtracking.MousePressed(e.X, 1, e.Y)
			}
		} else {
			c.MouseClickButton = 1
			c.MouseButton = 1
			if inputtracking.Enabled {
				inputtracking.MousePressed(e.X, 0, e.Y)
			}
		}
		return
	}
	// Java captures the meta state at release-time too; the platform
	// event at Release describes the still-pressed buttons (i.e. excludes
	// the just-released one), so we instead infer from c.MouseButton,
	// the value latched at Press.
	releasedRight := c.MouseButton == 2
	c.MouseButton = 0
	if inputtracking.Enabled {
		if releasedRight {
			inputtracking.MouseReleased(1)
		} else {
			inputtracking.MouseReleased(0)
		}
	}
}

// handleMouseMove maps a platform move/drag event onto mouseX/Y and
// InputTracking. Java's mouseDragged and mouseMoved are identical at this rev
// (GameShell.java:308-336): both set mouseX/Y and call
// InputTracking.mouseMoved(y, x) — note the (y, x) swap. The Go port
// preserves that swap.
func (c *Client) handleMouseMove(e platform.MouseMove) {
	c.IdleCycles = 0
	c.MouseX = e.X
	c.MouseY = e.Y
	if inputtracking.Enabled {
		inputtracking.MouseMoved(e.Y, e.X) // Java: InputTracking.mouseMoved(y, x) — note the swap
	}
}

// handleMouseCross maps a platform enter/leave event to InputTracking.
// Java: mouseEntered/mouseExited on GameShell (GameShell.java:323-336).
func (c *Client) handleMouseCross(e platform.MouseCross) {
	if !inputtracking.Enabled {
		return
	}
	if e.Entered {
		inputtracking.MouseEntered()
	} else {
		inputtracking.MouseExited()
	}
}

// handleKey ports keyPressed/keyReleased (GameShell.java:338-439). Java's
// pipeline: getKeyCode (var2) → translate to a Java code (var3) via a series
// of `if (var2 == N) var3 = ...` overrides; chars below 30 are zeroed; the
// final var3 drives actionKey, the keyQueue, and InputTracking.
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

	// Java: GameShell.java:338-397 (keyPressed) applies the F-key, Home,
	// End, PgUp, PgDown overrides AFTER the var2 == 10 line; GameShell.java:
	// 399-439 (keyReleased) deliberately stops at var2 == 10 and does NOT
	// apply those overrides. The prior Go port collapsed both into one
	// override sequence and then branched on Press/Release, causing the
	// release branch to pick up press-only overrides and feed the
	// override-mapped sentinel value (1008+offset, 1000, 1001, 1002, 1003)
	// to inputtracking.KeyReleased — Java would have fed it the raw
	// keyChar (typically CHAR_UNDEFINED = 0xFFFF). Recorded input-tracking
	// byte streams therefore diverged on every release of a special key.
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
		// ActionKey records held-state for any in-range key, including
		// text characters — non-text "letter held" checks elsewhere in
		// the game still need this (e.g. while the chat input cursor
		// is active).
		if var3 > 0 && var3 < 128 {
			c.ActionKey[var3] = 1
		}
		// Text characters (printable ASCII range, post-overrides) come
		// in via CharInput with proper keyboard-layout / shift /
		// dead-key resolution; only the AWT-style sentinels for
		// non-text keys (Ctrl=5, Backspace=8, Tab=9, Enter=10,
		// Home/End/PgUp/PgDown=1000..1003, F1..F12=1008..1019) are
		// pushed to KeyQueue and inputtracking from KeyPress.
		isSentinel := var3 == 5 || var3 == 8 || var3 == 9 || var3 == 10 || var3 >= 1000
		if isSentinel {
			c.KeyQueue[c.KeyQueueWritePos] = var3
			c.KeyQueueWritePos = (c.KeyQueueWritePos + 1) & 0x7F
			if inputtracking.Enabled {
				inputtracking.KeyPressed(var3)
			}
		}
		return
	}

	// key release — note: no F-key / Home / End / PgUp / PgDown overrides
	// here, matching Java's keyReleased.
	if var3 > 0 && var3 < 128 {
		c.ActionKey[var3] = 0
	}
	if inputtracking.Enabled {
		inputtracking.KeyReleased(var3)
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
	if e.Key != platform.KeyRune {
		return 0
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
func (c *Client) RunShell() {
	c.DrawProgress("Loading...", 0)
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
	// for literal-port hygiene.
	var1 := time.Now().UnixMilli()
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
		if var4 > 256 {
			var4 = 256
			var5 = int(int64(c.DelTime) - (var1-c.OTim[var3])/10)
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
			c.Update()
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
	if c.State == -1 || platform.Active.ShouldClose() {
		c.Shutdown()
	}
}

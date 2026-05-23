package client

import (
	"log"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/unit"

	"goscape-client/pkg/jagex2/client/inputtracking"
	"goscape-client/pkg/jagex2/graphics/bootfont"
	"goscape-client/pkg/jagex2/graphics/pix2d"
	"goscape-client/pkg/jagex2/graphics/pixmap"
)

type GameShell struct {
}

func (c *Client) InitApplication(height int, width int) {
	c.ScreenWidth = width
	c.ScreenHeight = height
	// Java: the AWT base component owned both a Frame and a Graphics
	// field (this.frame / this.graphics) that AWT painted automatically
	// when the component was added to the window. The Gio port uploads
	// directly via OverlayPixMap and the per-FrameEvent pixmap blit; see
	// viewbox.go for the same architectural deviation. Java's drawArea
	// PixMap field is a deob artifact (write-only) and is omitted here.
	c.buildInputFilters()

	// The window/event-loop goroutine below opens the Gio window before
	// Run() takes control of the calling goroutine, matching Java's
	// "open frame, then start game thread" ordering.
	go func() {
		// Create new window
		w := new(app.Window)
		w.Option(app.Title("Jagex"))
		w.Option(app.Size(unit.Dp(c.ScreenWidth), unit.Dp(c.ScreenHeight)))
		w.Option(app.MinSize(unit.Dp(c.ScreenWidth), unit.Dp(c.ScreenHeight)))
		w.Option(app.MaxSize(unit.Dp(c.ScreenWidth), unit.Dp(c.ScreenHeight)))

		if err := c.draw(w); err != nil {
			log.Fatalf("gameshell: %v", err)
		}
		os.Exit(0)
	}()
	// app.Main() is no longer called here; cmd/client/main.go owns it on
	// the process's main goroutine, as required on macOS (see
	// https://gioui.org/app). InitApplication runs from one of two
	// goroutines spawned by main(), and the window event loop runs in
	// the nested goroutine above. Only the game loop blocks this
	// goroutine.
	c.Run()
}

func (c *Client) draw(w *app.Window) error {
	// for every frame, clear the screen and redraw the whole layout

	//// ops are the operations from the UI
	//var ops op.Ops

	// Listen for events in the window
	for {
		// detect what type of event
		switch e := w.Event().(type) {

		case app.DestroyEvent:
			// Java: GameShell.windowClosing (GameShell.java:474-476) calls
			// destroy() → sets state = -1 → game loop drains state and
			// calls Shutdown() → Unload() → sleep → os.Exit. The prior
			// Go DestroyEvent handler returned directly, then the parent
			// goroutine called os.Exit(0) immediately — bypassing Unload
			// entirely, so sockets / MIDI / cache handles leaked.
			//
			// Signal the game loop to run its Shutdown sequence (which
			// closes c.Stream, drops cache references, calls
			// signlink.StopMidi). Sleep briefly to let it complete in
			// the common case; the fallback os.Exit(0) at the caller
			// still fires if the game loop is wedged.
			c.State = -1
			time.Sleep(1500 * time.Millisecond)
			return e.Err

		case app.FrameEvent:
			// Drain pending input events through Gio's pull-per-frame model.
			// event.Op declares `c` as a tag for the current clip area (the
			// full window since we push no clip), and e.Source.Event(filters)
			// returns one queued event per call. Java: AWT pushed events to
			// listener callbacks; Gio inverts that to a per-frame poll.
			//
			// Hold pixmap.OpsMu across the input-op write, the event drain,
			// and e.Frame so any concurrent PixMap.Draw on the game goroutine
			// can't race with the Gio compositor consuming c.Ops.
			pixmap.OpsMu.Lock()
			event.Op(&c.Ops, c)
			// Request keyboard focus on the client tag so Gio routes
			// key.EditEvent (OS-level typed-text dispatch, layout/IME/
			// dead-key aware) to us. Re-issued every frame so focus
			// re-asserts itself after alt-tab or window-manager focus
			// changes. See `c.handleEditEvent`.
			e.Source.Execute(key.FocusCmd{Tag: c})
			for {
				ev, ok := e.Source.Event(c.inputFilters...)
				if !ok {
					break
				}
				c.dispatchInputEvent(ev)
			}

			// A request to draw the window state.
			// Gio only issues FrameEvents when the window is resized or
			// the user interacts with the window.
			//
			// When the program receives a FrameEvent, it is responsible
			// for updating the display by calling the e.Frame function with
			// an operation list representing the new state.

			// This is sent when the application should re-render.
			// Resets the layout.Context for a new frame.
			//gtx := app.NewContext(&c.Ops, e)
			//gtx := app.NewContext(&ops, e)

			// PlotSprite the state into ops based on events in e.Queue.
			//draw(gtx)
			//draw(&c.Ops)
			//gtx.Execute(op.InvalidateCmd{})

			// Update the display - pass the operations list to the window driver
			//e.Frame(gtx.Ops)
			e.Frame(&c.Ops)
			pixmap.OpsMu.Unlock()
			w.Invalidate()
		}
	}
}

func (c *Client) RunGameShell() {
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
	for c.State >= 0 {
		if c.State > 0 {
			c.State--
			if c.State == 0 {
				c.Shutdown()
				return
			}
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
		c.Draw()
	}
	if c.State == -1 {
		c.Shutdown()
	}

}

func (c *Client) Shutdown() {
	c.State = -2
	c.Unload()
	time.Sleep(1 * time.Second)
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

	c.OverlayPixMap.Draw(&c.Ops, 0, 0)
}

// buildInputFilters constructs the per-Client filter list once. Java declared
// listener interfaces (MouseListener/KeyListener) and registered them via
// addMouseListener etc.; Gio's modern (post-2024-02) input API replaces that
// with a set of event.Filter values pulled per frame via source.Event(...).
func (c *Client) buildInputFilters() {
	const allMods = key.ModShift | key.ModCtrl | key.ModAlt | key.ModSuper | key.ModCommand

	c.inputFilters = []event.Filter{
		pointer.Filter{
			Target: c,
			Kinds:  pointer.Press | pointer.Release | pointer.Move | pointer.Drag | pointer.Enter | pointer.Leave,
		},
		// FocusFilter routes key.EditEvent (OS-aware text input) plus
		// FocusEvent / SnippetEvent / SelectionEvent to us. EditEvent
		// is the proper path for typed characters with correct keyboard
		// layout / Shift / dead-key / IME handling; key.Event below
		// stays focused on non-text keys (arrows, F-keys, etc.) where
		// AWT-style sentinels are still meaningful.
		key.FocusFilter{Target: c},
	}

	named := []key.Name{
		key.NameLeftArrow, key.NameRightArrow, key.NameUpArrow, key.NameDownArrow,
		key.NameReturn, key.NameEnter, key.NameEscape,
		key.NameHome, key.NameEnd, key.NameDeleteBackward, key.NameDeleteForward,
		key.NamePageUp, key.NamePageDown,
		key.NameTab, key.NameSpace,
		key.NameCtrl, key.NameShift, key.NameAlt, key.NameSuper, key.NameCommand,
		key.NameF1, key.NameF2, key.NameF3, key.NameF4, key.NameF5, key.NameF6,
		key.NameF7, key.NameF8, key.NameF9, key.NameF10, key.NameF11, key.NameF12,
		key.NameBack,
	}
	for _, n := range named {
		c.inputFilters = append(c.inputFilters, key.Filter{Name: n, Optional: allMods})
	}
	// Letters A-Z. Gio reports the uppercase letter as Name regardless of
	// Shift; we synthesize the lowercase keyChar below based on Modifiers.
	for r := 'A'; r <= 'Z'; r++ {
		c.inputFilters = append(c.inputFilters, key.Filter{Name: key.Name(string(r)), Optional: allMods})
	}
	// Digits 0-9.
	for r := '0'; r <= '9'; r++ {
		c.inputFilters = append(c.inputFilters, key.Filter{Name: key.Name(string(r)), Optional: allMods})
	}
}

// dispatchInputEvent routes a single Gio event to the matching Java handler.
// Java: separate listener methods (mousePressed, keyPressed, ...) on
// GameShell; Go switches on the dynamic event type.
func (c *Client) dispatchInputEvent(ev event.Event) {
	switch e := ev.(type) {
	case pointer.Event:
		c.handlePointer(e)
	case key.Event:
		c.handleKey(e)
	case key.EditEvent:
		c.handleEditEvent(e)
	case key.FocusEvent:
		c.handleFocus(e)
	}
}

// handleFocus mirrors Java's focusGained / focusLost on
// GameShell (GameShell.java:444-456). Java sets `refresh = true`
// and calls `refresh()` on focus gained, then forwards both
// events to InputTracking. Gio delivers a key.FocusEvent via
// FocusFilter when the window's focus state changes.
func (c *Client) handleFocus(e key.FocusEvent) {
	if e.Focus {
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

// handleEditEvent processes a Gio text-input event — the OS-level
// typed-text channel that resolves keyboard layout, modifier combos,
// dead keys, AltGr, and IME composition. Each rune in `e.Text` is the
// character the user actually intended to type, regardless of physical
// key layout. This is the proper port of Java's
// `KeyEvent.getKeyChar()` semantics; key.Event below remains for the
// modal sentinel keys (arrows, F-keys, Enter, Backspace, etc.) that
// the Java client maps to specific numeric IDs.
//
// ActionKey held-state for letters/digits/punctuation is still
// maintained by handleKey on key.Event (since EditEvent has no
// release counterpart). For Press, handleKey skips KeyQueue and
// inputtracking writes for text characters so this function is the
// sole producer of typed-text entries in those streams.
func (c *Client) handleEditEvent(e key.EditEvent) {
	c.IdleCycles = 0
	for _, r := range e.Text {
		var3 := int(r)
		if var3 <= 4 {
			continue
		}
		c.KeyQueue[c.KeyQueueWritePos] = var3
		c.KeyQueueWritePos = (c.KeyQueueWritePos + 1) & 0x7F
		if inputtracking.Enabled {
			inputtracking.KeyPressed(var3)
		}
	}
}

// handlePointer maps a Gio pointer.Event onto the same mouse* fields and
// InputTracking calls Java's mousePressed/mouseReleased/mouseMoved/
// mouseDragged/mouseEntered/mouseExited used. Java reference:
// GameShell.java:263-336.
func (c *Client) handlePointer(e pointer.Event) {
	// Gio gives sub-pixel coordinates; the game uses integer pixel coords.
	x := int(e.Position.X)
	y := int(e.Position.Y)

	switch e.Kind {
	case pointer.Press:
		// Java distinguishes the right ("meta") button via isMetaDown(); in
		// Gio the pressed button is in e.Buttons. ButtonSecondary maps to
		// AWT's right-click (mouseButton == 2).
		c.IdleCycles = 0
		c.MouseClickX = x
		c.MouseClickY = y
		if e.Buttons.Contain(pointer.ButtonSecondary) {
			c.MouseClickButton = 2
			c.MouseButton = 2
			if inputtracking.Enabled {
				inputtracking.MousePressed(x, 1, y)
			}
		} else {
			c.MouseClickButton = 1
			c.MouseButton = 1
			if inputtracking.Enabled {
				inputtracking.MousePressed(x, 0, y)
			}
		}
	case pointer.Release:
		c.IdleCycles = 0
		// Java captures the meta state at release-time too; Gio's e.Buttons
		// at Release describes the still-pressed buttons (i.e. excludes the
		// just-released one), so we instead infer from c.MouseButton, the
		// value latched at Press.
		releasedRight := c.MouseButton == 2
		c.MouseButton = 0
		if inputtracking.Enabled {
			if releasedRight {
				inputtracking.MouseReleased(1)
			} else {
				inputtracking.MouseReleased(0)
			}
		}
	case pointer.Move, pointer.Drag:
		// Java's mouseDragged and mouseMoved are identical at this rev
		// (GameShell.java:308-336): both set mouseX/Y and call
		// InputTracking.mouseMoved(y, x) — note the (y, x) swap. The Go port
		// preserves that swap.
		c.IdleCycles = 0
		c.MouseX = x
		c.MouseY = y
		if inputtracking.Enabled {
			inputtracking.MouseMoved(y, x)
		}
	case pointer.Enter:
		if inputtracking.Enabled {
			inputtracking.MouseEntered()
		}
	case pointer.Leave:
		if inputtracking.Enabled {
			inputtracking.MouseExited()
		}
	}
}

// handleKey ports keyPressed/keyReleased (GameShell.java:338-439). Java's
// pipeline: getKeyCode (var2) → translate to a Java code (var3) via a series
// of `if (var2 == N) var3 = ...` overrides; chars below 30 are zeroed; the
// final var3 drives actionKey, the keyQueue, and InputTracking.
//
// Gio doesn't expose AWT keycodes, so we synthesize them via keyNameToAwt,
// then apply Java's exact override sequence, preserving the bug-for-bug
// translation principle.
func (c *Client) handleKey(e key.Event) {
	c.IdleCycles = 0

	var2 := keyNameToAwt(e.Name) // AWT keyCode (Java: arg0.getKeyCode())
	var3 := keyCharFor(e)        // initial keyChar (Java: arg0.getKeyChar())

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
	if e.State == key.Press {
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
		// in via key.EditEvent with proper keyboard-layout / shift /
		// dead-key resolution; only the AWT-style sentinels for
		// non-text keys (Ctrl=5, Backspace=8, Tab=9, Enter=10,
		// Home/End/PgUp/PgDown=1000..1003, F1..F12=1008..1019) are
		// pushed to KeyQueue and inputtracking from key.Event.
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

	// key.Release — note: no F-key / Home / End / PgUp / PgDown overrides
	// here, matching Java's keyReleased.
	if var3 > 0 && var3 < 128 {
		c.ActionKey[var3] = 0
	}
	if inputtracking.Enabled {
		inputtracking.KeyReleased(var3)
	}
}

// keyNameToAwt maps a Gio key.Name back to the AWT keyCode the Java client
// expected. Returns 0 for any key without a Java override — the call sites
// only branch on the specific codes listed below, so 0 is a safe sentinel.
func keyNameToAwt(name key.Name) int {
	switch name {
	case key.NameLeftArrow:
		return 37
	case key.NameRightArrow:
		return 39
	case key.NameUpArrow:
		return 38
	case key.NameDownArrow:
		return 40
	case key.NameCtrl:
		return 17
	case key.NameDeleteBackward:
		return 8
	case key.NameDeleteForward:
		return 127
	case key.NameTab:
		return 9
	case key.NameReturn, key.NameEnter:
		return 10
	case key.NameHome:
		return 36
	case key.NameEnd:
		return 35
	case key.NamePageUp:
		return 33
	case key.NamePageDown:
		return 34
	case key.NameF1:
		return 112
	case key.NameF2:
		return 113
	case key.NameF3:
		return 114
	case key.NameF4:
		return 115
	case key.NameF5:
		return 116
	case key.NameF6:
		return 117
	case key.NameF7:
		return 118
	case key.NameF8:
		return 119
	case key.NameF9:
		return 120
	case key.NameF10:
		return 121
	case key.NameF11:
		return 122
	case key.NameF12:
		return 123
	}
	return 0
}

// keyCharFor synthesizes Java's keyChar from a Gio key.Event. AWT delivers
// the typed character directly (case-shifted by Shift); Gio reports only the
// uppercase letter Name plus a Modifiers bitset, so we recreate the shift
// case-folding here. For non-text keys, returns 0; the override sequence in
// handleKey then sets var3 to the proper sentinel (1=←, 2=→, 8=BS, …).
// shiftedChar maps US-keyboard physical-key characters to their
// Shift-modified equivalents. AWT's KeyEvent.getKeyChar() (which
// the Java client consumes) reports the typed character directly;
// Gio's key.Event reports only the physical key plus the modifier
// bitmask, so the application has to resolve Shift itself. Keys not
// in this table (letters, the function keys, arrows, etc.) are
// handled separately by keyCharFor. Layout-dependent — covers US
// QWERTY only; other layouts (UK, AZERTY, Dvorak) would map keys
// differently and need their own table or an IME path.
var shiftedChar = map[rune]rune{
	'1': '!', '2': '@', '3': '#', '4': '$', '5': '%',
	'6': '^', '7': '&', '8': '*', '9': '(', '0': ')',
	'-': '_', '=': '+', '[': '{', ']': '}', '\\': '|',
	';': ':', '\'': '"', ',': '<', '.': '>', '/': '?',
	'`': '~',
}

func keyCharFor(e key.Event) int {
	s := string(e.Name)
	if len(s) != 1 {
		return 0
	}
	r := rune(s[0])
	shift := e.Modifiers.Contain(key.ModShift)
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

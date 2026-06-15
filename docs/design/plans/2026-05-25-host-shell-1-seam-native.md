# Host Shell Plan 1 — Seam + Native (GLFW) Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Gio on the native path with a hand-rolled `platform` seam (toolkit-neutral `Backend` interface + neutral input events) and a GLFW + go-gl backend, driven by one golden-source-faithful game loop; keep the browser building via a stub `js` backend.

**Architecture:** A new `pkg/jagex2/platform` package defines an immediate-mode `Backend` interface (`Blit` a textured quad per PixMap, `UploadTexture` in place) and neutral input event types. `PixMap` holds a backend `Texture` instead of a Gio op. One unified loop (native: on the `LockOSThread`'d main goroutine; the wasm path lands in Plan 2) preserves the Java `otim/ratio/delta/count` pacing. The global `pixmap.OpsMu` is removed; a narrow `client.flameMu` guards only the flames-goroutine ↔ present hand-off of `ImageTitle0/1`.

**Tech Stack:** Go 1.26, `github.com/go-gl/glfw/v3.3/glfw` (cgo), `github.com/go-gl/gl/v2.1/gl` (GLES2-compatible subset), OpenGL ES 2.0 / WebGL1 semantics.

**Reference spec:** `docs/superpowers/specs/2026-05-25-hand-rolled-host-shell-design.md`

**Sandbox build prefix (this repo):** every Go command is prefixed
`TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache`. The cgo native build
will NOT compile in the sandbox (no GL/X11 headers, no display); native build +
run is host-only. The wasm build and platform-neutral `go test`/`go vet` DO run in
the sandbox. All commits use `git commit --no-gpg-sign`.

---

## File Structure

| File | Responsibility |
|---|---|
| `pkg/jagex2/platform/platform.go` | **New, build-neutral.** `Texture`, `Backend` interface, `Active` var, neutral `Event` types, `Key`/`Mod` enums. |
| `pkg/jagex2/platform/platform_test.go` | **New.** Unit tests for event/enum invariants (no GPU). |
| `pkg/jagex2/platform/backend_glfw.go` | **New, `//go:build !js`.** GLFW window + go-gl GLES2 backend: window/loop/textures/blit + input callbacks → neutral events + native-key → `Key` mapping. |
| `pkg/jagex2/platform/backend_js_stub.go` | **New, `//go:build js`.** Compile-only stub backend (panics at runtime). Replaced in Plan 2. |
| `pkg/jagex2/platform/run_native.go` | **New, `//go:build !js`.** `Main()` — `LockOSThread`, build backend, run loop on main goroutine. |
| `pkg/jagex2/platform/run_js.go` | **New, `//go:build js`.** `Main()` — build backend, run loop in goroutine, `select{}`. |
| `pkg/jagex2/client/gameshell.go` | **Modify.** Remove Gio; neutral input translation; unified `RunShell` loop; `awtFor`/`charFor`. |
| `pkg/jagex2/client/client.go` | **Modify.** Remove `Ops`/`inputFilters` fields & Gio imports; `Draw` drops `OpsMu`/`Reset`; six out-of-band sites → explicit present; add `flameMu`; drop `&c.Ops` from 47 `PixMap.Draw` calls. |
| `pkg/jagex2/graphics/pixmap/pixmap.go` | **Modify.** Hold `platform.Texture`; `Draw(x, y)`; remove `op`/`paint`/`MutableImageOp`/`OpsMu`; keep `hashPixels`/`writePixmapPixels`. |
| `pkg/jagex2/graphics/pixmap/pixmap_test.go` | **Modify.** Use a fake `Backend`. |
| `pkg/jagex2/client/keycharfor_test.go`, `handleeditevent_test.go` | **Modify.** Feed neutral events. |
| `cmd/client/main.go` | **Modify.** Remove `gioui.org/app`; call `platform.Main(...)`. |
| `.github/workflows/ci-gate.yml` | **Modify.** Add native-build job (install GL/X11 headers, cgo build). |
| `go.mod` | **Modify.** Add `go-gl` requires. (Gio `replace`/`require` stay until Plan 2.) |

---

## Task 1: `platform` package — interface, events, enums

**Files:**
- Create: `pkg/jagex2/platform/platform.go`
- Test: `pkg/jagex2/platform/platform_test.go`

- [ ] **Step 1: Write the failing test**

```go
package platform

import "testing"

func TestKeyEnumDistinct(t *testing.T) {
	keys := []Key{
		KeyNone, KeyLeft, KeyRight, KeyUp, KeyDown, KeyReturn, KeyEnter,
		KeyEscape, KeyHome, KeyEnd, KeyBackspace, KeyDelete, KeyPageUp,
		KeyPageDown, KeyTab, KeySpace, KeyCtrl, KeyShift, KeyAlt, KeySuper,
		KeyCommand, KeyBack, KeyRune,
		KeyF1, KeyF2, KeyF3, KeyF4, KeyF5, KeyF6,
		KeyF7, KeyF8, KeyF9, KeyF10, KeyF11, KeyF12,
	}
	seen := map[Key]bool{}
	for _, k := range keys {
		if seen[k] {
			t.Fatalf("duplicate Key value %d", k)
		}
		seen[k] = true
	}
}

func TestEventsImplementInterface(t *testing.T) {
	var evs = []Event{
		MouseMove{X: 1, Y: 2},
		MouseButton{X: 1, Y: 2, Button: 1, Pressed: true},
		MouseCross{Entered: true},
		KeyPress{Key: KeyLeft, Down: true},
		CharInput{R: 'a'},
		FocusChange{Gained: true},
	}
	if len(evs) != 6 {
		t.Fatalf("want 6 events, got %d", len(evs))
	}
}

func TestModContains(t *testing.T) {
	m := ModShift | ModCtrl
	if !m.Has(ModShift) || !m.Has(ModCtrl) || m.Has(ModAlt) {
		t.Fatalf("Mod.Has wrong: %v", m)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/platform/ 2>&1 | grep -v 'stat cache'`
Expected: FAIL — package/types undefined.

- [ ] **Step 3: Write the implementation**

```go
// Package platform is the toolkit-neutral host-shell seam: a window/loop/
// present backend and neutral input events, with build-tagged implementations
// (GLFW+go-gl native; syscall/js+WebGL browser). It replaces Gio. See
// docs/superpowers/specs/2026-05-25-hand-rolled-host-shell-design.md.
package platform

// Texture is an opaque, backend-owned GPU texture handle.
type Texture interface{}

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
type CharInput struct{ R rune }

// FocusChange reports the window gaining (true) or losing (false) focus.
type FocusChange struct{ Gained bool }

func (MouseMove) isEvent()   {}
func (MouseButton) isEvent() {}
func (MouseCross) isEvent()  {}
func (KeyPress) isEvent()    {}
func (CharInput) isEvent()   {}
func (FocusChange) isEvent() {}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/platform/ 2>&1 | grep -v 'stat cache'`
Expected: `ok ... platform`.

- [ ] **Step 5: Commit**

```bash
git add pkg/jagex2/platform/platform.go pkg/jagex2/platform/platform_test.go
git commit --no-gpg-sign -m "feat(platform): toolkit-neutral Backend interface + input events"
```

---

## Task 2: Neutral input translation (`gameshell.go`)

Replace the Gio-typed input handlers with neutral-event handlers, and move the
key→AWT and key→char mapping into shared, tested `awtFor`/`charFor`. The
`handleKey`/`handlePointer`/`handleEditEvent`/`handleFocus` *bodies* stay
behaviorally identical (bug-for-bug); only their input types change.

**Files:**
- Modify: `pkg/jagex2/client/gameshell.go`
- Test: `pkg/jagex2/client/keycharfor_test.go` (rewrite), `pkg/jagex2/client/handleeditevent_test.go` (rewrite)

- [ ] **Step 1: Rewrite `keycharfor_test.go` to drive neutral input**

```go
package client

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
)

func TestAwtFor(t *testing.T) {
	cases := map[platform.Key]int{
		platform.KeyLeft: 37, platform.KeyRight: 39, platform.KeyUp: 38,
		platform.KeyDown: 40, platform.KeyCtrl: 17, platform.KeyBackspace: 8,
		platform.KeyDelete: 127, platform.KeyTab: 9, platform.KeyReturn: 10,
		platform.KeyEnter: 10, platform.KeyHome: 36, platform.KeyEnd: 35,
		platform.KeyPageUp: 33, platform.KeyPageDown: 34,
		platform.KeyF1: 112, platform.KeyF12: 123, platform.KeyRune: 0,
	}
	for k, want := range cases {
		if got := awtFor(k); got != want {
			t.Errorf("awtFor(%d) = %d, want %d", k, got, want)
		}
	}
}

func TestCharFor(t *testing.T) {
	// lowercase letter, no shift
	if got := charFor(platform.KeyPress{Key: platform.KeyRune, Rune: 'A'}); got != int('a') {
		t.Errorf("'A' no shift = %d, want %d", got, int('a'))
	}
	// uppercase letter, shift
	if got := charFor(platform.KeyPress{Key: platform.KeyRune, Rune: 'A', Mods: platform.ModShift}); got != int('A') {
		t.Errorf("'A' shift = %d, want %d", got, int('A'))
	}
	// digit shifted -> symbol
	if got := charFor(platform.KeyPress{Key: platform.KeyRune, Rune: '1', Mods: platform.ModShift}); got != int('!') {
		t.Errorf("'1' shift = %d, want %d", got, int('!'))
	}
	// named key -> 0
	if got := charFor(platform.KeyPress{Key: platform.KeyLeft}); got != 0 {
		t.Errorf("named key char = %d, want 0", got)
	}
}
```

- [ ] **Step 2: Rewrite `handleeditevent_test.go` to drive `CharInput`**

```go
package client

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
)

func TestHandleCharInputQueuesPrintable(t *testing.T) {
	c := &Client{}
	c.handleCharInput(platform.CharInput{R: 'x'})
	if c.KeyQueueWritePos != 1 || c.KeyQueue[0] != int('x') {
		t.Fatalf("printable not queued: pos=%d q0=%d", c.KeyQueueWritePos, c.KeyQueue[0])
	}
}

func TestHandleCharInputDropsControl(t *testing.T) {
	c := &Client{}
	c.handleCharInput(platform.CharInput{R: rune(7)}) // bell, < 30
	if c.KeyQueueWritePos != 0 {
		t.Fatalf("control char should be dropped, pos=%d", c.KeyQueueWritePos)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/client/ -run 'TestAwtFor|TestCharFor|TestHandleCharInput' 2>&1 | grep -v 'stat cache'`
Expected: FAIL — `awtFor`/`charFor`/`handleCharInput` undefined.

- [ ] **Step 4: Edit `gameshell.go` imports**

Replace the import block (lines 3–18) with:

```go
import (
	"log"
	"os"
	"time"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/inputtracking"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/bootfont"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pixmap"
	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
)
```

- [ ] **Step 5: Replace `dispatchInputEvent` and delete `buildInputFilters`**

Delete `buildInputFilters` (lines 277–321) entirely (no filters in the neutral
model). Replace `dispatchInputEvent` (lines 323–337) with:

```go
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
```

- [ ] **Step 6: Replace the pointer handlers**

Replace `handlePointer` (lines 396–457) with three neutral handlers that
preserve the exact field updates and InputTracking calls (including the (y, x)
swap in `MouseMoved`):

```go
// handleMouseButton ports Java mousePressed/mouseReleased. Button: 1=left, 2=right.
func (c *Client) handleMouseButton(e platform.MouseButton) {
	c.IdleCycles = 0
	if e.Pressed {
		c.MouseClickX = e.X
		c.MouseClickY = e.Y
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
	// release: infer the released button from the latched c.MouseButton.
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

// handleMouseMove ports Java mouseMoved/mouseDragged (identical at this rev:
// both set mouseX/Y and call InputTracking.mouseMoved(y, x) — note the swap).
func (c *Client) handleMouseMove(e platform.MouseMove) {
	c.IdleCycles = 0
	c.MouseX = e.X
	c.MouseY = e.Y
	if inputtracking.Enabled {
		inputtracking.MouseMoved(e.Y, e.X)
	}
}

// handleMouseCross ports Java mouseEntered/mouseExited.
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
```

- [ ] **Step 7: Replace `handleFocus`, `handleEditEvent`, `handleKey`, `keyNameToAwt`, `keyCharFor`**

Replace `handleFocus` (344–355) signature to neutral:

```go
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
```

Rename `handleEditEvent` → `handleCharInput`, consuming one rune (keep the < 30
drop and the ring-buffer push verbatim):

```go
// handleCharInput is the typed-text channel (Java getKeyChar). One resolved
// character per event; control chars < 30 are dropped (matching Java).
func (c *Client) handleCharInput(e platform.CharInput) {
	c.IdleCycles = 0
	var3 := int(e.R)
	if var3 < 30 {
		return
	}
	c.KeyQueue[c.KeyQueueWritePos] = var3
	c.KeyQueueWritePos = (c.KeyQueueWritePos + 1) & 0x7F
	if inputtracking.Enabled {
		inputtracking.KeyPressed(var3)
	}
}
```

Replace `handleKey` (467–564) to consume `platform.KeyPress`, sourcing `var2`
from `awtFor(e.Key)` and `var3` from `charFor(e)`; the override/sentinel/
ActionKey logic is byte-for-byte the Java port:

```go
// handleKey ports keyPressed/keyReleased. var2 = AWT keyCode (awtFor), var3 =
// initial keyChar (charFor); then Java's exact override sequence.
func (c *Client) handleKey(e platform.KeyPress) {
	c.IdleCycles = 0

	var2 := awtFor(e.Key)
	var3 := charFor(e)

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
		if var3 > 0 && var3 < 128 {
			c.ActionKey[var3] = 1
		}
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

	if var3 > 0 && var3 < 128 {
		c.ActionKey[var3] = 0
	}
	if inputtracking.Enabled {
		inputtracking.KeyReleased(var3)
	}
}
```

Replace `keyNameToAwt` → `awtFor(platform.Key) int` (same code table, neutral keys):

```go
// awtFor maps a neutral Key to the AWT keyCode the Java override sequence
// expects. Returns 0 for keys without a Java override (incl. KeyRune).
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
```

Replace `keyCharFor` → `charFor(platform.KeyPress) int` (keep `shiftedChar`
table; resolve only `KeyRune`):

```go
// shiftedChar maps US-QWERTY physical characters to their Shift equivalents.
// Native GLFW also delivers layout-resolved text via CharInput; this table is
// the fallback used to synthesize var3 (held-key char) for KeyRune presses.
var shiftedChar = map[rune]rune{
	'1': '!', '2': '@', '3': '#', '4': '$', '5': '%',
	'6': '^', '7': '&', '8': '*', '9': '(', '0': ')',
	'-': '_', '=': '+', '[': '{', ']': '}', '\\': '|',
	';': ':', '\'': '"', ',': '<', '.': '>', '/': '?',
	'`': '~',
}

// charFor synthesizes Java's keyChar from a neutral KeyPress. Non-KeyRune keys
// return 0 (the override sequence then assigns the sentinel).
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
```

- [ ] **Step 8: Run the input tests**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/client/ -run 'TestAwtFor|TestCharFor|TestHandleCharInput' 2>&1 | grep -v 'stat cache'`
Expected: PASS. (The package as a whole will not yet build until Tasks 3–5 remove the remaining Gio references; that is expected mid-refactor. Run only these tests, which compile within the package once Tasks 3–5 land. If the package fails to compile here, proceed — Task 5 closes it; do NOT commit until Step 9.)

> Note for the implementer: Tasks 2–5 form one compile unit (the `client` and
> `pixmap` packages stop compiling until all four land, because `c.Ops` and
> `OpsMu` are removed across them). Treat Tasks 2–5 as a single commit boundary:
> make all edits, then build/test/commit once at the end of Task 5. The
> per-task "run the test" steps describe the target behavior; the green bar is
> reached at Task 5 Step N.

- [ ] **Step 9: (deferred commit — see Task 5)**

---

## Task 3: `PixMap` holds a backend texture

**Files:**
- Modify: `pkg/jagex2/graphics/pixmap/pixmap.go`
- Test: `pkg/jagex2/graphics/pixmap/pixmap_test.go`

- [ ] **Step 1: Rewrite `pixmap_test.go` with a fake Backend**

```go
package pixmap

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
)

type fakeTex struct{ id int }

type fakeBackend struct {
	uploads int
	blits   int
	nextID  int
}

func (f *fakeBackend) PollEvents() []platform.Event { return nil }
func (f *fakeBackend) ShouldClose() bool             { return false }
func (f *fakeBackend) Size() (int, int)              { return 532, 789 }
func (f *fakeBackend) NewTexture(w, h int) platform.Texture {
	f.nextID++
	return &fakeTex{id: f.nextID}
}
func (f *fakeBackend) UploadTexture(t platform.Texture, rgba []byte) { f.uploads++ }
func (f *fakeBackend) BeginFrame()                                   {}
func (f *fakeBackend) Blit(t platform.Texture, x, y int)             { f.blits++ }
func (f *fakeBackend) EndFrame()                                     {}
func (f *fakeBackend) Destroy()                                      {}

func TestPixMapUploadsOnlyOnChange(t *testing.T) {
	fb := &fakeBackend{}
	platform.Active = fb
	p := NewPixMap(4, 4) // NewPixMap performs one upload via Bind/initial state

	base := fb.uploads
	p.Draw(0, 0) // first Draw: uploaded flag may already be set by NewPixMap
	p.Draw(0, 0) // unchanged -> no new upload
	if fb.uploads != base+0 && fb.uploads != base+1 {
		// allow the initial upload, but a second identical Draw must not upload
	}
	uploadsAfterStable := fb.uploads
	p.Draw(0, 0)
	if fb.uploads != uploadsAfterStable {
		t.Fatalf("unchanged Draw re-uploaded: %d -> %d", uploadsAfterStable, fb.uploads)
	}

	blitsBefore := fb.blits
	p.Data[0] = 0x123456
	p.Draw(0, 0) // changed -> exactly one upload
	if fb.uploads != uploadsAfterStable+1 {
		t.Fatalf("changed Draw should upload once: %d -> %d", uploadsAfterStable, fb.uploads)
	}
	if fb.blits != blitsBefore+1 {
		t.Fatalf("every Draw must blit once: %d -> %d", blitsBefore, fb.blits)
	}
}

func TestHashPixelsDetectsChange(t *testing.T) {
	a := []int{1, 2, 3}
	if hashPixels(a) != hashPixels([]int{1, 2, 3}) {
		t.Fatal("equal data hashed differently")
	}
	if hashPixels(a) == hashPixels([]int{1, 2, 4}) {
		t.Fatal("changed data hashed the same")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/graphics/pixmap/ 2>&1 | grep -v 'stat cache'`
Expected: FAIL — `Draw` signature mismatch / `op` import gone.

- [ ] **Step 3: Rewrite `pixmap.go`**

```go
package pixmap

import (
	"encoding/binary"
	"image"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
)

// PixMap is a CPU-side pixel buffer blitted to the screen via the active
// platform Backend. Java: PixMap (drawImage per frame). The texture is created
// once and re-uploaded in place only when the pixels change (hashPixels), so
// there is no per-frame GPU texture churn.
type PixMap struct {
	Data   []int
	Width  int
	Height int

	imgBuf   *image.RGBA      // reusable RGBA staging buffer for uploads
	tex      platform.Texture // backend texture handle (created once)
	lastHash uint64
	uploaded bool
}

// NewPixMap allocates a width*height pixel buffer and its backend texture.
func NewPixMap(width, height int) *PixMap {
	var m PixMap
	m.Width = width
	m.Height = height
	m.Data = make([]int, width*height)
	m.imgBuf = image.NewRGBA(image.Rect(0, 0, width, height))
	m.tex = platform.Active.NewTexture(width, height)
	m.Bind()
	return &m
}

// Bind sets this PixMap as the active pix2d draw target.
func (p *PixMap) Bind() {
	pix2d.Bind(p.Width, p.Data, p.Height)
}

// Draw uploads the pixels (only if changed since last Draw) and blits the
// texture with its top-left at (x, y). Java: Graphics.drawImage(image, x, y).
func (p *PixMap) Draw(x, y int) {
	h := hashPixels(p.Data)
	if !p.uploaded || h != p.lastHash {
		writePixmapPixels(p.imgBuf, p.Data)
		platform.Active.UploadTexture(p.tex, p.imgBuf.Pix)
		p.lastHash = h
		p.uploaded = true
	}
	platform.Active.Blit(p.tex, x, y)
}

// hashPixels is FNV-1a 64-bit over the packed 0x00RRGGBB pixels, used to detect
// whether the buffer changed since the last GPU upload. Allocation-free.
func hashPixels(data []int) uint64 {
	const (
		offset uint64 = 14695981039346656037 // FNV-1a 64-bit offset basis
		prime  uint64 = 1099511628211        // FNV-1a 64-bit prime
	)
	h := offset
	for _, v := range data {
		h = (h ^ uint64(uint32(v))) * prime
	}
	return h
}

// writePixmapPixels fills dst in place from packed 0x00RRGGBB ints. 0x00RRGGBB
// -> 0xRRGGBBFF via a big-endian store laying bytes [R, G, B, 0xFF].
func writePixmapPixels(dst *image.RGBA, javaPixels []int) {
	pix := dst.Pix
	for i, argb := range javaPixels {
		binary.BigEndian.PutUint32(pix[i*4:], uint32(argb)<<8|0xFF)
	}
}
```

- [ ] **Step 4: Run the pixmap test**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/graphics/pixmap/ 2>&1 | grep -v 'stat cache'`
Expected: PASS.

- [ ] **Step 5: Commit (pixmap is self-contained and compiles)**

```bash
git add pkg/jagex2/graphics/pixmap/pixmap.go pkg/jagex2/graphics/pixmap/pixmap_test.go
git commit --no-gpg-sign -m "refactor(pixmap): blit via platform.Texture, drop Gio op list"
```

---

## Task 4: `flameMu` — narrow flames hand-off lock

**Files:**
- Modify: `pkg/jagex2/client/client.go`

- [ ] **Step 1: Add the field**

In the `Client` struct, replace the now-removed `Ops`/`inputFilters` fields
(client.go:154 and 158) with:

```go
	// flameMu guards the hand-off of the title flame buffers (ImageTitle0/1)
	// between the RunFlames goroutine (writer, via DrawFlames) and the game
	// loop (reader, via ImageTitle0/1.Draw → writePixmapPixels). It replaces
	// the former global pixmap.OpsMu, which is gone with the Gio op list.
	// Java animated flames on a dedicated thread; this lock is the minimal
	// serialization that thread now requires against the single render loop.
	flameMu sync.Mutex
```

(`sync` is already imported in client.go.)

- [ ] **Step 2: Lock in `DrawFlames`**

In `DrawFlames` (client.go:1616), replace the `pixmap.OpsMu.Lock()/Unlock()`
pair (1627–1628) with:

```go
	c.flameMu.Lock()
	defer c.flameMu.Unlock()
```

- [ ] **Step 3: Lock the flame reads in `DrawTitleScreen`**

In `DrawTitleScreen` (around client.go:3430), wrap ONLY the two flame-strip
draws — `c.ImageTitle0.Draw(...)` and `c.ImageTitle1.Draw(...)` — in the lock so
their `writePixmapPixels` read of `Data` cannot race `DrawFlames`. (Other
`ImageTitle*` draws are not written by the flames thread and stay unlocked.)
After Task 5 changes the call signatures, these become:

```go
	c.flameMu.Lock()
	c.ImageTitle0.Draw(0, 0)
	c.ImageTitle1.Draw(661, 0)
	c.flameMu.Unlock()
```

> Verify against the real file: the two flame strips are `ImageTitle0` (left,
> offset 0,0) and `ImageTitle1` (right, offset 661,0) per client.go:3430–3431.
> Keep their existing offsets; only add the lock and drop the `&c.Ops` arg.

- [ ] **Step 4: (deferred commit — see Task 5)**

---

## Task 5: Unified loop + present integration (`gameshell.go`, `client.go`, close the compile)

This task removes `c.Ops`, `OpsMu`, and the Gio window/event loop, replaces them
with the single `RunShell` loop and explicit per-call presents, and rewrites the
47 `PixMap.Draw(&c.Ops, ...)` call sites. It closes the compile opened in Task 2.

**Files:**
- Modify: `pkg/jagex2/client/gameshell.go`, `pkg/jagex2/client/client.go`

- [ ] **Step 1: Remove `c.Ops` field and Gio imports from `client.go`**

Delete the `Ops op.Ops` field (client.go:154) and the `inputFilters` field
(158) — both replaced by `flameMu` in Task 4. Remove the imports
`gioui.org/io/event` and `gioui.org/op` from the client.go import block.

- [ ] **Step 2: Rewrite `Draw()` to drop `OpsMu`/`Reset`**

`Draw()` (client.go:2371) becomes a pure per-frame draw; framing is done by the
loop (Step 5). Replace its body:

```go
func (c *Client) Draw() {
	if c.ErrorStarted || c.ErrorLoading || c.ErrorHost {
		c.DrawError()
		return
	}
	if c.InGame {
		c.DrawGame()
	} else {
		c.DrawTitleScreen()
	}
	c.DragCycles = 0
}
```

- [ ] **Step 3: Rewrite the six out-of-band repaint sites to present explicitly**

Each formerly wrote into `c.Ops` under `OpsMu` and relied on the Gio present
goroutine. Now each must frame itself. Define a small helper in `gameshell.go`:

```go
// present issues one full frame: BeginFrame, the supplied draw, EndFrame. Used
// by out-of-band repaints (loading/connection messages) that must show
// immediately, before a blocking operation, while the main loop is not the
// caller. Runs on the loop goroutine (all six call sites do).
func (c *Client) present(draw func()) {
	platform.Active.BeginFrame()
	draw()
	platform.Active.EndFrame()
}
```

Then at each site replace the `pixmap.OpsMu.Lock()/...Draw(&c.Ops, ...)/Unlock()`
trio:

- `DrawProgressGameShell` (gameshell.go:274) — change the final
  `c.OverlayPixMap.Draw(&c.Ops, 0, 0)` to be wrapped by the loop's framing; since
  `DrawProgress` is itself an out-of-band repaint, wrap its overlay draw:
  ```go
  c.present(func() { c.OverlayPixMap.Draw(0, 0) })
  ```
- `DrawProgress` (client.go:10406–10409 region) — this function holds
  `OpsMu.Lock()/defer Unlock()` for its whole body and ends by drawing the
  progress PixMap(s) into `c.Ops`. Mechanical transform: remove the
  `pixmap.OpsMu.Lock()` and its `defer pixmap.OpsMu.Unlock()`; wrap the function's
  existing draw body in `c.present(func() { ... })` where the body is the same
  statements as before with each `PixMap.Draw(&c.Ops, x, y)` changed to
  `PixMap.Draw(x, y)` (the `&c.Ops`-drop is the blanket sed in Step 4; the only
  manual edit here is removing the lock and adding the `c.present(func(){ })`
  wrapper around the draw statements).
- `TryReconnect` (client.go:8077): replace
  ```go
  pixmap.OpsMu.Lock()
  c.AreaViewport.Draw(&c.Ops, 8, 11)
  pixmap.OpsMu.Unlock()
  ```
  with
  ```go
  c.present(func() { c.AreaViewport.Draw(8, 11) })
  ```
- `Read` (client.go:9257, 9383, 9443): each identical shape — replace with
  `c.present(func() { c.AreaViewport.Draw(8, 11) })`.
- `LoginFunc` (client.go:6413): replace
  ```go
  pixmap.OpsMu.Lock()
  c.DrawTitleScreen()
  pixmap.OpsMu.Unlock()
  ```
  with
  ```go
  c.present(func() { c.DrawTitleScreen() })
  ```

- [ ] **Step 4: Rewrite all `PixMap.Draw(&c.Ops, x, y)` call sites → `Draw(x, y)`**

There are 47 sites across `client.go` and `gameshell.go`. Mechanically drop the
first argument. Find them all and confirm zero remain:

```bash
cd .
grep -rn "\.Draw(&c\.Ops" pkg/jagex2/client/   # expect: (none) after editing
```

Use a guarded sed for the mechanical rewrite, then eyeball the diff:

```bash
grep -rl "\.Draw(&c\.Ops" pkg/jagex2/client/ | while read -r f; do
  sed -i -E 's/\.Draw\(&c\.Ops, /.Draw(/g' "$f"
done
grep -rn "\.Draw(&c\.Ops" pkg/jagex2/client/ || echo "all rewritten"
```

(The two flame-strip draws in Task 4 Step 3 already use the new signature; the
sed leaves them correct.)

- [ ] **Step 5: Replace `InitApplication`/`draw`/`RunGameShell` with `RunShell`**

In `gameshell.go`, delete `InitApplication` (23–57), `draw` (59–149), and fold
the pacing loop of `RunGameShell` into a single `RunShell` that polls input and
presents each frame. `Run()` (wherever it currently calls `RunGameShell`) calls
`RunShell`. Replace `RunGameShell` (151–218) with:

```go
// RunShell is the single game loop: poll input, run catch-up logic ticks, draw,
// present, sleep. Faithful to Java GameShell.run() / the TS client (no
// requestAnimationFrame). Runs on the loop goroutine established by
// platform.Main (the LockOSThread'd main goroutine on native).
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

		platform.Active.BeginFrame()
		c.Draw()
		platform.Active.EndFrame()
	}
	if c.State == -1 || platform.Active.ShouldClose() {
		c.Shutdown()
	}
}
```

The backend closure in `main.go` (Task 8) calls `c.RunShell()` directly, so the
old entry points are no longer in the path. Find every caller and remove the dead
wrappers:

```bash
grep -rn "RunGameShell\|InitApplication\|func (c \*Client) Run(" pkg/jagex2/client/ cmd/
```

Delete `InitApplication` (already done above). If `Run()` only delegated to
`RunGameShell` (check its body — it should, per the original
`InitApplication`→`Run`→`RunGameShell` chain), delete `Run()` too. If `Run()`
does additional setup, move that setup to the top of `RunShell` and then delete
`Run()`. The sole loop entry point after this task is `c.RunShell()`.

- [ ] **Step 6: Build the native-neutral packages, run all platform-neutral tests**

The `client` package now references `platform.Active` (set at runtime) but does
not import any backend, so it compiles target-independently. Verify via the wasm
build (which excludes the cgo backend) and the test suite:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./pkg/... 2>&1 | grep -v 'stat cache'; echo "wasm-pkgs ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/client/ ./pkg/jagex2/graphics/pixmap/ ./pkg/jagex2/platform/ 2>&1 | grep -v 'stat cache'; echo "test ${PIPESTATUS[0]}"
```

Expected: both exit 0. (`./cmd/client` will not build yet — main.go still imports
Gio; fixed in Task 8. `./pkg/...` excludes cmd.)

- [ ] **Step 7: Commit Tasks 2, 4, 5 together**

```bash
git add pkg/jagex2/client/gameshell.go pkg/jagex2/client/client.go \
        pkg/jagex2/client/keycharfor_test.go pkg/jagex2/client/handleeditevent_test.go
git commit --no-gpg-sign -m "refactor(client): neutral input + unified RunShell loop; remove OpsMu/op.Ops, add flameMu"
```

---

## Task 6: Native GLFW + go-gl backend

Not unit-testable without a display; verification is the host build + a manual
run (Task 10). Provide the complete implementation.

**Files:**
- Create: `pkg/jagex2/platform/backend_glfw.go` (`//go:build !js`)
- Modify: `go.mod` (add go-gl requires)

- [ ] **Step 1: Add dependencies**

```bash
cd .
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go get github.com/go-gl/glfw/v3.3/glfw@latest
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go get github.com/go-gl/gl/v2.1/gl@latest
```

(These only update go.mod/go.sum; the cgo compile happens on the host.)

- [ ] **Step 2: Implement the backend**

```go
//go:build !js

package platform

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type glTexture struct {
	id   uint32
	w, h int
}

type glfwBackend struct {
	win    *glfw.Window
	w, h   int
	prog   uint32
	vbo    uint32
	events []Event
}

const vertexShaderSrc = `
attribute vec2 aPos;     // pixel coords
attribute vec2 aUV;
uniform vec2 uScreen;    // viewport size in pixels
varying vec2 vUV;
void main() {
	// pixel space -> clip space (flip Y so (0,0) is top-left)
	vec2 ndc = vec2(aPos.x / uScreen.x * 2.0 - 1.0,
	                1.0 - aPos.y / uScreen.y * 2.0);
	gl_Position = vec4(ndc, 0.0, 1.0);
	vUV = aUV;
}` + "\x00"

const fragmentShaderSrc = `
#ifdef GL_ES
precision mediump float;
#endif
varying vec2 vUV;
uniform sampler2D uTex;
void main() { gl_FragColor = texture2D(uTex, vUV); }
` + "\x00"

// newGLFWBackend creates the window and GL context. MUST be called on the
// LockOSThread'd main goroutine (see run_native.go).
func newGLFWBackend(width, height int, title string) *glfwBackend {
	if err := glfw.Init(); err != nil {
		panic(fmt.Sprintf("glfw init: %v", err))
	}
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.Resizable, glfw.False)
	win, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		panic(fmt.Sprintf("glfw window: %v", err))
	}
	win.MakeContextCurrent()
	glfw.SwapInterval(0) // vsync OFF — the RunShell sleep pacing governs fps
	if err := gl.Init(); err != nil {
		panic(fmt.Sprintf("gl init: %v", err))
	}
	b := &glfwBackend{win: win, w: width, h: height}
	b.prog = buildProgram()
	gl.GenBuffers(1, &b.vbo)
	b.installCallbacks()
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.BLEND)
	gl.ClearColor(0, 0, 0, 1)
	return b
}

func buildProgram() uint32 {
	vs := compileShader(vertexShaderSrc, gl.VERTEX_SHADER)
	fs := compileShader(fragmentShaderSrc, gl.FRAGMENT_SHADER)
	p := gl.CreateProgram()
	gl.AttachShader(p, vs)
	gl.AttachShader(p, fs)
	gl.LinkProgram(p)
	var ok int32
	gl.GetProgramiv(p, gl.LINK_STATUS, &ok)
	if ok == gl.FALSE {
		var n int32
		gl.GetProgramiv(p, gl.INFO_LOG_LENGTH, &n)
		log := strings.Repeat("\x00", int(n+1))
		gl.GetProgramInfoLog(p, n, nil, gl.Str(log))
		panic("link: " + log)
	}
	gl.DeleteShader(vs)
	gl.DeleteShader(fs)
	return p
}

func compileShader(src string, kind uint32) uint32 {
	s := gl.CreateShader(kind)
	csrc, free := gl.Strs(src)
	gl.ShaderSource(s, 1, csrc, nil)
	free()
	gl.CompileShader(s)
	var ok int32
	gl.GetShaderiv(s, gl.COMPILE_STATUS, &ok)
	if ok == gl.FALSE {
		var n int32
		gl.GetShaderiv(s, gl.INFO_LOG_LENGTH, &n)
		log := strings.Repeat("\x00", int(n+1))
		gl.GetShaderInfoLog(s, n, nil, gl.Str(log))
		panic("shader compile: " + log)
	}
	return s
}

func (b *glfwBackend) NewTexture(w, h int) Texture {
	var id uint32
	gl.GenTextures(1, &id)
	gl.BindTexture(gl.TEXTURE_2D, id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(w), int32(h), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	return &glTexture{id: id, w: w, h: h}
}

func (b *glfwBackend) UploadTexture(t Texture, rgba []byte) {
	tex := t.(*glTexture)
	gl.BindTexture(gl.TEXTURE_2D, tex.id)
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, int32(tex.w), int32(tex.h),
		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba))
}

func (b *glfwBackend) BeginFrame() {
	w, h := b.win.GetFramebufferSize()
	b.w, b.h = w, h
	gl.Viewport(0, 0, int32(w), int32(h))
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.UseProgram(b.prog)
	gl.Uniform2f(gl.GetUniformLocation(b.prog, gl.Str("uScreen\x00")), float32(w), float32(h))
}

func (b *glfwBackend) Blit(t Texture, x, y int) {
	tex := t.(*glTexture)
	x0, y0 := float32(x), float32(y)
	x1, y1 := float32(x+tex.w), float32(y+tex.h)
	// two triangles: pos.xy, uv.xy
	verts := []float32{
		x0, y0, 0, 0,
		x1, y0, 1, 0,
		x0, y1, 0, 1,
		x1, y0, 1, 0,
		x1, y1, 1, 1,
		x0, y1, 0, 1,
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, b.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.DYNAMIC_DRAW)
	aPos := uint32(gl.GetAttribLocation(b.prog, gl.Str("aPos\x00")))
	aUV := uint32(gl.GetAttribLocation(b.prog, gl.Str("aUV\x00")))
	gl.EnableVertexAttribArray(aPos)
	gl.VertexAttribPointerWithOffset(aPos, 2, gl.FLOAT, false, 16, 0)
	gl.EnableVertexAttribArray(aUV)
	gl.VertexAttribPointerWithOffset(aUV, 2, gl.FLOAT, false, 16, 8)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, tex.id)
	gl.Uniform1i(gl.GetUniformLocation(b.prog, gl.Str("uTex\x00")), 0)
	gl.DrawArrays(gl.TRIANGLES, 0, 6)
}

func (b *glfwBackend) EndFrame() {
	b.win.SwapBuffers()
}

// (Threading note: this backend must be constructed and driven on the
// LockOSThread'd main goroutine — see run_native.go. No runtime import here.)

func (b *glfwBackend) PollEvents() []Event {
	b.events = b.events[:0]
	glfw.PollEvents() // fires the callbacks below, appending to b.events
	out := make([]Event, len(b.events))
	copy(out, b.events)
	return out
}

func (b *glfwBackend) ShouldClose() bool { return b.win.ShouldClose() }
func (b *glfwBackend) Size() (int, int)  { return b.w, b.h }

func (b *glfwBackend) Destroy() {
	b.win.Destroy()
	glfw.Terminate()
}

func (b *glfwBackend) installCallbacks() {
	b.win.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		b.events = append(b.events, MouseMove{X: int(x), Y: int(y)})
	})
	b.win.SetMouseButtonCallback(func(_ *glfw.Window, btn glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		x, y := b.win.GetCursorPos()
		button := 1
		if btn == glfw.MouseButtonRight {
			button = 2
		}
		b.events = append(b.events, MouseButton{X: int(x), Y: int(y), Button: button, Pressed: action == glfw.Press})
	})
	b.win.SetCursorEnterCallback(func(_ *glfw.Window, entered bool) {
		b.events = append(b.events, MouseCross{Entered: entered})
	})
	b.win.SetFocusCallback(func(_ *glfw.Window, focused bool) {
		b.events = append(b.events, FocusChange{Gained: focused})
	})
	b.win.SetCharCallback(func(_ *glfw.Window, r rune) {
		b.events = append(b.events, CharInput{R: r})
	})
	b.win.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Repeat {
			return // Java had no auto-repeat for these sentinels
		}
		k, r := glfwKeyToNeutral(key)
		if k == KeyNone {
			return
		}
		b.events = append(b.events, KeyPress{
			Key:  k,
			Rune: r,
			Mods: glfwMods(mods),
			Down: action == glfw.Press,
		})
	})
}

func glfwMods(m glfw.ModifierKey) Mod {
	var out Mod
	if m&glfw.ModShift != 0 {
		out |= ModShift
	}
	if m&glfw.ModControl != 0 {
		out |= ModCtrl
	}
	if m&glfw.ModAlt != 0 {
		out |= ModAlt
	}
	if m&glfw.ModSuper != 0 {
		out |= ModSuper
	}
	return out
}

// glfwKeyToNeutral maps a GLFW key to a neutral Key. Letters/digits map to
// KeyRune with the rune set (charFor resolves Shift). Returns KeyNone for keys
// the game ignores.
func glfwKeyToNeutral(key glfw.Key) (Key, rune) {
	switch {
	case key >= glfw.KeyA && key <= glfw.KeyZ:
		return KeyRune, rune('A' + (key - glfw.KeyA))
	case key >= glfw.Key0 && key <= glfw.Key9:
		return KeyRune, rune('0' + (key - glfw.Key0))
	}
	switch key {
	case glfw.KeyLeft:
		return KeyLeft, 0
	case glfw.KeyRight:
		return KeyRight, 0
	case glfw.KeyUp:
		return KeyUp, 0
	case glfw.KeyDown:
		return KeyDown, 0
	case glfw.KeyEnter, glfw.KeyKPEnter:
		return KeyReturn, 0
	case glfw.KeyEscape:
		return KeyEscape, 0
	case glfw.KeyHome:
		return KeyHome, 0
	case glfw.KeyEnd:
		return KeyEnd, 0
	case glfw.KeyBackspace:
		return KeyBackspace, 0
	case glfw.KeyDelete:
		return KeyDelete, 0
	case glfw.KeyPageUp:
		return KeyPageUp, 0
	case glfw.KeyPageDown:
		return KeyPageDown, 0
	case glfw.KeyTab:
		return KeyTab, 0
	case glfw.KeyLeftControl, glfw.KeyRightControl:
		return KeyCtrl, 0
	case glfw.KeyLeftShift, glfw.KeyRightShift:
		return KeyShift, 0
	case glfw.KeyLeftAlt, glfw.KeyRightAlt:
		return KeyAlt, 0
	case glfw.KeyF1:
		return KeyF1, 0
	case glfw.KeyF2:
		return KeyF2, 0
	case glfw.KeyF3:
		return KeyF3, 0
	case glfw.KeyF4:
		return KeyF4, 0
	case glfw.KeyF5:
		return KeyF5, 0
	case glfw.KeyF6:
		return KeyF6, 0
	case glfw.KeyF7:
		return KeyF7, 0
	case glfw.KeyF8:
		return KeyF8, 0
	case glfw.KeyF9:
		return KeyF9, 0
	case glfw.KeyF10:
		return KeyF10, 0
	case glfw.KeyF11:
		return KeyF11, 0
	case glfw.KeyF12:
		return KeyF12, 0
	}
	return KeyNone, 0
}

```

- [ ] **Step 3: Verify (host only — cannot compile in sandbox)**

On the host:
```bash
CGO_ENABLED=1 go build ./pkg/jagex2/platform/
```
Expected: builds. (In the sandbox this step is skipped; the wasm build in Task 7
keeps the sandbox green.)

- [ ] **Step 4: Commit**

```bash
git add pkg/jagex2/platform/backend_glfw.go go.mod go.sum
git commit --no-gpg-sign -m "feat(platform): native GLFW + go-gl GLES2 backend"
```

---

## Task 7: Stub `js` backend + run glue

**Files:**
- Create: `pkg/jagex2/platform/backend_js_stub.go` (`//go:build js`)
- Create: `pkg/jagex2/platform/run_native.go` (`//go:build !js`)
- Create: `pkg/jagex2/platform/run_js.go` (`//go:build js`)

- [ ] **Step 1: Stub js backend (keeps `GOOS=js` compiling until Plan 2)**

```go
//go:build js

package platform

// jsStub is a compile-only placeholder so GOOS=js builds during Plan 1. Plan 2
// replaces this file with the real syscall/js + WebGL backend.
type jsStub struct{}

func newJSBackend(width, height int, title string) Backend {
	panic("platform(js): WebGL backend not yet implemented (Plan 2)")
}

func (jsStub) PollEvents() []Event              { panic("stub") }
func (jsStub) ShouldClose() bool                { panic("stub") }
func (jsStub) Size() (int, int)                 { panic("stub") }
func (jsStub) NewTexture(w, h int) Texture      { panic("stub") }
func (jsStub) UploadTexture(t Texture, b []byte){ panic("stub") }
func (jsStub) BeginFrame()                      { panic("stub") }
func (jsStub) Blit(t Texture, x, y int)         { panic("stub") }
func (jsStub) EndFrame()                        { panic("stub") }
func (jsStub) Destroy()                         { panic("stub") }
```

- [ ] **Step 2: Native run glue**

```go
//go:build !js

package platform

import "runtime"

// Main builds the native backend and runs loop on the main OS thread. GLFW and
// OpenGL are thread-affine, so the window, all GL calls, and the game loop must
// share the locked main goroutine. Blocks until loop returns.
func Main(width, height int, title string, loop func()) {
	runtime.LockOSThread()
	Active = newGLFWBackend(width, height, title)
	loop()
}
```

- [ ] **Step 3: Browser run glue (used by Plan 2; compiles now against the stub)**

```go
//go:build js

package platform

// Main builds the browser backend and runs loop in a goroutine. The loop yields
// to the JS event loop via time.Sleep (Go's wasm runtime parks the goroutine on
// timers), so the page composites and DOM input fires — the TS-client model, no
// requestAnimationFrame. main() never returns (select{} blocks the program).
func Main(width, height int, title string, loop func()) {
	Active = newJSBackend(width, height, title)
	go loop()
	select {}
}
```

- [ ] **Step 4: Verify wasm build of the platform package**

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./pkg/jagex2/platform/ 2>&1 | grep -v 'stat cache'; echo "exit ${PIPESTATUS[0]}"
```
Expected: exit 0.

- [ ] **Step 5: Commit**

```bash
git add pkg/jagex2/platform/backend_js_stub.go pkg/jagex2/platform/run_native.go pkg/jagex2/platform/run_js.go
git commit --no-gpg-sign -m "feat(platform): native/js run glue + compile-only js stub backend"
```

---

## Task 8: `main.go` — wire `platform.Main`

**Files:**
- Modify: `cmd/client/main.go`

- [ ] **Step 1: Remove Gio, call `platform.Main`**

Remove `"gioui.org/app"` from imports and add
`"github.com/zsrv/goscape-client/pkg/jagex2/platform"`. Replace the client
goroutine + `app.Main()` tail (main.go:88–123) with: keep the signlink/audio
goroutines, then drive the shell via `platform.Main` on the main goroutine.

```go
	var wg sync.WaitGroup
	wg.Go(func() { signlink.StartPriv() })
	wg.Go(func() {
		if client.LowMemory {
			audio.DisableForLowMemory()
			return
		}
		audio.Start()
	})

	// platform.Main owns the threading model: native locks the OS thread,
	// builds the GLFW backend, and runs the loop on the main goroutine; wasm
	// builds the WebGL backend and runs the loop in a goroutine, blocking on
	// select{}. The game client is created inside the loop closure so it exists
	// only once a backend is Active (NewClient allocates PixMaps → textures).
	platform.Main(532, 789, "Jagex", func() {
		c := client.NewClient()
		c.RunShell()
		os.Exit(0)
	})
```

> Note: `client.NewClient()` must allocate PixMaps only after `platform.Active`
> is set. `platform.Main` sets `Active` before invoking the closure, so creating
> the client inside the closure is required. If `NewClient` is currently called
> elsewhere eagerly, confirm there is no package-level PixMap allocation at
> import time (grep `NewPixMap` for `var ... = NewPixMap`); there is none today.

- [ ] **Step 2: Build both targets**

```bash
cd .
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm ${PIPESTATUS[0]}"
# native build is host-only (cgo); in the sandbox confirm only that the js build is green.
```
Expected: wasm exit 0. On the host also run `CGO_ENABLED=1 go build ./cmd/client` → exit 0.

- [ ] **Step 3: Commit**

```bash
git add cmd/client/main.go
git commit --no-gpg-sign -m "refactor(cmd): drive the client via platform.Main; drop gioui.org/app"
```

---

## Task 9: CI — native build job

**Files:**
- Modify: `.github/workflows/ci-gate.yml`

- [ ] **Step 1: Add a native-build job**

Add a job that installs GL/X11/Wayland headers and compiles the native target
(compile-only — no display needed). Use the repo's existing Go setup step style;
the new job body:

```yaml
  native-build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26.x'
      - name: Install GL/X11/Wayland headers
        run: |
          sudo apt-get update
          sudo apt-get install -y \
            libgl1-mesa-dev libx11-dev libxcursor-dev libxrandr-dev \
            libxinerama-dev libxi-dev libwayland-dev libxkbcommon-dev
      - name: Build native (cgo)
        run: CGO_ENABLED=1 go build ./...
```

Keep the existing wasm-build, test, and lint jobs unchanged.

- [ ] **Step 2: Validate the YAML locally**

```bash
cd .
python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci-gate.yml')); print('yaml ok')"
```
Expected: `yaml ok`.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci-gate.yml
git commit --no-gpg-sign -m "ci: build the cgo native target with GL/X11 headers"
```

---

## Task 10: Full verification + native run handoff

- [ ] **Step 1: Sandbox gates (wasm build + neutral tests + vet)**

```bash
cd .
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/... 2>&1 | grep -vE 'stat cache|no test files|^ok|cached'; echo "test ${PIPESTATUS[0]} (empty=pass)"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go vet ./pkg/jagex2/client/ ./pkg/jagex2/graphics/pixmap/ ./pkg/jagex2/platform/ 2>&1 | grep -v 'stat cache'; echo "vet ${PIPESTATUS[0]}"
```
Expected: wasm 0, tests pass, vet 0.

- [ ] **Step 2: Confirm no Gio remains on the native path**

```bash
grep -rn "gioui.org" pkg/jagex2/client/ pkg/jagex2/graphics/pixmap/ cmd/client/ || echo "client/pixmap/cmd are Gio-free"
```
Expected: Gio-free. (The vendored `third_party/gioui.org` and its `replace`
remain until Plan 2 — that is intended; nothing imports them on the native path
now, but the browser still uses the stub.)

- [ ] **Step 3: Host gate — native build + run (handoff to user)**

On the host (cannot run in sandbox):
```bash
go build ./... && go run ./cmd/client 10 0 highmem members
```
Acceptance: window opens at 532×789; title screen renders; **flames animate**
(proves the flameMu hand-off + change-detection re-upload); mouse + keyboard
work (login, typing); no stale frames; memory flat. `make lint` passes.

- [ ] **Step 4: Mark Plan 1 complete; Plan 2 (browser WebGL + Gio deletion) is next.**

---

## Self-Review notes (run before execution)

- Tasks 2/4/5 share a compile unit; they commit together at Task 5 Step 7. The
  per-task green-bar steps describe target behavior, not an independently
  buildable state — this is called out in Task 2 Step 8.
- Type/name consistency: `RunShell`, `platform.Active`, `platform.Main`,
  `awtFor`, `charFor`, `handleCharInput`, `handleMouseButton/Move/Cross`,
  `flameMu`, `PixMap.Draw(x, y)` are used consistently across tasks.
- The 47 `PixMap.Draw(&c.Ops, …)` rewrite (Task 5 Step 4) includes the two
  flame-strip draws that Task 4 Step 3 also touches; the sed leaves them with
  the correct new signature.

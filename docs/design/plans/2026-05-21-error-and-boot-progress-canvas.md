# Error & Boot-Progress Canvas Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire visible output for `DrawError` (`client.go:8242`) and `DrawProgressGameShell` (`gameshell.go:241`) so the fatal-error screen and boot-time loading bar render through Gio, closing follow-up item 2 from `.claude/resume/2026-05-21-all-phases-done-pick-followup.md`.

**Architecture:** Add one shared `*pixmap.PixMap` overlay (`ScreenWidth × ScreenHeight`) to `Client`. Both functions bind that buffer, use `pix2d` for rectangles, render text (via existing `pixfont` for `DrawError`, via a new `bootfont` package backed by `basicfont.Face7x13` for `DrawProgressGameShell`), then composite via `OverlayPixMap.Draw(&c.Ops, 0, 0)`.

**Tech Stack:** Go 1.26, Gio (`gioui.org`), `golang.org/x/image/font/basicfont` (already a transitive dep — promoted to direct), existing internal packages `pix2d`, `pixmap`, `pixfont`.

---

## File Structure

**Create:**
- `pkg/jagex2/graphics/bootfont/bootfont.go` — `basicfont.Face7x13` wrapper. Exports `DrawString`, `StringWidth`, `Height`.
- `pkg/jagex2/graphics/bootfont/bootfont_test.go` — unit tests for the wrapper.
- `docs/superpowers/plans/2026-05-21-error-and-boot-progress-canvas.md` — this file (already created).

**Modify:**
- `pkg/jagex2/client/client.go` — add `OverlayPixMap *pixmap.PixMap` field on `Client`, add `ensureOverlay` helper, port `DrawError` body.
- `pkg/jagex2/client/gameshell.go` — port `DrawProgressGameShell` body.
- `pkg/jagex2/client/client_test.go` (create if missing) — smoke test for `DrawProgressGameShell`.
- `go.mod` — promote `golang.org/x/image` from `// indirect` to direct via `go mod tidy`.

**Read-only references:**
- `pkg/jagex2/graphics/pix2d/pix2d.go:57,83` — confirm `FillRect(y, x, colour, w, h)` and `DrawRect(x, colour, h, y, w)` signatures while writing draw calls.
- `pkg/jagex2/graphics/pixmap/pixmap.go` — pattern for `Bind` + `Draw` lifecycle.
- `pkg/jagex2/client/client.go:10111-10154` — existing working `DrawProgress` is the visual template; copy its shape.
- `$HOME/Code/github.com/LostCityRS/Client-Java/src/main/java/jagex2/client/GameShell.java:529-560` — Java `drawProgress`, authoritative reference.
- `$HOME/Code/github.com/LostCityRS/Client-Java/src/main/java/jagex2/client/client.java:8727-8781` — Java `drawError`, authoritative reference for the three branches.

---

## Task 1: Add `OverlayPixMap` field and `ensureOverlay` helper

**Files:**
- Modify: `pkg/jagex2/client/client.go` (near line 119-125 where `ScreenWidth`, `Refresh` live)

- [ ] **Step 1.1: Add `OverlayPixMap` field to the `Client` struct**

Locate the struct field block around `client.go:119` containing `ScreenWidth int`, `Refresh bool`, etc. Add this field alongside them (alphabetical ordering not required — match local convention; place adjacent to other PixMap fields if any, otherwise next to `Refresh`):

```go
	OverlayPixMap   *pixmap.PixMap
```

If `pixmap` isn't already imported in `client.go`, confirm: `goscape-client/pkg/jagex2/graphics/pixmap` should already be in the import block (used elsewhere). No new import expected.

- [ ] **Step 1.2: Add the `ensureOverlay` helper at the bottom of `client.go`**

Append to `client.go`:

```go
// ensureOverlay lazily allocates the fullscreen overlay PixMap used by
// DrawError and DrawProgressGameShell. Lazy because ScreenWidth/Height
// are set by GameShell.SetCanvasSize during InitApplication, which runs
// after NewClient. If the screen size changed since the last allocation
// (currently unreachable but cheap to guard), reallocate.
func (c *Client) ensureOverlay() {
	if c.OverlayPixMap == nil ||
		c.OverlayPixMap.Width != c.ScreenWidth ||
		c.OverlayPixMap.Height != c.ScreenHeight {
		c.OverlayPixMap = pixmap.NewPixMap(c.ScreenWidth, c.ScreenHeight)
	}
}
```

- [ ] **Step 1.3: Verify the package still builds**

Run:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go build ./...
```

Expected: clean build, no errors.

- [ ] **Step 1.4: Commit**

```bash
git add pkg/jagex2/client/client.go
git commit --no-gpg-sign -m "$(cat <<'EOF'
port(client): add OverlayPixMap field and ensureOverlay helper

Backs the upcoming DrawError and DrawProgressGameShell rendering.
Lazy allocation because ScreenWidth/Height are set by InitApplication
after NewClient runs.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Create `bootfont` package skeleton with `Height` and `StringWidth`

**Files:**
- Create: `pkg/jagex2/graphics/bootfont/bootfont.go`
- Create: `pkg/jagex2/graphics/bootfont/bootfont_test.go`

- [ ] **Step 2.1: Write the failing tests**

Create `pkg/jagex2/graphics/bootfont/bootfont_test.go`:

```go
package bootfont

import "testing"

func TestHeight(t *testing.T) {
	if got := Height(); got != 13 {
		t.Fatalf("Height() = %d, want 13", got)
	}
}

func TestStringWidthEmpty(t *testing.T) {
	if got := StringWidth(""); got != 0 {
		t.Fatalf("StringWidth(\"\") = %d, want 0", got)
	}
}

func TestStringWidthASCII(t *testing.T) {
	// basicfont.Face7x13 advances 7 pixels per glyph.
	if got := StringWidth("hello"); got != 35 {
		t.Fatalf("StringWidth(\"hello\") = %d, want 35", got)
	}
}
```

- [ ] **Step 2.2: Run tests to verify they fail**

Run:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/jagex2/graphics/bootfont/...
```

Expected: FAIL with "no Go files" or "undefined: Height / StringWidth".

- [ ] **Step 2.3: Implement the minimal package**

Create `pkg/jagex2/graphics/bootfont/bootfont.go`:

```go
// Package bootfont renders text during the boot phase before c.JagTitle
// (and thus the RuneScape pixel fonts in pixfont) has been loaded. It
// wraps golang.org/x/image/font/basicfont.Face7x13, a monospace 7x13
// font shipped in x/image. Used exclusively by DrawProgressGameShell.
package bootfont

import (
	"unicode/utf8"

	"golang.org/x/image/font/basicfont"
)

// Height returns the font's inter-line height in pixels.
func Height() int {
	return basicfont.Face7x13.Height
}

// StringWidth returns the rendered pixel width of s, assuming
// basicfont.Face7x13's fixed 7-pixel advance per glyph.
func StringWidth(s string) int {
	return utf8.RuneCountInString(s) * basicfont.Face7x13.Advance
}
```

- [ ] **Step 2.4: Run tests to verify they pass**

Run:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/jagex2/graphics/bootfont/...
```

Expected: PASS — 3 tests.

If `go test` complains about missing module entries for `golang.org/x/image`, run:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go mod tidy
```

then re-run the tests.

- [ ] **Step 2.5: Commit**

```bash
git add pkg/jagex2/graphics/bootfont/ go.mod go.sum
git commit --no-gpg-sign -m "$(cat <<'EOF'
port(bootfont): add basicfont-backed text renderer for pre-JagTitle UI

Provides Height() and StringWidth() over basicfont.Face7x13. Used by
DrawProgressGameShell, where the RuneScape pixfont assets are not yet
loaded.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Implement `bootfont.DrawString`

**Files:**
- Modify: `pkg/jagex2/graphics/bootfont/bootfont.go`
- Modify: `pkg/jagex2/graphics/bootfont/bootfont_test.go`

- [ ] **Step 3.1: Write the failing test**

Append to `pkg/jagex2/graphics/bootfont/bootfont_test.go`:

```go
import (
	"testing"

	"goscape-client/pkg/jagex2/graphics/pixmap"
)

func TestDrawStringWritesPixels(t *testing.T) {
	p := pixmap.NewPixMap(200, 50)
	// Pre-fill with sentinel so we can detect any writes from DrawString.
	for i := range p.Data {
		p.Data[i] = 0x0000FF // blue
	}
	DrawString(p, 10, 20, 0xFFFFFF, "A")
	// Scan the bounding box for the "A" glyph. With Face7x13:
	// Advance=7, Width=6, Height=13, Ascent=11.
	// At dot=(10,20), glyph occupies x in [10..15], y in [20-11..20+2]=[9..22].
	written := 0
	for y := 9; y <= 22; y++ {
		for x := 10; x <= 15; x++ {
			if p.Data[y*p.Width+x] == 0xFFFFFF {
				written++
			}
		}
	}
	if written == 0 {
		t.Fatalf("DrawString wrote no white pixels in glyph bounding box; expected at least one")
	}
}
```

Replace the existing single `import "testing"` line in the file with the consolidated form above (one `import` block).

- [ ] **Step 3.2: Run test to verify it fails**

Run:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/jagex2/graphics/bootfont/...
```

Expected: FAIL with "undefined: DrawString".

- [ ] **Step 3.3: Implement `DrawString`**

Add to `pkg/jagex2/graphics/bootfont/bootfont.go`. Extend the import block:

```go
import (
	"image"
	"image/color"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"goscape-client/pkg/jagex2/graphics/pixmap"
)
```

Append:

```go
// DrawString rasterizes s onto p starting with the baseline at (x, y),
// in the given 0x00RRGGBB color. Matches AWT Graphics.drawString
// semantics: y is the glyph baseline, not the top of the box.
//
// The renderer rasterizes the glyphs into a temp image.NRGBA sized to
// the message bounding box, then copies set pixels into p.Data as
// 0x00RRGGBB ints. Runs at most a few times per boot frame, so the
// allocation cost is not in the hot path.
func DrawString(p *pixmap.PixMap, x, y, hexColor int, s string) {
	if s == "" {
		return
	}

	width := StringWidth(s)
	height := basicfont.Face7x13.Ascent + basicfont.Face7x13.Descent
	if width <= 0 || height <= 0 {
		return
	}

	src := image.NewNRGBA(image.Rect(0, 0, width, height))
	drawer := font.Drawer{
		Dst: src,
		Src: image.NewUniform(color.NRGBA{
			R: uint8(hexColor >> 16),
			G: uint8(hexColor >> 8),
			B: uint8(hexColor),
			A: 0xFF,
		}),
		Face: basicfont.Face7x13,
		Dot: fixed.Point26_6{
			X: fixed.I(0),
			Y: fixed.I(basicfont.Face7x13.Ascent),
		},
	}
	drawer.DrawString(s)

	// Copy non-transparent pixels into the PixMap, offset so that the
	// drawer-baseline (Ascent rows down in src) lands at y in p.
	topLeftY := y - basicfont.Face7x13.Ascent
	for srcY := range height {
		dstY := topLeftY + srcY
		if dstY < 0 || dstY >= p.Height {
			continue
		}
		for srcX := range width {
			dstX := x + srcX
			if dstX < 0 || dstX >= p.Width {
				continue
			}
			off := (srcY*width + srcX) * 4
			if src.Pix[off+3] == 0 {
				continue
			}
			p.Data[dstY*p.Width+dstX] = hexColor
		}
	}
}

```

`unicode/utf8` is still used inside `StringWidth` from Task 2, so the import stays valid — no shim needed.

- [ ] **Step 3.4: Run tests to verify they pass**

Run:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/jagex2/graphics/bootfont/...
```

Expected: PASS — 4 tests including `TestDrawStringWritesPixels`.

- [ ] **Step 3.5: Vet**

Run:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go vet ./pkg/jagex2/graphics/bootfont/...
```

Expected: no output (clean).

- [ ] **Step 3.6: Commit**

```bash
git add pkg/jagex2/graphics/bootfont/
git commit --no-gpg-sign -m "$(cat <<'EOF'
port(bootfont): implement DrawString rasterizer onto PixMap

Renders basicfont glyphs through font.Drawer into a temp NRGBA, then
copies non-transparent pixels into the int-packed PixMap buffer using
AWT drawString baseline semantics.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Port `DrawProgressGameShell`

**Files:**
- Modify: `pkg/jagex2/client/gameshell.go:241` (replace empty body)
- Modify: `pkg/jagex2/client/client.go:10117` (remove debug `Println`)
- Create or extend: `pkg/jagex2/client/gameshell_test.go`

- [ ] **Step 4.1: Write the failing test**

Create `pkg/jagex2/client/gameshell_test.go` (or extend if it already exists with other tests — check first with `ls pkg/jagex2/client/*_test.go`):

```go
package client

import "testing"

func TestDrawProgressGameShell_ClearsRefreshAndPopulatesOverlay(t *testing.T) {
	c := &Client{}
	c.ScreenWidth = 789
	c.ScreenHeight = 532
	c.Refresh = true

	c.DrawProgressGameShell("Connecting to fileserver", 25)

	if c.Refresh {
		t.Errorf("Refresh = true after DrawProgressGameShell; want false")
	}
	if c.OverlayPixMap == nil {
		t.Fatalf("OverlayPixMap nil after DrawProgressGameShell")
	}
	if c.OverlayPixMap.Width != 789 || c.OverlayPixMap.Height != 532 {
		t.Errorf("OverlayPixMap size = (%d,%d); want (789,532)",
			c.OverlayPixMap.Width, c.OverlayPixMap.Height)
	}
	// Verify at least one red pixel from the bar fill exists somewhere
	// in the overlay buffer (indicating draw calls actually fired).
	foundRed := false
	for _, px := range c.OverlayPixMap.Data {
		if px == 0x8C1111 {
			foundRed = true
			break
		}
	}
	if !foundRed {
		t.Errorf("no 0x8C1111 (bar red) pixels found in overlay; draw did not fire")
	}
}
```

- [ ] **Step 4.2: Run test to verify it fails**

Run:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/jagex2/client/... -run TestDrawProgressGameShell
```

Expected: FAIL — `Refresh = true after DrawProgressGameShell` and/or `OverlayPixMap nil` (the current function body is empty).

- [ ] **Step 4.3: Port `DrawProgressGameShell`**

In `pkg/jagex2/client/gameshell.go`, replace the body of `DrawProgressGameShell` at line 241. The full function (preserve the docstring above it):

```go
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
```

Add the new imports to the top of `gameshell.go` (extend the existing import block — confirm `pix2d` is already imported; if not, add it):

```go
	"goscape-client/pkg/jagex2/graphics/bootfont"
	"goscape-client/pkg/jagex2/graphics/pix2d"
```

- [ ] **Step 4.4: Remove the boot-time debug line**

In `pkg/jagex2/client/client.go` at line 10117, delete the line:

```go
		fmt.Println("DrawProgressGameShell called") // debug
```

If removing this leaves `fmt` unused in `client.go`, do NOT remove the `fmt` import — `fmt` is heavily used elsewhere. Verify with `go build`.

Also delete the leftover debug line at `client.go:10112`:

```go
	fmt.Printf("DrawProgress %v: %v\n", message, percent) // debug
```

— only if it's still safe (no other dependencies). If you'd rather leave it for now, that's fine; it's out of scope for this plan.

(Skip deleting `DrawProgress %v` if uncertain; leave it for a follow-up.)

- [ ] **Step 4.5: Run tests to verify they pass**

Run:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/jagex2/client/... -run TestDrawProgressGameShell
```

Expected: PASS.

- [ ] **Step 4.6: Verify full package still builds**

Run:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go build ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go vet ./...
```

Expected: clean.

- [ ] **Step 4.7: Commit**

```bash
git add pkg/jagex2/client/gameshell.go pkg/jagex2/client/client.go pkg/jagex2/client/gameshell_test.go
git commit --no-gpg-sign -m "$(cat <<'EOF'
port(gameshell): wire DrawProgressGameShell to overlay PixMap

Implements the boot-time loading-bar fallback via the shared overlay
PixMap, pix2d rectangles, and bootfont text. Mirrors Java
GameShell.drawProgress at lines 529-560. Removes the temporary
"DrawProgressGameShell called" debug Println.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Port `DrawError`

**Files:**
- Modify: `pkg/jagex2/client/client.go:8242` (replace `// Java: ...` comment lines with real draw calls)

- [ ] **Step 5.1: Replace the `DrawError` body**

In `pkg/jagex2/client/client.go`, replace the entire body of `DrawError` starting at line 8242 (preserve the existing docstring above it). The new body:

```go
func (c *Client) DrawError() {
	c.ensureOverlay()
	c.OverlayPixMap.Bind()
	pix2d.FillRect(0, 0, 0x000000, c.ScreenWidth, c.ScreenHeight)

	c.SetFrameRate(1)

	if c.ErrorLoading {
		c.FlameActive = false
		// Java: Font Helvetica BOLD 16, yellow header; BOLD 12 white body.
		// Go: FontBold12 throughout — same divergence as elsewhere.
		c.FontBold12.DrawString(30, 35, 0xFFFF00,
			"Sorry, an error has occured whilst loading RuneScape")
		c.FontBold12.DrawString(30, 85, 0xFFFFFF,
			"To fix this try the following (in order):")
		c.FontBold12.DrawString(30, 135, 0xFFFFFF,
			"1: Try closing ALL open web-browser windows, and reloading")
		c.FontBold12.DrawString(30, 165, 0xFFFFFF,
			"2: Try clearing your web-browsers cache from tools->internet options")
		c.FontBold12.DrawString(30, 195, 0xFFFFFF,
			"3: Try using a different game-world")
		c.FontBold12.DrawString(30, 225, 0xFFFFFF,
			"4: Try rebooting your computer")
		c.FontBold12.DrawString(30, 255, 0xFFFFFF,
			"5: Try selecting a different version of Java from the play-game menu")
	}
	if c.ErrorHost {
		c.FlameActive = false
		// Java: Font Helvetica BOLD 20, white. Go: FontBold12.
		c.FontBold12.DrawString(50, 50, 0xFFFFFF, "Error - unable to load game!")
		c.FontBold12.DrawString(50, 100, 0xFFFFFF, "To play RuneScape make sure you play from")
		c.FontBold12.DrawString(50, 150, 0xFFFFFF, "http://www.runescape.com")
	}
	if !c.ErrorStarted {
		c.OverlayPixMap.Draw(&c.Ops, 0, 0)
		return
	}
	c.FlameActive = false
	c.FontBold12.DrawString(30, 35, 0xFFFF00,
		"Error a copy of RuneScape already appears to be loaded")
	c.FontBold12.DrawString(30, 85, 0xFFFFFF,
		"To fix this try the following (in order):")
	c.FontBold12.DrawString(30, 135, 0xFFFFFF,
		"1: Try closing ALL open web-browser windows, and reloading")
	c.FontBold12.DrawString(30, 165, 0xFFFFFF,
		"2: Try rebooting your computer, and reloading")
	c.OverlayPixMap.Draw(&c.Ops, 0, 0)
}
```

Note the early-return reshape: Java had `if (!this.errorStarted) { return; }` — we must still composite the overlay before that early return, since the `ErrorLoading` or `ErrorHost` branch may have just drawn into it. The Java code didn't need this because each draw call painted directly to the window.

Verify `pix2d` is already imported by `client.go`. If not, add `"goscape-client/pkg/jagex2/graphics/pix2d"` to the import block.

Confirm `pixfont.PixFont.DrawString` signature with:

```bash
grep -n "func (p \*PixFont) DrawString" pkg/jagex2/graphics/pixfont/pixfont.go
```

The signature is `DrawString(arg0, arg1, arg3 int, arg4 string)`. From existing call sites (e.g. `client.go:843`: `c.FontBold12.DrawString(c.ProjectX+50-var12, c.ProjectY+1, 0, var15)`), the argument order is `(x, y, color, text)`. The code above uses that order.

- [ ] **Step 5.2: Build and vet**

Run:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go build ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go vet ./...
```

Expected: clean.

- [ ] **Step 5.3: Run the full test suite**

Run:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./...
```

Expected: all tests pass. No regression in any package.

- [ ] **Step 5.4: Commit**

```bash
git add pkg/jagex2/client/client.go
git commit --no-gpg-sign -m "$(cat <<'EOF'
port(client): wire DrawError to overlay PixMap

Replaces the placeholder // Java: reference comments with real pixfont
DrawString calls. Renders the three error branches (ErrorLoading,
ErrorHost, ErrorStarted) into the shared overlay PixMap. FontBold12
substitutes for Java's Helvetica BOLD 16/20 — same divergence as
elsewhere in the port.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Verify `go.mod` is clean and final sweep

**Files:**
- Modify: `go.mod`, `go.sum` (via `go mod tidy`)

- [ ] **Step 6.1: Run `go mod tidy`**

Run:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go mod tidy
```

This will:
- Promote `golang.org/x/image` from `// indirect` to a direct `require` (because `bootfont` now imports `golang.org/x/image/font/basicfont`).
- Update `go.sum` if needed.

- [ ] **Step 6.2: Inspect the diff**

Run:

```bash
git diff go.mod
```

Expected: `golang.org/x/image v0.40.0` (or current version) moved from the `// indirect` block to a direct require, OR a new direct require line added. The `golang.org/x/image` line should no longer have `// indirect`.

If unexpected modifications appear (e.g. version bumps to unrelated packages), revert with `git checkout go.mod go.sum` and investigate — `go mod tidy` should be minimal here.

- [ ] **Step 6.3: Full verification sweep**

Run all three in sequence:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go build ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go vet ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./...
```

Expected: all clean, all pass.

- [ ] **Step 6.4: Commit (only if go.mod/go.sum changed)**

```bash
git status --short go.mod go.sum
```

If both are modified:

```bash
git add go.mod go.sum
git commit --no-gpg-sign -m "$(cat <<'EOF'
chore(deps): promote golang.org/x/image to direct require

Now used directly by pkg/jagex2/graphics/bootfont for the basicfont
boot-time text renderer. Previously transitive via Gio.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

If `go mod tidy` produced no changes (because something else in the build chain already pulled `x/image` as direct), skip this commit.

---

## Out of scope (revisit later)

- Pixel-perfect Java parity for `DrawError`. Helvetica BOLD 16/20 vs. our `FontBold12` produces visibly different output. Documented inline; acceptable per the project's existing translation philosophy.
- Live window resize handling for `OverlayPixMap`. The window is fixed-size for the port; the `ensureOverlay` size check is defensive but won't be exercised.
- Unit tests for `DrawError` branches. Would require a real `pixfont.PixFont` fixture; cost exceeds value here. Visual correctness lands via the eventual integrated smoke test (resume follow-up item 3).
- Removing the `fmt.Printf("DrawProgress %v: %v\n", ...)` debug line at `client.go:10112`. Left in place to keep this plan focused; trivial follow-up.

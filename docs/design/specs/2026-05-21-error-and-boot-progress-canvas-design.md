# Design: Canvas for `DrawError` + `DrawProgressGameShell`

**Date:** 2026-05-21
**Scope:** Wire up the two remaining boot-time/error rendering paths
identified in `.claude/resume/2026-05-21-all-phases-done-pick-followup.md`
as follow-up item 2.

## Motivation

After all six PORTING.md phases closed, two functions in
`pkg/jagex2/client/` remain stubbed with their state-changing parts ported
but their visible output left as inline `// Java:` reference comments:

- `DrawError` (`client.go:8242`) — paints the fatal-error screen on top of
  the base component for the three error modes (`ErrorLoading`,
  `ErrorHost`, `ErrorStarted`).
- `DrawProgressGameShell` (`gameshell.go:241`) — the boot-time loading-bar
  fallback used by `DrawProgress` when `c.JagTitle` has not been
  downloaded yet (no fonts loaded).

The Java versions paint directly to the AWT base component's `Graphics`.
The Go port has no equivalent surface; the previous porter deferred the
decision pending an architectural call on which Go rendering primitive
should host them.

## Constraints discovered during exploration

Two facts shape the design more than anything else:

1. **`DrawError` runs late**, after `LoadTitle()` has finished and
   `c.FontBold12` / `c.FontPlain12` / `c.FontPlain11` are populated. It
   can reuse the exact `pix2d` + `pixfont` + `PixMap` pipeline already in
   use at `client.go:10122-10153` for the working post-boot
   `DrawProgress`.
2. **`DrawProgressGameShell` runs early**, before `JagTitle` exists. None
   of the pixfont fonts are loaded yet. Any text drawn during this phase
   must use a font that ships with the Go binary.

## Approach: shared overlay PixMap + per-function text source

Both functions paint full-screen overlays. We add **one** shared overlay
`*pixmap.PixMap` (`ScreenWidth × ScreenHeight`) to `Client`. Both
functions follow the same shape:

1. `c.OverlayPixMap.Bind()` — make `pix2d` write to this buffer.
2. Use `pix2d.FillRect` / `pix2d.DrawRect` for rectangles and fills.
3. Render text via a renderer chosen by the function (see below).
4. `c.OverlayPixMap.Draw(&c.Ops, 0, 0)` — composite into the frame's
   `op.Ops`.

Mutual exclusion: the two functions are never live concurrently (boot vs.
fatal error), so sharing one buffer is safe.

The text renderer differs per function:

- `DrawError`: existing `pixfont.PixFont` instances on `Client`
  (`FontBold12`, `FontPlain12`, `FontPlain11`). Already loaded by the
  time this function runs.
- `DrawProgressGameShell`: a new tiny `bootfont` package wrapping
  `golang.org/x/image/font/basicfont.Face7x13`, which is monospace 7x13,
  shipped in `golang.org/x/image` (already a transitive dependency via
  Gio's text shaper). No `JagTitle` required.

## Why these choices over the alternatives

- **`basicfont.Face7x13` over Gio text overlay.** A Gio text overlay
  would introduce a second rendering pipeline used by exactly one
  rarely-seen boot-time screen. `basicfont` keeps a single composition
  target (the PixMap) and matches the rest of the codebase.
- **`basicfont` over hand-rolled font.** Hand-rolling reinvents what
  `basicfont` gives us for ~0 dependency cost. The only argument for a
  custom font would be matching Java's Helvetica BOLD 13, but the
  post-boot `DrawProgress` path already diverges from Helvetica (uses
  RuneScape's own pixel font). Font fidelity is not a project goal.
- **`basicfont` over bar-only-no-text.** Skipping the message rendering
  is bad UX — boot errors like "Error loading - Will retry in 30 secs"
  become invisible.
- **`pixfont` over `bootfont` for `DrawError`.** `DrawError`'s call site
  guarantees `JagTitle` is loaded. Using `pixfont` matches every other
  text-drawing call site in the client.
- **One shared overlay PixMap over two.** The two functions are
  mutually exclusive. Sharing saves an allocation and keeps the
  rendering pattern uniform.

## New code: `pkg/jagex2/graphics/bootfont`

New package wrapping `basicfont.Face7x13`. Three functions:

```go
func DrawString(p *pixmap.PixMap, x, y, color int, msg string)
func StringWidth(msg string) int  // pixel width, for centering
func Height() int                  // font line height (13)
```

Implementation: rasterize via `font.Drawer{Dst, Src, Face, Dot}` into a
temp `image.NRGBA` sized to the message bounding box, then copy
non-zero-alpha pixels into `p.Data` as `0x00RRGGBB` ints. The temp alloc
runs at most a few times per boot frame and never on the game's hot
path, so allocation cost is negligible.

`go.mod`: promote `golang.org/x/image` from `// indirect` to a direct
`require`.

## `DrawError` changes (`client.go:8242`)

Replace the existing `// Java: ...` reference-comment lines with real
calls. Function shape:

1. `c.SetFrameRate(1)` (unchanged, already present).
2. `c.OverlayPixMap.Bind()`.
3. `pix2d.FillRect(0, 0, 0x000000, c.ScreenWidth, c.ScreenHeight)` —
   black background.
4. Per-branch text rendering using `c.FontBold12`. The Java reference
   uses Helvetica BOLD 16 (header) / BOLD 12 (body) / BOLD 20 (host
   error); the Go port uses `FontBold12` throughout — same divergence
   as the rest of the client.
   - `c.ErrorLoading`: yellow header + 5-line white body
     (`c.FlameActive = false`).
   - `c.ErrorHost`: white header + 2-line white body
     (`c.FlameActive = false`).
   - `c.ErrorStarted`: yellow header + 2-line white body
     (`c.FlameActive = false`).
5. `c.OverlayPixMap.Draw(&c.Ops, 0, 0)`.

Existing side-effect ordering between the three branches is preserved
exactly; the only change is wiring the visible output.

## `DrawProgressGameShell` changes (`gameshell.go:241`)

Replace the empty function body. Function shape, matching Java's
`GameShell.drawProgress()` at GameShell.java:529-560:

1. `c.OverlayPixMap.Bind()`.
2. If `c.Refresh`: `pix2d.FillRect` black background; clear
   `c.Refresh`. (See *Open questions* below — verify field name.)
3. Red outline rect at center: `pix2d.DrawRect(c.ScreenWidth/2-152,
   0x8C1111, 34, c.ScreenHeight/2-18, 304)`.
4. Red progress fill: `pix2d.FillRect(c.ScreenHeight/2-16,
   c.ScreenWidth/2-150, 0x8C1111, percent*3, 30)`.
5. Black remainder: `pix2d.FillRect(c.ScreenHeight/2-16,
   c.ScreenWidth/2-150+percent*3, 0, 300-percent*3, 30)`.
6. Centered message above the bar via `bootfont.DrawString` (white,
   `0xFFFFFF`), positioned at
   `(c.ScreenWidth - bootfont.StringWidth(message))/2, c.ScreenHeight/2 - 18 + 22`.
7. `c.OverlayPixMap.Draw(&c.Ops, 0, 0)`.

Removes the existing `fmt.Println("DrawProgressGameShell called")` debug
line at `client.go:10117`.

## Allocation / lifecycle of `OverlayPixMap`

`c.OverlayPixMap` is allocated lazily on first use (in either
`DrawError` or `DrawProgressGameShell`). Reason: `ScreenWidth` /
`ScreenHeight` are set by `GameShell.SetCanvasSize` (`gameshell.go:22`)
during `InitApplication`, which precedes `Run`. Eager allocation in
`NewClient` would run *before* the canvas size is known. Lazy
allocation in a small helper avoids that ordering problem and keeps
`NewClient` lean.

```go
func (c *Client) ensureOverlay() {
    if c.OverlayPixMap == nil ||
        c.OverlayPixMap.Width != c.ScreenWidth ||
        c.OverlayPixMap.Height != c.ScreenHeight {
        c.OverlayPixMap = pixmap.NewPixMap(c.ScreenWidth, c.ScreenHeight)
    }
}
```

The size-equality check handles a future resize (unlikely in this
port, but free to add).

## Testing

- `bootfont`:
  - `StringWidth` — deterministic from basicfont's monospace 7-pixel
    advance; assert on a handful of strings.
  - Rasterization — render a known glyph and assert at least one
    expected pixel is set in the target `PixMap`. Avoids brittle
    full-bitmap snapshots.
- `DrawError`:
  - Smoke test per branch (`ErrorLoading`, `ErrorHost`, `ErrorStarted`)
    asserting (a) no panic, (b) `c.FlameActive` ends `false`, (c)
    `c.OverlayPixMap` is populated.
- `DrawProgressGameShell`:
  - Smoke test with a representative `percent` value, asserting
    `c.Refresh` is cleared and the overlay is populated.
- Visual correctness is left to the eventual integrated smoke test
  (`.claude/resume/2026-05-21-all-phases-done-pick-followup.md` follow-up
  item 3). The current spec does not promise pixel-perfect Java parity;
  it promises a working canvas for the boot-time/error paths using the
  same rendering shape the rest of the client uses.

## Risks and open questions

1. **`c.Refresh` field existence.** Java's `this.refresh` controls
   whether the screen is cleared on this `drawProgress` call (set true
   in resize handlers, cleared after first fill). Need to confirm the
   field exists on `Client` and is wired to the same triggers. If
   missing, port it as part of this work. Action: grep
   `Refresh\\|refresh` on `Client` during step 1 of the plan.
2. **Font-size divergence at `DrawError`.** Java uses Helvetica BOLD
   16 / 20 for the headers. The port uses `FontBold12` for everything.
   Visual divergence is accepted (and matches the rest of the
   codebase). Documented inline in `DrawError`.
3. **`basicfont` doesn't support bold or color-styled runs.** Acceptable
   because `DrawProgressGameShell` only ever renders one plain white
   string; no bold/italic needed during boot.
4. **`go.mod` change.** Promoting `golang.org/x/image` from indirect to
   direct will be picked up by `go mod tidy`; no version pinning
   needed (Gio's existing constraint applies).

## Out of scope

- Helvetica fidelity for `DrawError`. If desired later, add a separate
  `helveticafont` package backed by `golang.org/x/image/font/sfnt` and a
  bundled Helvetica TrueType file. Not required for this work.
- Live resize handling for `OverlayPixMap` beyond the lazy size check
  above. The window is fixed-size for the port.
- Replacing the working post-boot `DrawProgress` path. It already
  works; touching it would be unrelated refactoring.

# Hand-Rolled Host Shell (Replace Gio) — Design

**Date:** 2026-05-25
**Status:** Approved (pending implementation plan)
**Branch:** rev-225

## Problem / Goal

The client uses the Gio UI toolkit (`gioui.org`) only as a host shell — open a window,
blit the software-rendered framebuffer, deliver input. It uses none of Gio's
retained-mode UI machinery. That mismatch has cost us real work: a wasm/WebGL
texture-churn leak (fixed by vendoring + patching Gio with a mutable-image path),
the `OpsMu` frame-serialization invariant, and a `key.SoftKeyboardCmd`-every-frame
hack to make browser keyboard input work.

This design replaces Gio with a minimal, purpose-built host shell that does exactly
what the client needs and nothing more, on **both** targets:

- **Native:** GLFW (`go-gl/glfw`) + OpenGL (`go-gl/gl`), cgo.
- **Browser:** `syscall/js` + WebGL, no third-party dependency.

Both are driven by **one** game loop that is faithful to the original Java
`GameShell` and the TypeScript client. The vendored+patched Gio tree is deleted at
the end.

## Why this is faithful, not a deviation

The original Java client and the TS client both render **immediate-mode**, frame by
frame, bypassing any retained component/scene model:

- **Java:** `GameShell` is a single dedicated game thread (`implements Runnable`,
  `extends Applet`). Its `run()` loop does timing → `Thread.sleep` → logic tick(s) →
  `draw()`. `PixMap.draw(y, Graphics, x)` calls `Graphics.drawImage(image, x, y)` —
  an immediate blit — once per PixMap per frame. AWT's *component* tree is
  retained, but this client never uses it; it pushes pixels directly on the game
  thread. Input is delivered to the same thread via AWT listeners.
- **TS (browser):** `async run()` is a synchronous `while (state >= 0)` loop that
  keeps the Java pacing math verbatim and replaces `Thread.sleep(delta)` with
  `await sleep(delta)`. Crucially it does **not** use `requestAnimationFrame`.
  `PixMap.draw(x, y)` calls `ctx.putImageData(img, x, y)` — immediate. Input via
  `addEventListener` into queues drained by the loop.

Gio's op list was the only retained layer in the stack, and it was the source of
the texture-churn leak. Going immediate-mode is a **return** to the golden-source
model, not a departure from it.

## Decisions (resolved during brainstorming)

1. **Scope:** one full spec covering all three sub-systems (seam, native backend,
   browser backend). Implementation is decomposed into two sequential plans (§13).
2. **Loop/threading:** one golden-source-faithful loop, sleep-driven, **no
   `requestAnimationFrame`**. Remove `OpsMu` and `op.Ops`.
3. **GL dialect:** WebGL1 / OpenGL ES 2.0 baseline — a single trivial shader pair
   serves both backends; widest compatibility; no VAO requirement.
4. **Native + CI:** accept cgo for GLFW; CI installs GL/X11/Wayland dev headers and
   `go build`s the native backend (compile-only, no display).

## Architecture

```
                 ┌─────────────────────────────────────┐
   game logic ── │ GameShell loop (one, faithful)       │
   (unchanged)   │   poll → tick(s) → draw → present →  │
                 │   time.Sleep(delta)                  │
                 └───────────────┬─────────────────────┘
                                 │ platform.Backend (interface)
                                 │ + neutral input events
                  ┌──────────────┴───────────────┐
        //go:build !js                      //go:build js
   ┌──────────────────────┐         ┌──────────────────────┐
   │ glfwBackend (cgo)     │         │ webglBackend          │
   │ GLFW window + go-gl   │         │ syscall/js + WebGL    │
   │ GLES2 quad blit       │         │ canvas, GLES2 quad    │
   └──────────────────────┘         └──────────────────────┘
```

The seam follows the **existing `_native.go`/`_js.go` build-tag pattern** already
used in the repo for sockets, URLs, storage, and codebase resolution.

### Current Gio integration surface (what this replaces)

Only four production files import `gioui.org`:

| File | Gio usage | Becomes |
|---|---|---|
| `cmd/client/main.go` | `app.Main()` | native: `LockOSThread` + run loop on main goroutine; wasm: start loop goroutine, `select{}` |
| `pkg/jagex2/client/gameshell.go` | `app.Window`, event loop, `pointer`/`key` input, `e.Frame` | unified loop + neutral-event dispatch (translation logic kept) |
| `pkg/jagex2/client/client.go` | `op.Ops` field, `event.Op` | `op.Ops` removed; `c.Draw` calls `BeginFrame`/`Blit`/`EndFrame` |
| `pkg/jagex2/graphics/pixmap/pixmap.go` | `op`, `paint.MutableImageOp` | holds a `platform.Texture`; `Draw(x,y)` blits |

Test files importing `gioui.org/io/key` (`handleeditevent_test.go`, `keycharfor_test.go`)
and `gioui.org/op` (`pixmap_test.go`) are updated to the neutral types / fake backend.

## Components

### 1. `pkg/jagex2/platform` — the seam

Build-neutral interface and neutral event types; two build-tagged implementations.

```go
// Texture is an opaque, backend-owned GPU texture handle.
type Texture interface{}

type Backend interface {
    // Input
    PollEvents() []Event   // drain input accumulated since the last call
    ShouldClose() bool

    // Geometry
    Size() (w, h int)

    // Textures
    NewTexture(w, h int) Texture
    UploadTexture(t Texture, rgba []byte) // in-place (glTexSubImage2D); caller gates on change

    // Frame
    BeginFrame()              // bind default framebuffer + viewport
    Blit(t Texture, x, y int) // textured quad at pixel offset, GL_NEAREST
    EndFrame()                // SwapBuffers (native) / implicit flush on yield (wasm)

    Destroy()
}

// Active is the process-wide current backend (mirrors the TS module-level
// canvas context). Set once at startup before the loop runs.
var Active Backend
```

Rationale — **immediate-mode, stateless `Blit`**: matches Java `drawImage(img,x,y)`
and TS `putImageData(img,x,y)`. RuneScape composites its screen from ~a dozen
full-rectangle PixMaps at fixed offsets, so `Blit` is the entire drawing API. No op
list, no retained scene, no compositor.

Performance note: immediate-mode's two traps are (a) re-uploading texture data every
frame and (b) high draw-call counts. Neither applies: (a) `UploadTexture` is gated by
the per-PixMap change-detection hash (texture uploaded once, re-uploaded only on pixel
change), so an unchanged PixMap costs only a 4-vertex quad draw per frame; (b) the
screen is ~a dozen large blits, not thousands of small ones. Documented escape hatch
if draw-call count ever grows: batch blits into a single vertex buffer (YAGNI now).

### 2. Neutral input events (in `platform`)

```go
type Event interface{ isEvent() }

type MouseMove   struct{ X, Y int }
type MouseButton struct{ X, Y, Button int; Pressed bool } // Button: 1=left, 2=right
type MouseCross  struct{ Entered bool }                    // enter / leave
type KeyPress    struct{ Key int; Down bool }              // named keys → AWT key codes
type CharInput   struct{ R rune }                          // layout/shift-resolved text
type FocusChange struct{ Gained bool }
```

The existing `gameshell.go` translation logic (`handlePointer`, `handleKey`,
`keyNameToAwt`, `keyCharFor`, the shift table, `KeyQueue`/`ActionKey` bookkeeping)
is kept almost verbatim — it consumes these neutral structs instead of Gio types.
Each backend maps its native key identifiers to the AWT codes the game expects:

- **Named keys:** GLFW key constants (native) / `KeyboardEvent.code` (browser) → AWT codes.
- **Text:** GLFW `SetCharCallback` (native) / DOM `keydown` `event.key` (browser) —
  both layout- and shift-resolved by the platform. This **removes the
  `key.SoftKeyboardCmd`/hidden-textarea hack** entirely.

### 3. The unified loop (`gameshell.go`)

`RunGameShell` (pacing) and `draw` (Gio events) collapse into one loop preserving the
exact `otim[10]`/`ratio`/`delta`/`count` pacing math from Java/TS:

```go
for !platform.Active.ShouldClose() && c.State >= 0 {
    for _, ev := range platform.Active.PollEvents() {
        c.dispatchInputEvent(ev)
    }
    // ---- unchanged otim/ratio/delta/count pacing math ----
    for count < 256 {
        c.Update()
        c.MouseClickButton = 0
        c.keyQueueReadPos = c.keyQueueWritePos
        count += ratio
    }
    count &= 0xFF
    platform.Active.BeginFrame()
    c.Draw() // issues PixMap.Draw(x,y) → Blit
    platform.Active.EndFrame()
    time.Sleep(time.Duration(delta) * time.Millisecond)
}
```

- **Native:** `main.go` calls `runtime.LockOSThread()` and runs this loop on the
  main goroutine (GLFW + GL are thread-affine). `glfw.PollEvents()` fires input
  callbacks synchronously; they fill the queue `PollEvents()` returns.
- **Browser:** the loop runs in a goroutine. `time.Sleep` parks it and yields to the
  JS event loop (page composites; DOM listeners fire) — Go's wasm equivalent of TS
  `await sleep`. `main()` blocks on `select{}` so the program never exits.

### 4. Present path (`pixmap.go`)

`PixMap` drops `op.Ops`/`paint`/`MutableImageOp`; holds a `platform.Texture`:

```go
func (p *PixMap) Draw(x, y int) { // no *op.Ops argument
    h := hashPixels(p.Data)
    if !p.uploaded || h != p.lastHash {
        writePixmapPixels(p.imgBuf, p.Data)
        platform.Active.UploadTexture(p.tex, p.imgBuf.Pix) // upload only on change
        p.lastHash, p.uploaded = h, true
    }
    platform.Active.Blit(p.tex, x, y) // blit every frame
}
```

`NewPixMap` allocates `p.tex = platform.Active.NewTexture(w, h)`. The ~46
`PixMap.Draw(&c.Ops, x, y)` call sites lose their first argument → `PixMap.Draw(x, y)`.
The change-detection hash (`hashPixels`, `writePixmapPixels`) is retained verbatim —
the texture-leak fix's intent survives the toolkit swap.

### 5. Native backend (`platform/*_native.go`, `//go:build !js`)

- `go-gl/glfw/v3.3/glfw` window: 532×789, non-resizable, title "Jagex".
- `go-gl/gl` (ES 2.0 / 2.1 profile). One static texture per PixMap; `glTexSubImage2D`
  on upload; a single textured-quad shader; `GL_NEAREST`.
- **`glfwSwapInterval(0)`** — vsync OFF, so `SwapBuffers` never blocks. The
  sleep-based pacing (`delta`) governs frame rate, matching the `deltime` math. A
  vsync'd blocking swap would fight the `otim`/`ratio`/`delta` loop.
- Input via GLFW callbacks (`SetCursorPosCallback`, `SetMouseButtonCallback`,
  `SetKeyCallback`, `SetCharCallback`, `SetCursorEnterCallback`, `SetFocusCallback`)
  → neutral-event queue, drained by `PollEvents()` (called right after
  `glfw.PollEvents()`).

### 6. Browser backend (`platform/*_js.go`, `//go:build js`)

- `syscall/js`: locate the `<canvas>`, `getContext("webgl")`. Same texture/quad/shader
  sequence as native (GLES2 ≈ WebGL1).
- Upload: `js.CopyBytesToJS` the RGBA into a `Uint8Array`, then `texSubImage2D`. The
  per-frame Go→JS copy happens only for *changed* PixMaps (hash-gated).
- Input: `addEventListener` via `js.FuncOf` for `mousemove`/`mousedown`/`mouseup`/
  `keydown`/`keyup`/`mouseenter`/`mouseleave`/`focus`/`blur` → neutral-event queue.
- **No `requestAnimationFrame`** — the Go loop's `time.Sleep` yields, per the TS
  precedent.

## What gets deleted

- `third_party/gioui.org/` (the vendored, patched copy)
- the `replace gioui.org => ./third_party/gioui.org` directive and `require gioui.org`
- `paint.MutableImageOp` and the gpu/ops/paint patch (no longer needed)
- `pixmap.OpsMu`
- `op.Ops` usage; `event.Op`, `key.FocusCmd`, `key.SoftKeyboardCmd`

## Out of scope (assumptions)

- **Audio** (oto) — already cross-platform; unchanged.
- **`signlink`, storage/IndexedDB, networking** — unchanged (already platform-split).
- **Window is fixed-size, non-resizable** (matches today's `MinSize == MaxSize`).
  Resize / HiDPI / DPR scaling is explicitly deferred to a future spec.

## Concurrency

A single loop thread owns all input, draw, present, and GL state — so **`OpsMu` is
removed** (it only existed to serialize the game goroutine against Gio's separate
present goroutine; Java/TS have no such lock). Background goroutines (`signlink.StartPriv`,
`audio`) never touch GL or PixMaps.

**Invariant to enforce (audit gate):** every `PixMap.Draw` / present call MUST run on
the loop thread. Out-of-band repaints flagged in prior work — `drawProgress` during
`load()`, and repaints in `Read` / `TryReconnect` / `LoginFunc` — must be funneled
onto the loop thread (e.g., via a "draw progress text" flag the loop reads, or by
ensuring those code paths only run on the loop). On wasm, a network goroutine calling
WebGL would be a correctness bug, not just a race. The implementation plan must audit
every present caller and confirm loop-thread affinity.

## CI

`ci-gate.yml` gains a native-build job: `apt-get` GL/X11/Wayland dev headers
(`libgl1-mesa-dev`, `libx11-dev`, `libxcursor-dev`, `libxrandr-dev`, `libxinerama-dev`,
`libxi-dev`, `libwayland-dev`, `libxkbcommon-dev`), then `CGO_ENABLED=1 go build` the
native target (compile-only — no display needed). The existing wasm build, the
platform-neutral `go test`, and lint stay.

## Testing

- **Platform-neutral unit tests** (no GPU): input translation (`keyNameToAwt`,
  `keyCharFor`, shift table) fed neutral events; `PixMap` hash-gating via a **fake
  `Backend`** capturing `UploadTexture`/`Blit` calls — assert upload happens only when
  `p.Data` changes, blit happens every frame.
- **Build gates:** `go build` native (with cgo + headers, CI) and `GOOS=js GOARCH=wasm
  go build ./cmd/client`.
- **Manual native run (host):** visual parity — flames animate, mouse/keyboard work,
  no stale frames, memory flat.
- **Manual browser run (host):** acceptance gates — renders, input works (typing +
  clicking), memory plateaus (no leak), audio plays.
- **Audit gate:** confirm all present callers run on the loop thread (§Concurrency).

## Risk / rollback

- The change is large but the seam is narrow (4 production files import Gio today).
  The riskiest part is the loop/thread restructure and the out-of-band-repaint audit.
- cgo for native adds a build dependency; mitigated by the CI header-install job and
  the host-run policy.
- Rollback: the work is staged as two plans on `rev-225`; until Gio is deleted (the
  final step of Plan 2), the branch can revert to the Gio shell.

## Decomposition into implementation plans

One spec, **two sequential plans** (each produces a working, testable client — no
throwaway Gio-backed shim):

1. **Seam + loop/PixMap refactor + native backend.** Create the `platform` package
   (interface + neutral events). Refactor `gameshell.go` translation to consume neutral
   events, collapse the loop, refactor `PixMap.Draw(x,y)`, remove `OpsMu`/`op.Ops`.
   Implement the **native GLFW + go-gl backend** in the same plan so the refactor has a
   real backend to run and test against (no temporary shim). Add a **compile-only stub
   `js` backend** (builds, panics "not implemented" at runtime) so `GOOS=js go build`
   stays green until Plan 2. Add the CI native-build job. Audit out-of-band repaints.
   *Deliverable:* a working native client with Gio fully removed from the native path
   (Gio still vendored but no longer imported by native code).
2. **Browser `syscall/js` + WebGL backend + Gio deletion.** Replace the stub with the
   real WebGL backend. Then **delete** `third_party/gioui.org/`, the `replace`
   directive, and the `gioui.org` require. *Deliverable:* a working browser client and
   a Gio-free repository. Browser acceptance-gate run.

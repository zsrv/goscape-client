# WASM WebGL Texture-Churn Leak — Patched-Gio In-Place Upload — Design

**Date:** 2026-05-24
**Status:** Approved (pending implementation)
**Branch:** rev-225

## Problem

In the `GOOS=js GOARCH=wasm` build, the browser tab's memory climbs to multiple
GB within seconds on the title screen. **Root cause (confirmed):** every frame,
each `PixMap.Draw` calls `paint.NewImageOp(imgBuf)`, which assigns a *fresh*
`handle: new(int)` (Gio's texture-cache key). Gio therefore creates a new GL
texture, uploads it, and one frame later deletes the previous frame's handle
(`gpu.textureCache.frame()` evicts handles unused that frame → `DeleteTexture`).
On the desktop GL backend this churn is reclaimed (flat memory); on the **WebGL
backend the per-frame `createTexture`/`deleteTexture` churn is not reclaimed**,
so memory grows unbounded. Confirmed wasm-specific: native does not leak; the
code is identical on both targets.

The leak rate matches the signature: ~one framebuffer-equivalent (532×789×4 ≈
1.7 MB) of texture churn per image per frame × ~9 title images × ~50 fps.

## Why this needs a Gio patch

Gio v0.10.0 (the latest release — there is no newer version to upgrade to)
models images as **immutable**: `gpu.texHandle` uploads a texture once per handle
and never re-reads it (`if tex.tex != nil { return tex.tex }`). The only two
options the public `paint.ImageOp` API offers are:
- **fresh handle every frame** → re-uploads correctly, but churns textures (the
  leak), or
- **stable handle** → cached, never re-uploaded → shows a stale frame (our
  software rasterizer mutates `imgBuf` in place, which Gio never re-reads).

We use Gio against its grain — blitting a live, per-frame-mutated framebuffer —
and there is no public API for "keep this texture, but re-read its pixels." So
the fix is a small, targeted patch to a vendored copy of Gio that adds a
**mutable-image** path: a stable handle whose texture is re-uploaded *in place*
when its content changes. This eliminates per-frame texture creation/deletion
entirely (the leak's source), on every backend.

## Design

### 1. Vendoring (in-tree copy + replace)

Copy the Gio v0.10.0 module into the repo at `third_party/gioui.org/` (the whole
module — Gio's packages are interdependent, so a module `replace` requires all of
it), apply the patch (§2), and add to the root `go.mod`:

```
replace gioui.org => ./third_party/gioui.org
```

The `require gioui.org v0.10.0` line stays. A local-filesystem `replace` is **not
checksummed**, so `go.sum` is unaffected and the build is self-contained and
offline-reproducible. The patched files are documented inline (a header comment
naming the upstream version and the change) so the patch can be re-applied or
upstreamed.

### 2. The Gio patch — a generation-gated mutable-image path

Three localized edits; no change to the op byte-encoding (the generation rides in
the handle struct, which is already carried as the op's ref):

- **`internal/ops/ops.go`** — add a shared marker type carrying a generation:
  ```go
  // MutableImageHandle is the handle of a paint.ImageOp whose backing image is
  // mutated in place. It is stable across frames (so its texture cache entry
  // persists — no per-frame create/delete churn); Gen is bumped by the producer
  // when the pixels change, signalling the GPU to re-upload to the existing
  // texture. Patch (goscape): mutable-image support over upstream gioui.org v0.10.0.
  type MutableImageHandle struct{ Gen uint64 }
  ```
  Both `op/paint` and `gpu` already import `internal/ops`, so the type is shared
  without a new package.

- **`op/paint/paint.go`** — add the mutable-op API:
  ```go
  type MutableImageOp struct {
      Filter ImageFilter
      src    *image.RGBA
      handle *ops.MutableImageHandle
  }
  // NewMutableImageOp creates an ImageOp over a buffer that the caller mutates in
  // place. The caller reuses the returned value across frames (stable handle) and
  // calls Invalidate when the pixels change.
  func NewMutableImageOp(src *image.RGBA) MutableImageOp {
      return MutableImageOp{src: src, handle: &ops.MutableImageHandle{}}
  }
  func (m MutableImageOp) Invalidate() { m.handle.Gen++ }
  func (m MutableImageOp) Add(o *op.Ops) {
      if m.src == nil || m.src.Bounds().Empty() { return }
      data := ops.Write2(&o.Internal, ops.TypeImageLen, m.src, m.handle)
      data[0] = byte(ops.TypeImage)
      data[1] = byte(m.Filter)
  }
  ```
  (Encoding is identical to the existing `ImageOp.Add` — same `TypeImage`, same
  two refs `{src, handle}`. The only difference is `handle` is the stable
  `*MutableImageHandle` instead of a fresh `*int`.)

- **`gpu/gpu.go`** — `texture` gains `gen uint64`; `texHandle` re-uploads in place
  when the handle is mutable and its generation advanced:
  ```go
  t, exists := cache.get(key)            // key = {filter, handle}; stable handle → hit
  if !exists { t = &texture{src: data.src}; cache.put(key, t) }
  tex = t.(*texture)
  if mh, ok := data.handle.(*ops.MutableImageHandle); ok {
      if tex.tex != nil && tex.gen == mh.Gen {
          return tex.tex                 // unchanged → no upload, no churn
      }
      // create on first use OR re-upload in place on a generation bump:
      if tex.tex == nil { tex.tex = r.ctx.NewTexture(...) }  // once, ever
      driver.UploadImage(tex.tex, image.Pt(0,0), data.src)   // texSubImage2D in place
      tex.gen = mh.Gen
      return tex.tex
  }
  // ... existing immutable path unchanged for normal ImageOps ...
  ```
  `decodeImageOp` is unchanged (it already passes `handle` through as `any`;
  `texHandle` type-asserts it). The stable handle keeps the cache entry "used"
  every frame, so `textureCache.frame()` never evicts/deletes it → **zero
  `NewTexture`/`DeleteTexture` per frame**.

**Net effect:** mutable images create their GL texture exactly once and update it
in place via `texSubImage2D` only when their generation advances. No churn, and
no re-upload when unchanged.

### 3. `PixMap` — reuse one mutable op + hash-based change detection

`pkg/jagex2/graphics/pixmap/pixmap.go`:

- Fields: replace the per-frame `paint.NewImageOp` with a stable
  `imageOp paint.MutableImageOp` (created once in `NewPixMap` over `imgBuf`),
  plus `lastHash uint64` and `uploaded bool`.
- `NewPixMap`: `m.imageOp = paint.NewMutableImageOp(m.imgBuf)` (Filter = Nearest).
- `Draw`:
  ```go
  func (p *PixMap) Draw(ops *op.Ops, x, y int) {
      defer op.Offset(image.Point{X: x, Y: y}).Push(ops).Pop()
      h := hashPixels(p.Data) // fast FNV-1a 64 over the pixel buffer
      if !p.uploaded || h != p.lastHash {
          writePixmapPixels(p.imgBuf, p.Data) // copy only when changed
          p.imageOp.Invalidate()              // bump generation -> GPU re-uploads
          p.lastHash = h
          p.uploaded = true
      }
      p.imageOp.Add(ops)        // always add (keeps the texture cache entry alive)
      paint.PaintOp{}.Add(ops)
  }
  ```
- Update the file's buffer-reuse rationale comment: the old "mint a fresh handle
  each frame to force re-read" note is now false — we hold a stable mutable op and
  re-upload only on a generation bump.

Unchanged PixMaps (the large static title/chrome tiles) cost one hash pass per
frame and **zero** GPU traffic; only genuinely-animated content (the flame
strips; the in-game viewport) re-uploads, in place, with no churn.

### Change detection — hashing

`hashPixels` is FNV-1a 64-bit over the byte view of `p.Data`. It is an O(N) read
with no allocation, strictly cheaper than the `texSubImage2D` it elides. Chosen
over a stored-snapshot `bytes.Equal` (which doubles per-PixMap memory) and over a
push-based dirty flag (which would require plumbing into every pixel-write call
site). If per-frame hashing ever shows up in a profile, a dirty flag is the
documented fallback — but hashing keeps change-detection self-contained in
`PixMap`.

## Concurrency

Unchanged from today's invariant: `writePixmapPixels` and the GPU re-upload both
run under `pixmap.OpsMu` (the caller holds it across the whole frame build; the
`FrameEvent` handler holds the same `OpsMu` across `e.Frame`, inside which
`texHandle`/`UploadImage` runs). The generation bump (`Invalidate`) also runs
under `OpsMu`. No new shared state or concurrency surface.

When a PixMap stops being drawn (e.g., title → gameplay), its op is no longer
`Add`ed → its handle is unused that frame → `textureCache.frame()` evicts and
deletes its texture. Cleanup still works.

## Testing

- **Native:** `go build ./...`, `go test ./...` (the pixmap tests/benchmarks),
  `go vet ./...`. A **visual parity check on the desktop build** is required — the
  patched mutable path renders on the desktop GL backend too, and must produce
  identical output (animated flames, no stale frames, no regression). This guards
  the working native path against the patch.
- **Native unit test:** `paint.NewMutableImageOp` returns a stable handle and
  `Invalidate` advances its `Gen` (testable without a GPU). The `PixMap`
  hash-gating logic (upload on change, skip when unchanged) can be unit-tested by
  asserting `imageOp` generation advances only when `p.Data` changes — a
  white-box test in the pixmap package, no GPU needed.
- **wasm:** `GOOS=js GOARCH=wasm go build ./cmd/client`; `go vet` on the touched
  js-tagged packages (none here — the patch is build-neutral Gio + pixmap).
- **Decisive browser test (acceptance gate):** sit on the title screen — memory
  must **plateau** (no climb to GB). Confirm the flames still animate and nothing
  is stale (proves re-upload-on-change works). Re-check after login (the gameplay
  viewport re-uploads in place each frame — verify it's smooth and memory-flat).

## Risk / rollback

The entire change is gated behind the `replace` directive plus the `PixMap`
`NewMutableImageOp` call. Reverting the `replace` (and restoring the per-frame
`paint.NewImageOp` in `PixMap`) returns to stock Gio behavior. The Gio patch is
small and isolated to three files, each carrying a header comment marking it as a
goscape patch over upstream v0.10.0 for future re-application or upstreaming.

## Files changed

| File | Change |
|---|---|
| `go.mod` | add `replace gioui.org => ./third_party/gioui.org` |
| `third_party/gioui.org/**` | **new** — in-tree copy of Gio v0.10.0 |
| `third_party/gioui.org/internal/ops/ops.go` | add `MutableImageHandle{Gen uint64}` |
| `third_party/gioui.org/op/paint/paint.go` | add `MutableImageOp` + `NewMutableImageOp`/`Invalidate`/`Add` |
| `third_party/gioui.org/gpu/gpu.go` | `texture.gen`; `texHandle` in-place re-upload for mutable handles |
| `pkg/jagex2/graphics/pixmap/pixmap.go` | stable `MutableImageOp` + FNV hash change-detection; reuse instead of per-frame `NewImageOp` |
| `pkg/jagex2/graphics/pixmap/pixmap_test.go` | unit-test the generation/hash gating |

## Open items for the implementation plan

- Exact `texHandle` edit must preserve the existing immutable-`ImageOp` path
  byte-for-byte (only add the mutable branch) — confirm against the current body.
- Confirm `int` width of `p.Data` on `js/wasm` for the hash byte-view (hash the
  backing bytes regardless of int size; use `unsafe`-free byte iteration or
  `hash/fnv` fed the pixel ints) — pick a portable, allocation-free formulation.
- Verify the desktop GL backend's `UploadImage` to an existing texture is a
  correct in-place update (it is `texSubImage2D`) — covered by the native visual
  parity check.
- Determine the precise mechanics of copying Gio v0.10.0 into `third_party/`
  (e.g., from the module cache) and that the local module's own `go.mod`/deps
  resolve under the `replace`.

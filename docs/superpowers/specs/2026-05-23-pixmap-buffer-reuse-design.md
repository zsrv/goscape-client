# PixMap upload-buffer reuse — design

**Date**: 2026-05-23
**Status**: Approved by user (brainstorming session)
**Author**: brainstorming session with Claude Code

## Purpose

Eliminate the per-frame heap allocation in the frame-upload path. After
the two preceding optimizations (commits `18f9a39`, `c03b85e`),
`pixmap.convertPixmapPixels` still calls `image.NewRGBA` on every
`PixMap.Draw`, allocating a fresh full-size RGBA buffer each frame. The
2026-05-23 profiles showed `image.NewRGBA` as the single dominant heap
allocator (>90% of `alloc_space` once Gio's internal conversion was
removed). This design reuses one buffer per PixMap so the steady-state
frame path performs zero image allocation.

## Background — why a single buffer is sufficient

Classic double-buffering exists to let a consumer read buffer N while a
producer writes buffer N+1 *without a lock*. That is not our situation.
Investigation of the Gio v0.10.0 render path established three facts:

1. **Producer and consumer share one lock.** `client.go` holds
   `pixmap.OpsMu` across the entire frame build (every `PixMap.Draw`
   write), and `gameshell.go` holds the *same* `OpsMu` across `e.Frame`.
   The buffer write and the buffer read therefore never overlap.
2. **The GPU upload is synchronous.** `e.Frame` → `gpu.texHandle` →
   `driver.UploadImage` → GL `TexSubImage2D`, which copies the pixel
   bytes into the texture during the call. Once `e.Frame` returns (still
   under `OpsMu`), the driver owns its copy and the CPU buffer is free
   to overwrite.
3. **The fresh handle per Draw is load-bearing.** Gio's texture cache
   (`gpu.go`) skips re-upload on a cache hit and evicts per frame via
   `g.cache.frame()`. `paint.NewImageOp` mints a fresh `new(int)` handle
   each call, which forces a re-read of the buffer every frame. This
   behavior is preserved — we keep calling `NewImageOp` per Draw.

Because `OpsMu` already serializes write vs. read and the upload is
synchronous, one reusable buffer per PixMap is safe. A second buffer
would add memory and complexity with no safety benefit under this
architecture.

## Scope

In scope:

- New private field `imgBuf *image.RGBA` on `PixMap`, allocated once in
  `NewPixMap` (Width × Height).
- Replace `convertPixmapPixels(width, height, javaPixels) *image.RGBA`
  with `writePixmapPixels(dst *image.RGBA, javaPixels []int)`, which
  fills `dst.Pix` in place using the existing big-endian `PutUint32`
  loop and allocates nothing.
- `Draw` fills `p.imgBuf` and passes it to `paint.NewImageOp`.
- Documentation of the three safety invariants at the field and in
  `Draw`.
- Tests and a benchmark proving zero per-frame allocation.

Out of scope (YAGNI):

- Double/triple buffering — unnecessary given `OpsMu` + synchronous
  upload (see Background).
- PixMap resize handling — dimensions are fixed at construction; there
  is no resize path. A recreated PixMap goes through `NewPixMap` and
  gets a fresh `imgBuf`.
- Reducing per-frame GPU texture churn (`NewTexture` per frame). That is
  inherent to Gio's immediate-mode cache for dynamic content and is a
  separate concern.

## Architecture

```
NewPixMap(w, h):
  Data   = make([]int, w*h)
  imgBuf = image.NewRGBA(image.Rect(0, 0, w, h))   ← allocated ONCE

PixMap.Draw(ops, x, y):            [caller holds OpsMu]
  writePixmapPixels(p.imgBuf, p.Data)   ← fills in place, no alloc
  op := paint.NewImageOp(p.imgBuf)      ← fresh handle, forces re-upload
  op.Filter = FilterNearest
  op.Add(ops); PaintOp{}.Add(ops)

e.Frame (Gio goroutine):           [same OpsMu hold]
  TexSubImage2D reads p.imgBuf synchronously → buffer free afterward
```

## Components

- **`PixMap.imgBuf *image.RGBA`** — reusable upload destination; lifetime
  equals the PixMap's. Sized at construction, never reallocated.
- **`writePixmapPixels(dst *image.RGBA, javaPixels []int)`** — pure fill,
  no allocation. Same conversion as today: `0x00RRGGBB → 0xRRGGBBFF` via
  `binary.BigEndian.PutUint32(dst.Pix[i*4:], …)`.

## Safety invariants (documented in code)

1. `OpsMu` serializes the fill (write) against `e.Frame`'s upload (read).
2. GL `TexSubImage2D` copies synchronously; the buffer is reusable once
   `e.Frame` returns.
3. Each PixMap instance is drawn at most once per frame (verified: ~28
   fixed-size region instances, each drawn once to its own position), so
   there is no intra-frame snapshot aliasing.

## Testing

- **Adapt** `TestConvertPixmapPixelsProducesRGBA` to the new signature:
  allocate a destination `*image.RGBA`, call `writePixmapPixels`, assert
  exact `Pix` bytes (channel order + opaque alpha unchanged).
- **New** `TestWritePixmapPixelsReusesBuffer`: fill one buffer twice with
  different inputs; assert the buffer holds only the latest content and
  the backing pointer is unchanged (documents reuse, guards against an
  accidental realloc).
- **Benchmark** `BenchmarkWritePixmapPixels`: allocate the destination
  `*image.RGBA` once *before* `b.Loop()`, then call `writePixmapPixels`
  inside the loop. This measures the steady-state per-frame fill and
  should report **0 allocs/op** — the measurable proof the per-frame
  allocation is gone. (Allocating the buffer inside the loop would
  reintroduce the very allocation we are removing and defeat the test.)

## Verification plan

1. `go test ./pkg/jagex2/graphics/pixmap/` green (correctness + reuse).
2. `go build ./...`, `go vet ./...`, `go test -race ./...` all clean.
3. Benchmark shows 0 allocs/op.
4. Live: capture a profile and confirm `image.NewRGBA` is gone from the
   steady-state `alloc_space` top (only construction-time allocation
   remains). Live capture is user-driven (sandbox cannot open a window).

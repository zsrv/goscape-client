# PixMap upload-buffer reuse Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate the per-frame `image.NewRGBA` allocation in `PixMap.Draw` by reusing one `*image.RGBA` buffer per PixMap.

**Architecture:** Add a private `imgBuf *image.RGBA` to `PixMap`, allocated once in `NewPixMap`. Replace the allocating `convertPixmapPixels` with a fill-in-place `writePixmapPixels(dst, src)`. `Draw` fills `p.imgBuf` and hands it to `paint.NewImageOp`. Safe because `OpsMu` serializes the write against `e.Frame`'s read and the GL upload (`TexSubImage2D`) copies synchronously.

**Tech Stack:** Go 1.26, Gio v0.10.0 (`gioui.org/op/paint`), `encoding/binary`, `image`.

**Spec:** `docs/superpowers/specs/2026-05-23-pixmap-buffer-reuse-design.md`

**Commit convention:** All commits use `git commit --no-gpg-sign`. Prefix Go commands with `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go`.

---

## File Structure

- `pkg/jagex2/graphics/pixmap/pixmap.go` — production code. Adds `imgBuf` field, `writePixmapPixels`; rewires `Draw`; removes `convertPixmapPixels`.
- `pkg/jagex2/graphics/pixmap/pixmap_test.go` — correctness + reuse + construction tests. Removes the obsolete `TestConvertPixmapPixelsProducesRGBA` once `convertPixmapPixels` is gone.
- `pkg/jagex2/graphics/pixmap/pixmap_bench_test.go` — benchmark proving 0 allocs/op for the fill.

---

## Task 1: Add `writePixmapPixels` fill-in-place helper

**Files:**
- Modify: `pkg/jagex2/graphics/pixmap/pixmap.go` (add new function; leave `convertPixmapPixels` in place for now)
- Modify: `pkg/jagex2/graphics/pixmap/pixmap_test.go` (add two tests)
- Modify: `pkg/jagex2/graphics/pixmap/pixmap_bench_test.go` (replace benchmark)

- [ ] **Step 1: Write the failing tests**

Add to `pkg/jagex2/graphics/pixmap/pixmap_test.go` (keep the existing `TestConvertPixmapPixelsProducesRGBA` for now):

```go
// TestWritePixmapPixelsFillsRGBA verifies the in-place fill writes the
// same opaque [R,G,B,0xFF] bytes the allocating converter produced.
func TestWritePixmapPixelsFillsRGBA(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 2, 1))
	pixels := []int{0x00FF8040, 0x00010203}

	writePixmapPixels(dst, pixels)

	want := []uint8{
		0xFF, 0x80, 0x40, 0xFF, // pixel 0: R, G, B, A(opaque)
		0x01, 0x02, 0x03, 0xFF, // pixel 1
	}
	if !bytes.Equal(dst.Pix, want) {
		t.Errorf("Pix = %v, want %v", dst.Pix, want)
	}
}

// TestWritePixmapPixelsReusesBuffer documents the reuse contract: a second
// fill overwrites the first's content and does NOT reallocate the backing
// array (the whole point of the optimization).
func TestWritePixmapPixelsReusesBuffer(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 2, 1))
	before := dst.Pix

	writePixmapPixels(dst, []int{0x00112233, 0x00445566})
	writePixmapPixels(dst, []int{0x00AABBCC, 0x00DDEEFF})

	want := []uint8{
		0xAA, 0xBB, 0xCC, 0xFF,
		0xDD, 0xEE, 0xFF, 0xFF,
	}
	if !bytes.Equal(dst.Pix, want) {
		t.Errorf("after reuse Pix = %v, want %v", dst.Pix, want)
	}
	if &dst.Pix[0] != &before[0] {
		t.Error("writePixmapPixels reallocated the backing array; expected in-place reuse")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/jagex2/graphics/pixmap/ -run TestWritePixmapPixels -v`
Expected: build failure — `undefined: writePixmapPixels`.

- [ ] **Step 3: Implement `writePixmapPixels`**

Add to `pkg/jagex2/graphics/pixmap/pixmap.go`, immediately above the existing `convertPixmapPixels`:

```go
// writePixmapPixels fills dst in place from packed 0x00RRGGBB ints (Java
// pix2d format). dst must have at least len(javaPixels) pixels. Java's
// DirectColorModel has no alpha mask, so all pixels are fully opaque;
// premultiplied RGBA then equals straight NRGBA byte-for-byte.
//
// 0x00RRGGBB -> 0xRRGGBBFF; a big-endian 32-bit store lays the bytes down
// as [R, G, B, 0xFF]. One wide store per pixel is ~2.2x faster than four
// byte writes (benchmarked 2026-05-23), and writing into a caller-owned
// buffer avoids a per-frame allocation.
func writePixmapPixels(dst *image.RGBA, javaPixels []int) {
	pix := dst.Pix
	for i, argb := range javaPixels {
		binary.BigEndian.PutUint32(pix[i*4:], uint32(argb)<<8|0xFF)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/jagex2/graphics/pixmap/ -run TestWritePixmapPixels -v`
Expected: PASS (both `TestWritePixmapPixelsFillsRGBA` and `TestWritePixmapPixelsReusesBuffer`).

- [ ] **Step 5: Replace the benchmark to measure the alloc-free fill**

Replace the entire body of `pkg/jagex2/graphics/pixmap/pixmap_bench_test.go` with:

```go
package pixmap

import (
	"image"
	"testing"
)

// benchPixels builds a full client-window worth of varied opaque pixels.
const benchW, benchH = 532, 789

func benchPixels() []int {
	p := make([]int, benchW*benchH)
	for i := range p {
		// Spread values across all three channels so the conversion
		// can't be specialised by a constant-folding optimiser.
		p[i] = (i * 2654435761) & 0x00FFFFFF
	}
	return p
}

func BenchmarkWritePixmapPixels(b *testing.B) {
	pixels := benchPixels()
	// Allocate the destination ONCE outside the loop: the steady-state
	// per-frame cost is the fill alone, which must be 0 allocs/op.
	dst := image.NewRGBA(image.Rect(0, 0, benchW, benchH))
	for b.Loop() {
		writePixmapPixels(dst, pixels)
	}
}
```

- [ ] **Step 6: Run the benchmark to confirm 0 allocs/op**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/jagex2/graphics/pixmap/ -run '^$' -bench BenchmarkWritePixmapPixels -benchmem`
Expected: a line like `BenchmarkWritePixmapPixels-… <ns> ns/op   0 B/op   0 allocs/op`.

- [ ] **Step 7: Commit**

```bash
git add pkg/jagex2/graphics/pixmap/pixmap.go pkg/jagex2/graphics/pixmap/pixmap_test.go pkg/jagex2/graphics/pixmap/pixmap_bench_test.go
git commit --no-gpg-sign -m "feat(pixmap): add alloc-free writePixmapPixels fill"
```

---

## Task 2: Allocate the reusable buffer in `NewPixMap`

**Files:**
- Modify: `pkg/jagex2/graphics/pixmap/pixmap.go` (add `imgBuf` field + allocation)
- Modify: `pkg/jagex2/graphics/pixmap/pixmap_test.go` (add construction test)

- [ ] **Step 1: Write the failing test**

Add to `pkg/jagex2/graphics/pixmap/pixmap_test.go`:

```go
// TestNewPixMapAllocatesImageBuffer verifies the reusable upload buffer is
// created at construction time, sized to the PixMap, so Draw never allocates.
func TestNewPixMapAllocatesImageBuffer(t *testing.T) {
	p := NewPixMap(4, 3)

	if p.imgBuf == nil {
		t.Fatal("NewPixMap did not allocate imgBuf")
	}
	if b := p.imgBuf.Bounds(); b.Dx() != 4 || b.Dy() != 3 {
		t.Errorf("imgBuf bounds = %v, want 4x3", b)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/jagex2/graphics/pixmap/ -run TestNewPixMapAllocatesImageBuffer -v`
Expected: build failure — `p.imgBuf undefined (type *PixMap has no field or method imgBuf)`.

- [ ] **Step 3: Add the field and allocate it**

In `pkg/jagex2/graphics/pixmap/pixmap.go`, change the `PixMap` struct from:

```go
// PixMap is a CPU-side pixel buffer that can be efficiently uploaded to GPU.
type PixMap struct {
	Data   []int
	Width  int
	Height int
}
```

to:

```go
// PixMap is a CPU-side pixel buffer that can be efficiently uploaded to GPU.
type PixMap struct {
	Data   []int
	Width  int
	Height int

	// imgBuf is the reusable RGBA buffer handed to paint.NewImageOp each
	// frame. Allocated once here and refilled in place by Draw, so the
	// steady-state frame path performs no image allocation. See Draw for
	// the safety invariants that make in-place reuse correct.
	imgBuf *image.RGBA
}
```

Then change `NewPixMap` from:

```go
func NewPixMap(width, height int) *PixMap {
	var m PixMap
	m.Width = width
	m.Height = height
	m.Data = make([]int, width*height)
	m.Bind()
	return &m
}
```

to:

```go
func NewPixMap(width, height int) *PixMap {
	var m PixMap
	m.Width = width
	m.Height = height
	m.Data = make([]int, width*height)
	m.imgBuf = image.NewRGBA(image.Rect(0, 0, width, height))
	m.Bind()
	return &m
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/jagex2/graphics/pixmap/ -run TestNewPixMapAllocatesImageBuffer -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/jagex2/graphics/pixmap/pixmap.go pkg/jagex2/graphics/pixmap/pixmap_test.go
git commit --no-gpg-sign -m "feat(pixmap): allocate reusable imgBuf in NewPixMap"
```

---

## Task 3: Rewire `Draw` to reuse the buffer; remove `convertPixmapPixels`

**Files:**
- Modify: `pkg/jagex2/graphics/pixmap/pixmap.go` (rewrite `Draw`, delete `convertPixmapPixels`)
- Modify: `pkg/jagex2/graphics/pixmap/pixmap_test.go` (delete obsolete `TestConvertPixmapPixelsProducesRGBA`)

- [ ] **Step 1: Rewrite `Draw` to fill and use `imgBuf`**

In `pkg/jagex2/graphics/pixmap/pixmap.go`, replace the body of `Draw` (currently lines ~69-76). Change:

```go
func (p *PixMap) Draw(ops *op.Ops, x, y int) {
	defer op.Offset(image.Point{X: x, Y: y}).Push(ops).Pop()
	img := convertPixmapPixels(p.Width, p.Height, p.Data)
	imageOp := paint.NewImageOp(img)
	imageOp.Filter = paint.FilterNearest
	imageOp.Add(ops)
	paint.PaintOp{}.Add(ops)
}
```

to:

```go
func (p *PixMap) Draw(ops *op.Ops, x, y int) {
	defer op.Offset(image.Point{X: x, Y: y}).Push(ops).Pop()
	// Fill the reused buffer in place instead of allocating a fresh image
	// every frame. This is safe because:
	//   1. OpsMu serializes this write against e.Frame's read (the caller
	//      holds OpsMu for the whole frame build; the FrameEvent handler
	//      holds the same OpsMu across e.Frame).
	//   2. The GL upload (TexSubImage2D, inside e.Frame) copies the bytes
	//      synchronously, so imgBuf is free to overwrite once e.Frame
	//      returns.
	//   3. paint.NewImageOp still mints a fresh handle, which forces Gio
	//      to re-read imgBuf every frame (a stable handle would show a
	//      frozen frame).
	// See docs/superpowers/specs/2026-05-23-pixmap-buffer-reuse-design.md.
	writePixmapPixels(p.imgBuf, p.Data)
	imageOp := paint.NewImageOp(p.imgBuf)
	imageOp.Filter = paint.FilterNearest
	imageOp.Add(ops)
	paint.PaintOp{}.Add(ops)
}
```

- [ ] **Step 2: Delete `convertPixmapPixels`**

In `pkg/jagex2/graphics/pixmap/pixmap.go`, delete the entire `convertPixmapPixels` function together with its doc comment (currently the block starting at `// convertPixmapPixels converts packed ...` through the function's closing brace). `writePixmapPixels` fully replaces it.

- [ ] **Step 3: Delete the obsolete test**

In `pkg/jagex2/graphics/pixmap/pixmap_test.go`, delete `TestConvertPixmapPixelsProducesRGBA` in full (it references the now-removed `convertPixmapPixels`; `TestWritePixmapPixelsFillsRGBA` covers the same correctness).

- [ ] **Step 4: Verify build, vet, and the full race suite**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go build ./... && TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go vet ./pkg/jagex2/graphics/pixmap/ && TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test -race ./...`
Expected: build clean, vet clean (no "unused function" or "undefined"), all packages `ok` / `PASS`.

- [ ] **Step 5: Commit**

```bash
git add pkg/jagex2/graphics/pixmap/pixmap.go pkg/jagex2/graphics/pixmap/pixmap_test.go
git commit --no-gpg-sign -m "perf(pixmap): reuse one RGBA buffer per PixMap, drop per-frame alloc"
```

---

## Final verification (after all tasks)

- [ ] `TMPDIR=… go test ./pkg/jagex2/graphics/pixmap/ -v` — all tests pass.
- [ ] `TMPDIR=… go test ./pkg/jagex2/graphics/pixmap/ -run '^$' -bench BenchmarkWritePixmapPixels -benchmem` — confirms `0 allocs/op`.
- [ ] `TMPDIR=… go build ./... && go vet ./... && go test -race ./...` — all clean.
- [ ] **Live (user-driven):** capture a profile and confirm `image.NewRGBA` no longer dominates steady-state `alloc_space` (only construction-time allocation remains). Sandbox cannot open a window, so this step is run by the user outside Claude Code.

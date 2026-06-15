# WASM Texture-Churn Leak Fix (Patched-Gio In-Place Upload) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate the wasm/WebGL multi-GB memory leak caused by per-frame GL texture create/delete churn, by vendoring Gio v0.10.0 in-tree and adding a mutable-image path (stable handle → in-place re-upload, generation-gated), then rewiring `PixMap` to reuse one mutable op with hash-based change detection.

**Architecture:** `paint.NewImageOp` mints a fresh texture-cache handle every frame, so Gio creates+deletes a GL texture per image per frame; WebGL doesn't reclaim that churn → unbounded growth (native is fine). The patch adds a stable `MutableImageHandle{Gen}` whose `gpu.texHandle` re-uploads to the *existing* texture (`texSubImage2D`) only when `Gen` advances — zero create/delete churn. `PixMap` holds one `MutableImageOp` and bumps its generation only when an FNV hash of its pixels changes.

**Tech Stack:** Go 1.26, Gio v0.10.0 (vendored in-tree via `replace`), `syscall/js` (browser).

**Sandbox note:** prefix `go` commands with `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache`. The `GOOS=js GOARCH=wasm` build prints a harmless `writing stat cache: ... read-only file system` line — filter `| grep -v 'stat cache'`, check `${PIPESTATUS[0]}`. Commit with `git commit --no-gpg-sign`. Spec: `docs/superpowers/specs/2026-05-24-wasm-texture-leak-patched-gio-design.md`.

**Sequencing:** Task 1 vendors *pristine* Gio + the `replace` and commits it unpatched (clean baseline, per request). Task 2 patches the vendored Gio. Task 3 rewires `PixMap`. Each builds green before the next.

---

## File Structure

| File | Task | Responsibility |
|---|---|---|
| `third_party/gioui.org/**` | 1 | in-tree pristine copy of Gio v0.10.0 |
| `go.mod` | 1 | `replace gioui.org => ./third_party/gioui.org` |
| `third_party/gioui.org/internal/ops/ops.go` | 2 | `MutableImageHandle{Gen uint64}` |
| `third_party/gioui.org/op/paint/paint.go` | 2 | `MutableImageOp` + constructor/Invalidate/Generation/Add |
| `third_party/gioui.org/gpu/gpu.go` | 2 | `texture.gen`; mutable branch in `texHandle` |
| `pkg/jagex2/graphics/pixmap/pixmap.go` | 3 | stable `MutableImageOp` + `hashPixels` change-detection |
| `pkg/jagex2/graphics/pixmap/pixmap_test.go` | 3 | unit-test the generation/hash gating |

---

## Task 1: Vendor pristine Gio v0.10.0 + replace (clean baseline)

**Files:** create `third_party/gioui.org/**`; modify `go.mod`.

- [ ] **Step 1: Copy the module out of the read-only cache**

```bash
cd .
mkdir -p third_party
cp -r $HOME/go/pkg/mod/gioui.org@v0.10.0 third_party/gioui.org
chmod -R u+w third_party/gioui.org
ls third_party/gioui.org/go.mod && head -1 third_party/gioui.org/go.mod
```
Expected: `third_party/gioui.org/go.mod` exists and its first line is `module gioui.org`.

- [ ] **Step 2: Add the replace directive**

Append to the root `go.mod` (after the `require` block):
```
replace gioui.org => ./third_party/gioui.org
```
Leave the existing `require gioui.org v0.10.0` line. (A local-path replace is not checksummed, so `go.sum` is untouched.)

- [ ] **Step 3: Verify both targets build against the in-tree copy**

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... 2>&1 | grep -v 'stat cache'; echo "native ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/... 2>&1 | grep -vE 'stat cache|no test files|^ok|cached'; echo "test ${PIPESTATUS[0]} (empty=pass)"
```
Expected: native 0, wasm 0, tests pass. (The build now compiles Gio from `third_party/`. If `go` complains about a missing `go.sum` entry, run `TMPDIR=… GOCACHE=… go mod tidy` once and re-run — but a local replace normally needs no sum change.)

- [ ] **Step 4: Commit the pristine copy (unpatched)**

```bash
git add third_party/gioui.org go.mod
git commit --no-gpg-sign -m "vendor: add pristine gioui.org v0.10.0 in-tree (replace directive)

Unmodified upstream copy, committed clean before any patch so the mutable-image
patch (next commit) is a reviewable diff against pristine Gio."
```

---

## Task 2: Patch the vendored Gio — mutable-image in-place upload

**Files:** modify `third_party/gioui.org/internal/ops/ops.go`, `third_party/gioui.org/op/paint/paint.go`, `third_party/gioui.org/gpu/gpu.go`.

- [ ] **Step 1: Add the `MutableImageHandle` marker type**

In `third_party/gioui.org/internal/ops/ops.go`, add this type (place it just after the `const ( TypeMacro OpType = iota ... )` op-type block — anywhere at package scope is fine):

```go
// MutableImageHandle is the handle of a paint.ImageOp whose backing image is
// mutated in place (e.g. a software rasterizer's framebuffer). It is stable
// across frames so its GPU texture-cache entry persists — no per-frame texture
// create/delete churn (which the WebGL backend does not reclaim). Gen is bumped
// by the producer when the pixels change, signalling gpu.texHandle to re-upload
// to the existing texture (texSubImage2D) instead of allocating a new one.
//
// goscape patch over upstream gioui.org v0.10.0 (mutable-image support).
type MutableImageHandle struct{ Gen uint64 }
```

- [ ] **Step 2: Add the `MutableImageOp` API**

In `third_party/gioui.org/op/paint/paint.go`, add this immediately after the existing `func (i ImageOp) Add(o *op.Ops) { ... }` method:

```go
// MutableImageOp draws an *image.RGBA whose pixels the caller mutates in place.
// Unlike ImageOp (immutable; uploaded once per handle), its GPU texture is
// created once and re-uploaded in place when the content changes, avoiding the
// per-frame texture churn the WebGL backend never reclaims. Reuse one
// MutableImageOp across frames (stable handle) and call Invalidate when the
// backing pixels change. goscape patch over upstream gioui.org v0.10.0.
type MutableImageOp struct {
	Filter ImageFilter
	src    *image.RGBA
	handle *ops.MutableImageHandle
}

// NewMutableImageOp creates a MutableImageOp backed by src (mutated in place).
func NewMutableImageOp(src *image.RGBA) MutableImageOp {
	return MutableImageOp{src: src, handle: &ops.MutableImageHandle{}}
}

// Invalidate signals that src's pixels changed; the next frame re-uploads to
// the existing GPU texture.
func (m MutableImageOp) Invalidate() { m.handle.Gen++ }

// Generation reports the current generation. Exposed for tests.
func (m MutableImageOp) Generation() uint64 { return m.handle.Gen }

// Add records the op. Encoding is identical to ImageOp.Add (same TypeImage and
// {src, handle} refs); only the handle is the stable mutable one.
func (m MutableImageOp) Add(o *op.Ops) {
	if m.src == nil || m.src.Bounds().Empty() {
		return
	}
	data := ops.Write2(&o.Internal, ops.TypeImageLen, m.src, m.handle)
	data[0] = byte(ops.TypeImage)
	data[1] = byte(m.Filter)
}
```

- [ ] **Step 3: Add `gen` to the texture struct**

In `third_party/gioui.org/gpu/gpu.go`, change:
```go
type texture struct {
	src *image.RGBA
	tex driver.Texture
}
```
to:
```go
type texture struct {
	src *image.RGBA
	tex driver.Texture
	gen uint64 // goscape patch: last-uploaded MutableImageHandle generation
}
```

- [ ] **Step 4: Add the mutable branch in `texHandle`**

In `third_party/gioui.org/gpu/gpu.go`, replace the whole `texHandle` function with this (the immutable path below the new branch is byte-for-byte the original):

```go
func (r *renderer) texHandle(cache *textureCache, data imageOpData) driver.Texture {
	key := textureCacheKey{
		filter: data.filter,
		handle: data.handle,
	}

	var tex *texture
	t, exists := cache.get(key)
	if !exists {
		t = &texture{
			src: data.src,
		}
		cache.put(key, t)
	}
	tex = t.(*texture)

	// goscape patch (mutable-image support over upstream gioui.org v0.10.0):
	// a *ops.MutableImageHandle keeps a STABLE cache key, so its entry is never
	// evicted/deleted (no per-frame texture churn — which WebGL never reclaims,
	// leaking GBs). The texture is created once and re-uploaded in place via
	// UploadImage (texSubImage2D) only when the handle's generation advances.
	if mh, ok := data.handle.(*ops.MutableImageHandle); ok {
		if tex.tex != nil && tex.gen == mh.Gen {
			return tex.tex
		}
		if tex.tex == nil {
			var minFilter, magFilter driver.TextureFilter
			switch data.filter {
			case filterLinear:
				minFilter, magFilter = driver.FilterLinearMipmapLinear, driver.FilterLinear
			case filterNearest:
				minFilter, magFilter = driver.FilterNearest, driver.FilterNearest
			}
			handle, err := r.ctx.NewTexture(driver.TextureFormatSRGBA,
				data.src.Bounds().Dx(), data.src.Bounds().Dy(),
				minFilter, magFilter,
				driver.BufferBindingTexture,
			)
			if err != nil {
				panic(err)
			}
			tex.tex = handle
		}
		driver.UploadImage(tex.tex, image.Pt(0, 0), data.src)
		tex.gen = mh.Gen
		return tex.tex
	}

	if tex.tex != nil {
		return tex.tex
	}

	var minFilter, magFilter driver.TextureFilter
	switch data.filter {
	case filterLinear:
		minFilter, magFilter = driver.FilterLinearMipmapLinear, driver.FilterLinear
	case filterNearest:
		minFilter, magFilter = driver.FilterNearest, driver.FilterNearest
	}

	handle, err := r.ctx.NewTexture(driver.TextureFormatSRGBA,
		data.src.Bounds().Dx(), data.src.Bounds().Dy(),
		minFilter, magFilter,
		driver.BufferBindingTexture,
	)
	if err != nil {
		panic(err)
	}
	driver.UploadImage(handle, image.Pt(0, 0), data.src)
	tex.tex = handle
	return tex.tex
}
```

- [ ] **Step 5: Verify both targets build with the patched Gio**

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... 2>&1 | grep -v 'stat cache'; echo "native ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go vet ./... 2>&1 | grep -v 'stat cache'; echo "vet ${PIPESTATUS[0]}"
gofmt -l third_party/gioui.org/internal/ops/ops.go third_party/gioui.org/op/paint/paint.go third_party/gioui.org/gpu/gpu.go
```
Expected: native 0, wasm 0, vet 0, gofmt prints nothing. (Behavior is verified by Task 3's test + Task 4's native-visual + browser-memory checks — the patch is in vendored library code.)

- [ ] **Step 6: Commit**

```bash
git add third_party/gioui.org/internal/ops/ops.go third_party/gioui.org/op/paint/paint.go third_party/gioui.org/gpu/gpu.go
git commit --no-gpg-sign -m "patch(gio): add mutable-image in-place upload (stable handle, generation-gated)

Adds ops.MutableImageHandle{Gen}, paint.MutableImageOp, and a texHandle branch
that re-uploads to the existing GL texture when the generation advances instead
of creating a new texture per frame. Eliminates the per-frame texture churn that
the WebGL backend never reclaimed (multi-GB wasm leak). Immutable ImageOp path
unchanged."
```

---

## Task 3: Rewire PixMap to the mutable op + hash change-detection

**Files:** modify `pkg/jagex2/graphics/pixmap/pixmap.go`; create test in `pkg/jagex2/graphics/pixmap/pixmap_test.go`.

- [ ] **Step 1: Write the failing tests**

Add to `pkg/jagex2/graphics/pixmap/pixmap_test.go` (create the file if absent; package `pixmap`, white-box). If the file exists, append these two functions and ensure `"gioui.org/op"` is imported:

```go
package pixmap

import (
	"testing"

	"gioui.org/op"
)

func TestHashPixelsDetectsChange(t *testing.T) {
	a := []int{1, 2, 3, 0xFFFFFF}
	if hashPixels(a) != hashPixels([]int{1, 2, 3, 0xFFFFFF}) {
		t.Fatal("identical data hashed differently")
	}
	if hashPixels(a) == hashPixels([]int{1, 2, 3, 0xFFFFFE}) {
		t.Fatal("different data hashed identically")
	}
}

func TestPixMapUploadsOnlyOnChange(t *testing.T) {
	p := NewPixMap(4, 4)
	var ops op.Ops

	p.Draw(&ops, 0, 0) // first draw must upload
	g1 := p.imageOp.Generation()
	if g1 == 0 {
		t.Fatalf("first Draw should bump generation, got %d", g1)
	}

	ops.Reset()
	p.Draw(&ops, 0, 0) // unchanged -> no re-upload
	if g2 := p.imageOp.Generation(); g2 != g1 {
		t.Fatalf("unchanged Draw re-uploaded: %d -> %d", g1, g2)
	}

	p.Data[5] = 0x123456 // change a pixel
	ops.Reset()
	p.Draw(&ops, 0, 0) // changed -> re-upload
	if g3 := p.imageOp.Generation(); g3 == g1 {
		t.Fatalf("changed Draw did not re-upload (gen stayed %d)", g3)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/graphics/pixmap/ -run 'TestHashPixels|TestPixMapUploads' -v`
Expected: FAIL — `undefined: hashPixels` and `p.imageOp undefined`.

- [ ] **Step 3: Add the new PixMap fields**

In `pkg/jagex2/graphics/pixmap/pixmap.go`, change the struct from:
```go
type PixMap struct {
	Data   []int
	Width  int
	Height int

	// imgBuf is the reusable RGBA buffer that Draw fills in place and
	// hands to paint.NewImageOp each frame, so the steady-state render
	// path performs no image allocation. See Draw for the safety
	// invariants that make in-place reuse correct.
	imgBuf *image.RGBA
}
```
to:
```go
type PixMap struct {
	Data   []int
	Width  int
	Height int

	// imgBuf is the reusable RGBA buffer that Draw fills in place, backing a
	// stable mutable image op (imageOp). The steady-state render path performs
	// no image allocation and no GPU texture churn — the texture is created
	// once and re-uploaded in place only when the pixels change.
	imgBuf *image.RGBA

	// imageOp is one stable mutable op reused every frame. A fresh ImageOp per
	// frame (the old approach) made Gio create+delete a GL texture per frame,
	// which the WebGL backend never reclaimed (multi-GB wasm leak). See
	// docs/superpowers/specs/2026-05-24-wasm-texture-leak-patched-gio-design.md.
	imageOp  paint.MutableImageOp
	lastHash uint64
	uploaded bool
}
```

- [ ] **Step 4: Create the mutable op in NewPixMap**

In `NewPixMap`, after `m.imgBuf = image.NewRGBA(image.Rect(0, 0, width, height))`, add:
```go
	m.imageOp = paint.NewMutableImageOp(m.imgBuf)
	m.imageOp.Filter = paint.FilterNearest
```

- [ ] **Step 5: Rewrite Draw + add hashPixels**

Replace the body of `Draw` (keep its signature; replace the implementation that minted `paint.NewImageOp` each frame) with:
```go
func (p *PixMap) Draw(ops *op.Ops, x, y int) {
	defer op.Offset(image.Point{X: x, Y: y}).Push(ops).Pop()
	// Re-upload only when the pixel content changed since the last upload. The
	// stable imageOp keeps one GPU texture alive across frames (no per-frame
	// texture create/delete churn), and hashPixels detects change without any
	// plumbing into the pix2d write paths.
	//
	// CONCURRENCY: callers hold OpsMu across the whole frame build, and the
	// FrameEvent handler holds the same OpsMu across e.Frame (inside which the
	// patched gpu.texHandle does the in-place UploadImage). So the write to
	// imgBuf, the Invalidate, and the upload are all serialized — same invariant
	// as before. See the OpsMu comment above.
	h := hashPixels(p.Data)
	if !p.uploaded || h != p.lastHash {
		writePixmapPixels(p.imgBuf, p.Data)
		p.imageOp.Invalidate()
		p.lastHash = h
		p.uploaded = true
	}
	// Always add the op so the texture stays "used" this frame and is not
	// evicted/deleted by Gio's texture cache.
	p.imageOp.Add(ops)
	paint.PaintOp{}.Add(ops)
}

// hashPixels is a fast FNV-1a-style 64-bit hash over the packed 0x00RRGGBB
// pixels, used to detect whether the buffer changed since the last GPU upload.
// Allocation-free and int-width-independent (each pixel fits in uint32). A
// collision would at worst skip one frame's re-upload of changed content (a
// 1-frame stale flicker), which is negligible for change detection.
func hashPixels(data []int) uint64 {
	const (
		offset uint64 = 1469598103934665603
		prime  uint64 = 1099511628211
	)
	h := offset
	for _, v := range data {
		h = (h ^ uint64(uint32(v))) * prime
	}
	return h
}
```
(`writePixmapPixels` and the imports `image`/`op`/`paint` are already present in the file; no import change needed.)

- [ ] **Step 6: Run the tests + builds**

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/graphics/pixmap/ -v 2>&1 | grep -v 'stat cache'; echo "pixmap test ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... 2>&1 | grep -v 'stat cache'; echo "native ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm ${PIPESTATUS[0]}"
gofmt -l pkg/jagex2/graphics/pixmap/pixmap.go pkg/jagex2/graphics/pixmap/pixmap_test.go
```
Expected: pixmap tests PASS (incl. the existing bench/tests), native 0, wasm 0, gofmt clean.

- [ ] **Step 7: Commit**

```bash
git add pkg/jagex2/graphics/pixmap/pixmap.go pkg/jagex2/graphics/pixmap/pixmap_test.go
git commit --no-gpg-sign -m "fix(pixmap): reuse one mutable image op + hash change-detection (fixes wasm leak)

PixMap now holds a stable paint.MutableImageOp and re-uploads (in place, via the
patched Gio) only when an FNV hash of its pixels changes, instead of minting a
fresh paint.NewImageOp every frame. Removes the per-frame GL texture churn that
leaked GBs on WebGL; static tiles cost one hash pass and zero GPU traffic."
```

---

## Task 4: Full verification

- [ ] **Step 1: Native gate**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./... && TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./... && TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go vet ./...`
Expected: all pass, vet clean.

- [ ] **Step 2: WASM build**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "exit ${PIPESTATUS[0]}"`
Expected: `exit 0`.

- [ ] **Step 3: Manual native visual-parity check (human-run, host)**

The patch changes Gio's render path on the desktop GL backend too — confirm no regression. Run `make run ARGS="10 0 highmem members …"`, and verify: the title screen renders correctly, the **flames animate** (proves in-place re-upload works), nothing is stale/frozen, and gameplay renders normally. Watch RSS on the title screen — should stay flat (it always did on native).

- [ ] **Step 4: Manual browser memory check (human-run, host) — the acceptance gate**

`make wasm && make wasm-serve`, open the client, sit on the title screen:
1. Browser memory must **plateau** — no climb to GB (this is the fix's acceptance criterion).
2. Flames still animate; nothing stale.
3. Log in and confirm the gameplay viewport renders smoothly and memory stays flat.
4. (Optional) DevTools Performance monitor: JS heap / GPU memory flat over a minute on the title screen.

---

## Notes / rollback

- Rollback = revert the `replace` directive and the `PixMap` `NewMutableImageOp` rewiring; stock Gio behavior returns.
- The patch is isolated to three Gio files, each marked "goscape patch over upstream gioui.org v0.10.0" for future re-application/upstreaming. Re-applying after a (future) Gio bump means re-copying the module (Task 1) and re-applying Task 2's three edits.
- Skip-when-unchanged (hashing) is the bandwidth optimization on top of the churn fix; if hashing ever shows in a profile, a push-based dirty flag is the documented fallback.
- Run `make lint` on the host before pushing (golangci-lint absent in sandbox).

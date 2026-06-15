# wasm Allocation Quick Wins Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove two behavior-preserving allocation churn sources from the wasm build: a per-draw rand source in `pixfont`, and a per-scene-build texel-pool reallocation in `pix3d`.

**Architecture:** Two independent changes. (1) `pixfont.DrawStringTooltip` reuses its `*rand.Rand` via `Seed` instead of allocating a fresh source each call. (2) `pix3d.ClearTexels` reclaims bound texel buffers back into the pool instead of nil-ing it, and `InitPool` reuses an existing matching pool instead of reallocating — with a size/detail guard that safely falls back to allocation on any mismatch. Applies to native and js alike.

**Tech Stack:** Go (native + `GOOS=js GOARCH=wasm`), standard `testing`. Reference spec: `docs/superpowers/specs/2026-05-25-wasm-alloc-quick-wins-design.md`.

**Conventions (every command + commit):**
- Prefix Go commands with `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache`.
- Commit with `git commit --no-gpg-sign`.
- Both builds must stay green: native `go build ./...` and `GOOS=js GOARCH=wasm go build ./...`.
- Bug-for-bug faithful port: no behavior change beyond the stated optimization.

The two tasks are independent and can be done in either order.

---

## Task 1: `pixfont` rand reuse

**Files:**
- Modify: `pkg/jagex2/graphics/pixfont/pixfont.go:256` (in `DrawStringTooltip`)
- Test: `pkg/jagex2/graphics/pixfont/pixfont_test.go` (append; file exists)

Context: `PixFont.Random` is typed `*rand.Rand` (pixfont.go:46); `math/rand` is already imported. `DrawStringTooltip` is the only site that assigns or reads `p.Random`.

- [ ] **Step 1: Write the failing test**

Append to `pkg/jagex2/graphics/pixfont/pixfont_test.go`:

```go
func TestRandReseedMatchesFreshSource(t *testing.T) {
	// The DrawStringTooltip optimization reseeds a reused *rand.Rand instead of
	// allocating a new source each call. This pins the stdlib guarantee that
	// makes that behavior-preserving: Seed(seed) yields the same stream as
	// New(NewSource(seed)).
	const seed = int64(0x5eed)
	fresh := rand.New(rand.NewSource(seed))
	reused := rand.New(rand.NewSource(1))
	reused.Seed(seed)
	for i := 0; i < 10; i++ {
		if a, b := fresh.Int(), reused.Int(); a != b {
			t.Fatalf("draw %d: fresh=%d reused=%d (reseed diverges from fresh source)", i, a, b)
		}
	}
}
```

Ensure the test file imports `"math/rand"` and `"testing"` (add to its import block if missing).

- [ ] **Step 2: Run the test to verify it passes immediately**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./pkg/jagex2/graphics/pixfont/ -run TestRandReseedMatchesFreshSource -v`
Expected: PASS. (This test pins a stdlib invariant the production change relies on; it passes before and after — it is a guard, not a red-then-green test. That is intentional and correct for documenting the equivalence.)

- [ ] **Step 3: Apply the production change**

In `pkg/jagex2/graphics/pixfont/pixfont.go`, inside `DrawStringTooltip`, replace line 256:

```go
	p.Random = rand.New(rand.NewSource(int64(arg0)))
```

with:

```go
	if p.Random == nil {
		p.Random = rand.New(rand.NewSource(int64(arg0)))
	} else {
		p.Random.Seed(int64(arg0))
	}
```

Leave the rest of `DrawStringTooltip` (the `p.Random.Int()` use on the next line, etc.) unchanged.

- [ ] **Step 4: Build + full suite + gofmt**

Run:
```
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go build ./...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOOS=js GOARCH=wasm go build ./...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./pkg/jagex2/graphics/pixfont/
gofmt -l pkg/jagex2/graphics/pixfont/pixfont.go pkg/jagex2/graphics/pixfont/pixfont_test.go
```
Expected: both builds OK, tests `ok`, gofmt prints nothing (`gofmt -w` any listed file).

- [ ] **Step 5: Commit**

```bash
git add pkg/jagex2/graphics/pixfont/pixfont.go pkg/jagex2/graphics/pixfont/pixfont_test.go
git commit --no-gpg-sign -m "perf(pixfont): reseed reused rand.Rand instead of allocating per tooltip draw"
```

---

## Task 2: `pix3d` texel pool reuse

**Files:**
- Modify: `pkg/jagex2/graphics/pix3d/pix3d.go` — `ClearTexels` (currently ~lines 115-120) and `InitPool` (currently ~lines 122-141)
- Test: `pkg/jagex2/graphics/pix3d/pix3d_test.go` (append; file exists)

Context (verified): `package pix3d`; globals `LowDetail bool`, `ActiveTexels [][]int` (len 50), `PoolSize int`, `TexelPool [][]int`. `PoolSize` is both pool capacity and a free-list stack pointer — free buffers in `TexelPool[0..PoolSize-1]`, bound buffers in `ActiveTexels[i]`; the `size` buffers are conserved between them. `GetTexels` pops (`PoolSize--`, `TexelPool[PoolSize]=nil`); `PushTexture` pushes back (`TexelPool[PoolSize]=buf; PoolSize++`). `LowDetail` selects buffer length (16384 vs 65536) and changes only on init/reset, never mid-gameplay.

- [ ] **Step 1: Write the failing tests**

Append to `pkg/jagex2/graphics/pix3d/pix3d_test.go`:

```go
func TestInitPoolReusesAfterClearTexels(t *testing.T) {
	// Mirror the scene-rebuild cycle: InitPool -> (textures bound) -> ClearTexels
	// -> InitPool. The second InitPool must REUSE the existing buffers, not
	// reallocate them.
	LowDetail = true
	TexelPool = nil
	for i := range ActiveTexels {
		ActiveTexels[i] = nil
	}

	InitPool(2)
	if len(TexelPool) != 2 || len(TexelPool[0]) != 16384 {
		t.Fatalf("InitPool(2) gave len=%d slot0len=%d, want 2 / 16384", len(TexelPool), len(TexelPool[0]))
	}
	slot0 := &TexelPool[0][0]
	slot1 := &TexelPool[1][0]

	// Simulate both buffers bound to textures (drain the free pool like GetTexels).
	ActiveTexels[5] = TexelPool[1]
	TexelPool[1] = nil
	PoolSize--
	ActiveTexels[9] = TexelPool[0]
	TexelPool[0] = nil
	PoolSize--
	if PoolSize != 0 {
		t.Fatalf("after draining both buffers PoolSize=%d, want 0", PoolSize)
	}

	ClearTexels()
	if PoolSize != 2 {
		t.Fatalf("ClearTexels left PoolSize=%d, want 2 (all buffers reclaimed)", PoolSize)
	}
	if TexelPool[0] == nil || TexelPool[1] == nil {
		t.Fatal("ClearTexels left a nil slot; buffers not fully reclaimed")
	}
	if ActiveTexels[5] != nil || ActiveTexels[9] != nil {
		t.Fatal("ClearTexels did not clear ActiveTexels")
	}

	InitPool(2)
	// The two backing arrays must be the SAME as before (reused, not realloc'd).
	got := map[*int]bool{&TexelPool[0][0]: true, &TexelPool[1][0]: true}
	if !got[slot0] || !got[slot1] {
		t.Error("InitPool reallocated buffers instead of reusing the reclaimed pool")
	}
	if PoolSize != 2 {
		t.Fatalf("InitPool reuse path left PoolSize=%d, want 2", PoolSize)
	}
}

func TestInitPoolReallocatesOnDetailChange(t *testing.T) {
	// If LowDetail changes (so the required buffer length differs), the guard
	// must fall through and reallocate rather than reuse wrong-sized buffers.
	LowDetail = true
	TexelPool = nil
	for i := range ActiveTexels {
		ActiveTexels[i] = nil
	}
	InitPool(2) // 16384-length buffers
	old0 := &TexelPool[0][0]

	LowDetail = false // now wants 65536-length buffers
	InitPool(2)
	if len(TexelPool[0]) != 65536 {
		t.Fatalf("slot len=%d after detail change, want 65536 (should have reallocated)", len(TexelPool[0]))
	}
	if &TexelPool[0][0] == old0 {
		t.Error("InitPool reused 16384 buffers after detail change to high detail")
	}
}
```

Ensure the test file imports `"testing"` (it already exists; add if its block lacks it).

- [ ] **Step 2: Run the tests to verify they fail**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./pkg/jagex2/graphics/pix3d/ -run "TestInitPoolReuses|TestInitPoolReallocates" -v`
Expected: FAIL on both. `TestInitPoolReusesAfterClearTexels` fails because the current `ClearTexels` sets `TexelPool = nil` and never updates `PoolSize`, so the post-`ClearTexels` `PoolSize == 2` assertion fails. `TestInitPoolReallocatesOnDetailChange` fails because the current `InitPool`'s `if TexelPool != nil { return }` guard returns early on the second call, leaving 16384-length buffers instead of reallocating to 65536.

- [ ] **Step 3: Rewrite `ClearTexels` to reclaim**

In `pkg/jagex2/graphics/pix3d/pix3d.go`, replace the current `ClearTexels`:

```go
func ClearTexels() {
	TexelPool = nil
	for i := range 50 {
		ActiveTexels[i] = nil
	}
}
```

with:

```go
// ClearTexels readies the texel pool for a fresh scene by reclaiming every
// bound buffer back into the free pool instead of releasing them, so the
// following InitPool can reuse the allocation. The `size` buffers are conserved
// between TexelPool (free) and ActiveTexels (bound), so after this the pool is
// full again (PoolSize == capacity) with no allocation.
func ClearTexels() {
	if TexelPool == nil {
		return
	}
	for i := range ActiveTexels {
		if ActiveTexels[i] != nil {
			TexelPool[PoolSize] = ActiveTexels[i]
			PoolSize++
			ActiveTexels[i] = nil
		}
	}
}
```

- [ ] **Step 4: Rewrite `InitPool` to reuse a matching pool**

Replace the current `InitPool`:

```go
func InitPool(size int) {
	if TexelPool != nil {
		return
	}
	PoolSize = size
	if LowDetail {
		TexelPool = make([][]int, PoolSize)
		for i := range TexelPool {
			TexelPool[i] = make([]int, 16384)
		}
	} else {
		TexelPool = make([][]int, PoolSize)
		for i := range TexelPool {
			TexelPool[i] = make([]int, 65536)
		}
	}
	for i := range 50 {
		ActiveTexels[i] = nil
	}
}
```

with:

```go
func InitPool(size int) {
	texelLen := 65536
	if LowDetail {
		texelLen = 16384
	}
	// Reuse an existing pool only when it already matches the requested slot
	// count and the current detail level's buffer length. Any mismatch (detail
	// change, or a partially drained pool whose slot 0 was popped to nil ->
	// len 0) falls through to (re)allocation, so correctness never depends on
	// the reuse firing. ClearTexels repopulates all slots first in the scene-
	// rebuild path, so this hits the reuse branch.
	if TexelPool != nil && len(TexelPool) == size && len(TexelPool[0]) == texelLen {
		PoolSize = size
		for i := range 50 {
			ActiveTexels[i] = nil
		}
		return
	}
	PoolSize = size
	TexelPool = make([][]int, size)
	for i := range TexelPool {
		TexelPool[i] = make([]int, texelLen)
	}
	for i := range 50 {
		ActiveTexels[i] = nil
	}
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./pkg/jagex2/graphics/pix3d/ -run "TestInitPoolReuses|TestInitPoolReallocates" -v`
Expected: PASS (both).

- [ ] **Step 6: Build + full suite + gofmt**

Run:
```
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go build ./...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOOS=js GOARCH=wasm go build ./...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./...
gofmt -l pkg/jagex2/graphics/pix3d/pix3d.go pkg/jagex2/graphics/pix3d/pix3d_test.go
```
Expected: both builds OK, all tests `ok`, gofmt clean.

- [ ] **Step 7: Commit**

```bash
git add pkg/jagex2/graphics/pix3d/pix3d.go pkg/jagex2/graphics/pix3d/pix3d_test.go
git commit --no-gpg-sign -m "perf(pix3d): reuse the texel pool across scene rebuilds instead of realloc

ClearTexels reclaims bound buffers into the free pool; InitPool reuses a
matching pool, falling back to allocation on size/detail mismatch."
```

---

## Final verification (after both tasks)

- [ ] Lint the two touched packages (native is fine; no js-only code here):
```
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOLANGCI_LINT_CACHE=$TMPDIR/golangci-cache \
  $TMPDIR/go/bin/golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 \
  ./pkg/jagex2/graphics/pixfont/ ./pkg/jagex2/graphics/pix3d/
```
Expected: `0 issues`.

- [ ] Native + js full builds and `go test ./...` green.

- [ ] **Browser confirmation (user, host-only):** a fresh `goscapeDumpAllocs()` (`-tags goscapedebug` build) should show `math/rand.NewSource` and `pix3d.InitPool` no longer among the churn leaders; tooltips and textured scenes render unchanged.

# wasm Allocation Quick Wins — Design

**Date:** 2026-05-25
**Branch:** rev-225
**Status:** Approved (design); spec under review

## Goal

Eliminate two churn sources found in the post-model-optimization `allocs` profile,
with no change to behavior: a per-draw rand allocation in `pixfont`, and a per-
scene-build reallocation of the `pix3d` texel pool. Combined ~96 MB of cumulative
allocation removed.

## Context

After the model-path allocation work (`NewModel6` reuse + `VertexNormal` value
slices), the wasm `allocs` profile's remaining churn includes two independent,
contained items unrelated to the larger `[]int`→`[]int32` migration (a separate
future sub-project):

- `math/rand.NewSource` — 23 MB, entirely from `pixfont.DrawStringTooltip`.
- `pix3d.InitPool` — 83 MB cumulative (~73 MB avoidable), from the
  `ClearTexels` → `InitPool` cycle on each scene rebuild.

Both are behavior-preserving optimizations. This is one spec with two independent
parts.

---

## Part 1 — `pixfont` rand reuse

### Current
`pkg/jagex2/graphics/pixfont/pixfont.go:256`, inside `DrawStringTooltip`:
```go
p.Random = rand.New(rand.NewSource(int64(arg0)))
```
Go's `rand.NewSource` allocates a ~5 KB `rngSource` (607 `int64`s) on every call.
`DrawStringTooltip` runs every frame a tooltip/jittered string is drawn, so this
is per-frame churn (~23 MB cumulative). (Java's `Random` is a single `long`; the
size mismatch is a Go-stdlib artifact, not a port bug.)

### Change
Reuse the `PixFont`'s `Random` and reseed it in place:
```go
if p.Random == nil {
    p.Random = rand.New(rand.NewSource(int64(arg0)))
} else {
    p.Random.Seed(int64(arg0))
}
```
`rand.New(rand.NewSource(seed))` and `(*rand.Rand).Seed(seed)` both seed the same
`rngSource` algorithm, producing an identical value stream. The per-draw jitter
is therefore byte-identical to before; only the allocation is removed.

`DrawStringTooltip` is the only site that assigns or reads `p.Random` (confirmed
by grep), so no other call path is affected.

### Test
`pkg/jagex2/graphics/pixfont/pixfont_test.go` (create or append): assert a
reseeded `*rand.Rand` produces the same first 10 `Int()` values as a freshly
`rand.New(rand.NewSource(seed))`-constructed one — pinning the stdlib equivalence
the optimization relies on.

---

## Part 2 — `pix3d` texel pool reuse

### Current
`pkg/jagex2/graphics/pix3d/pix3d.go`. `PoolSize` is overloaded as both pool
capacity and a free-list stack pointer: free texel buffers live in
`TexelPool[0..PoolSize-1]`, bound ones in `ActiveTexels[i]`; the `size` (=20)
buffers are conserved between the two (`GetTexels` pops + nils a slot,
`PushTexture` pushes one back). `ClearTexels` (called once per scene rebuild at
`client.go:8215`) does `TexelPool = nil`, so the following `InitPool(20)`
(`client.go:8285`, also `5799`) reallocates all 20 buffers (~10.5 MB each cycle,
~73 MB cumulative). `LowDetail` (which sets buffer length 16384 vs 65536) changes
only on init/reset paths (`client.go:1607`/`7312`), never mid-gameplay.

### Change

**`ClearTexels` — reclaim instead of free:**
```go
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
Pushing every bound buffer back leaves `TexelPool` fully populated with
`PoolSize == size` — a fresh-`InitPool` state — without allocating.

**`InitPool` — reuse when it matches, else (re)allocate:**
```go
func InitPool(size int) {
    texelLen := 65536
    if LowDetail {
        texelLen = 16384
    }
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

### Safety property
Correctness never depends on the reuse firing. The guard matches only when the
existing pool has the requested slot count and the current detail level's buffer
length; any mismatch (a `LowDetail` change, or `InitPool` called on a partially
drained pool where `TexelPool[0]` is a popped `nil` slot → `len 0`) falls through
to the unchanged `make` path. Worst case is "no reuse this once," never a
wrong-sized buffer or corruption. `len(nil)` is `0`, so the guard never panics.
The scene-rebuild sequence `ClearTexels` (8215) → `InitPool(20)` (8285) hits the
reuse path because `ClearTexels` repopulates all slots first.

### Test
`pkg/jagex2/graphics/pix3d/pix3d_test.go` (create or append), white-box (same
package):
- `LowDetail = true`; `TexelPool = nil`; `InitPool(2)`; record `&TexelPool[0][0]`.
- Simulate a bound state by moving buffers into `ActiveTexels` and decrementing
  `PoolSize` (mirroring `GetTexels`).
- `ClearTexels()`; assert `PoolSize == 2` and all `TexelPool` slots non-nil.
- `InitPool(2)`; assert `&TexelPool[0][0]` is unchanged (reused, not reallocated)
  and `ActiveTexels` is cleared.
- Separately: with `LowDetail` toggled so `texelLen` differs from the existing
  buffers, assert `InitPool` reallocates (guard correctly falls through).

---

## Files touched
- `pkg/jagex2/graphics/pixfont/pixfont.go` — rand reuse.
- `pkg/jagex2/graphics/pixfont/pixfont_test.go` — rand equivalence test.
- `pkg/jagex2/graphics/pix3d/pix3d.go` — `ClearTexels` reclaim + `InitPool` guard.
- `pkg/jagex2/graphics/pix3d/pix3d_test.go` — pool reuse test.

## Out of scope
The `[]int`→`[]int32` model array migration (separate sub-project, to be
brainstormed next). GC tuning. Any rendering-logic change.

## Verification
Native + `GOOS=js` build, `go test ./...`, golangci-lint, gofmt — all in-sandbox.
The memory win itself is confirmed by a future browser `allocs` capture, but
correctness is fully testable in-sandbox (no browser dependency).

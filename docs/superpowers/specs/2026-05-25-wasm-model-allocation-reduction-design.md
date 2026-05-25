# wasm Model-Path Allocation Reduction — Design

**Date:** 2026-05-25
**Branch:** rev-225
**Status:** Approved (design); spec under review

## Goal

Cut the dominant Go-wasm allocation in the 3D model render path so the browser's
linear-memory floor stops climbing during normal play, and GC churn drops. No
change to rendered output (bug-for-bug behavior preserved).

## Background / Evidence

A Go `allocs` profile captured from the live wasm build (`go tool pprof` on
`allocs.pb.gz`, via the `goscapeDumpAllocs()` debug hook) showed:

- **Retained heap ≈ 86 MB**, almost entirely one-time init/load structures
  (pixmap buffers, the pix3d pool, the soundfont, cache archives). Retention is
  *not* the problem.
- **Cumulative allocation ≈ 1.6 GB** — a ~19:1 churn-to-live ratio. Two sources
  dominate:
  - `model.NewModel6` — **1097 MB, 67.5% of all allocation**, entirely from
    `NpcType.GetSequencedModel` (927 MB) and `PlayerEntity.GetSequencedModel`
    (170 MB): a fresh transformed model is built **every frame for every visible
    NPC/player**. Frame-local (built, drawn, discarded; never cached — only the
    *base* model is cached in the size-30 LRU).
  - `vertexnormal.NewVertexNormal` — **2.0 M objects, 48% of all objects**, via
    `NewModel5` (loc model build, scene-retained) and `CalculateNormals` (entity
    base-model build, bursty). Each `VertexNormal` is individually heap-allocated
    because the field is `[]*VertexNormal`.

Go-wasm punishes churn twice: linear memory is **peak-driven and never returned**
to the browser, and the GC must scan 2 M tiny objects. In the Java original this
is invisible (generational GC eats transient garbage cheaply).

## Scope

Both changes ship in one spec/plan (user decision):

- **Part A** — per-entity reuse of the per-frame transformed model (`NewModel6`).
- **Part B** — `VertexNormal`/`VertexNormalOriginal` representation change
  (`[]*VertexNormal` → `[]VertexNormal`).

Out of scope: `[]int` → `[]int32` vertex storage (a separate, larger structural
project); GC tuning (`SetGCPercent`/`SetMemoryLimit`) — kept in reserve as a
later complement if still needed after these land.

---

## Part A — Per-entity reuse of the per-frame transformed model

### Current

`NpcType.GetSequencedModel` (`npctype.go:193`) and
`PlayerEntity.GetSequencedModel` (`playerentity.go:255`) call
`model.NewModel6(base, retainAlpha)` every frame. `NewModel6`
(`model.go:840`) allocates fresh `VertexX/Y/Z` int slices (deep-copied so the
animation transform doesn't corrupt the shared cached base) plus `FaceAlpha`
when `!retainAlpha`, and shares the remaining face fields by reference. The
result is transformed (`ApplyTransform`/`ApplyTransforms`/`Scale`), bounds
recomputed, returned, drawn, then discarded.

### Change

1. **Per-entity scratch model.** Add a lazily-created reusable field:
   - `NpcEntity.seqModel *model.Model`
   - `PlayerEntity.seqModel *model.Model`

2. **In-place reset primitive** on `*model.Model`:

   ```go
   // ResetFromModel6 re-initializes m as a transformable copy of src, reusing
   // m's owned vertex buffers (grown only when too small). Java: Model(Model, boolean).
   func (m *Model) ResetFromModel6(src *Model, retainAlpha bool)
   ```

   - Set `VertexCount`, `FaceCount`, `TexturedFaceCount` from `src`.
   - Ensure `cap(m.VertexX/Y/Z) >= src.VertexCount` (grow via a small
     `growInts(s, n) []int` helper that reallocates only when `cap < n`, else
     re-slices to length `n`); copy `src` verts in.
   - `FaceAlpha`: when `retainAlpha`, share `src.FaceAlpha` by reference; else
     reuse/grow `m.FaceAlpha` and copy (zero-fill if `src.FaceAlpha == nil`),
     matching `NewModel6`.
   - **Re-point every shared-reference field to `src` every call** (the base can
     change frame-to-frame with animation/appearance): `FaceInfo`, `FaceColour`,
     `FacePriority`, `Priority`, `LabelFaces`, `LabelVertices`, `FaceVertexA/B/C`,
     `FaceColourA/B/C`, `TexturedVertexA/B/C`. Enumerate against `NewModel6`'s
     field list exactly.

3. **Keep `NewModel6` as a thin wrapper** to preserve the faithful named
   constructor (nothing else calls it, but keep for fidelity):

   ```go
   func NewModel6(src *Model, retainAlpha bool) *Model {
       m := &Model{}
       m.ResetFromModel6(src, retainAlpha)
       return m
   }
   ```

4. **Thread the scratch through `GetSequencedModel`:**
   - `NpcType.GetSequencedModel(target *model.Model, arg0, arg1 int, arg2 []int) *model.Model`
     — fills `target` via `target.ResetFromModel6(base, !t.AnimHasAlpha)`, then
     `ApplyTransform(s)`/`Scale`/`CalculateBoundsCylinder` on `target`, returns it.
   - `PlayerEntity.GetSequencedModel` similarly uses `e.seqModel`.
   - Callers (`npcentity.go:60/66`, `playerentity.go`) pass the entity's
     `seqModel`, creating it on first use.

### Invariant (load-bearing)

The entity's `seqModel` is rebuilt and fully consumed within a single frame.
Frames run sequentially on the loop goroutine; the scene clears dynamic entities
each frame; `GetSequencedModel` results are never cached. Therefore no
cross-frame or cross-entity aliasing can occur. Off-screen entities hold an
idle, stable buffer (bounded, no churn).

---

## Part B — `VertexNormal` representation change

### Current

```go
VertexNormal         []*vertexnormal.VertexNormal
VertexNormalOriginal []*vertexnormal.VertexNormal
```

Each element is individually `vertexnormal.NewVertexNormal()`-allocated. Several
sites take a pointer alias and accumulate through it
(`x := m.VertexNormal[i]; x.X += …`). Confirmed across `model.go` and
`world3d.go MergeNormals` (`784-808`): **every cross-model interaction is
accumulation (`+=`) or field-copy through the alias — never pointer-identity
sharing** (no `modelB.VertexNormal[j] = modelA.VertexNormal[v]`). This is exactly
the case where `[]*T` → `[]T` is behavior-preserving.

### Change

1. Fields become value slices:

   ```go
   VertexNormal         []vertexnormal.VertexNormal
   VertexNormalOriginal []vertexnormal.VertexNormal
   ```

   Allocation: `m.VertexNormal = make([]vertexnormal.VertexNormal, n)` (zero-valued)
   — one allocation per model instead of N.

2. Rewrite every mutation-through-alias site to index form. Sites:
   - `model.go` NewModel5 copy (~799-808): `m.VertexNormal[i] = arg0.VertexNormal[i]`
     (struct copy).
   - `model.go` CalculateNormals build/accumulate/bake (~1252-1356): drop the
     per-element `NewVertexNormal()`; `var23 := m.VertexNormal[v]; var23.X += …`
     → `m.VertexNormal[v].X += …`; same for `VertexNormalOriginal`.
   - `world3d.go` MergeNormals (784-808): `normalA`/`var17` mutations →
     `modelA.VertexNormal[vertexA].X += …` / `modelB.VertexNormal[j].X += …`.
     `originalNormalA` stays a **value copy** (read-only; also immune to the
     in-loop accumulation, which is correct).

3. `nil` semantics preserved: a nil value-slice is valid; the `== nil` build
   guard, the `!= nil` guards in `world3d.go`, and the
   `m.VertexNormal = nil`/`VertexNormalOriginal = nil` free at end of
   `CalculateNormals` all behave identically.

4. Whole-slice share `m.VertexNormalOriginal = arg0.VertexNormalOriginal`
   (NewModel5) stays a slice-header share (read-only originals).

5. Remove now-unused `vertexnormal.NewVertexNormal` (only `model.go` used it).
   `VertexNormal` struct (`X,Y,Z,W int`) is unchanged.

---

## Testing

- **`ResetFromModel6`** (model package): output fields match `NewModel6` for a
  sample model; buffers reused (cap unchanged) across same-size rebuilds and
  grow on a larger src; shared fields point at `src`; `FaceAlpha` shared vs.
  owned per the `retainAlpha` flag.
- **Golden normal test**: `CalculateNormals` + `MergeNormals` on a tiny 2-model
  fixture, asserting identical numeric results to values captured from the
  pre-change pointer implementation — proves the index rewrite preserves
  accumulation order/values.
- Full `go test ./...` (native), native + `GOOS=js GOARCH=wasm` build & vet,
  golangci-lint (0 issues), gofmt.
- **Behavioral/memory verification is browser-side** on the user's host (sandbox
  cannot run wasm or read runtime memory). Success = identical visuals + a lower,
  stable `HeapSys` via `goscapeMemStats()` and a re-captured `allocs.pb.gz` with
  `NewModel6`/`NewVertexNormal` no longer dominating.

## Risks

- **Missed alias rewrite** (`x := slice[i]; x.f = v` left as a value copy)
  silently drops a write → guarded by the golden normal test and a per-site
  audit checklist in the plan.
- **`ResetFromModel6` stale shared reference**: must re-point *all* shared fields
  every call; enumerated against `NewModel6`'s exact field set.
- **Scratch under-grow**: must size off `src.VertexCount` (verts) and
  `src.FaceCount` (FaceAlpha); grow-on-demand reallocates when `cap` is short.

## Files touched

- `pkg/jagex2/graphics/model/model.go` — `ResetFromModel6`, `NewModel6` wrapper,
  `growInts`, `VertexNormal*` field types + alias rewrites, NewModel5 changes.
- `pkg/jagex2/graphics/vertexnormal/vertexnormal.go` — remove `NewVertexNormal`.
- `pkg/jagex2/dash3d/world3d/world3d.go` — MergeNormals alias rewrites.
- `pkg/jagex2/config/npctype/npctype.go` — `GetSequencedModel` target param.
- `pkg/jagex2/dash3d/entity/npcentity.go` — `seqModel` field + pass-through.
- `pkg/jagex2/dash3d/entity/playerentity/playerentity.go` — `seqModel` field +
  `GetSequencedModel` reuse.
- Tests under `pkg/jagex2/graphics/model/` (and a world3d normal fixture).

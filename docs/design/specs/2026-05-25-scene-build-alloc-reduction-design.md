# Spec A — scene-build allocation reduction round 2 (loc-model cache + Ground arrays)

**Branch:** `rev-225`
**Date:** 2026-05-25
**Status:** design, pending implementation

## Context

The wasm allocation-reduction effort earlier this session eliminated the dominant
*continuous per-frame* churn (`NewModel6` per-entity reuse + `VertexNormal`
`[]*`→`[]` value slice) and shipped two quick wins (`pixfont` rand reuse,
`pix3d` texel-pool reuse). A fresh `allocs.pb.gz` captured 2026-05-25 16:26 (after
all of the above landed) confirms those wins (`rand.NewSource` gone; `InitPool`
83 MB → 8.7 MB) and re-ranks what remains.

**Every remaining top allocator now flows from `loctype.GetModel` during scene
build (region crossings), not per frame.** Live heap is healthy (~82 MB). So this
is *region-load-hitch* territory (transient GC stutter when crossing map
boundaries), not steady-state framerate. The fresh profile (554 MB cumulative
`alloc_space`):

| Allocator | alloc_space | Runs on | Addressed here? |
|---|---|---|---|
| `Model.CalculateNormals` | 131 MB (23.7%) | dynamic-cache **miss** only | **Yes** (cache) |
| `NewModel5` | 121 MB (21.9%) | every hill-skewed placement (hit+miss) | No — **Spec B** |
| `NewModel4` | 67 MB (12.2%) | dynamic-cache **miss** only | **Yes** (cache) |
| `NewGround` | 27 MB / 27% of all **objects** | per ground tile (scene build) | **Yes** (arrays) |
| `CreateLabelReferences` | 23 MB | dynamic-cache **miss** only | **Yes** (cache) |
| `NewModel1`/`NewModel3` | 19 + 11 MB | dynamic-cache **miss** only | **Yes** (cache) |

The miss-only builders dominate because `ModelCacheDynamic` holds only **30**
entries — far smaller than a region's unique transformed-loc working set — so it
thrashes and rebuilds nearly every loc on each region crossing.

## Goal

Cut the bursty scene-build churn (region-load hitch) with two low-risk changes:

1. **Raise `ModelCacheDynamic`** so a region's unique loc-model working set fits,
   eliminating the miss-path rebuilds (`CalculateNormals`, `NewModel4`,
   `CreateLabelReferences`, `NewModel1/3`) — together ~45% of total churn.
2. **Inline `Ground.Locs` / `Ground.LocSpan`** as fixed arrays, removing two
   per-ground heap allocations (~11% of *all* allocated objects).

## Non-goals (explicitly deferred / dropped)

- **`NewModel5` cross-rebuild buffer pool — Spec B, gated behind a re-profile.**
  `NewModel5` (121 MB) is the per-tile hill-skew copy; it runs per placement
  regardless of cache size and its output is retained in the `World3D` scene
  (thousands alive at once), so it needs a pool that reclaims old-region buffers
  on scene clear — feasible but with real use-after-free risk. Designing it now
  would be against soon-to-be-stale numbers; after Spec A lands and we re-capture
  `allocs.pb.gz`, we design Spec B against the new profile and re-confirm the
  hitch still justifies the complexity.
- **`datastruct.NewLinkable` per-Ground node** (`g.DrawQueueNode`, ~8.6% of
  objects). Embedding it as a value field would change how the draw-queue linked
  list links Grounds (link via the embedded node, container-of-style access) —
  too invasive for this low-risk spec. Note for later.
- **`[]int` → `[]int32` model array migration** — evaluated and dropped earlier
  this session (high risk, browser-visual-only verification, mostly-transient
  benefit). Not revisited here.

## Change 1 — `ModelCacheDynamic` 30 → 256

`pkg/jagex2/config/loctype/loctype.go:24`, in `init()`:

```go
ModelCacheDynamic = datastruct.NewLruCache[*model.Model](256)
```

**This is a deliberate deviation from the faithful Java value of `30`**
(`Client-Java/.../config/LocType.java:88`, `new LruCache(30)`). It must carry an
inline comment documenting the intentional divergence and the reason (see
[[feedback_porting_check_java_first]] policy — faithful by default, deviate only
deliberately and documented; precedent: the documented cache-key deviation in the
storage layer). Suggested comment:

```go
// DEVIATION from Java's faithful LruCache(30) (LocType.java:88). 30 thrashes
// for a region's unique transformed-loc working set, making the miss-path
// builders (CalculateNormals/NewModel4/CreateLabelReferences) ~45% of scene-build
// allocation churn. 256 holds the working set; render-identical (the cache key
// encodes every transform parameter, so eviction only decides whether we
// rebuild, never what we render). Worst-case retained ~6 MB vs an ~82 MB live
// heap. See docs/superpowers/specs/2026-05-25-scene-build-alloc-reduction-design.md.
```

`ModelCacheStatic` stays at the faithful **500** (not a churn driver; only the
dynamic cache thrashes).

### Why this is render-identical (not just memory)

The dynamic cache key (`var10`) encodes shape + rotation + recolour + resize +
offset — every parameter that affects the built model. A larger cache only
retains more entries; eviction decides whether `GetModel` *rebuilds* a model, not
what the model looks like. For non-hill-skew / non-shared-light locs the cached
instance is returned directly (shared across placements, already the case at 30);
for hill-skew/shared-light locs the per-tile `NewModel5` copy still runs on both
hit and miss (unchanged). So output is bit-identical at any cache size ≥ 1.

### Risk / cost

- **Retained memory:** up to 226 extra cached models. Dynamic models share most
  arrays with their static base; worst case ~6 MB against an ~82 MB live heap.
  Negligible, and we confirm via the post-A `inuse_space` re-profile.
- **Correctness:** none beyond the above — pure capacity knob.

## Change 2 — `Ground.Locs` / `Ground.LocSpan` → fixed arrays

`pkg/jagex2/dash3d/typ/ground.go`:

```go
// struct fields (lines 17-18)
Locs    [5]*Location   // Java: Location[] locs = new Location[5] (Ground.java:43)
LocSpan [5]int         // Java: int[] locSpan = new int[5]       (Ground.java:46)
```

and delete the two allocations in `NewGround` (lines 41-42):

```go
g.Locs = make([]*Location, 5)     // REMOVE — now a fixed array field
g.LocSpan = make([]int, 5)        // REMOVE
```

### Why this is safe and faithful

- **Faithful:** Java declares these as fixed-size arrays (`new Location[5]`,
  `new int[5]`). The Go slice port was *less* faithful; fixed arrays match Java
  exactly. This is a fidelity improvement, **not** a deviation.
- **Behavior-identical:** all 18 use sites (audited in `world3d.go` and
  `ground.go`) are pure element index reads/writes through a `*Ground` pointer
  (`tile.Locs[i]`, `var5.Locs[l] = var5.Locs[l+1]`, `var5.Locs[n] = nil`,
  `tile.LocSpan[i] & ...`). No `len()`, no `append`, no slice assignment, no
  passing the field to a `[]`-typed parameter. Array index access through a
  pointer is identical to slice index access, and `nil` is a valid `[5]*Location`
  element. `Ground` is always heap-allocated via `&g`, so the arrays live inline
  in the same allocation — removing two allocations per ground with zero call-site
  changes.

### Risk / cost

- Removes ~141K objects/profile (~11% of all allocated objects). Tiny increase in
  the `Ground` struct's own size (now holds the arrays inline) — already heap
  allocated, so net allocation count drops by 2 per ground.

## Verification plan

In-sandbox gates (all must pass):

- `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go build ./...` (native)
- `GOOS=js GOARCH=wasm ... go build ./...` (wasm)
- `... go test ./...` (full suite; `loctype` and `world3d` tests exercise both
  changed files)
- golangci-lint (pinned v2.12.2, `--max-issues-per-linter=0 --max-same-issues=0`)
- `gofmt` clean

No new automated test is strictly required — both changes are behavior-preserving
and existing `loctype`/`world3d` tests cover the touched code. (If a cheap
characterization test for "cache returns same instance for repeated key" is easy,
add it; otherwise rely on existing coverage.)

Host verification (user):

- Build `-tags goscapedebug`, run in browser, cross several region boundaries,
  capture a fresh `allocs.pb.gz`.
- **Expected:** `CalculateNormals` / `NewModel4` / `CreateLabelReferences` /
  `NewModel1/3` collapse toward zero after the first region (cache now holds the
  working set); `NewGround` object count drops by ~2 allocations per ground;
  `inuse_space` rises only a few MB. Visuals **identical**.
- This fresh profile becomes the input for the Spec B (NewModel5 pool) decision.

## Out-of-scope reminder

Spec A is intentionally the two low-risk wins only. `NewModel5` (the single
biggest remaining churner) is **Spec B**, designed later against the post-A
profile.

# Scene-build Allocation Reduction (Spec A) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cut bursty scene-build (region-load) allocation churn with two behavior-preserving changes — inline `Ground.Locs`/`LocSpan` as fixed arrays, and raise the loc-model dynamic cache from 30 to 256.

**Architecture:** Both changes are in the loc/scenery scene-build path. (1) `Ground.Locs`/`LocSpan` become fixed `[5]` array fields living inline in the heap-allocated `Ground`, removing two `make` calls per ground (faithful to Java's `new[5]`). (2) `ModelCacheDynamic` capacity 30→256 so a region's unique transformed-loc working set fits, eliminating the miss-path rebuilds (`CalculateNormals`/`NewModel4`/`CreateLabelReferences`/`NewModel1/3`) — a documented deviation from Java's faithful `30`, render-identical because the cache key encodes every transform parameter.

**Tech Stack:** Go (native + `GOOS=js GOARCH=wasm`); `datastruct.LruCache[*model.Model]`; golangci-lint v2.12.2.

**Reference spec:** `docs/superpowers/specs/2026-05-25-scene-build-alloc-reduction-design.md`

**These are behavior-preserving changes** (a refactor and a capacity constant). There is no new behavior to test-drive, so verification is regression-oriented: the existing `loctype`/`world3d` test suites must still pass and **both** build targets must compile (a slice→array mistake such as an unspotted `len()`/`append` would fail to compile). Do **not** invent new unit tests for these — they would add no signal.

**Go command prefix (sandbox):** every `go` invocation below is shown with the required prefix `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache`. Lint uses `GOLANGCI_LINT_CACHE=$TMPDIR/golangci-cache`.

**Commits:** use `git commit --no-gpg-sign` (works in-sandbox; bypasses the GPG-agent read-only issue).

---

## Task 1: Inline `Ground.Locs` / `Ground.LocSpan` as fixed arrays

**Files:**
- Modify: `pkg/jagex2/dash3d/typ/ground.go:17-18` (field types) and `:41-42` (remove allocations)

This is faithful to Java (`Ground.java:43` `Location[] locs = new Location[5]`, `:46` `int[] locSpan = new int[5]`) — the slice port was *less* faithful. All 18 use sites in `world3d.go`/`ground.go` are pure element index reads/writes through a `*Ground` pointer (no `len()`, `append`, slice assignment, or `[]`-parameter passing), so the change is behavior-identical and requires no call-site edits.

- [ ] **Step 1: Change the field declarations to fixed arrays**

In `pkg/jagex2/dash3d/typ/ground.go`, replace lines 17-18:

```go
	Locs                 []*Location
	LocSpan              []int
```

with:

```go
	Locs                 [5]*Location // Java: Location[] locs = new Location[5] (Ground.java:43)
	LocSpan              [5]int       // Java: int[] locSpan = new int[5] (Ground.java:46)
```

- [ ] **Step 2: Remove the now-redundant allocations in `NewGround`**

In the same file, delete lines 41-42 (the two `make` calls). After the edit, `NewGround` reads:

```go
func NewGround(level, x, z int) *Ground {
	var g Ground

	g.Level = level
	g.OccludeLevel = g.Level
	g.X = x
	g.Z = z

	g.DrawQueueNode = datastruct.NewLinkable(&g)

	return &g
}
```

(The fixed arrays are zero-valued inline in `var g Ground` — `[5]*Location` elements are `nil`, `[5]int` elements are `0` — exactly what `make` produced.)

- [ ] **Step 3: gofmt the file**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache gofmt -w pkg/jagex2/dash3d/typ/ground.go`
Expected: no output (file already formatted, or re-aligned struct tags).

- [ ] **Step 4: Build native + wasm (this is the real safety check for slice→array)**

Run:
```bash
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go build ./...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOOS=js GOARCH=wasm go build ./...
```
Expected: both succeed with no output. A compile error here (e.g. `cannot use ... (variable of type [5]int)`) means a use site relied on slice semantics — stop and reassess against the spec's use-site audit.

- [ ] **Step 5: Run the affected test suites**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./pkg/jagex2/dash3d/... ./pkg/jagex2/graphics/...`
Expected: `ok` for all packages (notably `dash3d/world3d` and `dash3d/typ`).

- [ ] **Step 6: Commit**

```bash
git add pkg/jagex2/dash3d/typ/ground.go
git commit --no-gpg-sign -m "perf(ground): inline Locs/LocSpan as fixed [5] arrays (no per-ground make)

Faithful to Java (Ground.java:43/46 use new[5]); removes two heap allocations
per ground tile (~11% of all allocated objects in the scene-build profile).
Behavior-identical: all use sites are pure index access through *Ground.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Raise `ModelCacheDynamic` 30 → 256

**Files:**
- Modify: `pkg/jagex2/config/loctype/loctype.go:24`

This is a **documented deviation** from Java's faithful `LruCache(30)` (`LocType.java:88`). The comment is mandatory — it records the intentional divergence and why (project policy: faithful by default, deviate only deliberately and documented).

- [ ] **Step 1: Bump the capacity and add the deviation comment**

In `pkg/jagex2/config/loctype/loctype.go`, replace line 24:

```go
	ModelCacheDynamic = datastruct.NewLruCache[*model.Model](30)
```

with:

```go
	// DEVIATION from Java's faithful LruCache(30) (LocType.java:88). 30 thrashes
	// for a region's unique transformed-loc working set, making the miss-path
	// builders (CalculateNormals/NewModel4/CreateLabelReferences) ~45% of
	// scene-build allocation churn. 256 holds the working set; render-identical
	// (the cache key encodes every transform parameter, so eviction only decides
	// whether we rebuild, never what we render). Worst-case retained ~6 MB vs an
	// ~82 MB live heap. See docs/superpowers/specs/2026-05-25-scene-build-alloc-reduction-design.md.
	ModelCacheDynamic = datastruct.NewLruCache[*model.Model](256)
```

(Leave `ModelCacheStatic` at the faithful `500` on line 23 — it is not a churn driver.)

- [ ] **Step 2: gofmt the file**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache gofmt -w pkg/jagex2/config/loctype/loctype.go`
Expected: no output.

- [ ] **Step 3: Build native + wasm**

Run:
```bash
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go build ./...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOOS=js GOARCH=wasm go build ./...
```
Expected: both succeed with no output.

- [ ] **Step 4: Run the loctype tests**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./pkg/jagex2/config/loctype/...`
Expected: `ok` (or `no test files` if the package has none — either is acceptable; the constant change cannot regress logic).

- [ ] **Step 5: Commit**

```bash
git add pkg/jagex2/config/loctype/loctype.go
git commit --no-gpg-sign -m "perf(loctype): raise ModelCacheDynamic 30->256 (stop region-load thrashing)

Documented deviation from Java's LruCache(30) (LocType.java:88). 30 thrashes for
a region's unique transformed-loc working set; the miss-path builders
(CalculateNormals/NewModel4/CreateLabelReferences) were ~45% of scene-build
alloc churn. Render-identical (key encodes all transform params); ~6 MB worst-case
retained vs ~82 MB live heap.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Full-gate verification

**Files:** none (verification only).

Run the complete in-sandbox gate suite over the whole repo to confirm both changes together are clean.

- [ ] **Step 1: gofmt check (whole repo)**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache gofmt -l pkg cmd`
Expected: no output (no unformatted files).

- [ ] **Step 2: Build both targets**

Run:
```bash
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go build ./...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOOS=js GOARCH=wasm go build ./...
```
Expected: both succeed, no output.

- [ ] **Step 3: Full test suite**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./...`
Expected: all packages `ok` / `no test files`. No `FAIL`.

- [ ] **Step 4: Lint (pinned golangci-lint v2.12.2, no truncation)**

Run:
```bash
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOLANGCI_LINT_CACHE=$TMPDIR/golangci-cache \
  go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run \
  --max-issues-per-linter=0 --max-same-issues=0 ./...
```
Expected: `0 issues`. (If the standard config flags the new deviation comment line length or similar, fix formatting; do not weaken the comment content.)

- [ ] **Step 5: No commit needed**

Task 3 changes no files. If any gate failed, return to the responsible task and fix before proceeding.

---

## Host verification handoff (user, not an agent task)

After the plan lands, the user re-captures a profile on the host to confirm the win and to seed the Spec B (NewModel5 pool) decision:

1. `make wasm-debug` then `make wasm-serve`; open the browser, log in, cross several region boundaries.
2. In the browser console: `goscapeDumpAllocs()` → downloads `allocs.pb.gz`; place it at the repo root.

**Expected in the fresh profile:** `CalculateNormals` / `NewModel4` / `CreateLabelReferences` / `NewModel1`/`NewModel3` collapse toward zero after the first region (dynamic cache now holds the working set); `NewGround` drops 2 allocations per ground; `inuse_space` rises only a few MB; visuals identical. This profile is the input to the Spec B re-profile gate.

---

## Self-review notes

- **Spec coverage:** Spec Change 1 (cache 30→256 + documented comment) → Task 2. Spec Change 2 (`Ground` fixed arrays + remove makes) → Task 1. Verification plan (native+wasm build, full test, lint, gofmt) → embedded per-task + consolidated in Task 3. Host re-profile → handoff section. No gaps.
- **No new automated test:** intentional and stated — both changes are behavior-preserving and covered by existing `loctype`/`world3d` suites; a new test would add no signal (per the spec's verification section, which left it optional).
- **Out of scope (unchanged from spec):** `NewModel5` pool (Spec B, behind re-profile gate), `NewLinkable` per-Ground embedding, `[]int`→`[]int32` migration (dropped).

# wasm Model-Path Allocation Reduction — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate the dominant per-frame model-copy allocation (`NewModel6`) and the 2M-object `VertexNormal` churn in the 3D render path, with no change to rendered output.

**Architecture:** (A) Each NPC/player entity owns a reusable transformed `Model`; `GetSequencedModel` rewrites it in place via a new `ResetFromModel6` instead of allocating fresh vertex buffers every frame. (B) `Model.VertexNormal`/`VertexNormalOriginal` change from `[]*VertexNormal` to `[]VertexNormal` (one slice alloc per model instead of N), with all accumulate-through-alias sites rewritten to index form.

**Tech Stack:** Go (native + `GOOS=js GOARCH=wasm`), standard `testing`. Reference: spec `docs/superpowers/specs/2026-05-25-wasm-model-allocation-reduction-design.md`.

**Conventions (every command + commit):**
- Go commands are prefixed `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache`.
- Commits use `git commit --no-gpg-sign`.
- After each task, both builds must pass: native (`go build ./...`) and js (`GOOS=js GOARCH=wasm go build ./...`).

---

## Task 1: `ResetFromModel6` in-place model copy

**Files:**
- Modify: `pkg/jagex2/graphics/model/model.go` (replace `NewModel6` at lines 840-883; add `growInts` + `ResetFromModel6`)
- Test: `pkg/jagex2/graphics/model/model_test.go` (append)

- [ ] **Step 1: Write the failing tests**

Append to `pkg/jagex2/graphics/model/model_test.go`:

```go
// sampleBaseModel builds a minimal base model with the fields NewModel6 reads.
func sampleBaseModel(vertexCount, faceCount int) *Model {
	m := &Model{VertexCount: vertexCount, FaceCount: faceCount, TexturedFaceCount: 0}
	m.VertexX = make([]int, vertexCount)
	m.VertexY = make([]int, vertexCount)
	m.VertexZ = make([]int, vertexCount)
	for i := range vertexCount {
		m.VertexX[i] = i * 1
		m.VertexY[i] = i * 2
		m.VertexZ[i] = i * 3
	}
	m.FaceAlpha = make([]int, faceCount)
	for i := range faceCount {
		m.FaceAlpha[i] = i + 7
	}
	m.FaceColour = make([]int, faceCount)
	m.FaceVertexA = make([]int, faceCount)
	return m
}

func TestResetFromModel6CopiesVertsDeeply(t *testing.T) {
	src := sampleBaseModel(4, 2)
	var m Model
	m.ResetFromModel6(src, true)
	if m.VertexCount != 4 || m.FaceCount != 2 {
		t.Fatalf("counts = %d/%d, want 4/2", m.VertexCount, m.FaceCount)
	}
	m.VertexX[0] = 999
	if src.VertexX[0] == 999 {
		t.Error("VertexX not deep-copied: writing target mutated src")
	}
}

func TestResetFromModel6SharesFaceRefs(t *testing.T) {
	src := sampleBaseModel(4, 2)
	var m Model
	m.ResetFromModel6(src, true)
	// FaceColour is a shared reference (NewModel6 shares it).
	src.FaceColour[0] = 1234
	if m.FaceColour[0] != 1234 {
		t.Error("FaceColour should share src's backing array")
	}
}

func TestResetFromModel6AlphaShareVsOwn(t *testing.T) {
	src := sampleBaseModel(4, 2)
	var shared Model
	shared.ResetFromModel6(src, true) // retainAlpha -> share
	src.FaceAlpha[0] = 4321
	if shared.FaceAlpha[0] != 4321 {
		t.Error("retainAlpha=true should share src.FaceAlpha")
	}
	var owned Model
	owned.ResetFromModel6(src, false) // !retainAlpha -> own copy
	src.FaceAlpha[1] = 8888
	if owned.FaceAlpha[1] == 8888 {
		t.Error("retainAlpha=false should deep-copy FaceAlpha")
	}
}

func TestResetFromModel6ReusesBuffers(t *testing.T) {
	src := sampleBaseModel(4, 2)
	var m Model
	m.ResetFromModel6(src, false)
	capX := cap(m.VertexX)
	ptr := &m.VertexX[0]
	m.ResetFromModel6(src, false) // same size again
	if cap(m.VertexX) != capX || &m.VertexX[0] != ptr {
		t.Error("same-size rebuild reallocated VertexX instead of reusing it")
	}
	bigger := sampleBaseModel(64, 2)
	m.ResetFromModel6(bigger, false)
	if len(m.VertexX) != 64 {
		t.Fatalf("len after grow = %d, want 64", len(m.VertexX))
	}
}

func TestResetFromModel6ClearsStaleFields(t *testing.T) {
	src := sampleBaseModel(4, 2)
	var m Model
	m.Pickable = true
	m.VertexNormal = make([]vertexnormal.VertexNormal, 4)
	m.MaxY = 555
	m.ResetFromModel6(src, true)
	if m.Pickable || m.VertexNormal != nil || m.MaxY != 0 {
		t.Errorf("stale fields not cleared: Pickable=%v VertexNormal=%v MaxY=%d",
			m.Pickable, m.VertexNormal != nil, m.MaxY)
	}
}
```

Note: `TestResetFromModel6ClearsStaleFields` references `vertexnormal.VertexNormal` as a value type — that becomes valid in Task 3. Until Task 3, write that one field as `make([]*vertexnormal.VertexNormal, 4)` and the assertion `m.VertexNormal != nil`; flip it to the value form in Task 3 Step 4. Ensure `model_test.go` imports `"github.com/zsrv/goscape-client/pkg/jagex2/graphics/vertexnormal"`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./pkg/jagex2/graphics/model/ -run TestResetFromModel6 -v`
Expected: FAIL — `m.ResetFromModel6 undefined`.

- [ ] **Step 3: Implement `growInts` + `ResetFromModel6`, and reduce `NewModel6` to a wrapper**

In `pkg/jagex2/graphics/model/model.go`, replace the whole `NewModel6` function (currently lines 840-883) with:

```go
// growInts returns a length-n slice reusing s's backing array when cap allows,
// else a fresh allocation. Used to reuse per-frame model buffers.
func growInts(s []int, n int) []int {
	if cap(s) >= n {
		return s[:n]
	}
	return make([]int, n)
}

// ResetFromModel6 re-initializes m as a transformable copy of src, reusing m's
// owned vertex (and FaceAlpha) backing arrays so per-frame rebuilds don't
// allocate. The struct is cleared first so no field carries over from a prior
// frame (matching a fresh NewModel6), then the owned arrays are restored.
// Shared, read-only fields are re-pointed to src on every call because the base
// model can change frame-to-frame. retainAlpha must be constant for a given
// reused target (it is per entity/type). Java: Model(Model, boolean).
func (m *Model) ResetFromModel6(src *Model, retainAlpha bool) {
	vx, vy, vz, fa := m.VertexX, m.VertexY, m.VertexZ, m.FaceAlpha
	*m = Model{}

	m.VertexCount = src.VertexCount
	m.FaceCount = src.FaceCount
	m.TexturedFaceCount = src.TexturedFaceCount

	m.VertexX = growInts(vx, m.VertexCount)
	m.VertexY = growInts(vy, m.VertexCount)
	m.VertexZ = growInts(vz, m.VertexCount)
	for i := range m.VertexCount {
		m.VertexX[i] = src.VertexX[i]
		m.VertexY[i] = src.VertexY[i]
		m.VertexZ[i] = src.VertexZ[i]
	}

	if retainAlpha {
		m.FaceAlpha = src.FaceAlpha
	} else {
		fa = growInts(fa, m.FaceCount)
		if src.FaceAlpha == nil {
			for i := range m.FaceCount {
				fa[i] = 0
			}
		} else {
			for i := range m.FaceCount {
				fa[i] = src.FaceAlpha[i]
			}
		}
		m.FaceAlpha = fa
	}

	m.FaceInfo = src.FaceInfo
	m.FaceColour = src.FaceColour
	m.FacePriority = src.FacePriority
	m.Priority = src.Priority
	m.LabelFaces = src.LabelFaces
	m.LabelVertices = src.LabelVertices
	m.FaceVertexA = src.FaceVertexA
	m.FaceVertexB = src.FaceVertexB
	m.FaceVertexC = src.FaceVertexC
	m.FaceColourA = src.FaceColourA
	m.FaceColourB = src.FaceColourB
	m.FaceColourC = src.FaceColourC
	m.TexturedVertexA = src.TexturedVertexA
	m.TexturedVertexB = src.TexturedVertexB
	m.TexturedVertexC = src.TexturedVertexC
}

// NewModel6 allocates a fresh transformable copy of src. Hot paths reuse a
// target via ResetFromModel6 instead. Java: Model(Model, boolean).
func NewModel6(src *Model, retainAlpha bool) *Model {
	m := &Model{}
	m.ResetFromModel6(src, retainAlpha)
	return m
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./pkg/jagex2/graphics/model/ -run TestResetFromModel6 -v`
Expected: PASS (all 5).

- [ ] **Step 5: Full build + suite + gofmt**

Run:
```
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go build ./...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOOS=js GOARCH=wasm go build ./...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./pkg/jagex2/graphics/model/
gofmt -l pkg/jagex2/graphics/model/model.go pkg/jagex2/graphics/model/model_test.go
```
Expected: builds OK, tests `ok`, gofmt prints nothing.

- [ ] **Step 6: Commit**

```bash
git add pkg/jagex2/graphics/model/model.go pkg/jagex2/graphics/model/model_test.go
git commit --no-gpg-sign -m "perf(model): add ResetFromModel6 in-place copy; NewModel6 wraps it"
```

---

## Task 2: Per-entity reuse of the sequenced model

**Files:**
- Modify: `pkg/jagex2/config/npctype/npctype.go:172-208` (`GetSequencedModel` gains a `target` param)
- Modify: `pkg/jagex2/dash3d/entity/npcentity.go:10-14,53-69` (add `seqModel`, pass it through)
- Modify: `pkg/jagex2/dash3d/entity/playerentity/playerentity.go:24-47,255` (add `seqModel`, reuse it)

- [ ] **Step 1: Change `NpcType.GetSequencedModel` to fill a caller-provided target**

In `pkg/jagex2/config/npctype/npctype.go`, change the signature and the `NewModel6` line. Replace:

```go
func (t *NpcType) GetSequencedModel(arg0 int, arg1 int, arg2 []int) *model.Model {
```
with:
```go
func (t *NpcType) GetSequencedModel(target *model.Model, arg0 int, arg1 int, arg2 []int) *model.Model {
```

And replace (line 193):
```go
	var4 := model.NewModel6(var5, !t.AnimHasAlpha)
```
with:
```go
	target.ResetFromModel6(var5, !t.AnimHasAlpha)
	var4 := target
```

Everything else in the function (the `ApplyTransforms`/`Scale`/`CalculateBoundsCylinder` on `var4`, the `return var4`) is unchanged.

- [ ] **Step 2: Add `seqModel` to `NpcEntity` and pass it through**

In `pkg/jagex2/dash3d/entity/npcentity.go`, change the struct (lines 10-14):

```go
type NpcEntity struct {
	PathingEntity

	Type     *npctype.NpcType
	seqModel *model.Model // reused per-frame transformed model (avoids per-frame alloc)
}
```

In `NpcEntity.GetSequencedModel` (lines 53-69), create the scratch once and pass it to both call sites. Replace the two `e.Type.GetSequencedModel(...)` calls:

```go
	return e.Type.GetSequencedModel(var2, var4, seqtype.Instances[e.PrimarySeqID].WalkMerge)
```
→
```go
	if e.seqModel == nil {
		e.seqModel = &model.Model{}
	}
	return e.Type.GetSequencedModel(e.seqModel, var2, var4, seqtype.Instances[e.PrimarySeqID].WalkMerge)
```

and:
```go
	var3 := e.Type.GetSequencedModel(var2, -1, nil)
```
→
```go
	if e.seqModel == nil {
		e.seqModel = &model.Model{}
	}
	var3 := e.Type.GetSequencedModel(e.seqModel, var2, -1, nil)
```

- [ ] **Step 3: Add `seqModel` to `PlayerEntity` and reuse it**

In `pkg/jagex2/dash3d/entity/playerentity/playerentity.go`, add a field to the struct (after `LocModel *model.Model` at line 41):

```go
	seqModel           *model.Model // reused per-frame transformed model
```

Replace line 255:
```go
	var16 := model.NewModel6(var15, true)
```
with:
```go
	if e.seqModel == nil {
		e.seqModel = &model.Model{}
	}
	e.seqModel.ResetFromModel6(var15, true)
	var16 := e.seqModel
```

(The `e.LowMemory` early return at line 252 keeps returning the cached `var15` directly — unchanged.)

- [ ] **Step 4: Build + vet + suite**

Run:
```
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go build ./...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOOS=js GOARCH=wasm go build ./...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go vet ./pkg/jagex2/config/npctype/ ./pkg/jagex2/dash3d/entity/...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./...
```
Expected: all builds OK; vet clean; `go test` all `ok`. (Compile error here means a `GetSequencedModel` caller was missed — there are exactly the two in `npcentity.go`.)

- [ ] **Step 5: gofmt + commit**

```bash
gofmt -l pkg/jagex2/config/npctype/npctype.go pkg/jagex2/dash3d/entity/npcentity.go pkg/jagex2/dash3d/entity/playerentity/playerentity.go
git add pkg/jagex2/config/npctype/npctype.go pkg/jagex2/dash3d/entity/npcentity.go pkg/jagex2/dash3d/entity/playerentity/playerentity.go
git commit --no-gpg-sign -m "perf(entity): reuse per-frame sequenced model buffers (no per-frame NewModel6 alloc)"
```

---

## Task 3: `VertexNormal` value-slice representation

**Files:**
- Modify: `pkg/jagex2/graphics/model/model.go` (struct fields 167-168; `NewModel5` 799-809; `CalculateNormals` 1252-1316)
- Modify: `pkg/jagex2/graphics/vertexnormal/vertexnormal.go` (remove `NewVertexNormal`)
- Modify: `pkg/jagex2/dash3d/world3d/world3d.go` (`MergeNormals` 783-815)
- Test: `pkg/jagex2/graphics/model/model_test.go` (append normal-accumulation test)
- Test: `pkg/jagex2/dash3d/world3d/world3d_test.go` (create; merge test)

This task is **atomic**: the field type change is silent to the compiler (a value copy still compiles), so every accumulate-through-alias site must be fixed together or normals are silently wrong. The two characterization tests are the guard.

- [ ] **Step 1: Write the failing characterization tests (hand-computed values)**

Append to `pkg/jagex2/graphics/model/model_test.go`:

```go
// One face (0,0,0)-(100,0,0)-(0,0,100): the face normal reduces to (0,-256,0)
// and each of the 3 vertices accumulates it once (W=1). Computed by hand from
// CalculateNormals' cross-product + normalize-to-256 math.
func TestCalculateNormalsSingleFace(t *testing.T) {
	m := &Model{VertexCount: 3, FaceCount: 1}
	m.VertexX = []int{0, 100, 0}
	m.VertexY = []int{0, 0, 0}
	m.VertexZ = []int{0, 0, 100}
	m.FaceVertexA = []int{0}
	m.FaceVertexB = []int{1}
	m.FaceVertexC = []int{2}
	m.FaceColour = []int{0}

	m.CalculateNormals(64, 850, -30, -50, -30, false) // arg5=false: keep normals, build Original

	for i := range 3 {
		n := m.VertexNormal[i]
		if n.X != 0 || n.Y != -256 || n.Z != 0 || n.W != 1 {
			t.Errorf("VertexNormal[%d] = %+v, want {0 -256 0 1}", i, n)
		}
		o := m.VertexNormalOriginal[i]
		if o != n {
			t.Errorf("VertexNormalOriginal[%d] = %+v, want %+v", i, o, n)
		}
	}
}
```

Create `pkg/jagex2/dash3d/world3d/world3d_test.go`:

```go
package world3d

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/model"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/vertexnormal"
)

// MergeNormals cross-accumulates the ORIGINAL normals of coincident vertices:
// modelA's vertex gets modelB's original added, and vice-versa. One coincident
// vertex pair -> each side's VertexNormal becomes its own value plus the other's
// original. (merged < 3 so the face-flatten pass is skipped.)
func TestMergeNormalsAccumulatesCoincident(t *testing.T) {
	mkModel := func() *model.Model {
		m := &model.Model{VertexCount: 1, FaceCount: 0}
		m.VertexX = []int{0}
		m.VertexY = []int{0}
		m.VertexZ = []int{0}
		m.VertexNormal = []vertexnormal.VertexNormal{{X: 0, Y: 0, Z: 0, W: 0}}
		m.VertexNormalOriginal = []vertexnormal.VertexNormal{{X: 0, Y: 0, Z: 0, W: 0}}
		return m
	}
	a := mkModel()
	a.VertexNormal[0] = vertexnormal.VertexNormal{X: 10, Y: 0, Z: 0, W: 1}
	a.VertexNormalOriginal[0] = vertexnormal.VertexNormal{X: 10, Y: 0, Z: 0, W: 1}
	b := mkModel()
	b.VertexNormal[0] = vertexnormal.VertexNormal{X: 0, Y: 20, Z: 0, W: 1}
	b.VertexNormalOriginal[0] = vertexnormal.VertexNormal{X: 0, Y: 20, Z: 0, W: 1}

	w := &World3D{MergeIndexA: make([]int, 1), MergeIndexB: make([]int, 1)}
	w.MergeNormals(a, b, 0, 0, 0, false)

	if got := a.VertexNormal[0]; got != (vertexnormal.VertexNormal{X: 10, Y: 20, Z: 0, W: 2}) {
		t.Errorf("a.VertexNormal[0] = %+v, want {10 20 0 2}", got)
	}
	if got := b.VertexNormal[0]; got != (vertexnormal.VertexNormal{X: 10, Y: 20, Z: 0, W: 2}) {
		t.Errorf("b.VertexNormal[0] = %+v, want {10 20 0 2}", got)
	}
}
```

- [ ] **Step 2: Run — expect compile failure**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./pkg/jagex2/graphics/model/ ./pkg/jagex2/dash3d/world3d/ 2>&1 | head`
Expected: FAIL — the tests use `[]vertexnormal.VertexNormal` value literals, which don't match the current `[]*VertexNormal` fields (compile error). This confirms the tests pin the new representation.

- [ ] **Step 3: Change the struct fields**

In `pkg/jagex2/graphics/model/model.go` lines 167-168:

```go
	VertexNormal         []vertexnormal.VertexNormal
	VertexNormalOriginal []vertexnormal.VertexNormal
```

- [ ] **Step 4: Rewrite `NewModel5` normal copy (lines 799-809)**

Replace:
```go
		m.VertexNormal = make([]*vertexnormal.VertexNormal, m.VertexCount)
		for i := range m.VertexCount {
			m.VertexNormal[i] = vertexnormal.NewVertexNormal()
			var7 := m.VertexNormal[i]
			var8 := arg0.VertexNormal[i]
			var7.X = var8.X
			var7.Y = var8.Y
			var7.Z = var8.Z
			var7.W = var8.W
		}
		m.VertexNormalOriginal = arg0.VertexNormalOriginal
```
with:
```go
		m.VertexNormal = make([]vertexnormal.VertexNormal, m.VertexCount)
		for i := range m.VertexCount {
			m.VertexNormal[i] = arg0.VertexNormal[i]
		}
		m.VertexNormalOriginal = arg0.VertexNormalOriginal
```

Also in Task 1's `TestResetFromModel6ClearsStaleFields`, change the stale field setup to the value form: `m.VertexNormal = make([]vertexnormal.VertexNormal, 4)` (was `[]*vertexnormal.VertexNormal`).

- [ ] **Step 5: Rewrite `CalculateNormals` allocation + accumulation + Original copy**

In `CalculateNormals`, replace the build block (lines 1252-1257):
```go
	if m.VertexNormal == nil {
		m.VertexNormal = make([]*vertexnormal.VertexNormal, m.VertexCount)
		for i := range m.VertexCount {
			m.VertexNormal[i] = vertexnormal.NewVertexNormal()
		}
	}
```
with:
```go
	if m.VertexNormal == nil {
		m.VertexNormal = make([]vertexnormal.VertexNormal, m.VertexCount)
	}
```

Replace the accumulate block (lines 1284-1298):
```go
			var23 := m.VertexNormal[var10]
			var23.X += var19
			var23.Y += var20
			var23.Z += var21
			var23.W++
			var26 := m.VertexNormal[var11]
			var26.X += var19
			var26.Y += var20
			var26.Z += var21
			var26.W++
			var27 := m.VertexNormal[var12]
			var27.X += var19
			var27.Y += var20
			var27.Z += var21
			var27.W++
```
with:
```go
			m.VertexNormal[var10].X += var19
			m.VertexNormal[var10].Y += var20
			m.VertexNormal[var10].Z += var21
			m.VertexNormal[var10].W++
			m.VertexNormal[var11].X += var19
			m.VertexNormal[var11].Y += var20
			m.VertexNormal[var11].Z += var21
			m.VertexNormal[var11].W++
			m.VertexNormal[var12].X += var19
			m.VertexNormal[var12].Y += var20
			m.VertexNormal[var12].Z += var21
			m.VertexNormal[var12].W++
```

Replace the `VertexNormalOriginal` build (lines 1307-1316):
```go
		m.VertexNormalOriginal = make([]*vertexnormal.VertexNormal, m.VertexCount)
		for i := range m.VertexCount {
			var24 := m.VertexNormal[i]
			m.VertexNormalOriginal[i] = vertexnormal.NewVertexNormal()
			var25 := m.VertexNormalOriginal[i]
			var25.X = var24.X
			var25.Y = var24.Y
			var25.Z = var24.Z
			var25.W = var24.W
		}
```
with:
```go
		m.VertexNormalOriginal = make([]vertexnormal.VertexNormal, m.VertexCount)
		for i := range m.VertexCount {
			m.VertexNormalOriginal[i] = m.VertexNormal[i]
		}
```

`ApplyLighting` (lines 1325-1354) reads normals only (`var10 := m.VertexNormal[var7]` then reads `var10.X` etc.); a value copy reads identically, so **leave it unchanged**.

- [ ] **Step 6: Remove `NewVertexNormal`**

In `pkg/jagex2/graphics/vertexnormal/vertexnormal.go`, delete:
```go
func NewVertexNormal() *VertexNormal {
	return new(VertexNormal)
}
```
Keep the `VertexNormal` struct.

- [ ] **Step 7: Rewrite `MergeNormals` accumulation (world3d.go lines 783-815)**

Replace the outer loop body. Current:
```go
	for vertexA := range modelA.VertexCount {
		normalA := modelA.VertexNormal[vertexA]
		originalNormalA := modelA.VertexNormalOriginal[vertexA]

		if originalNormalA.W != 0 {
			y := modelA.VertexY[vertexA] - offsetY
			if y <= modelB.MinY {
				x := modelA.VertexX[vertexA] - arg2
				if x >= modelB.MinX && x <= modelB.MaxX {
					z := modelA.VertexZ[vertexA] - arg4
					if z >= modelB.MinZ && z <= modelB.MaxZ {
						for j := range vertexCountB {
							var17 := modelB.VertexNormal[j]
							var18 := modelB.VertexNormalOriginal[j]
							if x == vertexX[j] && z == modelB.VertexZ[j] && y == modelB.VertexY[j] && var18.W != 0 {
								normalA.X += var18.X
								normalA.Y += var18.Y
								normalA.Z += var18.Z
								normalA.W += var18.W
								var17.X += originalNormalA.X
								var17.Y += originalNormalA.Y
								var17.Z += originalNormalA.Z
								var17.W += originalNormalA.W
								merged++
								w.MergeIndexA[vertexA] = w.TmpMergeIndex
								w.MergeIndexB[j] = w.TmpMergeIndex
							}
						}
					}
				}
			}
		}
	}
```
Replace with (drop the mutable `normalA`/`var17` aliases; index-mutate; keep read-only `originalNormalA`/`var18` as value copies):
```go
	for vertexA := range modelA.VertexCount {
		originalNormalA := modelA.VertexNormalOriginal[vertexA]

		if originalNormalA.W != 0 {
			y := modelA.VertexY[vertexA] - offsetY
			if y <= modelB.MinY {
				x := modelA.VertexX[vertexA] - arg2
				if x >= modelB.MinX && x <= modelB.MaxX {
					z := modelA.VertexZ[vertexA] - arg4
					if z >= modelB.MinZ && z <= modelB.MaxZ {
						for j := range vertexCountB {
							var18 := modelB.VertexNormalOriginal[j]
							if x == vertexX[j] && z == modelB.VertexZ[j] && y == modelB.VertexY[j] && var18.W != 0 {
								modelA.VertexNormal[vertexA].X += var18.X
								modelA.VertexNormal[vertexA].Y += var18.Y
								modelA.VertexNormal[vertexA].Z += var18.Z
								modelA.VertexNormal[vertexA].W += var18.W
								modelB.VertexNormal[j].X += originalNormalA.X
								modelB.VertexNormal[j].Y += originalNormalA.Y
								modelB.VertexNormal[j].Z += originalNormalA.Z
								modelB.VertexNormal[j].W += originalNormalA.W
								merged++
								w.MergeIndexA[vertexA] = w.TmpMergeIndex
								w.MergeIndexB[j] = w.TmpMergeIndex
							}
						}
					}
				}
			}
		}
	}
```
(The `if merged < 3 || !arg5` block and the two FaceCount flatten loops after it are unchanged — they don't touch normals.)

- [ ] **Step 8: Run the characterization tests**

Run: `GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./pkg/jagex2/graphics/model/ ./pkg/jagex2/dash3d/world3d/ -run "TestCalculateNormalsSingleFace|TestMergeNormalsAccumulatesCoincident|TestResetFromModel6" -v`
Expected: PASS. A failure on either normal test means an alias rewrite was missed (a value copy is silently dropping a write).

- [ ] **Step 9: Full build (native + js) + full suite + gofmt**

Run:
```
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go build ./...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOOS=js GOARCH=wasm go build ./...
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache go test ./...
gofmt -l pkg/jagex2/graphics/model/ pkg/jagex2/graphics/vertexnormal/ pkg/jagex2/dash3d/world3d/
```
Expected: builds OK, all tests `ok`, gofmt prints nothing.

- [ ] **Step 10: Commit**

```bash
git add pkg/jagex2/graphics/model/ pkg/jagex2/graphics/vertexnormal/vertexnormal.go pkg/jagex2/dash3d/world3d/world3d.go pkg/jagex2/dash3d/world3d/world3d_test.go
git commit --no-gpg-sign -m "perf(model): VertexNormal []*->[] value slice (kills 2M per-build allocations)"
```

---

## Final verification (after all tasks)

- [ ] Lint the touched packages under js tags (the `init`-installed native binary from earlier sessions, or install per `reference_golangci_lint_sandbox`):

```
GOPATH=$TMPDIR/go GOCACHE=$TMPDIR/go-cache GOLANGCI_LINT_CACHE=$TMPDIR/golangci-cache \
  GOOS=js GOARCH=wasm $TMPDIR/go/bin/golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 \
  ./pkg/jagex2/graphics/model/ ./pkg/jagex2/graphics/vertexnormal/ ./pkg/jagex2/dash3d/world3d/ \
  ./pkg/jagex2/config/npctype/ ./pkg/jagex2/dash3d/entity/...
```
Expected: `0 issues`.

- [ ] Native + js full builds and `go test ./...` green (final sweep).

- [ ] **Browser verification (user, host-only — sandbox can't run wasm):** rebuild wasm; confirm NPCs/players/locs render identically (animation, lighting across loc seams), then capture a fresh `allocs.pb.gz` via `goscapeDumpAllocs()` and confirm `NewModel6`/`NewVertexNormal` no longer dominate `-alloc_space`/`-alloc_objects`, and `goscapeMemStats()` shows a lower, stable `HeapSys`.

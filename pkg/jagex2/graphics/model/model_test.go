package model

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/vertexnormal"
)

// TestSliceAlias_EqualValueDifferentBacking is the foundational guarantee
// for the H4 fix in abd652c. Java compares face-priority buckets via
// reference identity (`var14 != TmpPriorityFaces[11]`), which is a pointer
// comparison. The pre-fix Go port used slices.Equal, a value compare —
// two buckets that happened to share contents (e.g., empty buckets, or
// equal-by-coincidence with the priority-11 bucket) would be treated as
// the same bucket and skip the wrap-back-to-11 transition, silently
// dropping geometry.
//
// sliceAlias must return false for two distinct slices that happen to
// share values, and true only when the backing array is identical.
func TestSliceAlias_EqualValueDifferentBacking(t *testing.T) {
	a := []int{1, 2, 3, 4}
	b := []int{1, 2, 3, 4}

	if sliceAlias(a, b) {
		t.Error("sliceAlias(a, b) = true for distinct backing arrays with equal contents; want false")
	}
}

// TestSliceAlias_SameSlice verifies the reflexive case.
func TestSliceAlias_SameSlice(t *testing.T) {
	a := []int{1, 2, 3, 4}
	if !sliceAlias(a, a) {
		t.Error("sliceAlias(a, a) = false; want true")
	}
}

// TestSliceAlias_Subslice verifies that two slices sharing the same
// backing array (even with different lengths/offsets) alias each other.
// This is intentional: DrawFaces's bucket-11 detection uses the array
// reference, so any view into TmpPriorityFaces[11] qualifies as the same
// bucket.
func TestSliceAlias_Subslice(t *testing.T) {
	a := []int{1, 2, 3, 4, 5}
	b := a[:3]
	if !sliceAlias(a, b) {
		t.Error("sliceAlias(full, prefix) = false; want true (same backing array)")
	}
}

// TestSliceAlias_EmptySlices covers the edge case used heavily by the
// face-priority dispatch: two distinct empty buckets should be reference-
// unequal. The pre-fix slices.Equal would have treated them as equal.
func TestSliceAlias_EmptySlices(t *testing.T) {
	a := make([]int, 0, 8)
	b := make([]int, 0, 8)
	if sliceAlias(a, b) {
		t.Error("sliceAlias(empty1, empty2) = true; want false (distinct backings)")
	}
}

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
	m.VertexNormal = make([]*vertexnormal.VertexNormal, 4)
	m.MaxY = 555
	m.ResetFromModel6(src, true)
	if m.Pickable || m.VertexNormal != nil || m.MaxY != 0 {
		t.Errorf("stale fields not cleared: Pickable=%v VertexNormal=%v MaxY=%d",
			m.Pickable, m.VertexNormal != nil, m.MaxY)
	}
}

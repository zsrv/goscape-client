package model

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/vertexnormal"
)

// stubProvider is a no-op io.OnDemandProvider for tests: it records nothing and
// never faults in data, so TryGet/Request behave deterministically.
type stubProvider struct{}

func (stubProvider) RequestModel(id int) {}

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
	m.FaceColourA = make([]int, faceCount)
	m.FaceColourB = make([]int, faceCount)
	m.FaceColourC = make([]int, faceCount)
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
	src.FaceColourA[0] = 11
	src.FaceColourB[0] = 22
	src.FaceColourC[0] = 33
	if m.FaceColourA[0] != 11 || m.FaceColourB[0] != 22 || m.FaceColourC[0] != 33 {
		t.Error("FaceColourA/B/C should share src's backing arrays")
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
	ptrY := &m.VertexY[0]
	ptrZ := &m.VertexZ[0]
	m.ResetFromModel6(src, false) // same size again
	if cap(m.VertexX) != capX || &m.VertexX[0] != ptr || &m.VertexY[0] != ptrY || &m.VertexZ[0] != ptrZ {
		t.Error("same-size rebuild reallocated VertexX/Y/Z instead of reusing them")
	}
	bigger := sampleBaseModel(64, 2)
	m.ResetFromModel6(bigger, false)
	if len(m.VertexX) != 64 {
		t.Fatalf("len after grow = %d, want 64", len(m.VertexX))
	}
}

func TestResetFromModel6AlphaNoAliasAcrossModes(t *testing.T) {
	// Reuse a target with retainAlpha=true (shares srcA.FaceAlpha), then with
	// retainAlpha=false on a different src. The second call must NOT write into
	// srcA's backing array, and must own its alpha (not alias srcB).
	srcA := sampleBaseModel(4, 3) // FaceAlpha = {7,8,9}
	var m Model
	m.ResetFromModel6(srcA, true)
	srcB := sampleBaseModel(4, 3)
	for i := range srcB.FaceAlpha {
		srcB.FaceAlpha[i] = 100 + i
	}
	m.ResetFromModel6(srcB, false)
	for i, want := range []int{7, 8, 9} {
		if srcA.FaceAlpha[i] != want {
			t.Fatalf("srcA.FaceAlpha[%d] = %d, want %d (second call corrupted a prior shared src)", i, srcA.FaceAlpha[i], want)
		}
	}
	srcB.FaceAlpha[0] = 999
	if m.FaceAlpha[0] == 999 {
		t.Error("retainAlpha=false should own FaceAlpha, not alias srcB")
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

// One face (0,0,0)-(100,0,0)-(0,0,100): the face normal reduces to (0,-256,0)
// and each of the 3 vertices accumulates it once (W=1). Hand-computed from
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

// TestUnpackBlobRoundTrip synthesizes a minimal valid rev-244 per-id model blob
// (2 vertices, 1 face, 0 textured faces, all optional flags off), unpacks its
// metadata, decodes it via NewModel1, and asserts the decoded geometry matches
// what was encoded. This exercises the 244 blob format end-to-end including the
// 18-byte trailer and the front-to-back section offset walk.
//
// GSmart encodes a value v in [-64,63] as the single byte v+64 (decode is
// G1()-64). All deltas below stay in that range so each consumes one byte.
func TestUnpackBlobRoundTrip(t *testing.T) {
	// Encoded geometry:
	//   vertex 0: dx=10  dy=20  dz=5   -> X=10, Y=20, Z=5
	//   vertex 1: dx=3   dy=4   dz=6   -> X=13, Y=24, Z=11
	//   face 0: orientation 1, a=0 b=1 c=0, colour 0x1234
	gs := func(v int) byte { return byte(v + 64) } // GSmart single-byte encoding

	data := []byte{
		// vertexFlags (offset 0, len = vertexCount = 2)
		0x07, 0x07,
		// faceOrientations (len = faceCount = 1)
		0x01,
		// faceVertices (len = dataLengthFaceOrientations = 3): a=0, b-a=1, c-b=-1
		gs(0), gs(1), gs(-1),
		// faceColours (len = faceCount*2 = 2): 0x1234
		0x12, 0x34,
		// (faceTextureAxis: len 0)
		// vertexX (len dataLengthX = 2)
		gs(10), gs(3),
		// vertexY (len dataLengthY = 2)
		gs(20), gs(4),
		// vertexZ (len dataLengthZ = 2)
		gs(5), gs(6),
		// ---- 18-byte trailer ----
		0x00, 0x02, // vertexCount = 2 (g2)
		0x00, 0x01, // faceCount = 1 (g2)
		0x00,       // texturedFaceCount = 0 (g1)
		0x00,       // hasInfo = 0 (g1)
		0x00,       // priority = 0 (g1)
		0x00,       // hasAlpha = 0 (g1)
		0x00,       // hasFaceLabels = 0 (g1)
		0x00,       // hasVertexLabels = 0 (g1)
		0x00, 0x02, // dataLengthX = 2 (g2)
		0x00, 0x02, // dataLengthY = 2 (g2)
		0x00, 0x02, // dataLengthZ = 2 (g2)
		0x00, 0x03, // dataLengthFaceOrientations = 3 (g2)
	}

	Reset()
	Init(1, stubProvider{})
	Unpack(0, data)

	m := NewModel1(0)
	if m.VertexCount != 2 || m.FaceCount != 1 {
		t.Fatalf("counts = %d/%d, want 2/1", m.VertexCount, m.FaceCount)
	}
	if m.TexturedFaceCount != 0 {
		t.Fatalf("TexturedFaceCount = %d, want 0", m.TexturedFaceCount)
	}

	wantX := []int{10, 13}
	wantY := []int{20, 24}
	wantZ := []int{5, 11}
	for i := range 2 {
		if m.VertexX[i] != wantX[i] || m.VertexY[i] != wantY[i] || m.VertexZ[i] != wantZ[i] {
			t.Errorf("vertex %d = (%d,%d,%d), want (%d,%d,%d)",
				i, m.VertexX[i], m.VertexY[i], m.VertexZ[i], wantX[i], wantY[i], wantZ[i])
		}
	}

	if m.FaceVertexA[0] != 0 || m.FaceVertexB[0] != 1 || m.FaceVertexC[0] != 0 {
		t.Errorf("face verts = (%d,%d,%d), want (0,1,0)",
			m.FaceVertexA[0], m.FaceVertexB[0], m.FaceVertexC[0])
	}
	if m.FaceColour[0] != 0x1234 {
		t.Errorf("FaceColour[0] = %#x, want 0x1234", m.FaceColour[0])
	}
}

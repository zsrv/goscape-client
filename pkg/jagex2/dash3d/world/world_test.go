package world

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/config/loctype"
)

// TestChangeLocAvailableShapeNormalization verifies the shape normalization in
// ChangeLocAvailable (Java World.changeLocAvailable, World.java:1096-1105):
// shape 11 maps to 10, and shapes 5..8 all map to 4, before LocType.CheckModel
// is consulted.
//
// The fixture is a LocType registered at id 42 whose only declared shape is 4,
// backed by a model id whose metadata is absent (model.Request returns false
// when model.Metadata is nil). Therefore CheckModel(4) is false, while
// CheckModel(any other shape) returns true (the shape is not in Shapes, so
// CheckModel short-circuits to true). This lets us observe which shapes get
// normalized into 4.
func TestChangeLocAvailableShapeNormalization(t *testing.T) {
	const id = 42

	// Build a LocType cache by hand so loctype.Get(id) resolves without the
	// Jagfile decode pipeline. Get scans Cache for a matching Index.
	prevCache := loctype.Cache
	t.Cleanup(func() { loctype.Cache = prevCache })

	fixture := loctype.NewLocType()
	fixture.Index = id
	fixture.Shapes = []int{4}
	fixture.Models = []int{0} // model.Request(0) -> false (Metadata nil)

	loctype.Cache = make([]*loctype.LocType, 10)
	for i := range loctype.Cache {
		loctype.Cache[i] = loctype.NewLocType() // Index defaults to -1
	}
	loctype.Cache[0] = fixture

	// Sanity: shape 4 is the normalization target and is unavailable.
	if ChangeLocAvailable(id, 4) {
		t.Fatal("ChangeLocAvailable(id, 4) = true, want false (fixture model unavailable)")
	}

	// Shapes 5..8 all normalize to 4 -> unavailable.
	for shape := 5; shape <= 8; shape++ {
		if ChangeLocAvailable(id, shape) {
			t.Errorf("ChangeLocAvailable(id, %d) = true, want false (should normalize 5..8 -> 4)", shape)
		}
	}

	// Shape 11 normalizes to 10 (not in Shapes) -> CheckModel returns true.
	if !ChangeLocAvailable(id, 11) {
		t.Error("ChangeLocAvailable(id, 11) = false, want true (11 normalizes to 10, not 4)")
	}

	// A shape that is neither normalized nor declared (e.g. 9) -> true.
	if !ChangeLocAvailable(id, 9) {
		t.Error("ChangeLocAvailable(id, 9) = false, want true (shape 9 not in Shapes)")
	}
}

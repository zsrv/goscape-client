package world3d

import (
	"testing"

	"goscape-client/pkg/jagex2/dash3d"
)

// TestOccludedMode5_UsesZDeltaForZProjection is a regression test for the
// 57ad2b0 fix. Java's Occluded mode 5 (horizontal occluder, looking down
// along Y) projects both X and Z. The Java reference uses MinDeltaX /
// MaxDeltaX for the X projection AND MinDeltaZ / MaxDeltaZ for the Z
// projection. The pre-fix Go port copied MinDeltaX into the Z projection
// slot, which gave correct results only when X and Z had the same slope.
//
// This test constructs an occluder where X is locked (DeltaX = 0) and Z
// has a strong positive slope (DeltaZ = 256). With the fixed code the
// query point's Z must be projected forward by ~200 units; the buggy
// version would leave the Z window at the occluder's flat MinZ/MaxZ and
// reject the point.
func TestOccludedMode5_UsesZDeltaForZProjection(t *testing.T) {
	// Reset state for hermetic test. ActiveOccluders is a package-level
	// slice of length 500; we reuse it and pin the count.
	for i := range ActiveOccluders {
		ActiveOccluders[i] = nil
	}
	ActiveOccluderCount = 0
	t.Cleanup(func() { ActiveOccluderCount = 0 })

	occ := dash3d.NewOcclude()
	occ.Mode = 5
	occ.MinX, occ.MaxX = 0, 100
	occ.MinZ, occ.MaxZ = 0, 100
	occ.MinY, occ.MaxY = 0, 100
	occ.MinDeltaX, occ.MaxDeltaX = 0, 0     // X locked
	occ.MinDeltaZ, occ.MaxDeltaZ = 256, 256 // Z slope of +1 (256/256)
	occ.MinDeltaY, occ.MaxDeltaY = 0, 0

	ActiveOccluders[0] = occ
	ActiveOccluderCount = 1

	w := &World3D{}

	// arg0=X=50, arg1=Y=200, arg2=Z=200.
	// var6 = Y - MinY = 200.
	// Fixed: Z window = [0, 100] + (256*200)>>8 = [200, 300] — includes Z=200.
	// Buggy: Z window = [0, 100] + (0*200)>>8 = [0, 100] — excludes Z=200.
	if got := w.Occluded(50, 200, 200); !got {
		t.Errorf("Occluded(X=50, Y=200, Z=200) = false; want true (Z window should be projected forward by DeltaZ)")
	}
}

// TestOccludedMode5_RejectsOutsideZWindow confirms the projection works
// the other way: a point outside the projected Z window is correctly
// reported as not occluded.
func TestOccludedMode5_RejectsOutsideZWindow(t *testing.T) {
	for i := range ActiveOccluders {
		ActiveOccluders[i] = nil
	}
	ActiveOccluderCount = 0
	t.Cleanup(func() { ActiveOccluderCount = 0 })

	occ := dash3d.NewOcclude()
	occ.Mode = 5
	occ.MinX, occ.MaxX = 0, 100
	occ.MinZ, occ.MaxZ = 0, 100
	occ.MinY, occ.MaxY = 0, 100
	occ.MinDeltaX, occ.MaxDeltaX = 0, 0
	occ.MinDeltaZ, occ.MaxDeltaZ = 256, 256
	occ.MinDeltaY, occ.MaxDeltaY = 0, 0

	ActiveOccluders[0] = occ
	ActiveOccluderCount = 1

	w := &World3D{}

	// Z=50 with var6=200: projected Z window is [200, 300], Z=50 is below.
	if got := w.Occluded(50, 200, 50); got {
		t.Errorf("Occluded(X=50, Y=200, Z=50) = true; want false (Z=50 below projected window [200,300])")
	}
}

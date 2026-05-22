package model

import "testing"

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

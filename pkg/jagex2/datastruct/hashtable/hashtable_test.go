package hashtable

import "testing"

// TestPutGetRoundTrip verifies that a value put into the table can be looked
// up by its key.
func TestPutGetRoundTrip(t *testing.T) {
	tbl := NewHashTable(16)
	n := &Linkable{}
	tbl.Put(42, n)

	got := tbl.Find(42)
	if got != n {
		t.Fatalf("Get(42) = %p, want %p", got, n)
	}
	if got.Key != 42 {
		t.Fatalf("got.Key = %d, want 42", got.Key)
	}
}

// TestGetMissReturnsNil verifies that Get returns nil for keys that have
// never been put.
func TestGetMissReturnsNil(t *testing.T) {
	tbl := NewHashTable(16)
	if got := tbl.Find(1234); got != nil {
		t.Fatalf("Get(1234) on empty table = %v, want nil", got)
	}
	// Populate one bucket and confirm the miss in the same bucket is still
	// nil (i.e. the linear search terminates correctly at the sentinel).
	tbl.Put(0, &Linkable{})
	if got := tbl.Find(16); got != nil { // 16 & 15 == 0, same bucket as 0
		t.Fatalf("Get(16) = %v, want nil (collision miss)", got)
	}
}

// TestRemoveViaUnlink verifies that Unlink removes a node from the table so
// subsequent Get returns nil. (HashTable has no remove() method in Java;
// callers call Linkable.unlink() directly.)
func TestRemoveViaUnlink(t *testing.T) {
	tbl := NewHashTable(16)
	n := &Linkable{}
	tbl.Put(7, n)
	n.Unlink()
	if got := tbl.Find(7); got != nil {
		t.Fatalf("Get(7) after Unlink = %v, want nil", got)
	}
}

// TestPutRebucketsNode verifies that putting a node a second time at a new
// key moves it to the correct bucket. Java semantics: Put first calls
// Unlink on the node (if linked), then re-links it under the new key.
func TestPutRebucketsNode(t *testing.T) {
	tbl := NewHashTable(16)
	n := &Linkable{}
	tbl.Put(1, n)
	tbl.Put(2, n)
	if got := tbl.Find(1); got != nil {
		t.Fatalf("Get(1) after re-put at 2 = %v, want nil", got)
	}
	if got := tbl.Find(2); got != n {
		t.Fatalf("Get(2) after re-put = %p, want %p", got, n)
	}
	if n.Key != 2 {
		t.Fatalf("n.Key = %d, want 2", n.Key)
	}
}

// TestCollisionChain verifies that multiple nodes in the same bucket are all
// reachable. Bucket count is 4; keys 1, 5, 9 all hash to bucket 1.
func TestCollisionChain(t *testing.T) {
	tbl := NewHashTable(4)
	a := &Linkable{}
	b := &Linkable{}
	c := &Linkable{}
	tbl.Put(1, a)
	tbl.Put(5, b)
	tbl.Put(9, c)

	if got := tbl.Find(1); got != a {
		t.Errorf("Get(1) = %p, want %p", got, a)
	}
	if got := tbl.Find(5); got != b {
		t.Errorf("Get(5) = %p, want %p", got, b)
	}
	if got := tbl.Find(9); got != c {
		t.Errorf("Get(9) = %p, want %p", got, c)
	}

	// Verify they really all landed in bucket 1.
	sentinel := tbl.Buckets[1]
	count := 0
	for n := sentinel.Next; n != sentinel; n = n.Next {
		count++
	}
	if count != 3 {
		t.Errorf("bucket 1 chain length = %d, want 3", count)
	}

	// Unlink the middle node and verify the chain stays intact.
	b.Unlink()
	if got := tbl.Find(5); got != nil {
		t.Errorf("Get(5) after b.Unlink = %v, want nil", got)
	}
	if got := tbl.Find(1); got != a {
		t.Errorf("Get(1) after b.Unlink = %p, want %p", got, a)
	}
	if got := tbl.Find(9); got != c {
		t.Errorf("Get(9) after b.Unlink = %p, want %p", got, c)
	}
}

// TestIterateBucket verifies that callers can walk a bucket by following
// Next from the sentinel back to itself — the canonical Java-side iteration
// pattern (used inside HashTable.Get).
func TestIterateBucket(t *testing.T) {
	tbl := NewHashTable(8)
	keys := []int64{0, 8, 16, 24} // all hash to bucket 0
	for _, k := range keys {
		tbl.Put(k, &Linkable{})
	}

	sentinel := tbl.Buckets[0]
	var seen []int64
	for n := sentinel.Next; n != sentinel; n = n.Next {
		seen = append(seen, n.Key)
	}
	if len(seen) != len(keys) {
		t.Fatalf("seen %d keys, want %d", len(seen), len(keys))
	}
	// Java inserts at the tail (prev of sentinel), so iteration from
	// sentinel.next yields insertion order.
	for i, want := range keys {
		if seen[i] != want {
			t.Errorf("seen[%d] = %d, want %d", i, seen[i], want)
		}
	}
}

// TestNegativeKeyBucketing verifies that bucketing handles negative keys
// without crashing or escaping the bucket array. Java relies on the bitmask
// `key & (bucketCount - 1)` to produce a non-negative bucket index for any
// long key when bucketCount is a power of two.
func TestNegativeKeyBucketing(t *testing.T) {
	tbl := NewHashTable(16)
	n := &Linkable{}
	tbl.Put(-1, n)
	if got := tbl.Find(-1); got != n {
		t.Fatalf("Get(-1) = %p, want %p", got, n)
	}
}

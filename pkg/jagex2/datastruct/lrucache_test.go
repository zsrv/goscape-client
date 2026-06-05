package datastruct

import "testing"

// TestLruCachePutGet is a smoke test for the basic put/get path.
func TestLruCachePutGet(t *testing.T) {
	c := NewLruCache[int](3)
	c.Put(1, 100)
	c.Put(2, 200)
	c.Put(3, 300)

	for _, tc := range []struct {
		key  int64
		want int
	}{{1, 100}, {2, 200}, {3, 300}} {
		if got := c.Find(tc.key); got != tc.want {
			t.Errorf("Get(%d)=%d, want %d", tc.key, got, tc.want)
		}
	}
}

// TestLruCacheGetMovesToFront verifies that Get on an existing key re-pushes
// the node to the tail (most-recently-used slot). Before the 2026-05-22
// rewrite, Get allocated a fresh Linkable2 on every call, so the
// original cached node never moved and LRU degenerated to FIFO.
func TestLruCacheGetMovesToFront(t *testing.T) {
	c := NewLruCache[int](3)
	c.Put(1, 100)
	c.Put(2, 200)
	c.Put(3, 300)

	// Touch key 1 — it should become MRU. Insertion order was 1,2,3, so
	// without the touch the next eviction victim would be 1; with the touch
	// it should be 2.
	if got := c.Find(1); got != 100 {
		t.Fatalf("Get(1)=%d, want 100", got)
	}

	c.Put(4, 400)

	// 2 should have been evicted, not 1.
	if got := c.Find(2); got != 0 {
		t.Errorf("Get(2)=%d after eviction, want zero", got)
	}
	if got := c.Find(1); got != 100 {
		t.Errorf("Get(1)=%d, want 100 (should still be present)", got)
	}
	if got := c.Find(4); got != 400 {
		t.Errorf("Get(4)=%d, want 400", got)
	}
}

// TestLruCachePutEvictsAndCleansMap verifies that when Put-with-full evicts
// an entry, the evicted key is removed from the underlying map (not just
// from the history list). Before the rewrite, the map grew unboundedly.
func TestLruCachePutEvictsAndCleansMap(t *testing.T) {
	c := NewLruCache[int](2)
	c.Put(1, 100)
	c.Put(2, 200)
	c.Put(3, 300) // evicts 1

	if len(c.HashTable) != 2 {
		t.Errorf("HashTable size=%d after eviction, want 2", len(c.HashTable))
	}
	if _, ok := c.HashTable[1]; ok {
		t.Error("evicted key 1 still present in HashTable")
	}
	if got := c.Find(1); got != 0 {
		t.Errorf("Get(1)=%d after eviction, want zero", got)
	}
}

// TestLruCacheDelete verifies that explicit deletion removes the entry from
// both the map and the history list, and frees a slot for new Puts.
// ObjType.GetSprite depends on this for stale icon invalidation.
func TestLruCacheDelete(t *testing.T) {
	c := NewLruCache[int](2)
	c.Put(1, 100)
	c.Put(2, 200)

	c.Delete(1)
	if _, ok := c.HashTable[1]; ok {
		t.Error("Delete did not remove key 1 from HashTable")
	}
	if c.Available != 1 {
		t.Errorf("Available=%d after Delete, want 1", c.Available)
	}

	// Put a new entry — should not evict anything, since Delete freed a slot.
	c.Put(3, 300)
	if got := c.Find(2); got != 200 {
		t.Errorf("Get(2)=%d after Delete(1)+Put(3), want 200 (no eviction expected)", got)
	}
	if got := c.Find(3); got != 300 {
		t.Errorf("Get(3)=%d, want 300", got)
	}
}

// TestLruCacheDeleteAbsent is a no-op safety check.
func TestLruCacheDeleteAbsent(t *testing.T) {
	c := NewLruCache[int](2)
	c.Put(1, 100)
	c.Delete(99) // should not panic, should not change Available
	if c.Available != 1 {
		t.Errorf("Available=%d after Delete(absent), want 1", c.Available)
	}
}

// TestLruCacheRepeatedGetSameKey verifies that calling Get many times on the
// same key does not corrupt history or map state. Before the rewrite, each
// Get allocated a fresh node, so repeated Gets would grow the history list
// unboundedly.
func TestLruCacheRepeatedGetSameKey(t *testing.T) {
	c := NewLruCache[int](3)
	c.Put(1, 100)
	c.Put(2, 200)

	for range 1000 {
		_ = c.Find(1)
	}

	// HashTable must still have exactly 2 entries.
	if len(c.HashTable) != 2 {
		t.Errorf("HashTable size=%d after 1000 Gets, want 2", len(c.HashTable))
	}
	// Available should be unchanged.
	if c.Available != 1 {
		t.Errorf("Available=%d after Gets, want 1", c.Available)
	}
}

// TestLruCacheClear empties the cache and resets the available count.
func TestLruCacheClear(t *testing.T) {
	c := NewLruCache[int](3)
	c.Put(1, 100)
	c.Put(2, 200)

	c.Clear()

	if len(c.HashTable) != 0 {
		t.Errorf("HashTable not empty after Clear: size=%d", len(c.HashTable))
	}
	if c.Available != c.Capacity {
		t.Errorf("Available=%d after Clear, want Capacity=%d", c.Available, c.Capacity)
	}
}

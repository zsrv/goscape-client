package datastruct

// LruCache ports jagex2.datastruct.LruCache. Capacity entries are held in a
// map (Java: HashTable); the order of last-access is tracked in a doubly-
// linked History list. On Get, the cached node is moved to the front; on
// Put when full, the head of History is evicted and removed from the map.
//
// Java stores Linkable2 subclass instances directly; the same node is
// in BOTH the HashTable bucket list AND the History list via two distinct
// pointer pairs. The Go port stores the wrapper node in the map and relies
// on the embedded Key for back-reference on eviction.
type LruCache[T any] struct {
	Capacity  int32
	Available int32
	// Java: notFound/found (LruCache.java) — hit/miss telemetry written by
	// get(); no reader exists at this rev (audit datastruct-05).
	NotFound  int32
	Found     int32
	HashTable map[int64]*Linkable2[T]
	History   *LinkList2[T]
}

func NewLruCache[T any](size int32) *LruCache[T] {
	return &LruCache[T]{
		Capacity:  size,
		Available: size,
		HashTable: make(map[int64]*Linkable2[T], 0x400),
		History:   NewLinkList2[T](),
	}
}

func (l *LruCache[T]) Get(key int64) T {
	node, ok := l.HashTable[key]
	if !ok {
		l.NotFound++
		var zero T
		return zero
	}
	l.Found++
	// Java: history.push(var3) — re-pushing an already-linked node
	// Uncaches it then re-links at the tail (the most-recently-used slot).
	l.History.Push(node)
	return node.Linkable.Value //nolint:staticcheck // QF1008: explicit embedded-field selector mirrors Java field access
}

// Put inserts v under key. CONSTRAINT (datastruct.md #29): callers must Get
// first and only Put on a miss — Put does not guard against a duplicate key.
// A duplicate-key Put would orphan the previous node in History and
// double-decrement Available. Java's HashTable.put unlinked the prior bucket
// node first; the Go map redesign drops that structural protection. All current
// callers (objtype/loctype/npctype/spottype/component/playerentity) follow
// the Get-then-Put-if-miss pattern, so this is latent, not a live bug.
// Re-confirmed by the 2026-06-04 audit (datastruct-07): Java's true duplicate-
// key behavior (both nodes coexist in bucket+history; the OLDER one wins get)
// is unreproducible on a Go map without restoring bucket chains — the caller
// constraint stands.
func (l *LruCache[T]) Put(key int64, v T) {
	if l.Available == 0 {
		evicted := l.History.Pop()
		// Java: var5.unlink() — also removes the node from the HashTable
		// bucket because it lives in both lists simultaneously. Go's map
		// has no such linkage, so we delete by the Key we stamped on Put.
		delete(l.HashTable, evicted.Key)
	} else {
		l.Available--
	}
	node := NewLinkable2(v)
	node.Key = key
	l.HashTable[key] = node
	l.History.Push(node)
}

// Delete removes the entry with the given key.
//
// INTENTIONAL DEVIATION from Java (kept deliberately — do not "fix" back to
// Java's behavior). Java's only delete-style caller is ObjType.getIcon, which
// calls node.unlink() (Linkable.unlink, Java LruCache caller path). unlink()
// touches ONLY the bucket-list pointers: it removes the node from the HashTable
// bucket but leaves it in the History list and does NOT change `available`. The
// node lingers as an orphan in history until a later Pop evicts it, so each
// delete-then-re-put permanently consumes one slot until eviction reclaims it —
// Java's effective live capacity shrinks between evictions (a leak-then-reclaim
// bug). Go instead removes from the map, Uncaches the node out of history, and
// increments Available, so a subsequent Put nets Available unchanged with no
// orphan. The audit (datastruct.md #8) classified Go as MORE correct here; the
// project chose to keep it. The only observable difference is icon-cache
// eviction timing, not correctness.
func (l *LruCache[T]) Delete(key int64) {
	node, ok := l.HashTable[key]
	if !ok {
		return
	}
	delete(l.HashTable, key)
	node.Uncache()
	l.Available++
}

func (l *LruCache[T]) Clear() {
	for {
		node := l.History.Pop()
		if node == nil {
			break
		}
		delete(l.HashTable, node.Key)
	}
	l.Available = l.Capacity
}

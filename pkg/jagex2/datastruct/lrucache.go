package datastruct

// LruCache ports jagex2.datastruct.LruCache. Capacity entries are held in a
// map (Java: HashTable); the order of last-access is tracked in a doubly-
// linked History list. On Get, the cached node is moved to the front; on
// Put when full, the head of History is evicted and removed from the map.
//
// Java stores DoublyLinkable subclass instances directly; the same node is
// in BOTH the HashTable bucket list AND the History list via two distinct
// pointer pairs. The Go port stores the wrapper node in the map and relies
// on the embedded Key for back-reference on eviction.
type LruCache[T any] struct {
	Capacity  int32
	Available int32
	HashTable map[int64]*DoublyLinkable[T]
	History   *DoublyLinkList[T]
}

func NewLruCache[T any](size int32) *LruCache[T] {
	return &LruCache[T]{
		Capacity:  size,
		Available: size,
		HashTable: make(map[int64]*DoublyLinkable[T], 0x400),
		History:   NewDoublyLinkList[T](),
	}
}

func (l *LruCache[T]) Get(key int64) T {
	node, ok := l.HashTable[key]
	if !ok {
		var zero T
		return zero
	}
	// Java: history.push(var3) — re-pushing an already-linked node
	// Uncaches it then re-links at the tail (the most-recently-used slot).
	l.History.Push(node)
	return node.Linkable.Value
}

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
	node := NewDoublyLinkable(v)
	node.Key = key
	l.HashTable[key] = node
	l.History.Push(node)
}

// Delete removes the entry with the given key. Java does this by calling
// node.unlink() on the DoublyLinkable, which removes it from both the
// HashTable bucket list and the History list at once. Go needs explicit
// map + history removal.
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

// Package hashtable ports jagex2/datastruct/HashTable.java faithfully.
//
// In the Java client, HashTable's buckets are circular doubly-linked lists of
// Linkable nodes, where each Linkable carries a long `key` used both for
// bucket selection (key & (bucketCount - 1)) and per-bucket lookup.
//
// The existing Go datastruct package has a Linkable[T] type, but it is a
// value-wrapper used by LinkList/DoublyLinkList and has no `key` field — its
// semantics differ from Java's Linkable, which is itself the superclass of all
// hash-tabled objects (callers extend Linkable rather than wrapping a value).
// To preserve the Java surface (and to keep this package self-contained), we
// re-declare Linkable here with the exact public fields Java exposes. Callers
// that need to be hashable should embed *Linkable.
//
// Java sources:
//   - jagex2/datastruct/HashTable.java
//   - jagex2/datastruct/Linkable.java
package hashtable

// Linkable mirrors jagex2.datastruct.Linkable. Java declares Key/Next/Prev as
// public fields; we keep them exported to match. Subclass-style usage in Go
// is via struct embedding (`type MyNode struct { *Linkable; ... }`).
type Linkable struct {
	// Key is the hash key used by HashTable to bucket and locate this node.
	// Java: long key.
	Key int64
	// Next is the next node in the bucket's circular doubly-linked list.
	// Java: Linkable next.
	Next *Linkable
	// Prev is the previous node in the bucket's circular doubly-linked list.
	// Java: Linkable prev.
	Prev *Linkable
}

// Unlink removes this node from whatever list it currently belongs to.
// Java: Linkable.unlink().
func (l *Linkable) Unlink() {
	if l.Prev != nil {
		l.Prev.Next = l.Next
		l.Next.Prev = l.Prev
		l.Next = nil
		l.Prev = nil
	}
}

// HashTable is a fixed-size bucket array of circular doubly-linked Linkable
// lists. BucketCount must be a power of two; the bucket index is computed as
// `key & (BucketCount - 1)` (faithful to Java).
type HashTable struct {
	// BucketCount is the number of buckets. Java: int bucketCount.
	BucketCount int32
	// Buckets holds the sentinel head node for each bucket; sentinels point
	// to themselves when the bucket is empty. Java: Linkable[] buckets.
	Buckets []*Linkable
}

// NewHashTable constructs a HashTable with bucketCount buckets, each
// initialized with a self-referential sentinel Linkable.
// Java: public HashTable(int bucketCount).
func NewHashTable(bucketCount int32) *HashTable {
	t := &HashTable{
		BucketCount: bucketCount,
		Buckets:     make([]*Linkable, bucketCount),
	}
	for i := range bucketCount {
		sentinel := &Linkable{}
		sentinel.Next = sentinel
		sentinel.Prev = sentinel
		t.Buckets[i] = sentinel
	}
	return t
}

// Get returns the Linkable whose Key matches arg0, or nil if no such node
// exists. Java: public Linkable get(long arg0).
func (t *HashTable) Get(arg0 int64) *Linkable {
	// Java: this.buckets[(int) (arg0 & (long) (this.bucketCount - 1))]
	// Go's bitwise-AND precedence matches Java here (both bind tighter than
	// the implicit conversion); we parenthesize defensively per CLAUDE.md.
	sentinel := t.Buckets[int32(arg0&int64(t.BucketCount-1))]
	for node := sentinel.Next; node != sentinel; node = node.Next {
		if node.Key == arg0 {
			return node
		}
	}
	return nil
}

// Put inserts arg2 into the bucket selected by arg0, unlinking it from any
// previous list first. The node is added at the tail of the bucket's circular
// doubly-linked list (i.e. immediately before the sentinel).
// Java: public void put(long arg0, Linkable arg2).
func (t *HashTable) Put(arg0 int64, arg2 *Linkable) {
	if arg2.Prev != nil {
		arg2.Unlink()
	}
	sentinel := t.Buckets[int32(arg0&int64(t.BucketCount-1))]
	arg2.Prev = sentinel.Prev
	arg2.Next = sentinel
	arg2.Prev.Next = arg2
	arg2.Next.Prev = arg2
	arg2.Key = arg0
}

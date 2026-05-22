package datastruct

type DoublyLinkable[T any] struct {
	*Linkable[T]

	// Key mirrors Java's Linkable.key (a long stored on the linkable itself
	// so HashTable can look it up by bucket-AND mask). Go's LruCache uses
	// it to find the right map entry when an LRU eviction pops a node off
	// the history list.
	Key int64

	next2 *DoublyLinkable[T]
	prev2 *DoublyLinkable[T]
}

func NewDoublyLinkable[T any](value T) *DoublyLinkable[T] {
	return &DoublyLinkable[T]{
		Linkable: NewLinkable(value),
	}
}

// Unlink2
func (d *DoublyLinkable[T]) Uncache() {
	if d.prev2 != nil {
		d.prev2.next2 = d.next2
		d.next2.prev2 = d.prev2
		d.next2 = nil
		d.prev2 = nil
	}
}

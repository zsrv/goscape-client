package datastruct

type LruCache[T any] struct {
	Capacity  int32
	Available int32
	HashTable map[int64]T
	History   *DoublyLinkList[T]
}

func NewLruCache[T any](size int32) *LruCache[T] {
	l := &LruCache[T]{
		Capacity:  size,
		Available: size,
		HashTable: make(map[int64]T, 0x400), // Java: HashTable(1024) bucket count — Go map auto-grows beyond hint
		History:   NewDoublyLinkList[T](),
	}
	return l
}

func (l *LruCache[T]) Get(key int64) T {
	v, ok := l.HashTable[key]
	if !ok {
		var zero T
		return zero
	}
	node := NewDoublyLinkable(v)
	l.History.Push(node)
	return v
}

func (l *LruCache[T]) Put(key int64, v T) {
	if l.Available == 0 {
		sentinel := l.History.Pop()
		sentinel.Unlink()
		sentinel.Uncache()
	} else {
		l.Available--
	}

	l.HashTable[key] = v
	l.History.Push(NewDoublyLinkable(v))
}

func (l *LruCache[T]) Clear() {
	for {
		node := l.History.Pop()
		if node == nil {
			l.Available = l.Capacity
			return
		}
		node.Unlink()
		node.Uncache()
	}
}

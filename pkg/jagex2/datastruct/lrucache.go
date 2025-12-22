package datastruct

type LruCache[T any] struct {
	Capacity  int32
	Available int32
	HashTable map[int64]T
	History   *DoublyLinkList[T]
}

func NewLruCache[T any](arg1 int32) *LruCache[T] {
	l := &LruCache[T]{
		Capacity:  arg1,
		Available: arg1,
		HashTable: make(map[int64]T, 1024), // TODO: not limited to 1024
		History:   NewDoublyLinkList[T](),
	}
	return l
}

func (l *LruCache[T]) Get(arg0 int64) T {
	var3, ok := l.HashTable[arg0]
	if !ok {
		var zero T
		return zero
	}
	v := NewDoublyLinkable(var3)
	l.History.Push(v)
	return var3
}

func (l *LruCache[T]) Put(arg1 int64, arg2 T) {
	if l.Available == 0 {
		var5 := l.History.Pop()
		var5.Unlink()
		var5.Uncache()
	} else {
		l.Available--
	}
	l.HashTable[arg1] = arg2
	l.History.Push(NewDoublyLinkable(arg2))
}

func (l *LruCache[T]) Clear() {
	for {
		var1 := l.History.Pop()
		if var1 == nil {
			l.Available = l.Capacity
			return
		}
		var1.Unlink()
		var1.Uncache()
	}
}

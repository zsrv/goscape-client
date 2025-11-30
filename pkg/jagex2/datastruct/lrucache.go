package datastruct

type LruCache[T any] struct {
	Capacity  int32
	Available int32
	HashTable map[int64]*DoublyLinkable[T]
	History   *DoublyLinkList[T]
}

func NewLruCache[T any](arg1 int32) *LruCache[T] {
	l := &LruCache[T]{
		Capacity:  arg1,
		HashTable: make(map[int64]*DoublyLinkable[T]),
		History:   NewDoublyLinkList[T](5),
	}
	l.Available = arg1
	return l
}

func (l *LruCache[T]) Get(arg0 int64) *DoublyLinkable[T] {
	var3 := l.HashTable[arg0]
	if var3 != nil {
		l.History.Push(var3)
	}
	return var3
}

func (l *LruCache[T]) Put(arg1 int64, arg2 *DoublyLinkable[T]) {
	if l.Available == 0 {
		var5 := l.History.Pop()
		var5.Unlink()
		var5.Uncache()
	} else {
		l.Available--
	}
	l.HashTable[arg1] = arg2
	l.History.Push(arg2)
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

package datastruct

type LruCache struct {
	Field282  bool
	Capacity  int32
	Available int32
	HashTable map[int64]*DoublyLinkable
	History   *DoublyLinkList
}

func NewLruCache(arg1 int32) *LruCache {
	l := &LruCache{
		Field282:  false,
		Capacity:  arg1,
		HashTable: make(map[int64]*DoublyLinkable),
		History:   NewDoublyLinkList(5),
	}
	l.Available = arg1
	return l
}

func (l *LruCache) Get(arg0 int64) *DoublyLinkable {
	var3 := l.HashTable[arg0]
	if var3 != nil {
		l.History.Push(var3)
	}
	return var3
}

func (l *LruCache) Put(arg0 int32, arg1 int64, arg2 *DoublyLinkable) {
	if l.Available == 0 {
		var5 := l.History.Pop()
		var5.Unlink()
		var5.Uncache()
	} else {
		l.Available--
	}
	l.HashTable[arg1] = arg2
	if arg0 != 6 {
		l.Field282 = !l.Field282
	}
	l.History.Push(arg2)
}

func (l *LruCache) Clear() {
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

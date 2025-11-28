package datastruct

type LinkList[T any] struct {
	Sentinel *Linkable[T]
	Cursor   *Linkable[T]
}

func NewLinkList[T any]() *LinkList[T] {
	l := &LinkList[T]{
		Sentinel: new(Linkable[T]),
	}
	l.Sentinel.Next = l.Sentinel
	l.Sentinel.Prev = l.Sentinel
	return l
}

func (l *LinkList[T]) AddTail(arg0 *Linkable[T]) {
	if arg0.Prev != nil {
		arg0.Unlink()
	}
	arg0.Prev = l.Sentinel.Prev
	arg0.Next = l.Sentinel
	arg0.Prev.Next = arg0
	arg0.Next.Prev = arg0
}

func (l *LinkList[T]) AddHead(arg0 *Linkable[T], arg1 int32) {
	if arg0.Prev != nil {
		arg0.Unlink()
	}
	arg0.Prev = l.Sentinel
	if arg1 == -26173 {
		arg0.Next = l.Sentinel.Next
		arg0.Prev.Next = arg0
		arg0.Next.Prev = arg0
	}
}

func (l *LinkList[T]) RemoveHead() *Linkable[T] {
	var1 := l.Sentinel.Next
	if var1 == l.Sentinel {
		return nil
	} else {
		var1.Unlink()
		return var1
	}
}

func (l *LinkList[T]) Head() *Linkable[T] {
	var1 := l.Sentinel.Next
	if var1 == l.Sentinel {
		l.Cursor = nil
		return nil
	} else {
		l.Cursor = var1.Next
		return var1
	}
}

func (l *LinkList[T]) Tail() *Linkable[T] {
	var2 := l.Sentinel.Prev
	if var2 == l.Sentinel {
		l.Cursor = nil
		return nil
	}
	l.Cursor = var2.Prev
	return var2
}

func (l *LinkList[T]) Next() *Linkable[T] {
	var2 := l.Cursor
	if var2 == l.Sentinel {
		l.Cursor = nil
		return nil
	} else {
		l.Cursor = var2.Next
		return var2
	}
}

func (l *LinkList[T]) Prev() *Linkable[T] {
	var2 := l.Cursor
	if var2 == l.Sentinel {
		l.Cursor = nil
		return nil
	} else {
		l.Cursor = var2.Prev
		return var2
	}
}

func (l *LinkList[T]) Clear() {
	for {
		var1 := l.Sentinel.Next
		if var1 == l.Sentinel {
			return
		}
		var1.Unlink()
	}
}

package datastruct

// TODO: LinkList was changed to receive and return values instead of
//  Linkable! AddTail and AddHead no longer unlink the input from existing lists!
//  Not sure if this will break anything

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

func (l *LinkList[T]) AddTail(v T) {
	node := &Linkable[T]{
		Value: v,
	}
	node.Prev = l.Sentinel.Prev
	node.Next = l.Sentinel
	node.Prev.Next = node
	node.Next.Prev = node
}

func (l *LinkList[T]) AddHead(v T) {
	node := &Linkable[T]{
		Value: v,
	}
	node.Prev = l.Sentinel
	node.Next = l.Sentinel.Next
	node.Prev.Next = node
	node.Next.Prev = node
}

func (l *LinkList[T]) RemoveHead() T {
	node := l.Sentinel.Next
	if node == l.Sentinel {
		// Returns the zero value for the type argument used for T
		// https://stackoverflow.com/a/70586169
		var zero T
		return zero
	}
	node.Unlink()
	return node.Value
}

func (l *LinkList[T]) Head() T {
	node := l.Sentinel.Next
	if node == l.Sentinel {
		l.Cursor = nil
		var zero T
		return zero
	}
	l.Cursor = node.Next
	return node.Value
}

func (l *LinkList[T]) Tail() T {
	node := l.Sentinel.Prev
	if node == l.Sentinel {
		l.Cursor = nil
		var zero T
		return zero
	}
	l.Cursor = node.Prev
	return node.Value
}

func (l *LinkList[T]) Next() T {
	node := l.Cursor
	if node == l.Sentinel {
		l.Cursor = nil
		var zero T
		return zero
	}
	l.Cursor = node.Next
	return node.Value
}

func (l *LinkList[T]) Prev() T {
	node := l.Cursor
	if node == l.Sentinel {
		l.Cursor = nil
		var zero T
		return zero
	}
	l.Cursor = node.Prev
	return node.Value
}

func (l *LinkList[T]) Clear() {
	for {
		node := l.Sentinel.Next
		if node == l.Sentinel {
			return
		}
		node.Unlink()
	}
}

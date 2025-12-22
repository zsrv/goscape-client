package datastruct

type LinkList[T any] struct {
	sentinel *Linkable[T]
	cursor   *Linkable[T]
}

func NewLinkList[T any]() *LinkList[T] {
	s := new(Linkable[T])
	s.next = s
	s.prev = s
	return &LinkList[T]{
		sentinel: s,
	}
}

func (l *LinkList[T]) AddTail(v *Linkable[T]) {
	if v.prev != nil {
		v.Unlink()
	}
	v.prev = l.sentinel.prev
	v.next = l.sentinel
	v.prev.next = v
	v.next.prev = v
}

func (l *LinkList[T]) AddHead(v *Linkable[T]) {
	if v.prev != nil {
		v.Unlink()
	}
	v.prev = l.sentinel
	v.next = l.sentinel.next
	v.prev.next = v
	v.next.prev = v
}

func (l *LinkList[T]) RemoveHead() *Linkable[T] {
	n := l.sentinel.next
	if n == l.sentinel {
		return nil
	}
	n.Unlink()
	return n
}

func (l *LinkList[T]) Head() *Linkable[T] {
	n := l.sentinel.next
	if n == l.sentinel {
		l.cursor = nil
		return nil
	}
	l.cursor = n.next
	return n
}

func (l *LinkList[T]) Tail() *Linkable[T] {
	p := l.sentinel.prev
	if p == l.sentinel {
		l.cursor = nil
		return nil
	}
	l.cursor = p.prev
	return p
}

func (l *LinkList[T]) Next() *Linkable[T] {
	c := l.cursor
	if c == l.sentinel {
		l.cursor = nil
		return nil
	}
	l.cursor = c.next
	return c
}

func (l *LinkList[T]) Prev() *Linkable[T] {
	c := l.cursor
	if c == l.sentinel {
		l.cursor = nil
		return nil
	}
	l.cursor = c.prev
	return c
}

func (l *LinkList[T]) Clear() {
	for {
		n := l.sentinel.next
		if n == l.sentinel {
			return
		}
		n.Unlink()
	}
}

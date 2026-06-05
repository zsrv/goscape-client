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

// Push appends v at the tail (the most-recent slot). Java: push
// (LinkList.java @32f3062; same name at 254 — the Go AddTail name was a
// port-era deviation, aligned during the 274 rename pass).
func (l *LinkList[T]) Push(v *Linkable[T]) {
	if v.prev != nil {
		v.Unlink()
	}
	v.prev = l.sentinel.prev
	v.next = l.sentinel
	v.prev.next = v
	v.next.prev = v
}

// PushFront inserts v at the head. Java: pushFront (LinkList.java @32f3062;
// was addHead at 254).
func (l *LinkList[T]) PushFront(v *Linkable[T]) {
	if v.prev != nil {
		v.Unlink()
	}
	v.prev = l.sentinel
	v.next = l.sentinel.next
	v.prev.next = v
	v.next.prev = v
}

// PopFront removes and returns the head node, or nil when empty.
// Java: popFront (LinkList.java @32f3062; was pop at 254).
func (l *LinkList[T]) PopFront() *Linkable[T] {
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
	// Java 274: early-return when already empty (LinkList.java:101-103
	// @32f3062) — functionally a no-op (the loop below exits immediately on
	// an empty list) but ported faithfully.
	if l.sentinel.next == l.sentinel {
		return
	}
	for {
		n := l.sentinel.next
		if n == l.sentinel {
			return
		}
		n.Unlink()
	}
}

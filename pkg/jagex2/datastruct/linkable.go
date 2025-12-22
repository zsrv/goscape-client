package datastruct

type Linkable[T any] struct {
	Value T
	next  *Linkable[T]
	prev  *Linkable[T]
}

func NewLinkable[T any](value T) *Linkable[T] {
	return &Linkable[T]{Value: value}
}

func (l *Linkable[T]) Unlink() {
	if l.prev != nil {
		l.prev.next = l.next
		l.next.prev = l.prev
		l.next = nil
		l.prev = nil
	}
}

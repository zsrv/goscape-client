package datastruct

type Linkable[T any] struct {
	Value T
	Next  *Linkable[T]
	Prev  *Linkable[T]
}

func (l *Linkable[T]) Unlink() {
	if l.Prev != nil {
		l.Prev.Next = l.Next
		l.Next.Prev = l.Prev
		l.Next = nil
		l.Prev = nil
	}
}

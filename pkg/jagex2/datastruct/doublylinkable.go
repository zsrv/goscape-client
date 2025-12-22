package datastruct

type DoublyLinkable[T any] struct {
	*Linkable[T]

	next2 *DoublyLinkable[T]
	prev2 *DoublyLinkable[T]
}

func NewDoublyLinkable[T any](value T) *DoublyLinkable[T] {
	return &DoublyLinkable[T]{
		Linkable: NewLinkable(value),
	}
}

func (d *DoublyLinkable[T]) Uncache() {
	if d.prev2 != nil {
		d.prev2.next2 = d.next2
		d.next2.prev2 = d.prev2
		d.next2 = nil
		d.prev2 = nil
	}
}

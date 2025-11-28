package datastruct

type DoublyLinkable[T any] struct {
	Linkable[T]

	Next2 *DoublyLinkable[T]
	Prev2 *DoublyLinkable[T]
}

func (d *DoublyLinkable[T]) Uncache() {
	if d.Prev2 != nil {
		d.Prev2.Next2 = d.Next2
		d.Next2.Prev2 = d.Prev2
		d.Next2 = nil
		d.Prev2 = nil
	}
}

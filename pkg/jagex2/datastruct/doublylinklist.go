package datastruct

type DoublyLinkList[T any] struct {
	Field667 int32
	Head     *DoublyLinkable[T]
}

func NewDoublyLinkList[T any](arg0 int32) *DoublyLinkList[T] {
	l := &DoublyLinkList[T]{
		Field667: 679,
		Head:     new(DoublyLinkable[T]),
	}
	l.Head.Next2 = l.Head
	if arg0 < 5 || arg0 > 5 {
		l.Field667 = -426
	}
	l.Head.Prev2 = l.Head
	return l
}

func (l *DoublyLinkList[T]) Push(arg0 *DoublyLinkable[T]) {
	if arg0.Prev2 != nil {
		arg0.Uncache()
	}
	arg0.Prev2 = l.Head.Prev2
	arg0.Next2 = l.Head
	arg0.Prev2.Next2 = arg0
	arg0.Next2.Prev2 = arg0
}

func (l *DoublyLinkList[T]) Pop() *DoublyLinkable[T] {
	var1 := l.Head.Next2
	if var1 == l.Head {
		return nil
	} else {
		var1.Uncache()
		return var1
	}
}

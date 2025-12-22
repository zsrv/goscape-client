package datastruct

type DoublyLinkList[T any] struct {
	head *DoublyLinkable[T]
}

func NewDoublyLinkList[T any]() *DoublyLinkList[T] {
	head := new(DoublyLinkable[T])
	head.next2 = head
	head.prev2 = head
	return &DoublyLinkList[T]{
		head: head,
	}
}

func (l *DoublyLinkList[T]) Push(arg0 *DoublyLinkable[T]) {
	if arg0.prev2 != nil {
		arg0.Uncache()
	}
	arg0.prev2 = l.head.prev2
	arg0.next2 = l.head
	arg0.prev2.next2 = arg0
	arg0.next2.prev2 = arg0
}

func (l *DoublyLinkList[T]) Pop() *DoublyLinkable[T] {
	var1 := l.head.next2
	if var1 == l.head {
		return nil
	}
	var1.Uncache()
	return var1
}

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

func (l *DoublyLinkList[T]) Push(node *DoublyLinkable[T]) {
	if node.prev2 != nil {
		node.Uncache()
	}
	node.prev2 = l.head.prev2
	node.next2 = l.head
	node.prev2.next2 = node
	node.next2.prev2 = node
}

func (l *DoublyLinkList[T]) Pop() *DoublyLinkable[T] {
	node := l.head.next2
	if node == l.head {
		return nil
	}
	node.Uncache()
	return node
}

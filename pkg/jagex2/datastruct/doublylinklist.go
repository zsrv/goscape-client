package datastruct

type DoublyLinkList[T any] struct {
	head   *DoublyLinkable[T]
	cursor *DoublyLinkable[T]
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

// Head returns the first node and seeds the cursor for Next, mirroring Java
// DoublyLinkList.head() (datastruct/DoublyLinkList.java).
func (l *DoublyLinkList[T]) Head() *DoublyLinkable[T] {
	n := l.head.next2
	if n == l.head {
		l.cursor = nil
		return nil
	}
	l.cursor = n.next2
	return n
}

// Next advances the cursor seeded by Head. Java DoublyLinkList.next().
func (l *DoublyLinkList[T]) Next() *DoublyLinkable[T] {
	n := l.cursor
	if n == l.head {
		l.cursor = nil
		return nil
	}
	l.cursor = n.next2
	return n
}

// Size counts live nodes. Java DoublyLinkList.size().
func (l *DoublyLinkList[T]) Size() int {
	count := 0
	for n := l.head.next2; n != l.head; n = n.next2 {
		count++
	}
	return count
}

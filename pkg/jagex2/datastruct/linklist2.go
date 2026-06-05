package datastruct

type LinkList2[T any] struct {
	head   *Linkable2[T]
	cursor *Linkable2[T]
}

func NewLinkList2[T any]() *LinkList2[T] {
	head := new(Linkable2[T])
	head.next2 = head
	head.prev2 = head
	return &LinkList2[T]{
		head: head,
	}
}

func (l *LinkList2[T]) Push(node *Linkable2[T]) {
	if node.prev2 != nil {
		node.Uncache()
	}
	node.prev2 = l.head.prev2
	node.next2 = l.head
	node.prev2.next2 = node
	node.next2.prev2 = node
}

func (l *LinkList2[T]) Pop() *Linkable2[T] {
	node := l.head.next2
	if node == l.head {
		return nil
	}
	node.Uncache()
	return node
}

// Head returns the first node and seeds the cursor for Next, mirroring Java
// LinkList2.head() (datastruct/LinkList2.java).
func (l *LinkList2[T]) Head() *Linkable2[T] {
	n := l.head.next2
	if n == l.head {
		l.cursor = nil
		return nil
	}
	l.cursor = n.next2
	return n
}

// Next advances the cursor seeded by Head. Java LinkList2.next().
func (l *LinkList2[T]) Next() *Linkable2[T] {
	n := l.cursor
	if n == l.head {
		l.cursor = nil
		return nil
	}
	l.cursor = n.next2
	return n
}

// Size counts live nodes. Java LinkList2.size().
func (l *LinkList2[T]) Size() int {
	count := 0
	for n := l.head.next2; n != l.head; n = n.next2 {
		count++
	}
	return count
}

package datastruct

type DoublyLinkList struct {
	Field667 int32
	Head     *DoublyLinkable
}

func NewDoublyLinkList(arg0 int32) *DoublyLinkList {
	l := &DoublyLinkList{
		Field667: 679,
		Head:     new(DoublyLinkable),
	}
	l.Head.Next2 = l.Head
	if arg0 < 5 || arg0 > 5 {
		l.Field667 = -426
	}
	l.Head.Prev2 = l.Head
	return l
}

func (l *DoublyLinkList) Push(arg0 *DoublyLinkable) {
	if arg0.Prev2 != nil {
		arg0.Uncache()
	}
	arg0.Prev2 = l.Head.Prev2
	arg0.Next2 = l.Head
	arg0.Prev2.Next2 = arg0
	arg0.Next2.Prev2 = arg0
}

func (l *DoublyLinkList) Pop() *DoublyLinkable {
	var1 := l.Head.Next2
	if var1 == l.Head {
		return nil
	} else {
		var1.Uncache()
		return var1
	}
}

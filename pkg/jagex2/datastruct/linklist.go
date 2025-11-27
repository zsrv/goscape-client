package datastruct

type LinkList struct {
	Field661 bool
	Field662 int8
	Field663 int32
	Field664 int32
	Sentinel *Linkable
	Cursor   *Linkable
}

func NewLinkList(arg0 int32) *LinkList {
	l := &LinkList{
		Field661: true,
		Field662: 2,
		Field663: -546,
		Field664: -676,
		Sentinel: new(Linkable),
	}
	if arg0 != 0 {
		l.Field661 = !l.Field661
	}
	l.Sentinel.Next = l.Sentinel
	l.Sentinel.Prev = l.Sentinel
	return l
}

func (l *LinkList) AddTail(arg0 *Linkable) {
	if arg0.Prev != nil {
		arg0.Unlink()
	}
	arg0.Prev = l.Sentinel.Prev
	arg0.Next = l.Sentinel
	arg0.Prev.Next = arg0
	arg0.Next.Prev = arg0
}

func (l *LinkList) AddHead(arg0 *Linkable, arg1 int32) {
	if arg0.Prev != nil {
		arg0.Unlink()
	}
	arg0.Prev = l.Sentinel
	if arg1 == -26173 {
		arg0.Next = l.Sentinel.Next
		arg0.Prev.Next = arg0
		arg0.Next.Prev = arg0
	}
}

func (l *LinkList) RemoveHead() *Linkable {
	var1 := l.Sentinel.Next
	if var1 == l.Sentinel {
		return nil
	} else {
		var1.Unlink()
		return var1
	}
}

func (l *LinkList) Head() *Linkable {
	var1 := l.Sentinel.Next
	if var1 == l.Sentinel {
		l.Cursor = nil
		return nil
	} else {
		l.Cursor = var1.Next
		return var1
	}
}

func (l *LinkList) Tail(arg0 int8) *Linkable {
	var2 := l.Sentinel.Prev
	if var2 == l.Sentinel {
		l.Cursor = nil
		return nil
	}
	l.Cursor = var2.Prev
	if arg0 != l.Field662 {
		l.Field664 = 112
	}
	return var2
}

func (l *LinkList) Next(arg0 int8) *Linkable {
	if arg0 <= 0 {
		panic("null pointer exception")
	}
	var2 := l.Cursor
	if var2 == l.Sentinel {
		l.Cursor = nil
		return nil
	} else {
		l.Cursor = var2.Next
		return var2
	}
}

func (l *LinkList) Prev(arg0 bool) *Linkable {
	var2 := l.Cursor
	if arg0 {
		for var3 := 1; var3 > 0; var3++ {
		}
	}
	if var2 == l.Sentinel {
		l.Cursor = nil
		return nil
	} else {
		l.Cursor = var2.Prev
		return var2
	}
}

func (l *LinkList) Clear() {
	for {
		var1 := l.Sentinel.Next
		if var1 == l.Sentinel {
			return
		}
		var1.Unlink()
	}
}

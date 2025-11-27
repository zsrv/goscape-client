package datastruct

type Linkable struct {
	Key  int64
	Next *Linkable
	Prev *Linkable
}

func (l *Linkable) Unlink() {
	if l.Prev != nil {
		l.Prev.Next = l.Next
		l.Next.Prev = l.Prev
		l.Next = nil
		l.Prev = nil
	}
}

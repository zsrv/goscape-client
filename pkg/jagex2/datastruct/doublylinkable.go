package datastruct

type DoublyLinkable struct {
	Linkable

	Next2 *DoublyLinkable
	Prev2 *DoublyLinkable
}

func (d *DoublyLinkable) Uncache() {
	if d.Prev2 != nil {
		d.Prev2.Next2 = d.Next2
		d.Next2.Prev2 = d.Prev2
		d.Next2 = nil
		d.Prev2 = nil
	}
}

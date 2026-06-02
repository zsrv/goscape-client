package datastruct

import "testing"

// TestDoublyLinkListSizeHeadNextPop exercises the three new iteration
// primitives (Head/Next/Size) and confirms FIFO order matches the push order.
func TestDoublyLinkListSizeHeadNextPop(t *testing.T) {
	l := NewDoublyLinkList[int]()

	a := NewDoublyLinkable(10)
	b := NewDoublyLinkable(20)
	c := NewDoublyLinkable(30)
	l.Push(a)
	l.Push(b)
	l.Push(c)

	if got := l.Size(); got != 3 {
		t.Fatalf("Size after 3 pushes: got %d, want 3", got)
	}

	// Head/Next must yield nodes in FIFO (insertion) order: a, b, c.
	want := []*DoublyLinkable[int]{a, b, c}
	i := 0
	for n := l.Head(); n != nil; n = l.Next() {
		if i >= len(want) {
			t.Fatalf("iteration yielded more than %d nodes", len(want))
		}
		if n != want[i] {
			t.Errorf("node[%d]: got value %d, want %d", i, n.Value, want[i].Value)
		}
		i++
	}
	if i != len(want) {
		t.Fatalf("iteration yielded %d nodes, want %d", i, len(want))
	}

	// Pop removes the head node; Size should shrink to 2.
	popped := l.Pop()
	if popped != a {
		t.Fatalf("Pop: got value %d, want %d", popped.Value, a.Value)
	}
	if got := l.Size(); got != 2 {
		t.Fatalf("Size after Pop: got %d, want 2", got)
	}
}

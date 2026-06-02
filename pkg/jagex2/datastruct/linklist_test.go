package datastruct

import "testing"

func linkListLen[T any](l *LinkList[T]) int {
	n := 0
	for x := l.Head(); x != nil; x = l.Next() {
		n++
	}
	return n
}

// TestLinkListAddHeadMovesExistingNode pins the contract that the
// post-2026-05-22 SortObjStacks fix relies on: AddHead on a *Linkable
// already linked into the list moves it to the head without growing the
// list. The pre-fix bug in client.SortObjStacks wrapped an
// already-linked entity in a fresh *Linkable via NewLinkable(value),
// which had prev==nil, so AddHead's unlink-on-add guard never fired
// and every call leaked a duplicate node. See deob/client.java:8490
// (Java's `ClientObj extends Linkable`, so addHead(var5) moves
// the existing node).
func TestLinkListAddHeadMovesExistingNode(t *testing.T) {
	l := NewLinkList[int]()
	a := NewLinkable(1)
	b := NewLinkable(2)
	c := NewLinkable(3)
	l.AddTail(a)
	l.AddTail(b)
	l.AddTail(c)

	if got := linkListLen(l); got != 3 {
		t.Fatalf("initial len=%d, want 3", got)
	}

	l.AddHead(c)

	if got := linkListLen(l); got != 3 {
		t.Fatalf("after AddHead(existing): len=%d, want 3 (duplicate leak)", got)
	}
	if got := l.Head(); got != c {
		t.Fatalf("after AddHead(existing): head=%v, want c", got)
	}
	if got := l.Next(); got != a {
		t.Fatalf("after AddHead(existing): second=%v, want a", got)
	}
	if got := l.Next(); got != b {
		t.Fatalf("after AddHead(existing): third=%v, want b", got)
	}
	if got := l.Next(); got != nil {
		t.Fatalf("after AddHead(existing): expected end of list, got %v", got)
	}
}

// TestLinkListAddHeadFreshDoesGrow is the deliberate contrast to
// TestLinkListAddHeadMovesExistingNode: a freshly-constructed
// *Linkable (prev==nil) is appended without unlinking anything, so the
// list grows by one. This is the path the SortObjStacks pre-fix took.
func TestLinkListAddHeadFreshDoesGrow(t *testing.T) {
	l := NewLinkList[int]()
	l.AddTail(NewLinkable(1))
	l.AddTail(NewLinkable(2))

	l.AddHead(NewLinkable(3))

	if got := linkListLen(l); got != 3 {
		t.Fatalf("after AddHead(fresh): len=%d, want 3", got)
	}
}

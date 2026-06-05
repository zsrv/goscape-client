package datastruct

import "testing"

func linkListLen[T any](l *LinkList[T]) int {
	n := 0
	for x := l.Head(); x != nil; x = l.Next() {
		n++
	}
	return n
}

// TestLinkListPushFrontMovesExistingNode pins the contract that the
// post-2026-05-22 SortObjStacks fix relies on: PushFront (ex-AddHead) on a *Linkable
// already linked into the list moves it to the head without growing the
// list. The pre-fix bug in client.SortObjStacks wrapped an
// already-linked entity in a fresh *Linkable via NewLinkable(value),
// which had prev==nil, so PushFront's unlink-on-add guard never fired
// and every call leaked a duplicate node. See deob/client.java:8490
// (Java's `ClientObj extends Linkable`, so addHead(var5) moves
// the existing node).
func TestLinkListPushFrontMovesExistingNode(t *testing.T) {
	l := NewLinkList[int]()
	a := NewLinkable(1)
	b := NewLinkable(2)
	c := NewLinkable(3)
	l.Push(a)
	l.Push(b)
	l.Push(c)

	if got := linkListLen(l); got != 3 {
		t.Fatalf("initial len=%d, want 3", got)
	}

	l.PushFront(c)

	if got := linkListLen(l); got != 3 {
		t.Fatalf("after PushFront(existing): len=%d, want 3 (duplicate leak)", got)
	}
	if got := l.Head(); got != c {
		t.Fatalf("after PushFront(existing): head=%v, want c", got)
	}
	if got := l.Next(); got != a {
		t.Fatalf("after PushFront(existing): second=%v, want a", got)
	}
	if got := l.Next(); got != b {
		t.Fatalf("after PushFront(existing): third=%v, want b", got)
	}
	if got := l.Next(); got != nil {
		t.Fatalf("after PushFront(existing): expected end of list, got %v", got)
	}
}

// TestLinkListPushFrontFreshDoesGrow is the deliberate contrast to
// TestLinkListPushFrontMovesExistingNode: a freshly-constructed
// *Linkable (prev==nil) is appended without unlinking anything, so the
// list grows by one. This is the path the SortObjStacks pre-fix took.
func TestLinkListPushFrontFreshDoesGrow(t *testing.T) {
	l := NewLinkList[int]()
	l.Push(NewLinkable(1))
	l.Push(NewLinkable(2))

	l.PushFront(NewLinkable(3))

	if got := linkListLen(l); got != 3 {
		t.Fatalf("after PushFront(fresh): len=%d, want 3", got)
	}
}

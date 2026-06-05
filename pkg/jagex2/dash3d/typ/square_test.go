package typ

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct"
)

// TestGroundDrawQueueNodeIdentity locks in the World.DrawTile parity fix.
// Because Java's `Square extends Linkable`, enqueuing an already-queued tile
// MOVES it to the tail (addTail unlinks first) and a Square can appear in
// drawTileQueue at most once. The Go Square owns one reusable node
// (DrawQueueNode), so re-adding it must move the existing entry rather than
// create a duplicate — the pre-fix behavior allocated a fresh wrapper per add,
// which left duplicates in the queue and never moved the original.
func TestGroundDrawQueueNodeIdentity(t *testing.T) {
	a := NewSquare(0, 1, 1)
	b := NewSquare(0, 2, 2)

	// The node's Value must point back at its owning Square.
	if a.DrawQueueNode.Value != a {
		t.Fatal("DrawQueueNode.Value should point back at its Square")
	}

	q := datastruct.NewLinkList[*Square]()
	q.Push(a.DrawQueueNode)
	q.Push(b.DrawQueueNode)
	q.Push(a.DrawQueueNode) // re-add A: must MOVE to tail, not duplicate

	var order []*Square
	for {
		n := q.PopFront()
		if n == nil {
			break
		}
		order = append(order, n.Value)
	}

	if len(order) != 2 {
		t.Fatalf("expected 2 entries (no duplicate of A), got %d", len(order))
	}
	if order[0] != b || order[1] != a {
		t.Fatalf("expected queue order [B, A] after re-adding A, got match=[%v, %v]",
			order[0] == b, order[1] == a)
	}
}

package typ

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct"
)

// TestGroundDrawQueueNodeIdentity locks in the World3D.DrawTile parity fix.
// Because Java's `Ground extends Linkable`, enqueuing an already-queued tile
// MOVES it to the tail (addTail unlinks first) and a Ground can appear in
// drawTileQueue at most once. The Go Ground owns one reusable node
// (DrawQueueNode), so re-adding it must move the existing entry rather than
// create a duplicate — the pre-fix behavior allocated a fresh wrapper per add,
// which left duplicates in the queue and never moved the original.
func TestGroundDrawQueueNodeIdentity(t *testing.T) {
	a := NewGround(0, 1, 1)
	b := NewGround(0, 2, 2)

	// The node's Value must point back at its owning Ground.
	if a.DrawQueueNode.Value != a {
		t.Fatal("DrawQueueNode.Value should point back at its Ground")
	}

	q := datastruct.NewLinkList[*Ground]()
	q.AddTail(a.DrawQueueNode)
	q.AddTail(b.DrawQueueNode)
	q.AddTail(a.DrawQueueNode) // re-add A: must MOVE to tail, not duplicate

	var order []*Ground
	for {
		n := q.RemoveHead()
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

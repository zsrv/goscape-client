package typ

import "github.com/zsrv/goscape-client/pkg/jagex2/datastruct"

type Square struct {
	Level                int
	X                    int
	Z                    int
	OccludeLevel         int
	Underlay             *QuickGround
	Overlay              *Ground
	Wall                 *Wall
	Decor                *Decor
	GroundDecor          *GroundDecor
	GroundObj            *GroundObject
	LocCount             int
	Locs                 [5]*Sprite // Java: Sprite[] locs = new Sprite[5] (Square.java:43)
	LocSpan              [5]int     // Java: int[] locSpan = new int[5] (Square.java:46)
	LocSpans             int
	DrawLevel            int
	Visible              bool
	Update               bool
	ContainsLocs         bool
	CheckLocSpans        int
	BlockLocSpans        int
	InverseBlockLocSpans int
	BackWallTypes        int
	Bridge               *Square

	// Java: `Square extends Linkable` (Square.java:7) — a Square IS its own
	// intrusive list node, so World3D.drawTileQueue.addTail moves an
	// already-queued tile to the tail (addTail unlinks first) and a Square can
	// appear in the queue at most once. Go can't embed the generic node as a
	// base class, so each Square owns a single reusable node whose Value points
	// back to itself; DrawTile enqueues this node rather than a fresh wrapper.
	DrawQueueNode *datastruct.Linkable[*Square]
}

func NewSquare(level, x, z int) *Square {
	var g Square
	g.Level = level
	g.OccludeLevel = g.Level
	g.X = x
	g.Z = z

	g.DrawQueueNode = datastruct.NewLinkable(&g)

	return &g
}

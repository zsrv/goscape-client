package typ

import "github.com/zsrv/goscape-client/pkg/jagex2/datastruct"

type Ground struct {
	Level                int
	X                    int
	Z                    int
	OccludeLevel         int
	Underlay             *TileUnderlay
	Overlay              *TileOverlay
	Wall                 *Wall
	Decor                *Decor
	GroundDecor          *GroundDecor
	GroundObj            *GroundObject
	LocCount             int
	Locs                 []*Location
	LocSpan              []int
	LocSpans             int
	DrawLevel            int
	Visible              bool
	Update               bool
	ContainsLocs         bool
	CheckLocSpans        int
	BlockLocSpans        int
	InverseBlockLocSpans int
	BackWallTypes        int
	Bridge               *Ground

	// Java: `Ground extends Linkable` (Ground.java:7) — a Ground IS its own
	// intrusive list node, so World3D.drawTileQueue.addTail moves an
	// already-queued tile to the tail (addTail unlinks first) and a Ground can
	// appear in the queue at most once. Go can't embed the generic node as a
	// base class, so each Ground owns a single reusable node whose Value points
	// back to itself; DrawTile enqueues this node rather than a fresh wrapper.
	DrawQueueNode *datastruct.Linkable[*Ground]
}

func NewGround(level, x, z int) *Ground {
	var g Ground
	g.Locs = make([]*Location, 5)
	g.LocSpan = make([]int, 5)

	g.Level = level
	g.OccludeLevel = g.Level
	g.X = x
	g.Z = z

	g.DrawQueueNode = datastruct.NewLinkable(&g)

	return &g
}

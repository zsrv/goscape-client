package typ

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
}

func NewGround(arg0, arg1, arg2 int) *Ground {
	var g Ground
	g.Locs = make([]*Location, 5)
	g.LocSpan = make([]int, 5)

	g.Level = arg0
	g.OccludeLevel = g.Level
	g.X = arg1
	g.Z = arg2

	return &g
}

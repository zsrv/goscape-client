package dash3d

type Occlude struct {
	MinTileX  int
	MaxTileX  int
	MinTileZ  int
	MaxTileZ  int
	Type      int
	MinX      int
	MaxX      int
	MinZ      int
	MaxZ      int
	MinY      int
	MaxY      int
	Mode      int
	MinDeltaX int
	MaxDeltaX int
	MinDeltaZ int
	MaxDeltaZ int
	MinDeltaY int
	MaxDeltaY int
}

func NewOcclude() *Occlude {
	return new(Occlude)
}

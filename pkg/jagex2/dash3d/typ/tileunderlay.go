package typ

type TileUnderlay struct {
	SouthwestColor int
	SoutheastColor int
	NortheastColor int
	NorthwestColor int
	TextureID      int
	Flat           bool
	RGB            int
}

func NewTileUnderlay(southwestColor, southeastColor, northeastColor, northwestColor, textureID, rgb int, flat bool) *TileUnderlay {
	return &TileUnderlay{
		SouthwestColor: southwestColor,
		SoutheastColor: southeastColor,
		NortheastColor: northeastColor,
		NorthwestColor: northwestColor,
		TextureID:      textureID,
		RGB:            rgb,
		Flat:           flat,
	}
}

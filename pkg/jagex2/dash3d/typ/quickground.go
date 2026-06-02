package typ

type QuickGround struct {
	SouthwestColor int
	SoutheastColor int
	NortheastColor int
	NorthwestColor int
	TextureID      int
	Flat           bool
	RGB            int
}

func NewQuickGround(southwestColor, southeastColor, northeastColor, northwestColor, textureID, rgb int, flat bool) *QuickGround {
	return &QuickGround{
		SouthwestColor: southwestColor,
		SoutheastColor: southeastColor,
		NortheastColor: northeastColor,
		NorthwestColor: northwestColor,
		TextureID:      textureID,
		RGB:            rgb,
		Flat:           flat,
	}
}

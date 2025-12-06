package typ

import "goscape-client/pkg/jagex2/graphics/model"

type GroundDecor struct {
	Y      int
	X      int
	Z      int
	Model  *model.Model
	BitSet int
	Info   byte
}

func NewGroundDecor() *GroundDecor {
	return new(GroundDecor)
}

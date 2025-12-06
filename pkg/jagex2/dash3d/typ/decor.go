package typ

import "goscape-client/pkg/jagex2/graphics/model"

type Decor struct {
	Y      int
	X      int
	Z      int
	Type   int
	Angle  int
	Model  *model.Model
	BitSet int
	Info   byte
}

func NewDecor() *Decor {
	return new(Decor)
}

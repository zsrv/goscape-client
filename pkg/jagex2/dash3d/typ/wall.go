package typ

import "goscape-client/pkg/jagex2/graphics/model"

type Wall struct {
	Y      int
	X      int
	Z      int
	TypeA  int
	TypeB  int
	ModelA *model.Model
	ModelB *model.Model
	BitSet int
	Info   byte
}

func NewWall() *Wall {
	return new(Wall)
}

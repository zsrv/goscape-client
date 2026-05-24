package typ

import "github.com/zsrv/goscape-client/pkg/jagex2/graphics/model"

type Wall struct {
	Y      int
	X      int
	Z      int
	TypeA  int
	TypeB  int
	ModelA *model.Model
	ModelB *model.Model
	BitSet int
	Info   int8 // Java: byte (signed); always read as int(Info)&0xFF
}

func NewWall() *Wall {
	return new(Wall)
}

package typ

import "github.com/zsrv/goscape-client/pkg/jagex2/dash3d/entity"

// Java: rev-244 Wall. ModelA/ModelB hold the rev-244 model1/model2 (now
// ModelSource, so a static *Model or a self-animating ClientLocAnim). Field
// names TypeA/TypeB/BitSet/Info are kept under their rev-225 Go names (244:
// angle1/angle2/typecode1/typecode2).
type Wall struct {
	Y      int
	X      int
	Z      int
	TypeA  int
	TypeB  int
	ModelA entity.ModelSource
	ModelB entity.ModelSource
	BitSet int
	Info   int8 // Java: byte (signed); always read as int(Info)&0xFF
}

func NewWall() *Wall {
	return new(Wall)
}

package typ

import "github.com/zsrv/goscape-client/pkg/jagex2/dash3d/entity"

// Java: rev-244 GroundDecor. Model is now a ModelSource (static *Model or
// self-animating ClientLocAnim).
type GroundDecor struct {
	Y      int
	X      int
	Z      int
	Model  entity.ModelSource
	BitSet int
	Info   int8 // Java: byte (signed); always read as int(Info)&0xFF
}

func NewGroundDecor() *GroundDecor {
	return new(GroundDecor)
}

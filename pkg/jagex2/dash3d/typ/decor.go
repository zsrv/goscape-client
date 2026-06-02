package typ

import "github.com/zsrv/goscape-client/pkg/jagex2/dash3d/entity"

// Java: rev-244 Decor. Model is now a ModelSource (static *Model or
// self-animating ClientLocAnim). Type/Angle keep their rev-225 Go names (244:
// angle1/angle2).
type Decor struct {
	Y      int
	X      int
	Z      int
	Type   int
	Angle  int
	Model  entity.ModelSource
	BitSet int
	Info   int8 // Java: byte (signed); always read as int(Info)&0xFF
	// MinY mirrors rev-244 ModelSource.minY (default 1000): seeded from a static
	// model at add time and refreshed when drawn, read by the decor visibility
	// cull (rev-244 swapped the rev-225 resolved model.maxY arg for cached minY).
	MinY int
}

func NewDecor() *Decor {
	return &Decor{MinY: 1000}
}

package typ

import "github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"

type Decor struct {
	Y      int
	X      int
	Z      int
	Type   int
	Angle  int
	Model  *model.Model
	BitSet int
	Info   int8 // Java: byte (signed); always read as int(Info)&0xFF
}

func NewDecor() *Decor {
	return new(Decor)
}

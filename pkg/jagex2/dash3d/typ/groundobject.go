package typ

import "goscape-client/pkg/jagex2/graphics/model"

type GroundObject struct {
	Y         int
	X         int
	Z         int
	TopObj    *model.Model
	BottomObj *model.Model
	MiddleObj *model.Model
	BitSet    int
	Offset    int
}

func NewGroundObject() *GroundObject {
	return new(GroundObject)
}

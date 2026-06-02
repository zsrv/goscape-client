package typ

import "github.com/zsrv/goscape-client/pkg/jagex2/dash3d/entity"

// Java: rev-244 GroundObject. top/bottom/middle are now ModelSource. The Go
// names TopObj/BottomObj/MiddleObj and Offset (244: height) are kept.
type GroundObject struct {
	Y         int
	X         int
	Z         int
	TopObj    entity.ModelSource
	BottomObj entity.ModelSource
	MiddleObj entity.ModelSource
	BitSet    int
	Offset    int
}

func NewGroundObject() *GroundObject {
	return new(GroundObject)
}

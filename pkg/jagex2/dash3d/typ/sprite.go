package typ

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/entity"
)

// Java: rev-244 Sprite. The rev-225 Model (*Model) + Entity (ModelSource) pair
// is collapsed into the single rev-244 ModelSource field Model (a static *Model
// or a self-animating ClientLocAnim).
type Sprite struct {
	Level         int
	Y             int
	X             int
	Z             int
	Model         entity.ModelSource
	Yaw           int
	MinSceneTileX int
	MaxSceneTileX int
	MinSceneTileZ int
	MaxSceneTileZ int
	Distance      int
	Cycle         int
	BitSet        int
	Info          int8 // Java: byte (signed); always read as int(Info)&0xFF
	// MinY mirrors rev-244 ModelSource.minY (default 1000) for this node's one
	// model: seeded from a static model at add time and refreshed when drawn,
	// then read by the loc visibility cull (rev-244 swapped the rev-225 resolved
	// model.maxY arg for the source's cached minY).
	MinY int
}

func NewSprite() *Sprite {
	return &Sprite{MinY: 1000}
}

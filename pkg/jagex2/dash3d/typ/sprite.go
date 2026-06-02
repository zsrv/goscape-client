package typ

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/entity"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
)

type Sprite struct {
	Level         int
	Y             int
	X             int
	Z             int
	Model         *model.Model
	Entity        entity.ModelSource
	Yaw           int
	MinSceneTileX int
	MaxSceneTileX int
	MinSceneTileZ int
	MaxSceneTileZ int
	Distance      int
	Cycle         int
	BitSet        int
	Info          int8 // Java: byte (signed); always read as int(Info)&0xFF
}

func NewSprite() *Sprite {
	return new(Sprite)
}

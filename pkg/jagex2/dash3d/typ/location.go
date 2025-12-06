package typ

import (
	"goscape-client/pkg/jagex2/dash3d/entity"
	"goscape-client/pkg/jagex2/graphics/model"
)

type Location struct {
	Level         int
	Y             int
	X             int
	Z             int
	Model         *model.Model
	Entity        entity.Entity
	Yaw           int
	MinSceneTileX int
	MaxSceneTileX int
	MinSceneTileZ int
	MaxSceneTileZ int
	Distance      int
	Cycle         int
	BitSet        int
	Info          byte
}

func NewLocation() *Location {
	return new(Location)
}

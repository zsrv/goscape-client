package entity

import "goscape-client/pkg/jagex2/datastruct"

type LocMergeEntity struct {
	datastruct.Linkable[*LocMergeEntity]

	Plane     int
	Layer     int
	X         int
	Z         int
	LocIndex  int
	Angle     int
	Shape     int
	LastCycle int
}

func NewLocMergeEntity(plane, angle, z, lastCycle, shape, locIndex, x, layer int) *LocMergeEntity {
	var e LocMergeEntity
	e.Plane = plane
	e.Layer = layer
	e.X = x
	e.Z = z
	e.LocIndex = locIndex
	e.Angle = angle
	e.Shape = shape
	e.LastCycle = lastCycle
	return &e
}

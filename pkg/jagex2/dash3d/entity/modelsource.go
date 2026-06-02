package entity

import "github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"

type ModelSource interface {
	GetModel() *model.Model
}

// ModelSourceOf wraps a concrete *Model as a ModelSource, returning a genuine
// nil interface (not a typed-nil) when m is nil. This is required wherever a
// possibly-nil static loc model flows into a ModelSource field: Java's
// `ModelSource ref = (Model) null` is null, but Go's interface holding a typed
// nil pointer compares != nil, which would defeat the scene's `!= nil` add/draw
// guards and panic at draw. Java: the `model = loc.getModel(...)` assignments in
// World.addLoc (rev-244) where getModel may return null.
func ModelSourceOf(m *model.Model) ModelSource {
	if m == nil {
		return nil
	}
	return m
}

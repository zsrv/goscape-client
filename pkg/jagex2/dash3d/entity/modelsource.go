package entity

import "github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"

type ModelSource interface {
	// GetTempModel resolves the source's current drawable model.
	// Java: ModelSource.getTempModel (ModelSource.java:25 @2e62978; was
	// getModel in ≤245.2).
	GetTempModel() *model.Model
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

// ObjSourceOf wraps a possibly-nil *ClientObj as a ModelSource, returning a
// genuine nil interface when o is nil — the same typed-nil trap ModelSourceOf
// guards (a typed-nil would pass the scene's `!= nil` draw guards and panic
// in GetTempModel's nil receiver). Java: showObject passes its second/third
// ClientObj refs, which may be null, straight into World.setObj
// (Client.java:3904-3915 @32f3062).
func ObjSourceOf(o *ClientObj) ModelSource {
	if o == nil {
		return nil
	}
	return o
}

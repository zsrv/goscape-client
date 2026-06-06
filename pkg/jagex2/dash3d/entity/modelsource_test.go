package entity

import "testing"

// TestObjSourceOfNil verifies ObjSourceOf normalizes a nil *ClientObj to a
// genuine nil interface — a typed-nil would pass the scene's `!= nil` draw
// guards and panic inside GetModel (nil receiver). Same trap class as
// ModelSourceOf (see its doc comment).
func TestObjSourceOfNil(t *testing.T) {
	if src := ObjSourceOf(nil); src != nil {
		t.Fatalf("ObjSourceOf(nil) = %#v, want nil interface", src)
	}
}

// TestObjSourceOfNonNil verifies a real ClientObj passes through unwrapped,
// preserving the lazy per-frame GetModel resolution.
func TestObjSourceOfNonNil(t *testing.T) {
	o := NewClientObj()
	if src := ObjSourceOf(o); src != ModelSource(o) {
		t.Fatalf("ObjSourceOf(o) = %#v, want the ClientObj itself", src)
	}
}

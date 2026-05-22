package client

import (
	"testing"

	"goscape-client/pkg/jagex2/config/component"
	"goscape-client/pkg/jagex2/graphics/pix2d"
)

// TestDrawInterface_RendersChildren pins the post-fix early-return
// polarity at client.go:3440. Java `arg3.childId == null` (return when
// there are no children) had been translated as Go `arg3.ChildID != nil`
// — the operator was flipped, so every Type-0 layer with children
// silently early-returned and no interface ever drew its contents (all
// tabs, message boxes, bank UI etc. appeared blank). The fix restores
// `== nil`. This test reproduces the failing scenario by handing
// DrawInterface a Type-0 parent with a single Type-3 (filled rect) child
// and verifying the child's pixels actually land in pix2d.Data.
func TestDrawInterface_RendersChildren(t *testing.T) {
	const w, h = 50, 50
	pix2d.Reset()
	pix2d.Bind(w, make([]int, w*h), h)
	t.Cleanup(pix2d.Reset)

	prevInstances := component.Instances
	t.Cleanup(func() { component.Instances = prevInstances })

	const childID, fillColour = 1, 0x00FF00
	child := &component.Component{
		Id:     childID,
		Type:   3,
		Width:  10,
		Height: 10,
		Colour: fillColour,
		Fill:   true,
	}
	component.Instances = make([]*component.Component, childID+1)
	component.Instances[childID] = child

	parent := &component.Component{
		Type:    0,
		Width:   30,
		Height:  30,
		ChildID: []int{childID},
		ChildX:  []int{5},
		ChildY:  []int{5},
	}

	c := NewClient()
	c.DrawInterface(0, 0, parent, 0)

	// If the early-return polarity is wrong, DrawInterface returned
	// before its child loop and pix2d.Data stays all zero. After the
	// fix, the child's FillRect writes a 10x10 block of fillColour
	// starting at (5,5).
	hits := 0
	for _, p := range pix2d.Data {
		if p == fillColour {
			hits++
		}
	}
	if hits == 0 {
		t.Fatalf("DrawInterface did not render the Type-3 child (no fillColour pixels in pix2d.Data) — the early-return polarity at client.go:3440 has regressed")
	}
}

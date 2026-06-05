package entity

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/config/spotanimtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/animframe"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
)

// MapSpotAnim
type MapSpotAnim struct {
	Type        *spotanimtype.SpotAnimType
	StartCycle  int
	Level       int
	X           int
	Z           int
	Y           int
	SeqFrame    int
	SeqCycle    int
	SeqComplete bool
}

func NewMapSpotAnim(x, arg1, z, arg4, y, level, arg7 int) *MapSpotAnim {
	return &MapSpotAnim{
		Type:        spotanimtype.List[arg1],
		Level:       level,
		X:           x,
		Z:           z,
		Y:           y,
		StartCycle:  arg7 + arg4,
		SeqComplete: false,
	}
}

func (e *MapSpotAnim) Update(arg0 int) {
	e.SeqCycle += arg0
	for {
		for ok := true; ok; ok = e.SeqFrame >= 0 && e.SeqFrame < e.Type.Seq.NumFrames {
			for ok2 := true; ok2; ok2 = e.SeqFrame < e.Type.Seq.NumFrames {
				if e.SeqCycle <= e.Type.Seq.GetDuration(e.SeqFrame) {
					return
				}
				e.SeqCycle -= e.Type.Seq.GetDuration(e.SeqFrame) + 1
				e.SeqFrame++
			}
		}
		e.SeqFrame = 0
		e.SeqComplete = true
	}
}

// Java: getTempModel (MapSpotAnim.java:64-97 @2e62978; was getModel) — 254
// hoists the resolved frame id before the ctor (it was only computed inside
// the !seqComplete branch at 245.2) and derives the alpha-share flag from it.
func (e *MapSpotAnim) GetTempModel() *model.Model {
	mdl := e.Type.GetTempModel()
	// Java: MapSpotAnim.java:66-68 @2e62978 — nil while the spotanim model
	// faults in.
	if mdl == nil {
		return nil
	}

	var3 := e.Type.Seq.Frames[e.SeqFrame] // Java: var3
	spot := model.NewModel4(mdl, true, animframe.ShareAlpha(var3), false)

	if !e.SeqComplete {
		spot.PrepareAnim()
		spot.Animate(var3)
		spot.LabelFaces = nil
		spot.LabelVertices = nil
	}

	if e.Type.ResizeH != 128 || e.Type.ResizeV != 128 {
		spot.Scale(e.Type.ResizeH, e.Type.ResizeV, e.Type.ResizeH)
	}

	if e.Type.Orientation != 0 {
		switch e.Type.Orientation {
		case 90:
			spot.Rotate90()
		case 180:
			spot.Rotate90()
			spot.Rotate90()
		case 270:
			spot.Rotate90()
			spot.Rotate90()
			spot.Rotate90()
		}
	}

	spot.CalculateNormals(e.Type.Ambient+64, e.Type.Contrast+850, -30, -50, -30, true)
	return spot
}

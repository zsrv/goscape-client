package entity

import (
	"goscape-client/pkg/jagex2/config/spotanimtype"
	"goscape-client/pkg/jagex2/graphics/model"
)

// MapSpotAnim
type SpotAnimEntity struct {
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

func NewSpotAnimEntity(x, arg1, z, arg4, y, level, arg7 int) *SpotAnimEntity {
	return &SpotAnimEntity{
		Type:        spotanimtype.Instances[arg1],
		Level:       level,
		X:           x,
		Z:           z,
		Y:           y,
		StartCycle:  arg7 + arg4,
		SeqComplete: false,
	}
}

func (e *SpotAnimEntity) Update(arg0 int) {
	e.SeqCycle += arg0
	for {
		for ok := true; ok; ok = e.SeqFrame >= 0 && e.SeqFrame < e.Type.Seq.FrameCount {
			for ok2 := true; ok2; ok2 = e.SeqFrame < e.Type.Seq.FrameCount {
				if e.SeqCycle <= e.Type.Seq.Delay[e.SeqFrame] {
					return
				}
				e.SeqCycle -= e.Type.Seq.Delay[e.SeqFrame] + 1
				e.SeqFrame++
			}
		}
		e.SeqFrame = 0
		e.SeqComplete = true
	}
}

// GetModel
func (e *SpotAnimEntity) Draw() *model.Model {
	mdl := e.Type.GetModel()

	spot := model.NewModel4(mdl, true, !e.Type.AnimHasAlpha, false)

	if !e.SeqComplete {
		spot.CreateLabelReferences()
		spot.ApplyTransform(e.Type.Seq.Frames[e.SeqFrame])
		spot.LabelFaces = nil
		spot.LabelVertices = nil
	}

	if e.Type.ResizeH != 128 || e.Type.ResizeV != 128 {
		spot.Scale(e.Type.ResizeH, e.Type.ResizeV, e.Type.ResizeH)
	}

	if e.Type.Orientation != 0 {
		switch e.Type.Orientation {
		case 90:
			spot.RotateY90()
		case 180:
			spot.RotateY90()
			spot.RotateY90()
		case 270:
			spot.RotateY90()
			spot.RotateY90()
			spot.RotateY90()
		}
	}

	spot.CalculateNormals(e.Type.Ambient+64, e.Type.Contrast+850, -30, -50, -30, true)
	return spot
}

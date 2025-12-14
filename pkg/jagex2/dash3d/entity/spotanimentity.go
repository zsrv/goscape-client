package entity

import (
	"goscape-client/pkg/jagex2/config/spotanimtype"
	"goscape-client/pkg/jagex2/datastruct"
	"goscape-client/pkg/jagex2/graphics/model"
)

type SpotAnimEntity struct {
	datastruct.Linkable[*SpotAnimEntity]

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

func (e *SpotAnimEntity) Draw() *model.Model {
	var2 := e.Type.GetModel()
	var3 := model.NewModel4(var2, true, !e.Type.AnimHasAlpha, false)
	if !e.SeqComplete {
		var3.CreateLabelReferences()
		var3.ApplyTransform(e.Type.Seq.Frames[e.SeqFrame])
		var3.LabelFaces = nil
		var3.LabelVertices = nil
	}
	if e.Type.ResizeH != 128 || e.Type.ResizeV != 128 {
		var3.Scale(e.Type.ResizeH, e.Type.ResizeV, e.Type.ResizeH)
	}
	if e.Type.Orientation != 0 {
		switch e.Type.Orientation {
		case 90:
			var3.RotateY90()
		case 180:
			var3.RotateY90()
			var3.RotateY90()
		case 270:
			var3.RotateY90()
			var3.RotateY90()
			var3.RotateY90()
		}
	}
	var3.CalculateNormals(e.Type.Ambient+64, e.Type.Contrast+850, -30, -50, -30, true)
	return var3
}

package entity

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/config/npctype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/seqtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/spotanimtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/model"
)

type NpcEntity struct {
	PathingEntity

	Type *npctype.NpcType
}

func NewNpcEntity() *NpcEntity {
	e := new(NpcEntity)
	e.PathingEntity = *NewPathingEntity()
	return e
}

func (e *NpcEntity) Draw() *model.Model {
	if e.Type == nil {
		return nil
	}
	if e.SpotanimID == -1 || e.SpotanimFrame == -1 {
		return e.GetSequencedModel()
	}
	var2 := e.GetSequencedModel()
	var3 := spotanimtype.Instances[e.SpotanimID]
	var4 := model.NewModel4(var3.GetModel(), true, !var3.AnimHasAlpha, false)
	var4.Translate(-e.SpotanimOffset, 0, 0)
	var4.CreateLabelReferences()
	var4.ApplyTransform(var3.Seq.Frames[e.SpotanimFrame])
	var4.LabelFaces = nil
	var4.LabelVertices = nil
	if var3.ResizeH != 128 || var3.ResizeV != 128 {
		var4.Scale(var3.ResizeH, var3.ResizeV, var3.ResizeH)
	}
	var4.CalculateNormals(var3.Ambient+64, var3.Contrast+850, -30, -50, -30, true)
	var5 := []*model.Model{var2, var4}
	// Java: NpcEntity.java:35 — `new Model(var5, (byte) -31, 2)`.
	// Java's `(byte) -31` is a deobfuscator overload disambiguator
	// that NewModel3 never reads; dropped per the deob-artifact
	// exclusion policy.
	var6 := model.NewModel3(var5, 2)
	if e.Type.Size == 1 {
		var6.Pickable = true
	}
	return var6
}

func (e *NpcEntity) GetSequencedModel() *model.Model {
	if e.PrimarySeqID >= 0 && e.PrimarySeqDelay == 0 {
		var2 := seqtype.Instances[e.PrimarySeqID].Frames[e.PrimarySeqFrame]
		var4 := -1
		if e.SecondarySeqID >= 0 && e.SecondarySeqID != e.SeqStandID {
			var4 = seqtype.Instances[e.SecondarySeqID].Frames[e.SecondarySeqFrame]
		}
		return e.Type.GetSequencedModel(var2, var4, seqtype.Instances[e.PrimarySeqID].WalkMerge)
	}
	var2 := -1
	if e.SecondarySeqID >= 0 {
		var2 = seqtype.Instances[e.SecondarySeqID].Frames[e.SecondarySeqFrame]
	}
	var3 := e.Type.GetSequencedModel(var2, -1, nil)
	e.Height = var3.MaxY
	return var3
}

func (e *NpcEntity) IsVisible() bool {
	return e.Type != nil
}

package entity

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/config/npctype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/seqtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/spotanimtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
)

type ClientNpc struct {
	ClientEntity

	Type     *npctype.NpcType
	seqModel *model.Model // reused per-frame transformed model (avoids per-frame alloc)
}

func NewClientNpc() *ClientNpc {
	e := new(ClientNpc)
	e.ClientEntity = *NewClientEntity()
	return e
}

// Java: getModel (ClientNpc.java:15-56). 244 nil-guards the lazily-loaded
// animated model (ClientNpc.java:21-23 — nil while the model file is still
// faulting in via OnDemand), reads height before the spotanim merge, and
// guards the spot model the same way (ClientNpc.java:31).
func (e *ClientNpc) GetModel() *model.Model {
	if e.Type == nil {
		return nil
	}
	var2 := e.GetSequencedModel()
	if var2 == nil {
		return nil
	}
	// Java: super.height = model.minY — 244 Model.minY is Go Model.MaxY
	// (the deob lineages name the -y bound oppositely).
	e.Height = var2.MaxY
	if e.SpotanimID != -1 && e.SpotanimFrame != -1 {
		var3 := spotanimtype.Instances[e.SpotanimID]
		if spotModel := var3.GetModel(); spotModel != nil {
			var4 := model.NewModel4(spotModel, true, !var3.AnimHasAlpha, false)
			var4.Translate(-e.SpotanimOffset, 0, 0)
			var4.PrepareAnim()
			var4.Animate(var3.Seq.Frames[e.SpotanimFrame])
			var4.LabelFaces = nil
			var4.LabelVertices = nil
			if var3.ResizeH != 128 || var3.ResizeV != 128 {
				var4.Scale(var3.ResizeH, var3.ResizeV, var3.ResizeH)
			}
			var4.CalculateNormals(var3.Ambient+64, var3.Contrast+850, -30, -50, -30, true)
			var5 := []*model.Model{var2, var4}
			// Java: `new Model(true, 2, models)` (ClientNpc.java:47).
			var2 = model.NewModel3(var5, 2)
		}
	}
	if e.Type.Size == 1 {
		var2.UseAABBMouseCheck = true
	}
	return var2
}

// Java: getAnimatedModel (ClientNpc.java:59-77) — may return nil while the
// npc model faults in; callers nil-guard. 244 moves the height update into
// getModel (it was set here on the idle branch in 225).
func (e *ClientNpc) GetSequencedModel() *model.Model {
	if e.PrimarySeqID >= 0 && e.PrimarySeqDelay == 0 {
		var2 := seqtype.Instances[e.PrimarySeqID].Frames[e.PrimarySeqFrame]
		var4 := -1
		if e.SecondarySeqID >= 0 && e.SecondarySeqID != e.SeqStandID {
			var4 = seqtype.Instances[e.SecondarySeqID].Frames[e.SecondarySeqFrame]
		}
		if e.seqModel == nil {
			e.seqModel = &model.Model{}
		}
		return e.Type.GetSequencedModel(e.seqModel, var2, var4, seqtype.Instances[e.PrimarySeqID].WalkMerge)
	}
	var2 := -1
	if e.SecondarySeqID >= 0 {
		var2 = seqtype.Instances[e.SecondarySeqID].Frames[e.SecondarySeqFrame]
	}
	if e.seqModel == nil {
		e.seqModel = &model.Model{}
	}
	return e.Type.GetSequencedModel(e.seqModel, var2, -1, nil)
}

func (e *ClientNpc) IsVisible() bool {
	return e.Type != nil
}

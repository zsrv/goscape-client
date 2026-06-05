package entity

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/config/npctype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/seqtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/spotanimtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/animframe"
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

// Java: getTempModel (ClientNpc.java:14-48 @2e62978; was getModel) —
// nil-guards the lazily-loaded animated model (nil while the model file is
// still faulting in via OnDemand), reads height before the spotanim merge,
// and guards the spot model the same way.
func (e *ClientNpc) GetTempModel() *model.Model {
	if e.Type == nil {
		return nil
	}
	var2 := e.GetTempModel2()
	if var2 == nil {
		return nil
	}
	// Java: super.height = model.minY — 244 Model.minY is Go Model.MaxY
	// (the deob lineages name the -y bound oppositely).
	e.Height = var2.MaxY
	if e.SpotanimID != -1 && e.SpotanimFrame != -1 {
		var3 := spotanimtype.Instances[e.SpotanimID]
		if spotModel := var3.GetTempModel(); spotModel != nil {
			// Java: var5 = var3.seq.frames[spotanimFrame] hoisted before the
			// ctor at 254; shareAlpha takes the RESOLVED frame id — unlike
			// ClientPlayer, which passes the raw index
			// (ClientNpc.java:27-29 @2e62978).
			var5 := var3.Seq.Frames[e.SpotanimFrame] // Java: var5
			var4 := model.NewModel4(spotModel, true, animframe.ShareAlpha(var5), false)
			var4.Translate(-e.SpotanimOffset, 0, 0)
			var4.PrepareAnim()
			var4.Animate(var5)
			var4.LabelFaces = nil
			var4.LabelVertices = nil
			if var3.ResizeH != 128 || var3.ResizeV != 128 {
				var4.Scale(var3.ResizeH, var3.ResizeV, var3.ResizeH)
			}
			var4.CalculateNormals(var3.Ambient+64, var3.Contrast+850, -30, -50, -30, true)
			var7 := []*model.Model{var2, var4}
			// Java: `new Model(2, var7, true)` (ClientNpc.java:39-40 @2e62978).
			var2 = model.NewModel3(var7, 2)
		}
	}
	if e.Type.Size == 1 {
		var2.UseAABBMouseCheck = true
	}
	return var2
}

// Java: getTempModel2 (ClientNpc.java:50-66 @2e62978; was getAnimatedModel)
// — may return nil while the npc model faults in; callers nil-guard. 244
// moved the height update into getTempModel (it was set here on the idle
// branch in 225).
func (e *ClientNpc) GetTempModel2() *model.Model {
	if e.PrimarySeqID >= 0 && e.PrimarySeqDelay == 0 {
		var2 := seqtype.Instances[e.PrimarySeqID].Frames[e.PrimarySeqFrame]
		var4 := -1
		if e.SecondarySeqID >= 0 && e.SecondarySeqID != e.SeqStandID {
			var4 = seqtype.Instances[e.SecondarySeqID].Frames[e.SecondarySeqFrame]
		}
		if e.seqModel == nil {
			e.seqModel = &model.Model{}
		}
		return e.Type.GetTempModel(e.seqModel, var2, var4, seqtype.Instances[e.PrimarySeqID].WalkMerge)
	}
	var2 := -1
	if e.SecondarySeqID >= 0 {
		var2 = seqtype.Instances[e.SecondarySeqID].Frames[e.SecondarySeqFrame]
	}
	if e.seqModel == nil {
		e.seqModel = &model.Model{}
	}
	return e.Type.GetTempModel(e.seqModel, var2, -1, nil)
}

// Java: isReady (ClientNpc.java:69 @2e62978; was isVisible in ≤245.2).
func (e *ClientNpc) IsReady() bool {
	return e.Type != nil
}

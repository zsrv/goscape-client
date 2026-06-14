package playerentity

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/idktype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/objtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/seqtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/spotanimtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/entity"
	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct"
	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct/jstring"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/model"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

var (
	ModelCache *datastruct.LruCache[*model.Model]
)

func init() {
	ModelCache = datastruct.NewLruCache[*model.Model](200)
}

type PlayerEntity struct {
	entity.PathingEntity

	Name               string
	Visible            bool
	Gender             int
	HeadIcons          int
	Appearances        []int
	Colors             []int
	CombatLevel        int
	AppearanceHashCode int64
	Y                  int
	LocStartCycle      int
	LocStopCycle       int
	LocOffsetX         int
	LocOffsetY         int
	LocOffsetZ         int
	LocModel           *model.Model
	seqModel           *model.Model // reused per-frame transformed model
	MinTileX           int
	MinTileZ           int
	MaxTileX           int
	LowMemory          bool
	MaxTileZ           int
}

func NewPlayerEntity() *PlayerEntity {
	return &PlayerEntity{
		PathingEntity: *entity.NewPathingEntity(),

		Appearances: make([]int, 12),
		Colors:      make([]int, 5),
	}
}

func (e *PlayerEntity) Read(arg1 *io.Packet) {
	arg1.Pos = 0
	e.Gender = arg1.G1()
	e.HeadIcons = arg1.G1()
	for i := range 12 {
		var4 := arg1.G1()
		if var4 == 0 {
			e.Appearances[i] = 0
		} else {
			var5 := arg1.G1()
			e.Appearances[i] = (var4 << 8) + var5
		}
	}
	for i := range 5 {
		var5 := arg1.G1()
		if var5 < 0 || var5 >= len(clientextras.Field1307[i]) {
			var5 = 0
		}
		e.Colors[i] = var5
	}
	e.SeqStandID = arg1.G2()
	if e.SeqStandID == 0xFFFF {
		e.SeqStandID = -1
	}
	e.SeqTurnID = arg1.G2()
	if e.SeqTurnID == 0xFFFF {
		e.SeqTurnID = -1
	}
	e.SeqWalkID = arg1.G2()
	if e.SeqWalkID == 0xFFFF {
		e.SeqWalkID = -1
	}
	e.SeqTurnAroundID = arg1.G2()
	if e.SeqTurnAroundID == 0xFFFF {
		e.SeqTurnAroundID = -1
	}
	e.SeqTurnLeftID = arg1.G2()
	if e.SeqTurnLeftID == 0xFFFF {
		e.SeqTurnLeftID = -1
	}
	e.SeqTurnRightId = arg1.G2()
	if e.SeqTurnRightId == 0xFFFF {
		e.SeqTurnRightId = -1
	}
	e.SeqRunID = arg1.G2()
	if e.SeqRunID == 0xFFFF {
		e.SeqRunID = -1
	}
	e.Name = jstring.FormatName(jstring.FromBase37(arg1.G8()))
	e.CombatLevel = arg1.G1()
	e.Visible = true
	e.AppearanceHashCode = 0
	for i := range 12 {
		e.AppearanceHashCode <<= 0x4
		if e.Appearances[i] >= 256 {
			e.AppearanceHashCode += int64(e.Appearances[i] - 256)
		}
	}
	if e.Appearances[0] >= 256 {
		e.AppearanceHashCode += int64((e.Appearances[0] - 256) >> 4)
	}
	if e.Appearances[1] >= 256 {
		e.AppearanceHashCode += int64((e.Appearances[1] - 256) >> 8)
	}
	for i := range 5 {
		e.AppearanceHashCode <<= 0x3
		e.AppearanceHashCode += int64(e.Colors[i])
	}
	e.AppearanceHashCode <<= 0x1
	e.AppearanceHashCode += int64(e.Gender)
}

func (e *PlayerEntity) Draw() *model.Model {
	if !e.Visible {
		return nil
	}
	var2 := e.GetSequencedModel()
	e.Height = var2.MinY
	var2.Pickable = true
	if e.LowMemory {
		return var2
	}
	if e.SpotanimID != -1 && e.SpotanimFrame != -1 {
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
		var2 = model.NewModel3(var5, 2)
	}
	if e.LocModel != nil {
		if clientextras.LoopCycle >= e.LocStopCycle {
			e.LocModel = nil
		}
		if clientextras.LoopCycle >= e.LocStartCycle && clientextras.LoopCycle < e.LocStopCycle {
			var6 := e.LocModel
			var6.Translate(e.LocOffsetY-e.Y, e.LocOffsetX-e.X, e.LocOffsetZ-e.Z)
			if e.DstYaw == 512 {
				var6.RotateY90()
				var6.RotateY90()
				var6.RotateY90()
			} else if e.DstYaw == 0x400 {
				var6.RotateY90()
				var6.RotateY90()
			} else if e.DstYaw == 1536 {
				var6.RotateY90()
			}
			var8 := []*model.Model{var2, var6}
			var2 = model.NewModel3(var8, 2)
			if e.DstYaw == 512 {
				var6.RotateY90()
			} else if e.DstYaw == 0x400 {
				var6.RotateY90()
				var6.RotateY90()
			} else if e.DstYaw == 1536 {
				var6.RotateY90()
				var6.RotateY90()
				var6.RotateY90()
			}
			var6.Translate(e.Y-e.LocOffsetY, e.X-e.LocOffsetX, e.Z-e.LocOffsetZ)
		}
	}
	var2.Pickable = true
	return var2
}

func (e *PlayerEntity) GetSequencedModel() *model.Model {
	var2 := e.AppearanceHashCode
	var4 := -1
	var5 := -1
	var6 := -1
	var7 := -1
	if e.PrimarySeqID >= 0 && e.PrimarySeqDelay == 0 {
		var8 := seqtype.Instances[e.PrimarySeqID]
		var4 = var8.Frames[e.PrimarySeqFrame]
		if e.SecondarySeqID >= 0 && e.SecondarySeqID != e.SeqStandID {
			var5 = seqtype.Instances[e.SecondarySeqID].Frames[e.SecondarySeqFrame]
		}
		if var8.RightHand >= 0 {
			var6 = var8.RightHand
			// Java: `var2 += var6 - appearances[5] << 8` is 32-bit int arithmetic,
			// sign-extended into the long var2. int32(...) reproduces that wrap; Go's
			// 64-bit int would otherwise diverge for high righthand/lefthand values.
			var2 += int64(int32((var6 - e.Appearances[5]) << 8))
		}
		if var8.LeftHand >= 0 {
			var7 = var8.LeftHand
			var2 += int64(int32((var7 - e.Appearances[3]) << 16))
		}
	} else if e.SecondarySeqID >= 0 {
		var4 = seqtype.Instances[e.SecondarySeqID].Frames[e.SecondarySeqFrame]
	}
	var15 := ModelCache.Get(var2)
	if var15 == nil {
		var9 := make([]*model.Model, 12)
		var10 := 0
		for i := range 12 {
			var12 := e.Appearances[i]
			if var7 >= 0 && i == 3 {
				var12 = var7
			}
			if var6 >= 0 && i == 5 {
				var12 = var6
			}
			if var12 >= 256 && var12 < 512 {
				var9[var10] = idktype.Instances[var12-256].GetModel()
				var10++
			}
			if var12 >= 512 {
				var13 := objtype.Get(var12 - 512)
				var14 := var13.GetWornModel(e.Gender)
				if var14 != nil {
					var9[var10] = var14
					var10++
				}
			}
		}
		var15 = model.NewModel2(var9, var10)
		for i := range 5 {
			if e.Colors[i] != 0 {
				var15.Recolor(clientextras.Field1307[i][0], clientextras.Field1307[i][e.Colors[i]])
				if i == 1 {
					var15.Recolor(clientextras.Field1438[0], clientextras.Field1438[e.Colors[i]])
				}
			}
		}
		var15.CreateLabelReferences()
		var15.CalculateNormals(64, 850, -30, -50, -30, true)
		ModelCache.Put(var2, var15)
	}
	if e.LowMemory {
		return var15
	}
	if e.seqModel == nil {
		e.seqModel = &model.Model{}
	}
	e.seqModel.ResetFromModel6(var15, true)
	var16 := e.seqModel
	if var4 != -1 && var5 != -1 {
		var16.ApplyTransforms(var5, var4, seqtype.Instances[e.PrimarySeqID].WalkMerge)
	} else if var4 != -1 {
		var16.ApplyTransform(var4)
	}
	var16.CalculateBoundsCylinder()
	var16.LabelFaces = nil
	var16.LabelVertices = nil
	return var16
}

func (e *PlayerEntity) GetHeadModel() *model.Model {
	if !e.Visible {
		return nil
	}
	var2 := make([]*model.Model, 12)
	var3 := 0
	for i := range 12 {
		var5 := e.Appearances[i]
		if var5 >= 256 && var5 < 512 {
			var2[var3] = idktype.Instances[var5-256].GetHeadModel()
			var3++
		}
		if var5 >= 512 {
			var6 := objtype.Get(var5 - 512).GetHeadModel(e.Gender)
			if var6 != nil {
				var2[var3] = var6
				var3++
			}
		}
	}
	var7 := model.NewModel2(var2, var3)
	for i := range 5 {
		if e.Colors[i] != 0 {
			var7.Recolor(clientextras.Field1307[i][0], clientextras.Field1307[i][e.Colors[i]])
			if i == 1 {
				var7.Recolor(clientextras.Field1438[0], clientextras.Field1438[e.Colors[i]])
			}
		}
	}
	return var7
}

func (e *PlayerEntity) IsVisible() bool {
	return e.Visible
}

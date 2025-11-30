package npctype

import (
	"strings"

	"goscape-client/pkg/jagex2/datastruct"
	"goscape-client/pkg/jagex2/graphics/model"
	"goscape-client/pkg/jagex2/io"
)

var (
	Count      int
	Offsets    []int
	Dat        *io.Packet
	Cache      []*NpcType
	CachePos   int
	ModelCache = datastruct.NewLruCache[*model.Model](30)
)

type NpcType struct {
	Index        int64
	Name         string
	Desc         []byte
	Size         byte
	Models       []int
	Heads        []int
	ReadyAnim    int
	WalkAnim     int
	WalkAnimB    int
	WalkAnimR    int
	WalkAnimL    int
	AnimHasAlpha bool
	RecolS       []int
	RecolD       []int
	Op           []string
	ResizeX      int
	ResizeY      int
	ResizeZ      int
	Minimap      bool
	VisLevel     int
	ResizeH      int
	ResizeV      int
}

func NewNpcType() *NpcType {
	return &NpcType{
		Index:        -1,
		Size:         1,
		ReadyAnim:    -1,
		WalkAnim:     -1,
		WalkAnimB:    -1,
		WalkAnimR:    -1,
		WalkAnimL:    -1,
		AnimHasAlpha: false,
		ResizeX:      -1,
		ResizeY:      -1,
		ResizeZ:      -1,
		Minimap:      true,
		VisLevel:     -1,
		ResizeH:      128,
		ResizeV:      128,
	}
}

func Unpack(arg0 io.Jagfile) {
	Dat = io.NewPacket(arg0.Read("npc.dat", nil))
	var1 := io.NewPacket(arg0.Read("npc.idx", nil))
	Count = var1.G2()
	Offsets = make([]int, Count)
	var2 := 2
	for i := range Count {
		Offsets[i] = var2
		var2 += var1.G2()
	}
	Cache = make([]*NpcType, 20)
	for i := range 20 {
		Cache[i] = NewNpcType()
	}
}

func Unload() {
	ModelCache = nil
	Offsets = nil
	Cache = nil
	Dat = nil
}

func Get(arg0 int) *NpcType {
	for i := range 20 {
		if Cache[i].Index == int64(arg0) {
			return Cache[i]
		}
	}
	CachePos = (CachePos + 1) % 20
	Cache[CachePos] = NewNpcType()
	var2 := Cache[CachePos]
	Dat.Pos = Offsets[arg0]
	var2.Index = int64(arg0)
	var2.Decode(Dat)
	return var2
}

func (t *NpcType) Decode(arg1 *io.Packet) {
	for {
		var3 := arg1.G1()
		switch var3 {
		case 0:
			return
		case 1:
			var4 := arg1.G1()
			t.Models = make([]int, var4)
			for i := range var4 {
				t.Models[i] = arg1.G2()
			}
		case 2:
			t.Name = arg1.GJStr()
		case 3:
			t.Desc = arg1.GStrByte()
		case 12:
			t.Size = arg1.G1B()
		case 13:
			t.ReadyAnim = arg1.G2()
		case 14:
			t.WalkAnim = arg1.G2()
		case 16:
			t.AnimHasAlpha = true
		case 17:
			t.WalkAnim = arg1.G2()
			t.WalkAnimB = arg1.G2()
			t.WalkAnimR = arg1.G2()
			t.WalkAnimL = arg1.G2()
		case 30, 31, 32, 33, 34, 35, 36, 37, 38, 39:
			if t.Op == nil {
				t.Op = make([]string, 5)
			}
			t.Op[var3-30] = arg1.GJStr()
			if strings.ToLower(t.Op[var3-30]) == "hidden" {
				t.Op[var3-30] = ""
			}
		case 40:
			var4 := arg1.G1()
			t.RecolS = make([]int, var4)
			t.RecolD = make([]int, var4)
			for i := range var4 {
				t.RecolS[i] = arg1.G2()
				t.RecolD[i] = arg1.G2()
			}
		case 60:
			var4 := arg1.G1()
			t.Heads = make([]int, var4)
			for i := range var4 {
				t.Heads[i] = arg1.G2()
			}
		case 90:
			t.ResizeX = arg1.G2()
		case 91:
			t.ResizeY = arg1.G2()
		case 92:
			t.ResizeZ = arg1.G2()
		case 93:
			t.Minimap = false
		case 95:
			t.VisLevel = arg1.G2()
		case 97:
			t.ResizeH = arg1.G2()
		case 98:
			t.ResizeV = arg1.G2()
		}
	}
}

func (t *NpcType) GetSequencedModel(arg0 int, arg1 int, arg2 []int) *model.Model {
	var5 := ModelCache.Get(t.Index).Value
	if var5 == nil {
		var6 := make([]*model.Model, len(t.Models))
		for i := range len(t.Models) {
			var6[i] = model.NewModel1(t.Models[i])
		}
		if len(var6) == 1 {
			var5 = var6[0]
		} else {
			var5 = model.NewModel2(var6, len(var6))
		}
		if t.RecolS != nil {
			for i := range len(t.RecolS) {
				var5.Recolor(t.RecolS[i], t.RecolD[i])
			}
		}
		var5.CreateLabelReferences()
		var5.CalculateNormals(64, 850, -30, -50, -30, true)
		//ModelCache.Put(t.Index, var5) // TODO
	}
	var4 := model.NewModel6(var5, !t.AnimHasAlpha)
	if arg0 != -1 && arg1 != -1 {
		var4.ApplyTransforms(arg1, arg0, arg2)
	} else if arg0 != -1 {
		var4.ApplyTransform(arg0)
	}
	if t.ResizeH != 128 || t.ResizeV != 128 {
		var4.Scale(t.ResizeH, t.ResizeV, t.ResizeH)
	}
	var4.CalculateBoundsCylinder()
	var4.LabelFaces = nil
	var4.LabelVertices = nil
	if t.Size == 1 {
		var4.Pickable = true
	}
	return var4
}

func (t *NpcType) GetHeadModel() *model.Model {
	if t.Heads == nil {
		return nil
	}
	var2 := make([]*model.Model, len(t.Heads))
	for i := range t.Heads {
		var2[i] = model.NewModel1(t.Heads[i])
	}
	var var4 *model.Model
	if len(var2) == 1 {
		var4 = var2[0]
	} else {
		var4 = model.NewModel2(var2, len(var2))
	}
	if t.RecolS != nil {
		for i := range t.RecolS {
			var4.Recolor(t.RecolS[i], t.RecolD[i])
		}
	}
	return var4
}

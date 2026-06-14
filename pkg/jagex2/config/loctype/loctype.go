package loctype

import (
	"strings"

	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
	"github.com/zsrv/goscape-client/pkg/jagex2/io/ondemand"
)

var (
	Reset             bool
	Count             int
	Offsets           []int
	Dat               *io.Packet
	Cache             []*LocType
	CachePos          int
	ModelCacheStatic  *datastruct.LruCache[*model.Model]
	ModelCacheDynamic *datastruct.LruCache[*model.Model]
)

func init() {
	ModelCacheStatic = datastruct.NewLruCache[*model.Model](500)
	// DEVIATION from Java's faithful LruCache(30) (LocType.java:88). 30 thrashes
	// for a region's unique transformed-loc working set, making the miss-path
	// builders (CalculateNormals/NewModel4/CreateLabelReferences) ~45% of
	// scene-build allocation churn. 256 holds the working set; render-identical
	// (the cache key encodes every transform parameter, so eviction only decides
	// whether we rebuild, never what we render). Worst-case retained ~6 MB vs an
	// ~82 MB live heap (profiled 2026-05-25).
	ModelCacheDynamic = datastruct.NewLruCache[*model.Model](256)
}

type LocType struct {
	Index         int
	Models        []int
	Shapes        []int
	Name          string
	Desc          []byte
	RecolS        []int
	RecolD        []int
	Width         int
	Length        int
	BlockWalk     bool
	BlockRange    bool
	Active        bool
	HillSkew      bool
	ShareLight    bool
	Occlude       bool
	Anim          int
	WallWidth     int
	Ambient       int8
	Contrast      int8
	MapFunction   int
	MapScene      int
	ResizeX       int
	ResizeY       int
	ResizeZ       int
	OffsetX       int
	OffsetY       int
	OffsetZ       int
	ForceApproach int
	AnimHasAlpha  bool
	Mirror        bool
	Shadow        bool
	ForceDecor    bool
	// Java: breakroutefinding (LocType.java:134 @176a85f) — new at 245.2
	BreakRouteFinding bool
	Op                []string
}

func NewLocType() *LocType {
	return &LocType{
		Index: -1,
	}
}

func Unpack(arg0 *io.Jagfile) {
	Dat = io.NewPacket(arg0.Read("loc.dat", nil))
	var1 := io.NewPacket(arg0.Read("loc.idx", nil))
	Count = var1.G2()
	Offsets = make([]int, Count)
	var2 := 2
	for i := range Count {
		Offsets[i] = var2
		var2 += var1.G2()
	}
	Cache = make([]*LocType, 10)
	for i := range 10 {
		Cache[i] = NewLocType()
	}
}

func Unload() {
	ModelCacheStatic = nil
	ModelCacheDynamic = nil
	Offsets = nil
	Cache = nil
	Dat = nil
}

func Get(arg0 int) *LocType {
	for i := range 10 {
		if Cache[i].Index == arg0 {
			return Cache[i]
		}
	}
	CachePos = (CachePos + 1) % 10
	var2 := Cache[CachePos]
	Dat.Pos = Offsets[arg0]
	var2.Index = arg0
	var2.Reset()
	var2.Decode(Dat)
	return var2
}

func (loc *LocType) Reset() {
	loc.Models = nil
	loc.Shapes = nil
	loc.Name = ""
	loc.Desc = nil
	loc.RecolS = nil
	loc.RecolD = nil
	loc.Width = 1
	loc.Length = 1
	loc.BlockWalk = true
	loc.BlockRange = true
	loc.Active = false
	loc.HillSkew = false
	loc.ShareLight = false
	loc.Occlude = false
	loc.Anim = -1
	loc.WallWidth = 16
	loc.Ambient = 0
	loc.Contrast = 0
	loc.Op = nil
	loc.AnimHasAlpha = false
	loc.MapFunction = -1
	loc.MapScene = -1
	loc.Mirror = false
	loc.Shadow = true
	loc.ResizeX = 128
	loc.ResizeY = 128
	loc.ResizeZ = 128
	loc.ForceApproach = 0
	loc.OffsetX = 0
	loc.OffsetY = 0
	loc.OffsetZ = 0
	loc.ForceDecor = false
	loc.BreakRouteFinding = false // Java: reset (LocType.java:220 @176a85f)
}

// Java: decode (LocType.java:224-345 @176a85f) — 245.2 moves the code==0
// end-of-stream handling out of the loop and appends the new
// breakroutefinding post-loop block.
func (loc *LocType) Decode(buf *io.Packet) {
	active := -1 // Java: active
loop:
	for {
		code := buf.G1() // Java: code
		switch code {
		case 0:
			break loop
		case 1:
			count := buf.G1() // Java: count
			loc.Shapes = make([]int, count)
			loc.Models = make([]int, count)
			for i := range count {
				loc.Models[i] = buf.G2()
				loc.Shapes[i] = buf.G1()
			}
		case 2:
			loc.Name = buf.GJStr()
		case 3:
			loc.Desc = buf.GStrByte()
		case 14:
			loc.Width = buf.G1()
		case 15:
			loc.Length = buf.G1()
		case 17:
			loc.BlockWalk = false
		case 18:
			loc.BlockRange = false
		case 19:
			active = buf.G1()
			if active == 1 {
				loc.Active = true
			}
		case 21:
			loc.HillSkew = true
		case 22:
			loc.ShareLight = true
		case 23:
			loc.Occlude = true
		case 24:
			loc.Anim = buf.G2()
			if loc.Anim == 0xFFFF {
				loc.Anim = -1
			}
		case 25:
			loc.AnimHasAlpha = true
		case 28:
			loc.WallWidth = buf.G1()
		case 29:
			loc.Ambient = buf.G1B()
		case 39:
			loc.Contrast = buf.G1B()
		case 30, 31, 32, 33, 34, 35, 36, 37, 38:
			if loc.Op == nil {
				loc.Op = make([]string, 5)
			}
			loc.Op[code-30] = buf.GJStr()
			// Java assigns op[i] = null here; Go uses "" as the absence marker.
			// All read sites compare via `!= ""`. The wire format never sends ""
			// as a legitimate option, so the two markers are equivalent in
			// practice. Same convention in NpcType and ObjType.
			if strings.ToLower(loc.Op[code-30]) == "hidden" {
				loc.Op[code-30] = ""
			}
		case 40:
			count := buf.G1() // Java: count
			loc.RecolS = make([]int, count)
			loc.RecolD = make([]int, count)
			for i := range count {
				loc.RecolS[i] = buf.G2()
				loc.RecolD[i] = buf.G2()
			}
		case 60:
			loc.MapFunction = buf.G2()
		case 62:
			loc.Mirror = true
		case 64:
			loc.Shadow = false
		case 65:
			loc.ResizeX = buf.G2()
		case 66:
			loc.ResizeY = buf.G2()
		case 67:
			loc.ResizeZ = buf.G2()
		case 68:
			loc.MapScene = buf.G2()
		case 69:
			loc.ForceApproach = buf.G1()
		case 70:
			loc.OffsetX = buf.G2B()
		case 71:
			loc.OffsetY = buf.G2B()
		case 72:
			loc.OffsetZ = buf.G2B()
		case 73:
			loc.ForceDecor = true
		case 74:
			loc.BreakRouteFinding = true
		}
	}
	if loc.Shapes == nil {
		loc.Shapes = make([]int, 0)
	}
	if active == -1 {
		loc.Active = false
		if len(loc.Shapes) > 0 && loc.Shapes[0] == 10 {
			loc.Active = true
		}
		if loc.Op != nil {
			loc.Active = true
		}
	}
	// Java: LocType.java:341-344 @176a85f — new at 245.2
	if loc.BreakRouteFinding {
		loc.BlockWalk = false
		loc.BlockRange = false
	}
}

func (loc *LocType) GetModel(arg0, arg1, arg2, arg3, arg4, arg5, arg6 int) *model.Model {
	var8 := -1
	for i := range len(loc.Shapes) {
		if loc.Shapes[i] == arg0 {
			var8 = i
			break
		}
	}
	if var8 == -1 {
		return nil
	}
	var10 := int64((loc.Index<<6)+(var8<<3)+arg1) + int64((arg6+1)<<32)
	if Reset {
		var10 = 0
	}
	var12 := ModelCacheDynamic.Get(var10)
	if var12 == nil {
		if var8 >= len(loc.Models) {
			return nil
		}
		var13 := loc.Models[var8]
		if var13 == -1 {
			return nil
		}
		var14 := loc.Mirror != (arg1 > 3)
		if var14 {
			var13 += 65536
		}
		var15 := ModelCacheStatic.Get(int64(var13))
		if var15 == nil {
			var15 = model.TryGet(var13 & 0xFFFF)
			if var15 == nil {
				return nil
			}
			if var14 {
				var15.RotateY180()
			}
			ModelCacheStatic.Put(int64(var13), var15)
		}
		var var16 bool
		if loc.ResizeX == 128 && loc.ResizeY == 128 && loc.ResizeZ == 128 {
			var16 = false
		} else {
			var16 = true
		}
		var var17 bool
		if loc.OffsetX == 0 && loc.OffsetY == 0 && loc.OffsetZ == 0 {
			var17 = false
		} else {
			var17 = true
		}
		var18 := model.NewModel4(var15, loc.RecolS == nil, !loc.AnimHasAlpha, arg1 == 0 && arg6 == -1 && !var16 && !var17)
		if arg6 != -1 {
			var18.CreateLabelReferences()
			var18.ApplyTransform(arg6)
			var18.LabelFaces = nil
			var18.LabelVertices = nil
		}
		for ; arg1 > 0; arg1-- {
			var18.RotateY90()
		}
		if loc.RecolS != nil {
			for i := range len(loc.RecolS) {
				var18.Recolor(loc.RecolS[i], loc.RecolD[i])
			}
		}
		if var16 {
			var18.Scale(loc.ResizeZ, loc.ResizeY, loc.ResizeX)
		}
		if var17 {
			var18.Translate(loc.OffsetY, loc.OffsetX, loc.OffsetZ)
		}
		var18.CalculateNormals(int(loc.Ambient)+64, int(loc.Contrast)*5+768, -50, -10, -50, !loc.ShareLight)
		if loc.BlockWalk {
			var18.ObjRaise = var18.MinY
		}
		ModelCacheDynamic.Put(var10, var18)
		if loc.HillSkew || loc.ShareLight {
			var18 = model.NewModel5(var18, loc.HillSkew, loc.ShareLight)
		}
		if loc.HillSkew {
			var19 := (arg2 + arg3 + arg4 + arg5) / 4
			for i := range var18.VertexCount {
				var21 := var18.VertexX[i]
				var22 := var18.VertexZ[i]
				var23 := arg2 + (arg3-arg2)*(var21+64)/128
				var24 := arg5 + (arg4-arg5)*(var21+64)/128
				var25 := var23 + (var24-var23)*(var22+64)/128
				var18.VertexY[i] += var25 - var19
			}
			var18.CalculateBoundsY()
		}
		return var18
	} else if Reset {
		return var12
	} else {
		if loc.HillSkew || loc.ShareLight {
			var12 = model.NewModel5(var12, loc.HillSkew, loc.ShareLight)
		}
		if loc.HillSkew {
			var13 := (arg2 + arg3 + arg4 + arg5) / 4
			for i := range var12.VertexCount {
				var27 := var12.VertexX[i]
				var28 := var12.VertexZ[i]
				var29 := arg2 + (arg3-arg2)*(var27+64)/128
				var30 := arg5 + (arg4-arg5)*(var27+64)/128
				var19 := var29 + (var30-var29)*(var28+64)/128
				var12.VertexY[i] += var19 - var13
			}
			var12.CalculateBoundsY()
		}
		return var12
	}
}

// CheckModel reports whether the model for the given shape is loaded.
// Java: LocType.checkModel(int shape) (ec.a(II)Z), lines 336-354.
// Sole caller is World.CheckLocations (WS2 scene-readiness path).
func (t *LocType) CheckModel(shape int) bool {
	index := -1
	for i := range len(t.Shapes) {
		if t.Shapes[i] == shape {
			index = i
			break
		}
	}
	if index == -1 {
		return true
	}
	if t.Models == nil {
		return true
	}
	m := t.Models[index]
	return m == -1 || model.Request(m&0xFFFF)
}

// CheckModelAll reports whether all models for this loc are loaded.
// Java: LocType.checkModelAll() (ec.b(I)Z), lines 357-371.
//
// NOTE: Java uses ready &= Model.request(...) — a bitwise AND that is
// NON-short-circuit, ensuring request() is called for every model even
// after ready becomes false (request has the side effect of queuing a
// network fetch on a miss). Ported as: always call model.Request, then
// clear ready if it returns false. Do NOT use ready = ready && model.Request
// (short-circuits, silently drops side effects on later models).
func (t *LocType) CheckModelAll() bool {
	ready := true
	if t.Models == nil {
		return true
	}
	for _, m := range t.Models {
		if m != -1 {
			if !model.Request(m & 0xFFFF) {
				ready = false
			}
		}
	}
	return ready
}

// Prefetch queues all models for this loc as low-priority on-demand prefetches.
// Java: LocType.prefetch(OnDemand od) (ec.a(ILvb;)V), lines 374-384.
func (t *LocType) Prefetch(od *ondemand.OnDemand) {
	if t.Models == nil {
		return
	}
	for _, m := range t.Models {
		if m != -1 {
			od.Prefetch(0, m&0xFFFF)
		}
	}
}

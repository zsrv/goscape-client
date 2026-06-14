package loctype

import (
	"strings"

	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/animframe"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
	"github.com/zsrv/goscape-client/pkg/jagex2/io/ondemand"
)

var (
	Count    int
	Offsets  []int
	Dat      *io.Packet
	Cache    []*LocType
	CachePos int
	// Temp is the multi-model merge scratch used by BuildModel's models-only
	// path. Java: temp (ec.h, LocType.java:30 @2e62978) — new at 254.
	Temp [4]*model.Model
	// Java: mc1/mc2 (LocType.java:87,90 @2e62978) — descriptive 245.2-era
	// names kept per the hybrid naming policy (mc1/mc2 is a regression).
	ModelCacheStatic  *datastruct.LruCache[*model.Model]
	ModelCacheDynamic *datastruct.LruCache[*model.Model]
)

func init() {
	ModelCacheStatic = datastruct.NewLruCache[*model.Model](500)
	// DEVIATION from Java's faithful LruCache(30) (mc2, LocType.java:90
	// @2e62978). 30 thrashes
	// for a region's unique transformed-loc working set, making the miss-path
	// builders (CalculateNormals/NewModel4/PrepareAnim) ~45% of
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
	// Java: raiseobject (ec.P, LocType.java:123 @2e62978) — new at 254;
	// replaces the blockwalk gate on Model.objRaise in BuildModel.
	RaiseObject int
	Mirror      bool
	Shadow      bool
	ForceDecor  bool
	// Java: breakroutefinding (LocType.java:134 @176a85f) — new at 245.2
	BreakRouteFinding bool
	Op                []string
}

func NewLocType() *LocType {
	return &LocType{
		Index: -1,
	}
}

func Unpack(arg0 *io.JagFile) {
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
	loc.BreakRouteFinding = false
	loc.RaiseObject = -1 // Java: reset (LocType.java:216 @2e62978) — new at 254
}

// Java: decode (LocType.java:220-348 @2e62978). 254 moves the end-of-stream
// finalization INTO the code==0 branch, gates opcode 1's allocation on
// count > 0 (the deob do-while; 245.2 allocated empty arrays), adds the
// models-only opcode 5 and raiseobject opcode 75, and drops opcode 25
// (animHasAlpha — replaced by AnimFrame.shareAlpha, WS3).
func (loc *LocType) Decode(buf *io.Packet) {
	active := -1 // Java: var3
	for {
		code := buf.G1() // Java: var4
		switch code {
		case 0:
			// Java: LocType.java:225-245 @2e62978 — finalization, new active
			// rule (shapes may stay nil), raiseobject default.
			if active == -1 {
				loc.Active = false
				if loc.Models != nil && (loc.Shapes == nil || loc.Shapes[0] == 10) {
					loc.Active = true
				}
				if loc.Op != nil {
					loc.Active = true
				}
			}
			if loc.BreakRouteFinding {
				loc.BlockWalk = false
				loc.BlockRange = false
			}
			if loc.RaiseObject == -1 {
				loc.RaiseObject = 0
				if loc.BlockWalk {
					loc.RaiseObject = 1
				}
			}
			return
		case 1:
			count := buf.G1() // Java: var5
			// Java 254 skips the allocation entirely when count <= 0
			// (LocType.java:340 `while (var5 <= 0)`); 245.2 allocated
			// empty arrays here.
			if count > 0 {
				loc.Shapes = make([]int, count)
				loc.Models = make([]int, count)
				for i := range count {
					loc.Models[i] = buf.G2()
					loc.Shapes[i] = buf.G1()
				}
			}
		case 2:
			loc.Name = buf.GStr()
		case 3:
			loc.Desc = buf.GStrByte()
		case 5:
			// Java: LocType.java:255-263 @2e62978 — new at 254: models-only
			// list (no shapes); BuildModel takes the shapes==nil path.
			count := buf.G1() // Java: var7
			if count > 0 {
				loc.Shapes = nil
				loc.Models = make([]int, count)
				for i := range count {
					loc.Models[i] = buf.G2()
				}
			}
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
			loc.Op[code-30] = buf.GStr()
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
		case 75:
			// Java: LocType.java:336-337 @2e62978 — new at 254.
			loc.RaiseObject = buf.G1()
		}
	}
}

// GetModel returns the fully transformed scene model for the given shape /
// angle / heightmap corners / frame, or nil while models are still loading.
// 254 splits the build into BuildModel; the hillskew/sharelight wrap is
// applied here on every call (BuildModel's cache holds the unwrapped model).
// Java: getModel (LocType.java:395-417 @2e62978).
func (loc *LocType) GetModel(arg0, arg1, arg2, arg3, arg4, arg5, arg6 int) *model.Model {
	var8 := loc.BuildModel(arg0, arg6, arg1) // Java: var8
	if var8 == nil {
		return nil
	}
	if loc.HillSkew || loc.ShareLight {
		var8 = model.NewModel5(var8, loc.HillSkew, loc.ShareLight)
	}
	if loc.HillSkew {
		var9 := (arg2 + arg3 + arg4 + arg5) / 4 // Java: var9
		for i := range var8.VertexCount {
			var11 := var8.VertexX[i]
			var12 := var8.VertexZ[i]
			var13 := arg2 + (arg3-arg2)*(var11+64)/128
			var14 := arg5 + (arg4-arg5)*(var11+64)/128
			var15 := var13 + (var14-var13)*(var12+64)/128
			var8.VertexY[i] += var15 - var9
		}
		var8.CalcHeight()
	}
	return var8
}

// BuildModel builds (or returns the cached) transformed model for the given
// shape (arg0), frame (arg2), and angle (arg3). New at 254: the Shapes==nil
// branch serves opcode-5 models-only locs (shape must be 10), merging
// multiple models through the Temp scratch; the classic shapes branch
// follows. The 245.2 -1/index-bounds guards are dropped (faithful to 254's
// unguarded array reads). Java: buildModel (LocType.java:419-533 @2e62978);
// param gap arg0/arg2/arg3 mirrors the deob signature.
func (loc *LocType) BuildModel(arg0, arg2, arg3 int) *model.Model {
	var var5 *model.Model // Java: var5
	var var7 int64        // Java: var7 (dynamic-cache key)
	if loc.Shapes == nil {
		if arg0 != 10 {
			return nil
		}
		var7 = int64((loc.Index<<6)+arg3) + int64((arg2+1)<<32)
		var9 := ModelCacheDynamic.Get(var7)
		if var9 != nil {
			return var9
		}
		if loc.Models == nil {
			return nil
		}
		var10 := loc.Mirror != (arg3 > 3) // Java: var10 (mirror ^ angle > 3)
		var11 := len(loc.Models)
		for var12 := range var11 {
			var13 := loc.Models[var12]
			if var10 {
				var13 += 65536
			}
			var5 = ModelCacheStatic.Get(int64(var13))
			if var5 == nil {
				var5 = model.Load(var13 & 0xFFFF)
				if var5 == nil {
					return nil
				}
				if var10 {
					var5.Rotate180()
				}
				ModelCacheStatic.Put(int64(var13), var5)
			}
			if var11 > 1 {
				Temp[var12] = var5
			}
		}
		if var11 > 1 {
			var5 = model.NewModel2(Temp[:], var11)
		}
	} else {
		var14 := -1
		for var15 := range len(loc.Shapes) {
			if loc.Shapes[var15] == arg0 {
				var14 = var15
				break
			}
		}
		if var14 == -1 {
			return nil
		}
		var7 = int64((loc.Index<<6)+(var14<<3)+arg3) + int64((arg2+1)<<32)
		var16 := ModelCacheDynamic.Get(var7)
		if var16 != nil {
			return var16
		}
		var17 := loc.Models[var14]
		var18 := loc.Mirror != (arg3 > 3) // Java: var18 (mirror ^ angle > 3)
		if var18 {
			var17 += 65536
		}
		var5 = ModelCacheStatic.Get(int64(var17))
		if var5 == nil {
			var5 = model.Load(var17 & 0xFFFF)
			if var5 == nil {
				return nil
			}
			if var18 {
				var5.Rotate180()
			}
			ModelCacheStatic.Put(int64(var17), var5)
		}
	}
	var var19 bool // Java: var19 (resize needed)
	if loc.ResizeX == 128 && loc.ResizeY == 128 && loc.ResizeZ == 128 {
		var19 = false
	} else {
		var19 = true
	}
	var var20 bool // Java: var20 (offset needed)
	if loc.OffsetX == 0 && loc.OffsetY == 0 && loc.OffsetZ == 0 {
		var20 = false
	} else {
		var20 = true
	}
	// Java: new Model(AnimFrame.shareAlpha(arg2), <isolated>, recol_s==null,
	// var5) (LocType.java:507 @2e62978) — the ctor arg reorder vs 245.2 is
	// signature-only churn; the real delta is the alpha-share flag, which
	// was !animHasAlpha at 245.2 (WS3).
	var21 := model.NewModel4(var5, loc.RecolS == nil, animframe.ShareAlpha(arg2), arg3 == 0 && arg2 == -1 && !var19 && !var20)
	if arg2 != -1 {
		var21.PrepareAnim()
		var21.Animate(arg2)
		var21.LabelFaces = nil
		var21.LabelVertices = nil
	}
	for ; arg3 > 0; arg3-- {
		var21.Rotate90()
	}
	if loc.RecolS != nil {
		for i := range len(loc.RecolS) {
			var21.Recolor(loc.RecolS[i], loc.RecolD[i])
		}
	}
	if var19 {
		var21.Scale(loc.ResizeZ, loc.ResizeY, loc.ResizeX)
	}
	if var20 {
		var21.Translate(loc.OffsetY, loc.OffsetX, loc.OffsetZ)
	}
	var21.CalculateNormals(int(loc.Ambient)+64, int(loc.Contrast)*5+768, -50, -10, -50, !loc.ShareLight)
	// Java: raiseobject == 1 gate (LocType.java:526 @2e62978) — was
	// blockwalk at 245.2. var21.MinY is Java's model.minY (max model
	// height above origin).
	if loc.RaiseObject == 1 {
		var21.ObjRaise = var21.MinY
	}
	ModelCacheDynamic.Put(var7, var21)
	return var21
}

// CheckModel reports whether the model for the given shape is loaded,
// requesting it otherwise. 254 adds the Shapes==nil (opcode-5) branch with
// its shape==10 all-models AND, and drops the 245.2 -1 guard (a -1 model id
// masks to 65535 and is requested like any other id).
// Java: checkModel (LocType.java:351-370 @2e62978). Sole caller is
// World.CheckLocations (scene-readiness path).
func (t *LocType) CheckModel(shape int) bool {
	if t.Shapes != nil {
		for var5 := range len(t.Shapes) {
			if t.Shapes[var5] == shape {
				return model.RequestDownload(t.Models[var5] & 0xFFFF)
			}
		}
		return true
	} else if t.Models == nil {
		return true
	} else if shape == 10 {
		// Java: var3 &= Model.requestDownload(...) — non-short-circuit AND;
		// every model must be requested even after the first miss.
		var3 := true // Java: var3
		for var4 := range len(t.Models) {
			if !model.RequestDownload(t.Models[var4] & 0xFFFF) {
				var3 = false
			}
		}
		return var3
	} else {
		return true
	}
}

// CheckModelAll reports whether all models for this loc are loaded.
// Java: checkModelAll (LocType.java:373-383 @2e62978). 254 drops the
// 245.2 per-model -1 guard.
//
// NOTE: Java uses var2 &= Model.requestDownload(...) — a bitwise AND that is
// NON-short-circuit, ensuring requestDownload() is called for every model
// even after var2 becomes false (it has the side effect of queuing a network
// fetch on a miss). Ported as: always call model.RequestDownload, then clear
// ready if it returns false. Do NOT use ready = ready && model.RequestDownload
// (short-circuits, silently drops side effects on later models).
func (t *LocType) CheckModelAll() bool {
	if t.Models == nil {
		return true
	}
	ready := true // Java: var2
	for _, m := range t.Models {
		if !model.RequestDownload(m & 0xFFFF) {
			ready = false
		}
	}
	return ready
}

// PrefetchModelAll queues all models for this loc as low-priority on-demand
// prefetches. Java: prefetchModelAll (LocType.java:386-392 @2e62978; was
// prefetch at 245.2) — 254 drops the -1 guard; the arg-order churn in the
// Java call is compensated by OnDemand.prefetch's own param reorder, and the
// Go call order is already correct (scope WS2; do not touch).
func (t *LocType) PrefetchModelAll(od *ondemand.OnDemand) {
	if t.Models == nil {
		return
	}
	for _, m := range t.Models {
		od.Prefetch(0, m&0xFFFF)
	}
}

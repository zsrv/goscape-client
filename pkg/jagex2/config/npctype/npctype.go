package npctype

import (
	"strings"

	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/animframe"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
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
	Index  int64
	Name   string
	Desc   []byte
	Size   int8
	Models []int
	Heads  []int
	// Java: 245.2 renames readyanim→runanim and swaps the walkanim_l/walkanim_r
	// declaration + opcode-17 read order (NpcType.java:46-58 @176a85f). COUPLED
	// with the direct (non-swapped) assignments in Client getNpcPos* — net
	// behaviour identical to 244.
	RunAnim   int
	WalkAnim  int
	WalkAnimB int
	WalkAnimL int
	WalkAnimR int
	RecolS    []int
	RecolD    []int
	Op        []string
	// Java: field998/field999/field1000 (gc.v/w/x @2e62978; were
	// 245.2's field1010/1011/1012) — assigned by opcodes 90/91/92 but
	// never read anywhere in Java or Go (re-verified at 2e62978).
	// Pure deobfuscator residue; fields omitted per the deob-artifact
	// exclusion policy. The wire reads in Decode() are preserved as
	// discards to keep packet-position alignment.
	Minimap  bool
	VisLevel int
	ResizeH  int
	ResizeV  int
	// Java: NpcType alwaysontop/headicon/ambient/contrast (rev-244 opcodes
	// 99-102). Ambient/Contrast are consumed by CalculateNormals in GetSequencedModel.
	AlwaysOnTop bool
	HeadIcon    int
	// Java: turnspeed (gc.G, NpcType.java:98 @2e62978) — new at 254;
	// copied onto the entity by getNpcPosNewVis/Extended (WS5).
	TurnSpeed int
	Ambient   int
	Contrast  int
}

func NewNpcType() *NpcType {
	return &NpcType{
		Index:     -1,
		Size:      1,
		RunAnim:   -1,
		WalkAnim:  -1,
		WalkAnimB: -1,
		WalkAnimL: -1,
		WalkAnimR: -1,
		Minimap:   true,
		VisLevel:  -1,
		ResizeH:   128,
		ResizeV:   128,
		// Java: NpcType field initializers (rev-244). AlwaysOnTop/Ambient/
		// Contrast default to the zero value; HeadIcon defaults to -1.
		HeadIcon: -1,
		// Java: turnspeed = 32 (NpcType.java:98 @2e62978) — new at 254.
		TurnSpeed: 32,
	}
}

func Unpack(arg0 *io.JagFile) {
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
			t.RunAnim = arg1.G2()
		case 14:
			t.WalkAnim = arg1.G2()
		// Java 254 drops opcode 16 (animHasAlpha) — WS3 shareAlpha rework.
		case 17:
			// Java: 245.2 reads walkanim_l before walkanim_r (NpcType.java:135-138
			// @176a85f); 244 read _r then _l. Cache bytes are unchanged — the swap
			// is compensated by the direct assigns in Client getNpcPos*.
			t.WalkAnim = arg1.G2()
			t.WalkAnimB = arg1.G2()
			t.WalkAnimL = arg1.G2()
			t.WalkAnimR = arg1.G2()
		case 30, 31, 32, 33, 34, 35, 36, 37, 38, 39:
			if t.Op == nil {
				t.Op = make([]string, 5)
			}
			t.Op[var3-30] = arg1.GJStr()
			// Java assigns op[i] = null here; Go uses "" — see LocType.Decode
			// for the convention's full rationale.
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
		// Java: NpcType.java:191-196 — opcodes 90/91/92 write resizex/y/z.
		// Fields are deob artifacts (never read); reads kept as discards
		// so packet-position alignment matches Java byte-for-byte.
		case 90, 91, 92:
			arg1.G2()
		case 93:
			t.Minimap = false
		case 95:
			t.VisLevel = arg1.G2()
		case 97:
			t.ResizeH = arg1.G2()
		case 98:
			t.ResizeV = arg1.G2()
		// Java: NpcType.decode opcodes 99-102 (rev-244).
		case 99:
			t.AlwaysOnTop = true
		case 100:
			t.Ambient = int(arg1.G1B())
		case 101:
			t.Contrast = int(arg1.G1B()) * 5
		case 102:
			t.HeadIcon = arg1.G2()
		case 103:
			// Java: NpcType.java:236-237 @2e62978 — new at 254.
			t.TurnSpeed = arg1.G2()
		}
	}
}

// GetTempModel builds the animated NPC model into target (the caller-owned
// reusable Model — Go deviation replacing Java's Model.empty static; see
// ResetFromModel6). arg0 is the primary frame, arg1 the secondary frame,
// arg2 the walkmerge label set.
// Java: getTempModel (NpcType.java:240-285 @2e62978; was getModel at 245.2).
func (t *NpcType) GetTempModel(target *model.Model, arg0 int, arg1 int, arg2 []int) *model.Model {
	var5 := ModelCache.Get(t.Index)
	if var5 == nil {
		// Java: NpcType.getModel precheck — loop calls Model.request on every
		// model (non-short-circuit); ready=true means "something missing".
		// Java: `boolean ready = false; ... if (!Model.request(...)) { ready = true; }`
		ready := false // Java: ready
		for i := 0; i < len(t.Models); i++ {
			if !model.RequestDownload(t.Models[i]) {
				ready = true
			}
		}
		if ready {
			return nil
		}
		var6 := make([]*model.Model, len(t.Models))
		for i := range len(t.Models) {
			var6[i] = model.Load(t.Models[i])
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
		var5.PrepareAnim()
		var5.CalculateNormals(t.Ambient+64, t.Contrast+850, -30, -50, -30, true)
		ModelCache.Put(t.Index, var5)
	}
	// Java: var11.set(AnimFrame.shareAlpha(arg1) & AnimFrame.shareAlpha(arg3),
	// var5) (NpcType.java:267 @2e62978) — was !animHasAlpha at 245.2 (WS3).
	// shareAlpha has no side effects, so Go && is equivalent to Java's &.
	target.ResetFromModel6(var5, animframe.ShareAlpha(arg0) && animframe.ShareAlpha(arg1))
	var4 := target
	if arg0 != -1 && arg1 != -1 {
		var4.MaskAnimate(arg1, arg0, arg2)
	} else if arg0 != -1 {
		var4.Animate(arg0)
	}
	if t.ResizeH != 128 || t.ResizeV != 128 {
		// Java: NpcType.getModel scale(resizev, resizeh, resizeh); Go Scale(arg0=z, arg2=y, arg3=x).
		// So z=resizeh=ResizeH, y=resizev=ResizeV, x=resizeh=ResizeH → Scale(ResizeH, ResizeV, ResizeH).
		var4.Scale(t.ResizeH, t.ResizeV, t.ResizeH)
	}
	var4.CalcBoundingCylinder()
	var4.LabelFaces = nil
	var4.LabelVertices = nil
	if t.Size == 1 {
		var4.UseAABBMouseCheck = true
	}
	return var4
}

// GetHead builds the chathead model.
// Java: getHead (NpcType.java:288-307 @2e62978; was getHeadModel at 245.2).
func (t *NpcType) GetHead() *model.Model {
	if t.Heads == nil {
		return nil
	}
	// Java: NpcType.getHeadModel precheck — loop calls Model.request on every
	// head model (non-short-circuit); exists=true means "something missing".
	// Java: `boolean exists = false; ... if (!Model.request(...)) { exists = true; }`
	exists := false // Java: exists
	for i := 0; i < len(t.Heads); i++ {
		if !model.RequestDownload(t.Heads[i]) {
			exists = true
		}
	}
	if exists {
		return nil
	}
	var2 := make([]*model.Model, len(t.Heads))
	for i := range t.Heads {
		var2[i] = model.Load(t.Heads[i])
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

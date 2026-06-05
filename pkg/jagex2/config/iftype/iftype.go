package iftype

import (
	"strconv"
	"strings"

	"github.com/zsrv/goscape-client/pkg/jagex2/config/npctype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/objtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/animframe"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/entity/playerentity"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct"
	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct/jstring"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix32"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pixfont"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

var (
	List       []*IfType
	ImageCache *datastruct.LruCache[*pix32.Pix32]
	// Java: IfType.modelCache = new LruCache(30) (IfType.java:119) — a static
	// field initializer created once at class load and never nulled. unpack() only
	// creates/nulls imageCache (IfType.java:204,443); modelCache must survive it.
	ModelCache = datastruct.NewLruCache[*model.Model](30)
)

type IfType struct {
	InvSlotObjId    []int
	InvSlotObjCount []int
	SeqFrame        int
	SeqCycle        int
	Id              int
	Layer           int
	Type            int
	ButtonType      int
	ClientCode      int
	Width           int
	Height          int
	// Java: IfType.trans (IfType.java:53 @176a85f; named alpha at 244) —
	// header field read between height and overlayer. int8 to match Java's
	// signed-byte sign extension.
	Trans            int8
	X                int
	Y                int
	Scripts          [][]int
	ScriptComparator []int
	ScriptOperand    []int
	OverLayer        int
	Scroll           int
	ScrollPosition   int
	Hide             bool
	ChildID          []int
	ChildX           []int
	ChildY           []int
	Zoom             int
	Xan              int
	Yan              int
	ActionVerb       string
	Action           string
	ActionTarget     int
	Option           string
	Colour           int
	ActiveColour     int
	OverColour       int
	// Java: activeOverColour (IfType.java:146 @176a85f) — new at 245.2
	ActiveOverColour int
	Anim             int
	ActiveAnim       int
	MarginX          int
	MarginY          int
	ModelType        int
	Model            int
	ActiveModelType  int
	ActiveModel      int
	Graphic          *pix32.Pix32
	ActiveGraphic    *pix32.Pix32
	Font             *pixfont.PixFont
	Text             string
	ActiveText       string
	// Java: IfType.java:130,160 declares unusedShort1/unusedBoolean1
	// (assigned for Type==1 components but never read). Pure deob
	// residue; fields omitted per the deob-artifact exclusion policy.
	// Decode preserves the wire reads as discards.
	Draggable    bool
	Interactable bool
	Usable       bool
	// Java: swappable (IfType.java:167 @176a85f) — new at 245.2
	Swappable      bool
	Fill           bool
	Center         bool
	Shadowed       bool
	InvSlotOffsetX []int
	InvSlotOffsetY []int
	InvSlotSprite  []*pix32.Pix32
	IOps           []string
}

func NewIfType() *IfType {
	return new(IfType)
}

// SwapObj swaps the inventory slots src and dst (id + count).
// Java: IfType.swapObj (IfType.java:447-455, new in 244).
func (c *IfType) SwapObj(src, dst int) {
	c.InvSlotObjId[src], c.InvSlotObjId[dst] = c.InvSlotObjId[dst], c.InvSlotObjId[src]
	c.InvSlotObjCount[src], c.InvSlotObjCount[dst] = c.InvSlotObjCount[dst], c.InvSlotObjCount[src]
}

func Unpack(arg0 *io.JagFile, arg1 []*pixfont.PixFont, arg3 *io.JagFile) {
	ImageCache = datastruct.NewLruCache[*pix32.Pix32](50000)
	var4 := io.NewPacket(arg3.Read("data", nil))
	var5 := -1
	var6 := var4.G2()
	List = make([]*IfType, var6)
	for {
		var com *IfType
		for ok := true; ok; ok = com.ButtonType != 1 && com.ButtonType != 4 && com.ButtonType != 5 && com.ButtonType != 6 {
			if var4.Pos >= len(var4.Data) {
				// Java: IfType.java:443 nulls only imageCache; modelCache stays alive.
				ImageCache = nil
				return
			}
			var7 := var4.G2()
			if var7 == 0xFFFF {
				var5 = var4.G2()
				var7 = var4.G2()
			}
			List[var7] = NewIfType()
			com = List[var7]
			com.Id = var7
			com.Layer = var5
			com.Type = var4.G1()
			com.ButtonType = var4.G1()
			com.ClientCode = var4.G2()
			com.Width = var4.G2()
			com.Height = var4.G2()
			// Java: com.trans = (byte) var4.g1() (IfType.java:235 @176a85f;
			// named alpha at 244) — header read between height and overlayer;
			// shifts every following field.
			com.Trans = int8(var4.G1())
			com.OverLayer = var4.G1()
			if com.OverLayer == 0 {
				com.OverLayer = -1
			} else {
				com.OverLayer = ((com.OverLayer - 1) << 8) + var4.G1()
			}
			var9 := var4.G1()
			var10 := 0
			if var9 > 0 {
				com.ScriptComparator = make([]int, var9)
				com.ScriptOperand = make([]int, var9)
				for i := range var9 {
					com.ScriptComparator[i] = var4.G1()
					com.ScriptOperand[i] = var4.G2()
				}
			}
			var10 = var4.G1()
			var11 := 0
			var12 := 0
			if var10 > 0 {
				com.Scripts = make([][]int, var10)
				for i := range var10 {
					var12 = var4.G2()
					com.Scripts[i] = make([]int, var12)
					for j := range var12 {
						com.Scripts[i][j] = var4.G2()
					}
				}
			}
			if com.Type == 0 {
				com.Scroll = var4.G2()
				com.Hide = var4.G1() == 1
				// Java: int childCount = data.g2() (IfType.java:265) —
				// rev-244 widens the Type==0 child count to g2 from 225's
				// g1 (225-clean IfType.java:253). Reading one byte here
				// desyncs the whole sequential stream.
				var11 = var4.G2()
				com.ChildID = make([]int, var11)
				com.ChildX = make([]int, var11)
				com.ChildY = make([]int, var11)
				for i := range var11 {
					com.ChildID[i] = var4.G2()
					com.ChildX[i] = var4.G2B()
					com.ChildY[i] = var4.G2B()
				}
			}
			if com.Type == 1 {
				// Java: IfType.java:264-265 — Type==1 reads g2 + g1
				// into unusedShort1 / unusedBoolean1. Reads kept as
				// discards so packet-position alignment matches Java.
				var4.G2()
				var4.G1()
			}
			if com.Type == 2 {
				com.InvSlotObjId = make([]int, com.Width*com.Height)
				com.InvSlotObjCount = make([]int, com.Width*com.Height)
				com.Draggable = var4.G1() == 1
				com.Interactable = var4.G1() == 1
				com.Usable = var4.G1() == 1
				// Java: IfType.java:285 @176a85f — new at 245.2; shifts all
				// later type-2 reads by 1 byte.
				com.Swappable = var4.G1() == 1
				com.MarginX = var4.G1()
				com.MarginY = var4.G1()
				com.InvSlotOffsetX = make([]int, 20)
				com.InvSlotOffsetY = make([]int, 20)
				com.InvSlotSprite = make([]*pix32.Pix32, 20)
				for i := range 20 {
					var12 = var4.G1()
					if var12 == 1 {
						com.InvSlotOffsetX[i] = var4.G2B()
						com.InvSlotOffsetY[i] = var4.G2B()
						var17 := var4.GStr()
						if arg0 != nil && len(var17) > 0 {
							var14 := strings.LastIndex(var17, ",")
							v, err := strconv.Atoi(var17[var14+1:])
							if err != nil {
								panic(err)
							}
							com.InvSlotSprite[i] = GetImage(arg0, v, var17[0:var14])
						}
					}
				}
				com.IOps = make([]string, 5)
				// Java assigns iops[i] = null on length()==0; Go uses "" — see
				// LocType.Decode for the convention's full rationale. The
				// `= ""` re-assignment is a no-op (already ""), kept to mirror
				// Java's nulling pass for readability.
				for i := range 5 {
					com.IOps[i] = var4.GStr()
					if len(com.IOps[i]) == 0 {
						com.IOps[i] = ""
					}
				}
			}
			if com.Type == 3 {
				com.Fill = var4.G1() == 1
			}
			if com.Type == 4 || com.Type == 1 {
				com.Center = var4.G1() == 1
				var11 = var4.G1()
				if arg1 != nil {
					com.Font = arg1[var11]
				}
				com.Shadowed = var4.G1() == 1
			}
			if com.Type == 4 {
				com.Text = var4.GStr()
				com.ActiveText = var4.GStr()
			}
			if com.Type == 1 || com.Type == 3 || com.Type == 4 {
				com.Colour = var4.G4()
			}
			if com.Type == 3 || com.Type == 4 {
				com.ActiveColour = var4.G4()
				com.OverColour = var4.G4()
				// Java: IfType.java:332 @176a85f — new at 245.2; shifts all
				// later type-3/4 reads by 4 bytes.
				com.ActiveOverColour = var4.G4()
			}
			if com.Type == 5 {
				var16 := var4.GStr()
				if arg0 != nil && len(var16) > 0 {
					var12 = strings.LastIndex(var16, ",")
					v, err := strconv.Atoi(var16[var12+1:])
					if err != nil {
						panic(err)
					}
					com.Graphic = GetImage(arg0, v, var16[0:var12])
				}
				var16 = var4.GStr()
				if arg0 != nil && len(var16) > 0 {
					var12 = strings.LastIndex(var16, ",")
					v, err := strconv.Atoi(var16[var12+1:])
					if err != nil {
						panic(err)
					}
					com.ActiveGraphic = GetImage(arg0, v, var16[0:var12])
				}
			}
			if com.Type == 6 {
				var7 = var4.G1()
				if var7 != 0 {
					com.ModelType = 1
					com.Model = ((var7 - 1) << 8) + var4.G1()
				}
				var7 = var4.G1()
				if var7 != 0 {
					com.ActiveModelType = 1
					com.ActiveModel = ((var7 - 1) << 8) + var4.G1()
				}
				var7 = var4.G1()
				if var7 == 0 {
					com.Anim = -1
				} else {
					com.Anim = ((var7 - 1) << 8) + var4.G1()
				}
				var7 = var4.G1()
				if var7 == 0 {
					com.ActiveAnim = -1
				} else {
					com.ActiveAnim = ((var7 - 1) << 8) + var4.G1()
				}
				com.Zoom = var4.G2()
				com.Xan = var4.G2()
				com.Yan = var4.G2()
			}
			if com.Type == 7 {
				com.InvSlotObjId = make([]int, com.Width*com.Height)
				com.InvSlotObjCount = make([]int, com.Width*com.Height)
				com.Center = var4.G1() == 1
				var11 = var4.G1()
				if arg1 != nil {
					com.Font = arg1[var11]
				}
				com.Shadowed = var4.G1() == 1
				com.Colour = var4.G4()
				com.MarginX = var4.G2B()
				com.MarginY = var4.G2B()
				com.Interactable = var4.G1() == 1
				com.IOps = make([]string, 5)
				for i := range 5 {
					com.IOps[i] = var4.GStr()
					if len(com.IOps[i]) == 0 {
						com.IOps[i] = ""
					}
				}
			}
			if com.ButtonType == 2 || com.Type == 2 {
				com.ActionVerb = var4.GStr()
				com.Action = var4.GStr()
				com.ActionTarget = var4.G2()
			}
		}
		com.Option = var4.GStr()
		if len(com.Option) == 0 {
			switch com.ButtonType {
			case 1:
				com.Option = "Ok"
			case 4, 5:
				com.Option = "Select"
			case 6:
				com.Option = "Continue"
			}
		}
	}
}

// Java: getTempModel (IfType.java:437-463 @2e62978; was Component.getModel
// at 245.2).
func (c *IfType) GetTempModel(arg0 int, arg1 int, arg2 bool, localPlayer *playerentity.ClientPlayer) *model.Model {
	var m *model.Model // Java: model — resolved deferred (type,id) pair
	if arg2 {
		m = c.LoadModel(c.ActiveModelType, c.ActiveModel, localPlayer)
	} else {
		m = c.LoadModel(c.ModelType, c.Model, localPlayer)
	}
	if m == nil {
		return nil
	}
	if arg0 == -1 && arg1 == -1 && m.FaceColour == nil {
		return m
	}
	// Java: new Model(AnimFrame.shareAlpha(arg1) & AnimFrame.shareAlpha(arg3),
	// false, true, var5) (IfType.java:450 @2e62978) — the alpha-share flag was
	// the constant true at 245.2 (WS3); the ctor arg reorder is signature-only.
	var5 := model.NewModel4(m, true, animframe.ShareAlpha(arg0) && animframe.ShareAlpha(arg1), false)
	if arg0 != -1 || arg1 != -1 {
		var5.PrepareAnim()
	}
	if arg0 != -1 {
		var5.Animate(arg0)
	}
	if arg1 != -1 {
		var5.Animate(arg1)
	}
	var5.CalculateNormals(64, 768, -50, -10, -50, true)
	return var5
}

// Java: IfType.loadModel (IfType.java:458-483 @176a85f).
func (c *IfType) LoadModel(arg0 int, arg1 int, localPlayer *playerentity.ClientPlayer) *model.Model {
	// Java: (long) ((arg0 << 16) + arg1) — 245.2 does int arithmetic and widens
	// AFTER the add (244 widened type before the shift); int32 wrap preserves
	// Java int overflow. Equivalent for valid ids.
	var3 := ModelCache.Find(int64(int32((arg0 << 16) + arg1)))
	if var3 != nil {
		return var3
	}
	if arg0 == 1 {
		var3 = model.Load(arg1)
	}
	if arg0 == 2 {
		var3 = npctype.Get(arg1).GetHead()
	}
	if arg0 == 3 {
		var3 = localPlayer.GetHeadModel()
	}
	if arg0 == 4 {
		var3 = objtype.Get(arg1).GetInvModel(50) // Java: IfType.loadModel uses ObjType.getInvModel (not getInterfaceModel) — IfType.java:472 @176a85f
	}
	if arg0 == 5 {
		var3 = nil
	}
	if var3 != nil {
		ModelCache.Put(int64(int32((arg0<<16)+arg1)), var3)
	}
	return var3
}

// Java: IfType.cacheModel (IfType.java:518-523).
func CacheModel(m *model.Model, id int, typ int) {
	ModelCache.Clear()
	if m != nil && typ != 4 {
		// Java: (long)((arg3 << 16) + arg1) — the add runs in 32-bit int and
		// wraps before widening, like LoadModel's key (audit config-B-04).
		ModelCache.Put(int64(int32((typ<<16)+id)), m)
	}
}

func GetImage(arg0 *io.JagFile, arg1 int, arg2 string) (result *pix32.Pix32) {
	var4 := (jstring.HashCode(arg2) << 8) + int64(arg1)
	var6 := ImageCache.Find(var4)
	if var6 != nil {
		return var6
	}
	// Java: IfType.java:433-439 — try/catch returning null on any
	// exception during Pix32 construction (missing/corrupt media entry,
	// bad format). The Go pix32 ctor can panic on malformed input; mirror
	// Java's tolerance so a single bad asset doesn't brick boot/Unpack.
	defer func() {
		if recover() != nil {
			result = nil
		}
	}()
	var6 = pix32.NewPix323(arg0, arg2, arg1)
	ImageCache.Put(var4, var6)
	return var6
}

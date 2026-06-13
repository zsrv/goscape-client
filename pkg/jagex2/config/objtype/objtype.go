package objtype

import (
	"strings"

	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix32"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix3d"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

var (
	Count        int
	Offsets      []int
	Dat          *io.Packet
	Cache        []*ObjType
	CachePos     int
	MembersWorld bool = true

	ModelCache  = datastruct.NewLruCache[*model.Model](50)
	SpriteCache = datastruct.NewLruCache[*pix32.Pix32](100) // Java: new LruCache(100)
)

type ObjType struct {
	Index  int
	Model  int
	Name   string
	Desc   []byte
	RecolS []int
	RecolD []int
	Zoom2D int
	Xan2D  int
	Yan2D  int
	Zan2D  int
	Xof2D  int
	Yof2D  int
	// Java: field1034 (ObjType.java @2e62978; was 245.2's field1045) —
	// assigned by opcode 10 but never read in Java or Go. Pure
	// deobfuscator residue; field omitted per the deob-artifact
	// exclusion policy; Decode keeps the G2 read as a discard. 254
	// deletes opcode 9 + its boolean (245.2 field1044) outright.
	Stackable        bool
	Cost             int
	ManWearOffset   int8
	WomanWearOffset int8
	ManWear          int
	ManWear2         int
	WomanWear        int
	WomanWear2       int
	ManWear3         int
	WomanWear3       int
	ManHead          int
	ManHead2         int
	WomanHead        int
	WomanHead2       int
	CertLink         int
	CertTemplate     int
	Members          bool
	CountObj         []int
	CountCo          []int
	Op               []string
	IOp              []string
	// Java: ObjType resizex/resizey/resizez/ambient/contrast (rev-244 opcodes
	// 110-114). Consumed by Scale + CalculateNormals in GetInterfaceModel.
	ResizeX  int
	ResizeY  int
	ResizeZ  int
	Ambient  int
	Contrast int
}

func NewObjType() *ObjType {
	return &ObjType{
		Index: -1,
	}
}

func Init(arg0 *io.JagFile) {
	Dat = io.NewPacket(arg0.Read("obj.dat", nil))
	var1 := io.NewPacket(arg0.Read("obj.idx", nil))
	Count = var1.G2()
	Offsets = make([]int, Count)
	var2 := 2
	for i := range Count {
		Offsets[i] = var2
		var2 += var1.G2()
	}
	Cache = make([]*ObjType, 10)
	for i := range 10 {
		Cache[i] = NewObjType()
	}
}

func Unload() {
	ModelCache = nil
	SpriteCache = nil
	Offsets = nil
	Cache = nil
	Dat = nil
}

// List fetches (decoding on miss) the definition for obj id arg0.
// Java 274: list (ObjType.java:183 @32f3062; was get at 254).
func List(arg0 int) *ObjType {
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
	if var2.CertTemplate != -1 {
		var2.ToCertificate()
	}
	if !MembersWorld && var2.Members {
		var2.Name = "Members Object"
		var2.Desc = []byte("Login to a members' server to use this object.")
		var2.Op = nil
		var2.IOp = nil
	}
	return var2
}

func (t *ObjType) Reset() {
	t.Model = 0
	t.Name = ""
	t.Desc = nil
	t.RecolS = nil
	t.RecolD = nil
	t.Zoom2D = 2000
	t.Xan2D = 0
	t.Yan2D = 0
	t.Zan2D = 0
	t.Xof2D = 0
	t.Yof2D = 0
	t.Stackable = false
	t.Cost = 1
	t.Members = false
	t.Op = nil
	t.IOp = nil
	t.ManWear = -1
	t.ManWear2 = -1
	t.ManWearOffset = 0
	t.WomanWear = -1
	t.WomanWear2 = -1
	t.WomanWearOffset = 0
	t.ManWear3 = -1
	t.WomanWear3 = -1
	t.ManHead = -1
	t.ManHead2 = -1
	t.WomanHead = -1
	t.WomanHead2 = -1
	t.CountObj = nil
	t.CountCo = nil
	t.CertLink = -1
	t.CertTemplate = -1
	// Java: ObjType.reset (rev-244)
	t.ResizeX = 128
	t.ResizeY = 128
	t.ResizeZ = 128
	t.Ambient = 0
	t.Contrast = 0
}

func (t *ObjType) Decode(arg1 *io.Packet) {
	for {
		var3 := arg1.G1()
		switch var3 {
		case 0:
			return
		case 1:
			t.Model = arg1.G2()
		case 2:
			t.Name = arg1.GStr()
		case 3:
			t.Desc = arg1.GStrByte()
		case 4:
			t.Zoom2D = arg1.G2()
		case 5:
			t.Xan2D = arg1.G2()
		case 6:
			t.Yan2D = arg1.G2()
		case 7:
			t.Xof2D = arg1.G2()
			if t.Xof2D > 32767 {
				t.Xof2D -= 65536
			}
		case 8:
			t.Yof2D = arg1.G2()
			if t.Yof2D > 32767 {
				t.Yof2D -= 65536
			}
		// Java: ObjType.java:278-279 @2e62978 — opcode 10 writes the dead
		// field1034; G2 read kept as a discard so packet-position
		// alignment matches Java. Opcode 9 (245.2 field1044) is deleted
		// at 254.
		case 10:
			arg1.G2()
		case 11:
			t.Stackable = true
		case 12:
			t.Cost = arg1.G4()
		case 16:
			t.Members = true
		case 23:
			t.ManWear = arg1.G2()
			t.ManWearOffset = arg1.G1B()
		case 24:
			t.ManWear2 = arg1.G2()
		case 25:
			t.WomanWear = arg1.G2()
			t.WomanWearOffset = arg1.G1B()
		case 26:
			t.WomanWear2 = arg1.G2()
		case 30, 31, 32, 33, 34:
			if t.Op == nil {
				t.Op = make([]string, 5)
			}
			t.Op[var3-30] = arg1.GStr()
			// Java assigns op[i] = null here; Go uses "" — see LocType.Decode
			// for the convention's full rationale.
			if strings.ToLower(t.Op[var3-30]) == "hidden" {
				t.Op[var3-30] = ""
			}
		case 35, 36, 37, 38, 39:
			if t.IOp == nil {
				t.IOp = make([]string, 5)
			}
			t.IOp[var3-35] = arg1.GStr()
		case 40:
			var4 := arg1.G1()
			t.RecolS = make([]int, var4)
			t.RecolD = make([]int, var4)
			for i := range var4 {
				t.RecolS[i] = arg1.G2()
				t.RecolD[i] = arg1.G2()
			}
		case 78:
			t.ManWear3 = arg1.G2()
		case 79:
			t.WomanWear3 = arg1.G2()
		case 90:
			t.ManHead = arg1.G2()
		case 91:
			t.WomanHead = arg1.G2()
		case 92:
			t.ManHead2 = arg1.G2()
		case 93:
			t.WomanHead2 = arg1.G2()
		case 95:
			t.Zan2D = arg1.G2()
		case 97:
			t.CertLink = arg1.G2()
		case 98:
			t.CertTemplate = arg1.G2()
		case 100, 101, 102, 103, 104, 105, 106, 107, 108, 109:
			if t.CountObj == nil {
				t.CountObj = make([]int, 10)
				t.CountCo = make([]int, 10)
			}
			t.CountObj[var3-100] = arg1.G2()
			t.CountCo[var3-100] = arg1.G2()
		// Java: ObjType.decode opcodes 110-114 (rev-244).
		case 110:
			t.ResizeX = arg1.G2()
		case 111:
			t.ResizeY = arg1.G2()
		case 112:
			t.ResizeZ = arg1.G2()
		case 113:
			t.Ambient = int(arg1.G1B())
		case 114:
			t.Contrast = int(arg1.G1B()) * 5
		}
	}
}

func (t *ObjType) ToCertificate() {
	var2 := List(t.CertTemplate)
	t.Model = var2.Model
	t.Zoom2D = var2.Zoom2D
	t.Xan2D = var2.Xan2D
	t.Yan2D = var2.Yan2D
	t.Zan2D = var2.Zan2D
	t.Xof2D = var2.Xof2D
	t.Yof2D = var2.Yof2D
	t.RecolS = var2.RecolS
	t.RecolD = var2.RecolD
	var3 := List(t.CertLink)
	t.Name = var3.Name
	t.Members = var3.Members
	t.Cost = var3.Cost
	var4 := "a"
	// Java: var3.name.charAt(0). RuneScape item names are ASCII, so byte
	// indexing matches charAt for valid inputs. A non-ASCII first char would
	// yield the leading UTF-8 byte (e.g. 0xC2 for £), which can never match
	// the vowel set — safe outcome for both clients.
	var5 := var3.Name[0]
	if var5 == 'A' || var5 == 'E' || var5 == 'I' || var5 == 'O' || var5 == 'U' {
		var4 = "an"
	}
	// Java: String.getBytes() emits ONE byte per char (platform/Latin-1);
	// []byte(string) would emit multi-byte UTF-8 for chars like '£', and the
	// Desc consumer transcodes Latin1→UTF8 on read (audit objtype-02).
	var6 := "Swap this note at any bank for " + var4 + " " + var3.Name + "."
	desc := make([]byte, 0, len(var6))
	for _, r := range var6 {
		desc = append(desc, byte(r))
	}
	t.Desc = desc
	t.Stackable = true
}

func (t *ObjType) GetInterfaceModel(arg0 int) *model.Model {
	if t.CountObj != nil && arg0 > 1 {
		var2 := -1
		for i := range 10 {
			if arg0 >= t.CountCo[i] && t.CountCo[i] != 0 {
				var2 = t.CountObj[i]
			}
		}
		if var2 != -1 {
			return List(var2).GetInterfaceModel(1)
		}
	}
	var4 := ModelCache.Find(int64(t.Index))
	if var4 != nil {
		return var4
	}
	var4 = model.Load(t.Model)
	if var4 == nil {
		return nil
	}
	if t.ResizeX != 128 || t.ResizeY != 128 || t.ResizeZ != 128 {
		// Java: ObjType.getModel scale(resizey, resizez, resizex); Scale(arg0=z, arg2=y, arg3=x)
		var4.Scale(t.ResizeZ, t.ResizeY, t.ResizeX)
	}
	if t.RecolS != nil {
		for i := range len(t.RecolS) {
			var4.Recolor(t.RecolS[i], t.RecolD[i])
		}
	}
	var4.CalculateNormals(t.Ambient+64, t.Contrast+768, -50, -10, -50, true)
	var4.UseAABBMouseCheck = true
	ModelCache.Put(int64(t.Index), var4)
	return var4
}

// GetSprite renders (or fetches from SpriteCache) the 32x32 inventory icon for
// obj `id` at stack `count`. outlineRgb selects the variant: 0 = plain
// (cacheable, dark shadow pass), >0 = selection outline painted in that
// colour (1.04x zoom), -1 = cert-link sub-icon (1.5x zoom, no outline/shadow).
// Java 274 moves id to the first param: getSprite(id, outlineRgb, count)
// (ObjType.java:208 @32f3062) — a real reorder, not compensated churn: 254's
// getSprite took (outlineRgb, count, id) (ObjType.java:442 @2e62978).
func GetSprite(id int, outlineRgb int, count int) *pix32.Pix32 {
	// Java: the icon cache is only consulted (and later populated) for the
	// plain outlineRgb==0 variant (ObjType.java:475-486, 602-604).
	if outlineRgb == 0 {
		var3 := SpriteCache.Find(int64(id))
		if var3 != nil && var3.OHi != count && var3.OHi != -1 {
			// Java: var3.unlink() — Linkable2's unlink() removes the node
			// from both the hashtable bucket and the history list. The Go port
			// of LruCache exposes Delete(key) for the same effect.
			SpriteCache.Delete(int64(id))
			var3 = nil
		}
		if var3 != nil {
			return var3
		}
	}
	var4 := List(id)
	if var4.CountObj == nil {
		count = -1
	}
	if count > 1 {
		var5 := -1
		for i := range 10 {
			if count >= var4.CountCo[i] && var4.CountCo[i] != 0 {
				var5 = var4.CountObj[i]
			}
		}
		if var5 != -1 {
			var4 = List(var5)
		}
	}
	// Java: ObjType.getIcon fetches and null-checks the model BEFORE creating
	// the Pix32 buffer or saving/mutating any pix2d/pix3d global state, so an
	// early return on a cache-miss leaves render state untouched.
	// Java: Client-Java ObjType.getIcon lines 507-508 (01f16088)
	var15 := var4.GetInterfaceModel(1)
	if var15 == nil {
		return nil
	}
	// Java: the cert-link sub-icon is fetched BEFORE binding the icon buffer —
	// the recursive call rebinds Pix2D itself (ObjType.java:512-519). The -1
	// variant applies the 1.5x zoom.
	var linkedIcon *pix32.Pix32
	if var4.CertTemplate != -1 {
		// Java 274: getSprite(var5.certlink, -1, 10) (ObjType.java:240 @32f3062).
		linkedIcon = GetSprite(var4.CertLink, -1, 10)
		if linkedIcon == nil {
			return nil
		}
	}
	var3 := pix32.NewPix321(32, 32)
	var5 := pix3d.CenterW3D
	var6 := pix3d.CenterH3D
	var7 := pix3d.LineOffset
	var8 := pix2d.Data
	var9 := pix2d.Width2D
	var10 := pix2d.Height2D
	var11 := pix2d.ClipMinX
	var12 := pix2d.ClipMaxX
	var13 := pix2d.ClipMinY
	var14 := pix2d.ClipMaxY
	pix3d.LowDetail = false
	pix2d.SetPixels(32, var3.Pixels, 32)
	pix2d.FillRect(0, 0, 0, 32, 32)
	pix3d.Init()
	// Java: zoom scaling per variant (ObjType.java:539-544, new in 244).
	zoom := var4.Zoom2D
	if outlineRgb == -1 {
		zoom = int(float64(zoom) * 1.5)
	} else if outlineRgb > 0 {
		zoom = int(float64(zoom) * 1.04)
	}
	// Java: `Pix3D.sinTable[xan2d] * zoom >> 16` is 32-bit int arithmetic; the
	// product overflows/wraps at 2^31 (reachable when zoom2d > 32768). int32(...)
	// reproduces that truncation before the arithmetic >>16, which Go's 64-bit int
	// would otherwise skip. Same fix as DrawInterface type-6 (client.go).
	var16 := int(int32(pix3d.SinTable[var4.Xan2D]*zoom)) >> 16
	var17 := int(int32(pix3d.CosTable[var4.Xan2D]*zoom)) >> 16
	var15.DrawSimple(0, var4.Yan2D, var4.Zan2D, var4.Xan2D, var4.Xof2D, var16+var15.MaxY/2+var4.Yof2D, var17+var4.Yof2D)
	for i := 31; i >= 0; i-- {
		for j := 31; j >= 0; j-- {
			if var3.Pixels[i+j*32] == 0 {
				if i > 0 && var3.Pixels[i-1+j*32] > 1 {
					var3.Pixels[i+j*32] = 1
				} else if j > 0 && var3.Pixels[i+(j-1)*32] > 1 {
					var3.Pixels[i+j*32] = 1
				} else if i < 31 && var3.Pixels[i+1+j*32] > 1 {
					var3.Pixels[i+j*32] = 1
				} else if j < 31 && var3.Pixels[i+(j+1)*32] > 1 {
					var3.Pixels[i+j*32] = 1
				}
			}
		}
	}
	if outlineRgb > 0 {
		// Java: ObjType.java:567-582 (new in 244) — paint the selection
		// outline around the silhouette.
		for i := 31; i >= 0; i-- {
			for j := 31; j >= 0; j-- {
				if var3.Pixels[i+j*32] == 0 {
					if i > 0 && var3.Pixels[i-1+j*32] == 1 {
						var3.Pixels[i+j*32] = outlineRgb
					} else if j > 0 && var3.Pixels[i+(j-1)*32] == 1 {
						var3.Pixels[i+j*32] = outlineRgb
					} else if i < 31 && var3.Pixels[i+1+j*32] == 1 {
						var3.Pixels[i+j*32] = outlineRgb
					} else if j < 31 && var3.Pixels[i+(j+1)*32] == 1 {
						var3.Pixels[i+j*32] = outlineRgb
					}
				}
			}
		}
	} else if outlineRgb == 0 {
		// Java: ObjType.java:583-591 — the dark drop-shadow pass is gated to
		// the plain variant in 244.
		for i := 31; i >= 0; i-- {
			for j := 31; j >= 0; j-- {
				if var3.Pixels[i+j*32] == 0 && i > 0 && j > 0 && var3.Pixels[i-1+(j-1)*32] > 0 {
					var3.Pixels[i+j*32] = 3153952
				}
			}
		}
	}
	if var4.CertTemplate != -1 {
		// Java: ObjType.java:593-601 — 1:1 plotSprite blit of the cert-link
		// icon over the note background (225 used the crop/scale routine).
		var21 := linkedIcon.OWi
		var22 := linkedIcon.OHi
		linkedIcon.OWi = 32
		linkedIcon.OHi = 32
		linkedIcon.PlotSprite(0, 0)
		linkedIcon.OWi = var21
		linkedIcon.OHi = var22
	}
	if outlineRgb == 0 { // Java: ObjType.java:602-604
		SpriteCache.Put(int64(id), var3)
	}
	pix2d.SetPixels(var9, var8, var10)
	pix2d.SetClipping(var14, var13, var12, var11)
	pix3d.CenterW3D = var5
	pix3d.CenterH3D = var6
	pix3d.LineOffset = var7
	pix3d.LowDetail = true
	if var4.Stackable {
		var3.OWi = 33
	} else {
		var3.OWi = 32
	}
	var3.OHi = count
	return var3
}

// Java: checkWearModel (ObjType.java:625-650) — 244 lazy-model load barrier
// for worn equipment models: requests all gendered wear parts from OnDemand
// and reports whether every one is resident. Non-short-circuit on purpose:
// each part is requested even after the first miss, like Java.
func (t *ObjType) CheckWearModel(gender int) bool {
	wear := t.ManWear
	wear2 := t.ManWear2
	wear3 := t.ManWear3
	if gender == 1 {
		wear = t.WomanWear
		wear2 = t.WomanWear2
		wear3 = t.WomanWear3
	}
	if wear == -1 {
		return true
	}
	ready := true //nolint:staticcheck // QF1007: kept split to mirror Java's flag shape (ObjType.java:639-648)
	if !model.RequestDownload(wear) {
		ready = false
	}
	if wear2 != -1 && !model.RequestDownload(wear2) {
		ready = false
	}
	if wear3 != -1 && !model.RequestDownload(wear3) {
		ready = false
	}
	return ready
}

// GetWearModelNoCheck builds the worn-equipment model for the given gender
// without a load barrier (callers gate on CheckWearModel first).
// Java: getWearModelNoCheck (ObjType.java:598-635 @2e62978; was getWearModel
// at 245.2 — net-equivalent branch restructure). The Translate arg order is
// the compensated Go body mapping — do not "fix" against the Java literals.
func (t *ObjType) GetWearModelNoCheck(arg1 int) *model.Model {
	var4 := t.ManWear  // Java: var4
	var5 := t.ManWear2 // Java: var5
	var6 := t.ManWear3 // Java: var6
	if arg1 == 1 {
		var4 = t.WomanWear
		var5 = t.WomanWear2
		var6 = t.WomanWear3
	}
	if var4 == -1 {
		return nil
	}
	var7 := model.Load(var4) // Java: var7
	if var5 != -1 {
		if var6 == -1 {
			var11 := model.Load(var5)
			var12 := []*model.Model{var7, var11}
			var7 = model.NewModel2(var12, 2)
		} else {
			var8 := model.Load(var5)
			var9 := model.Load(var6)
			var10 := []*model.Model{var7, var8, var9}
			var7 = model.NewModel2(var10, 3)
		}
	}
	if arg1 == 0 && t.ManWearOffset != 0 {
		var7.Translate(0, int(t.ManWearOffset), 0) // Java: translate(0, manwearOffset, 0) (ObjType.java:627 @32f3062)
	}
	if arg1 == 1 && t.WomanWearOffset != 0 {
		var7.Translate(0, int(t.WomanWearOffset), 0) // Java: translate(0, womanwearOffset, 0) (ObjType.java:630 @32f3062)
	}
	if t.RecolS != nil {
		for i := range len(t.RecolS) {
			var7.Recolor(t.RecolS[i], t.RecolD[i])
		}
	}
	return var7
}

// Java: checkHeadModel (ObjType.java:697-717) — load barrier for the chat
// head models, mirroring CheckWearModel.
func (t *ObjType) CheckHeadModel(gender int) bool {
	head := t.ManHead
	head2 := t.ManHead2
	if gender == 1 {
		head = t.WomanHead
		head2 = t.WomanHead2
	}
	if head == -1 {
		return true
	}
	ready := true //nolint:staticcheck // QF1007: kept split to mirror Java's flag shape (ObjType.java:709-716)
	if !model.RequestDownload(head) {
		ready = false
	}
	if head2 != -1 && !model.RequestDownload(head2) {
		ready = false
	}
	return ready
}

// GetHeadModelNoCheck builds the chathead model for the given gender without
// a load barrier (callers gate on CheckHeadModel first).
// Java: getHeadModelNoCheck (ObjType.java:659-681 @2e62978; was getHeadModel
// at 245.2).
func (t *ObjType) GetHeadModelNoCheck(arg1 int) *model.Model {
	var3 := t.ManHead  // Java: var3
	var4 := t.ManHead2 // Java: var4
	if arg1 == 1 {
		var3 = t.WomanHead
		var4 = t.WomanHead2
	}
	if var3 == -1 {
		return nil
	}
	var5 := model.Load(var3)
	if var4 != -1 {
		var6 := model.Load(var4)
		var7 := []*model.Model{var5, var6}
		var5 = model.NewModel2(var7, 2)
	}
	if t.RecolS != nil {
		for i := range len(t.RecolS) {
			var5.Recolor(t.RecolS[i], t.RecolD[i])
		}
	}
	return var5
}

// GetInvModel returns the inventory model for the obj, or nil if not yet available.
// Java: ObjType.getInvModel (rev-244). Unlike GetInterfaceModel, this does NOT
// scale/calculateNormals/cache — it is the raw recoloured model for Component use.
func (t *ObjType) GetInvModel(arg0 int) *model.Model {
	if t.CountObj != nil && arg0 > 1 {
		var2 := -1
		for i := range 10 {
			if arg0 >= t.CountCo[i] && t.CountCo[i] != 0 {
				var2 = t.CountObj[i]
			}
		}
		if var2 != -1 {
			return List(var2).GetInvModel(1)
		}
	}
	var4 := model.Load(t.Model)
	if var4 == nil {
		return nil
	}
	if t.RecolS != nil {
		for i := range len(t.RecolS) {
			var4.Recolor(t.RecolS[i], t.RecolD[i])
		}
	}
	return var4
}

package objtype

import (
	"strings"

	"goscape-client/pkg/jagex2/datastruct"
	"goscape-client/pkg/jagex2/graphics/model"
	"goscape-client/pkg/jagex2/graphics/pix2d"
	"goscape-client/pkg/jagex2/graphics/pix32"
	"goscape-client/pkg/jagex2/graphics/pix3d"
	"goscape-client/pkg/jagex2/io"
)

var (
	Count        int
	Offsets      []int
	Dat          *io.Packet
	Cache        []*ObjType
	CachePos     int
	MembersWorld bool = true

	ModelCache = datastruct.NewLruCache[*model.Model](50)
	IconCache  = datastruct.NewLruCache[*pix32.Pix32](200)
)

type ObjType struct {
	Index            int
	Model            int
	Name             string
	Desc             []byte
	RecolS           []int
	RecolD           []int
	Zoom2D           int
	Xan2D            int
	Yan2D            int
	Zan2D            int
	Xof2D            int
	Yof2D            int
	Code9            bool
	Code10           int
	Stackable        bool
	Cost             int
	ManWearOffsetY   byte
	WomanWearOffsetY byte
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
}

func NewObjType() *ObjType {
	return &ObjType{
		Index: -1,
	}
}

func Unpack(arg0 *io.Jagfile) {
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
	IconCache = nil
	Offsets = nil
	Cache = nil
	Dat = nil
}

func Get(arg0 int) *ObjType {
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
	t.Code9 = false
	t.Code10 = -1
	t.Stackable = false
	t.Cost = 1
	t.Members = false
	t.Op = nil
	t.IOp = nil
	t.ManWear = -1
	t.ManWear2 = -1
	t.ManWearOffsetY = 0
	t.WomanWear = -1
	t.WomanWear2 = -1
	t.WomanWearOffsetY = 0
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
			t.Name = arg1.GJStr()
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
		case 9:
			t.Code9 = true
		case 10:
			t.Code10 = arg1.G2()
		case 11:
			t.Stackable = true
		case 12:
			t.Cost = arg1.G4()
		case 16:
			t.Members = true
		case 23:
			t.ManWear = arg1.G2()
			t.ManWearOffsetY = arg1.G1B()
		case 24:
			t.ManWear2 = arg1.G2()
		case 25:
			t.WomanWear = arg1.G2()
			t.WomanWearOffsetY = arg1.G1B()
		case 26:
			t.WomanWear2 = arg1.G2()
		case 30, 31, 32, 33, 34:
			if t.Op == nil {
				t.Op = make([]string, 5)
			}
			t.Op[var3-30] = arg1.GJStr()
			if strings.ToLower(t.Op[var3-30]) == "hidden" {
				t.Op[var3-30] = ""
			}
		case 35, 36, 37, 38, 39:
			if t.IOp == nil {
				t.IOp = make([]string, 5)
			}
			t.IOp[var3-35] = arg1.GJStr()
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
		}
	}
}

func (t *ObjType) ToCertificate() {
	var2 := Get(t.CertTemplate)
	t.Model = var2.Model
	t.Zoom2D = var2.Zoom2D
	t.Xan2D = var2.Xan2D
	t.Yan2D = var2.Yan2D
	t.Zan2D = var2.Zan2D
	t.Xof2D = var2.Xof2D
	t.Yof2D = var2.Yof2D
	t.RecolS = var2.RecolS
	t.RecolD = var2.RecolD
	var3 := Get(t.CertLink)
	t.Name = var3.Name
	t.Members = var3.Members
	t.Cost = var3.Cost
	var4 := "a"
	var5 := var3.Name[0]
	if var5 == 'A' || var5 == 'E' || var5 == 'I' || var5 == 'O' || var5 == 'U' {
		var4 = "an"
	}
	t.Desc = []byte("Swap this note at any bank for " + var4 + " " + var3.Name + ".")
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
		if var2 != 1 {
			return Get(var2).GetInterfaceModel(1)
		}
	}
	var4 := ModelCache.Get(int64(t.Index)).Value
	if var4 != nil {
		return var4
	}
	var4 = model.NewModel1(t.Model)
	if t.RecolS != nil {
		for i := range len(t.RecolS) {
			var4.Recolor(t.RecolS[i], t.RecolD[i])
		}
	}
	var4.CalculateNormals(64, 768, -50, -10, -50, true)
	var4.Pickable = true
	//ModelCache.Put(int64(t.Index), var4) // TODO: LruCache
	return var4
}

func GetIcon(arg0, arg2 int) *pix32.Pix32 {
	var3 := IconCache.Get(int64(arg0)).Value
	if var3 != nil && var3.CropH != arg2 && var3.CropH != -1 {
		var3.Unlink()
		var3 = nil
	}
	if var3 != nil {
		return var3
	}
	var4 := Get(arg0)
	if var4.CountObj == nil {
		arg2 = -1
	}
	var5 := 0
	if arg2 > 1 {
		var5 = -1
		for i := range 10 {
			if arg2 >= var4.CountCo[i] && var4.CountCo[i] != 0 {
				var5 = var4.CountObj[i]
			}
		}
		if var5 != -1 {
			var4 = Get(var5)
		}
	}
	var3 = pix32.NewPix321(32, 32)
	var5 = pix3d.CenterW3D
	var6 := pix3d.CenterH3D
	var7 := pix3d.LineOffset
	var8 := pix2d.Data
	var9 := pix2d.Width2D
	var10 := pix2d.Height2D
	var11 := pix2d.BoundLeft
	var12 := pix2d.BoundRight
	var13 := pix2d.BoundTop
	var14 := pix2d.BoundBottom
	pix3d.Jagged = false
	pix2d.Bind(32, var3.Pixels, 32)
	pix2d.FillRect(0, 0, 0, 32, 32)
	pix3d.Init2D()
	var15 := var4.GetInterfaceModel(1)
	var16 := pix3d.SinTable[var4.Xan2D] * var4.Zoom2D >> 16
	var17 := pix3d.CosTable[var4.Xan2D] * var4.Zoom2D >> 16
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
	for i := 31; i >= 0; i-- {
		for j := 31; j >= 0; j-- {
			if var3.Pixels[i+j*32] == 0 && i > 0 && j > 0 && var3.Pixels[i-1+(j-1)*32] > 0 {
				var3.Pixels[i+j*32] = 3153952
			}
		}
	}
	if var4.CertTemplate != -1 {
		var20 := GetIcon(var4.CertLink, 10)
		var21 := var20.CropW
		var22 := var20.CropH
		var20.CropW = 32
		var20.CropH = 32
		var20.Crop(22, 5, 22, 5)
		var20.CropW = var21
		var20.CropH = var22
	}
	//IconCache.Put(arg0, var3) // TODO: LruCache
	pix2d.Bind(var9, var8, var10)
	pix2d.SetClipping(var14, var13, var12, var11)
	pix3d.CenterW3D = var5
	pix3d.CenterH3D = var6
	pix3d.LineOffset = var7
	pix3d.Jagged = true
	if var4.Stackable {
		var3.CropW = 33
	} else {
		var3.CropW = 32
	}
	var3.CropH = arg2
	return var3
}

func (t *ObjType) GetWornModel(arg1 int) *model.Model {
	var3 := t.ManWear
	if arg1 == 1 {
		var3 = t.WomanWear
	}
	if var3 == -1 {
		return nil
	}
	var4 := t.ManWear2
	var5 := t.ManWear3
	if arg1 == 1 {
		var4 = t.WomanWear2
		var5 = t.WomanWear3
	}
	var6 := model.NewModel1(var3)
	if var4 != -1 {
		var var7 *model.Model
		if var5 == -1 {
			var7 = model.NewModel1(var4)
			var11 := []*model.Model{var6, var7}
			var6 = model.NewModel2(var11, 2)
		} else {
			var7 = model.NewModel1(var4)
			var8 := model.NewModel1(var5)
			var9 := []*model.Model{var6, var7, var8}
			var6 = model.NewModel2(var9, 3)
		}
	}
	if arg1 == 0 && t.ManWearOffsetY != 0 {
		var6.Translate(int(t.ManWearOffsetY), 0, 0)
	}
	if arg1 == 1 && t.WomanWearOffsetY != 0 {
		var6.Translate(int(t.WomanWearOffsetY), 0, 0)
	}
	if t.RecolS != nil {
		for i := range len(t.RecolS) {
			var6.Recolor(t.RecolS[i], t.RecolD[i])
		}
	}
	return var6
}

func (t *ObjType) GetHeadModel(arg1 int) *model.Model {
	var3 := t.ManHead
	if arg1 == 1 {
		var3 = t.WomanHead
	}
	if var3 == -1 {
		return nil
	}
	var4 := t.ManHead2
	if arg1 == 1 {
		var4 = t.WomanHead2
	}
	var5 := model.NewModel1(var3)
	if var4 != -1 {
		var6 := model.NewModel1(var4)
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

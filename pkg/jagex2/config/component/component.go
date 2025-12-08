package component

import (
	"strconv"
	"strings"

	"goscape-client/pkg/jagex2/datastruct"
	"goscape-client/pkg/jagex2/graphics/model"
	"goscape-client/pkg/jagex2/graphics/pix32"
	"goscape-client/pkg/jagex2/graphics/pixfont"
	"goscape-client/pkg/jagex2/io"
)

var (
	Instances  []*Component
	ImageCache *datastruct.LruCache[*pix32.Pix32]
	ModelCache *datastruct.LruCache[*model.Model]
)

type Component struct {
	InvSlotObjId     []int
	InvSlotObjCount  []int
	SeqFrame         int
	SeqCycle         int
	Id               int
	Layer            int
	Type             int
	ButtonType       int
	ClientCode       int
	Width            int
	Height           int
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
	Anim             int
	ActiveAnim       int
	UnusedShort1     int
	MarginX          int
	MarginY          int
	Model            *model.Model
	ActiveModel      *model.Model
	Graphic          *pix32.Pix32
	ActiveGraphic    *pix32.Pix32
	Font             *pixfont.PixFont
	Text             string
	ActiveText       string
	UnusedBoolean1   bool
	Draggable        bool
	Interactable     bool
	Usable           bool
	Fill             bool
	Center           bool
	Shadowed         bool
	InvSlotOffsetX   []int
	InvSlotOffsetY   []int
	InvSlotSprite    []*pix32.Pix32
	IOps             []string
}

func NewComponent() *Component {
	return new(Component)
}

func Unpack(arg0 *io.Jagfile, arg1 []*pixfont.PixFont, arg3 *io.Jagfile) {
	ImageCache = datastruct.NewLruCache[*pix32.Pix32](50000)
	ModelCache = datastruct.NewLruCache[*model.Model](50000)
	var4 := io.NewPacket(arg3.Read("data", nil))
	var5 := -1
	var6 := var4.G2()
	Instances = make([]*Component, var6)
	for {
		var var8 *Component
		for ok := true; ok; ok = var8.ButtonType != -1 && var8.ButtonType != 4 && var8.ButtonType != 5 && var8.ButtonType != 6 {
			if var4.Pos >= len(var4.Data) {
				ImageCache = nil
				ModelCache = nil
				return
			}
			var7 := var4.G2()
			if var7 == 65535 {
				var5 = var4.G2()
				var7 = var4.G2()
			}
			Instances[var7] = NewComponent()
			var8.Id = var7
			var8.Layer = var5
			var8.Type = var4.G1()
			var8.ButtonType = var4.G1()
			var8.ClientCode = var4.G2()
			var8.Width = var4.G2()
			var8.Height = var4.G2()
			var8.OverLayer = var4.G1()
			if var8.OverLayer == 0 {
				var8.OverLayer = -1
			} else {
				var8.OverLayer = (var8.OverLayer - 1<<8) + var4.G1()
			}
			var9 := var4.G1()
			var10 := 0
			if var9 > 0 {
				var8.ScriptComparator = make([]int, var9)
				var8.ScriptOperand = make([]int, var9)
				for i := range var9 {
					var8.ScriptComparator[i] = var4.G1()
					var8.ScriptOperand[i] = var4.G2()
				}
			}
			var10 = var4.G1()
			var11 := 0
			var12 := 0
			if var10 > 0 {
				var8.Scripts = make([][]int, var10)
				for i := range var10 {
					var12 = var4.G2()
					var8.Scripts[i] = make([]int, var12)
					for j := 0; j < var12; j++ {
						var8.Scripts[i][j] = var4.G2()
					}
				}
			}
			if var8.Type == 0 {
				var8.Scroll = var4.G2()
				var8.Hide = var4.G1() == 1
				var11 = var4.G1()
				var8.ChildID = make([]int, var11)
				var8.ChildX = make([]int, var11)
				var8.ChildY = make([]int, var11)
				for i := range var11 {
					var8.ChildID[i] = var4.G2()
					var8.ChildX[i] = var4.G2B()
					var8.ChildY[i] = var4.G2B()
				}
			}
			if var8.Type == 1 {
				var8.UnusedShort1 = var4.G2()
				var8.UnusedBoolean1 = var4.G1() == 1
			}
			if var8.Type == 2 {
				var8.InvSlotObjId = make([]int, var8.Width*var8.Height)
				var8.InvSlotObjCount = make([]int, var8.Width*var8.Height)
				var8.Draggable = var4.G1() == 1
				var8.Interactable = var4.G1() == 1
				var8.Usable = var4.G1() == 1
				var8.MarginX = var4.G1()
				var8.MarginY = var4.G1()
				var8.InvSlotOffsetX = make([]int, 20)
				var8.InvSlotOffsetY = make([]int, 20)
				var8.InvSlotSprite = make([]*pix32.Pix32, 20)
				for i := range 20 {
					var12 = var4.G1()
					if var12 == 1 {
						var8.InvSlotOffsetX[i] = var4.G2B()
						var8.InvSlotOffsetY[i] = var4.G2B()
						var17 := var4.GJStr()
						if arg0 != nil && len(var17) > 0 {
							var14 := strings.LastIndex(var17, ",")
							v, err := strconv.Atoi(var17[var14+1:])
							if err != nil {
								panic(err)
							}
							var8.InvSlotSprite[i] = GetImage(arg0, v, var17[0:var14]) // TODO: check slicing logic
						}
					}
				}
				var8.IOps = make([]string, 5)
				for i := range 5 {
					var8.IOps[i] = var4.GJStr()
					if len(var8.IOps[i]) == 0 {
						var8.IOps[i] = ""
					}
				}
			}
			if var8.Type == 3 {
				var8.Fill = var4.G1() == 1
			}
			if var8.Type == 4 || var8.Type == 1 {
				var8.Center = var4.G1() == 1
				var11 = var4.G1()
				if arg1 != nil {
					var8.Font = arg1[var11]
				}
				var8.Shadowed = var4.G1() == 1
			}
			if var8.Type == 4 {
				var8.Text = var4.GJStr()
				var8.ActiveText = var4.GJStr()
			}
			if var8.Type == 1 || var8.Type == 3 || var8.Type == 4 {
				var8.Colour = var4.G4()
			}
			if var8.Type == 3 || var8.Type == 4 {
				var8.ActiveColour = var4.G4()
				var8.OverColour = var4.G4()
			}
			if var8.Type == 5 {
				var16 := var4.GJStr()
				if arg0 != nil && len(var16) > 0 {
					var12 = strings.LastIndex(var16, ",")
					v, err := strconv.Atoi(var16[var12+1:])
					if err != nil {
						panic(err)
					}
					var8.Graphic = GetImage(arg0, v, var16[0:var12]) // TODO: check slicing logic
				}
				var16 = var4.GJStr()
				if arg0 != nil && len(var16) > 0 {
					var12 = strings.LastIndex(var16, ",")
					v, err := strconv.Atoi(var16[var12+1:])
					if err != nil {
						panic(err)
					}
					var8.ActiveGraphic = GetImage(arg0, v, var16[0:var12]) // TODO: check slicing logic
				}
			}
			if var8.Type == 6 {
				var7 = var4.G1()
				if var7 != 0 {
					var8.Model = GetModel((var7 - 1<<8) + var4.G1())
				}
				var7 = var4.G1()
				if var7 != 0 {
					var8.ActiveModel = GetModel((var7 - 1<<8) + var4.G1())
				}
				var7 = var4.G1()
				if var7 == 0 {
					var8.Anim = -1
				} else {
					var8.Anim = (var7 - 1<<8) + var4.G1()
				}
				var7 = var4.G1()
				if var7 == 0 {
					var8.ActiveAnim = -1
				} else {
					var8.ActiveAnim = (var7 - 1<<8) + var4.G1()
				}
				var8.Zoom = var4.G2()
				var8.Xan = var4.G2()
				var8.Yan = var4.G2()
			}
			if var8.Type == 7 {
				var8.InvSlotObjId = make([]int, var8.Width*var8.Height)
				var8.InvSlotObjCount = make([]int, var8.Width*var8.Height)
				var8.Center = var4.G1() == 1
				var11 = var4.G1()
				if arg1 != nil {
					var8.Font = arg1[var11]
				}
				var8.Shadowed = var4.G1() == 1
				var8.Colour = var4.G4()
				var8.MarginX = var4.G2B()
				var8.MarginY = var4.G2B()
				var8.Interactable = var4.G1() == 1
				var8.IOps = make([]string, 5)
				for i := range 5 {
					var8.IOps[i] = var4.GJStr()
					if len(var8.IOps[i]) == 0 {
						var8.IOps[i] = ""
					}
				}
			}
			if var8.ButtonType == 2 || var8.Type == 2 {
				var8.ActionVerb = var4.GJStr()
				var8.Action = var4.GJStr()
				var8.ActionTarget = var4.G2()
			}
		}
		var8.Option = var4.GJStr()
		if len(var8.Option) == 0 {
			switch var8.ButtonType {
			case 1:
				var8.Option = "Ok"
			case 4, 5:
				var8.Option = "Select"
			case 6:
				var8.Option = "Continue"
			}
		}
	}
}

func (c *Component) GetModel(arg0 int, arg1 int, arg2 bool) *model.Model {
	var4 := c.Model
	if arg2 {
		var4 = c.ActiveModel
	}
	if var4 == nil {
		return nil
	}
	if arg0 == -1 && arg1 == -1 && var4.FaceColour == nil {
		return var4
	}
	var5 := model.NewModel4(var4, true, true, false)
	if arg0 != -1 || arg1 != -1 {
		var5.CreateLabelReferences()
	}
	if arg0 != -1 {
		var5.ApplyTransform(arg0)
	}
	if arg1 != -1 {
		var5.ApplyTransform(arg1)
	}
	var5.CalculateNormals(64, 768, -50, -10, -50, true)
	return var5
}

func GetImage(arg0 *io.Jagfile, arg1 int, arg2 string) *pix32.Pix32 {
	var4 := (datastruct.HashCode(arg2) << 8) + int64(arg1)
	var6 := ImageCache.Get(var4).Value
	if var6 != nil {
		return var6
	}
	var6 = pix32.NewPix323(arg0, arg2, arg1)
	//ImageCache.Put(var4, var6) // TODO
	return var6
}

func GetModel(arg1 int) *model.Model {
	var2 := ModelCache.Get(int64(arg1)).Value
	if var2 != nil {
		return var2
	}
	var2 = model.NewModel1(arg1)
	//ModelCache.Put(int64(arg1), var2) // TODO
	return var2
}

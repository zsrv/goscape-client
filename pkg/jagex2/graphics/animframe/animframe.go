package animframe

import (
	"goscape-client/pkg/jagex2/graphics/animbase"
	"goscape-client/pkg/jagex2/io"
)

var Instances []*AnimFrame

type AnimFrame struct {
	Delay  int
	Base   *animbase.AnimBase
	Length int
	Groups []int
	X      []int
	Y      []int
	Z      []int
}

func NewAnimFrame() *AnimFrame {
	return &AnimFrame{}
}

func Unpack(arg1 *io.Jagfile) {
	var2 := io.NewPacket(arg1.Read("frame_head.dat", nil))
	var3 := io.NewPacket(arg1.Read("frame_tran1.dat", nil))
	var4 := io.NewPacket(arg1.Read("frame_tran2.dat", nil))
	var5 := io.NewPacket(arg1.Read("frame_del.dat", nil))
	var6 := var2.G2()
	var7 := var2.G2()
	Instances = make([]*AnimFrame, var7+1)
	var8 := make([]int, 500)
	var9 := make([]int, 500)
	var10 := make([]int, 500)
	var11 := make([]int, 500)
	for range var6 {
		var13 := var2.G2()
		Instances[var13] = NewAnimFrame()
		var14 := Instances[var13]
		var14.Delay = var5.G1()
		var15 := var2.G2()
		var16 := animbase.Instances[var15]
		var14.Base = var16
		var17 := var2.G1()
		var18 := -1
		var19 := 0
		var21 := 0
		for j := range var17 {
			var21 = var3.G1()
			if var21 > 0 {
				if var16.Types[j] != 0 {
					for k := j - 1; k > var18; k-- {
						if var16.Types[k] == 0 {
							var8[var19] = k
							var9[var19] = 0
							var10[var19] = 0
							var11[var19] = 0
							var19++
							break
						}
					}
				}
				var8[var19] = j
				var23 := 0
				if var16.Types[var8[var19]] == 3 {
					var23 = 128
				}
				if var21&0x1 == 0 {
					var9[var19] = var23
				} else {
					var9[var19] = var4.GSmart()
				}
				if var21&0x2 == 0 {
					var10[var19] = var23
				} else {
					var10[var19] = var4.GSmart()
				}
				if var21&0x4 == 0 {
					var11[var19] = var23
				} else {
					var11[var19] = var4.GSmart()
				}
				var18 = j
				var19++
			}
		}
		var14.Length = var19
		var14.Groups = make([]int, var19)
		var14.X = make([]int, var19)
		var14.Y = make([]int, var19)
		var14.Z = make([]int, var19)
		for j := range var19 {
			var14.Groups[j] = var8[j]
			var14.X[j] = var9[j]
			var14.Y[j] = var10[j]
			var14.Z[j] = var11[j]
		}
	}
}

package animbase

import "github.com/zsrv/goscape-client/pkg/jagex2/io"

var Instances []*AnimBase

type AnimBase struct {
	Length int
	Types  []int
	Labels [][]int
}

func NewAnimBase() *AnimBase {
	return &AnimBase{}
}

func Unpack(arg1 *io.Jagfile) {
	var2 := io.NewPacket(arg1.Read("base_head.dat", nil))
	var3 := io.NewPacket(arg1.Read("base_type.dat", nil))
	var4 := io.NewPacket(arg1.Read("base_label.dat", nil))
	var5 := var2.G2()
	var6 := var2.G2()
	Instances = make([]*AnimBase, var6+1)
	for range var5 {
		var8 := var2.G2()
		var9 := var2.G1()
		var10 := make([]int, var9)
		var11 := make([][]int, var9)
		for i := range var9 {
			var10[i] = var3.G1()
			var13 := var4.G1()
			var11[i] = make([]int, var13)
			for j := range var13 {
				var11[i][j] = var4.G1()
			}
		}
		Instances[var8] = NewAnimBase()
		Instances[var8].Length = var9
		Instances[var8].Types = var10
		Instances[var8].Labels = var11
	}
}

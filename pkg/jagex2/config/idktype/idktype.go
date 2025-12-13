package idktype

import (
	"fmt"

	"goscape-client/pkg/jagex2/graphics/model"
	"goscape-client/pkg/jagex2/io"
)

var (
	Count     int
	Instances []*IdkType
)

type IdkType struct {
	Type    int
	Models  []int
	RecolS  []int
	RecolD  []int
	Heads   []int
	Disable bool
}

func NewIdkType() *IdkType {
	return &IdkType{
		Type:    -1,
		RecolS:  make([]int, 6),
		RecolD:  make([]int, 6),
		Heads:   []int{-1, -1, -1, -1, -1},
		Disable: false,
	}
}

func Unpack(arg0 *io.Jagfile) {
	var2 := io.NewPacket(arg0.Read("idk.dat", nil))
	Count = var2.G2()
	if Instances == nil {
		Instances = make([]*IdkType, Count)
	}
	for i := range Count {
		if Instances[i] == nil {
			Instances[i] = NewIdkType()
		}
		Instances[i].Decode(var2)
	}
}

func (idk *IdkType) Decode(arg1 *io.Packet) {
	for {
		var3 := arg1.G1()
		switch var3 {
		case 0:
			return
		case 1:
			idk.Type = arg1.G1()
		case 2:
			var4 := arg1.G1()
			idk.Models = make([]int, var4)
			for i := range var4 {
				idk.Models[i] = arg1.G2()
			}
		case 3:
			idk.Disable = true
		case 40, 41, 42, 43, 44, 45, 46, 47, 48, 49:
			idk.RecolS[var3-40] = arg1.G2()
		case 50, 51, 52, 53, 54, 55, 56, 57, 58, 59:
			idk.RecolD[var3-50] = arg1.G2()
		case 60, 61, 62, 63, 64, 65, 66, 67, 68, 69:
			idk.Heads[var3-60] = arg1.G2()
		default:
			fmt.Println("Error unrecognised config code:", var3)
		}
	}
}

func (idk *IdkType) GetModel() *model.Model {
	if idk.Models == nil {
		return nil
	}
	var1 := make([]*model.Model, len(idk.Models))
	for i := range len(idk.Models) {
		var1[i] = model.NewModel1(idk.Models[i])
	}
	var var3 *model.Model
	if len(var1) == 1 {
		var3 = var1[0]
	} else {
		var3 = model.NewModel2(var1, len(var1))
	}
	for i := 0; i < 6 && idk.RecolS[i] != 0; i++ {
		var3.Recolor(idk.RecolS[i], idk.RecolD[i])
	}
	return var3
}

func (idk *IdkType) GetHeadModel() *model.Model {
	var2 := make([]*model.Model, 5)
	var3 := 0
	for i := range 5 {
		if idk.Heads[i] != -1 {
			var2[var3] = model.NewModel1(idk.Heads[i])
			var3++
		}
	}
	var5 := model.NewModel2(var2, var3)
	for i := 0; i < 6 && idk.RecolS[i] != 0; i++ {
		var5.Recolor(idk.RecolS[i], idk.RecolD[i])
	}
	return var5
}

package varptype

import (
	"fmt"

	"goscape-client/pkg/jagex2/io"
)

var (
	Count      int
	Instances  []*VarpType
	Code3Count int
	Code3      []int
)

type VarpType struct {
	Code10     string
	Code1      int
	Code2      int
	HasCode3   bool
	Code4      bool
	ClientCode int
	Code6      bool
	Code7      int
	Code8      bool
}

func NewVarpType() *VarpType {
	return &VarpType{
		HasCode3: false,
		Code4:    true,
		Code6:    false,
		Code8:    false,
	}
}

func Unpack(arg0 *io.Jagfile) {
	var2 := io.NewPacket(arg0.Read("varp.dat", nil))
	Code3Count = 0
	Count = var2.G2()
	if Instances == nil {
		Instances = make([]*VarpType, Count)
	}
	if Code3 == nil {
		Code3 = make([]int, Count)
	}
	for i := range Count {
		if Instances[i] == nil {
			Instances[i] = NewVarpType()
		}
		Instances[i].Decode(i, var2)
	}
}

func (t *VarpType) Decode(arg1 int, arg2 *io.Packet) {
	for {
		var4 := arg2.G1()
		switch var4 {
		case 0:
			return
		case 1:
			t.Code1 = arg2.G1()
		case 2:
			t.Code2 = arg2.G1()
		case 3:
			t.HasCode3 = true
			Code3[Code3Count] = arg1
			Code3Count++
		case 4:
			t.Code4 = false
		case 5:
			t.ClientCode = arg2.G2()
		case 6:
			t.Code6 = true
		case 7:
			t.Code7 = arg2.G4()
		case 8:
			t.Code8 = true
		case 10:
			t.Code10 = arg2.GJStr()
		default:
			fmt.Println("Error unrecognised config code:", var4)
		}
	}
}

package spotanimtype

import (
	"fmt"

	"goscape-client/pkg/jagex2/config/seqtype"
	"goscape-client/pkg/jagex2/datastruct"
	"goscape-client/pkg/jagex2/graphics/model"
	"goscape-client/pkg/jagex2/io"
)

var (
	Count      int
	Instances  []*SpotAnimType
	ModelCache = datastruct.NewLruCache[*model.Model](30)
)

type SpotAnimType struct {
	Index        int
	Model        int
	Anim         int
	Seq          *seqtype.SeqType
	AnimHasAlpha bool
	RecolS       []int
	RecolD       []int
	ResizeH      int
	ResizeV      int
	Orientation  int
	Ambient      int
	Contrast     int
}

func NewSpotAnimType() *SpotAnimType {
	return &SpotAnimType{
		Anim:         -1,
		AnimHasAlpha: false,
		RecolS:       make([]int, 6),
		RecolD:       make([]int, 6),
		ResizeH:      128,
		ResizeV:      128,
	}
}

func Unpack(arg0 *io.Jagfile) {
	var2 := io.NewPacket(arg0.Read("spotanim.dat", nil))
	Count = var2.G2()
	if Instances == nil {
		Instances = make([]*SpotAnimType, Count)
	}
	for i := range Count {
		if Instances[i] == nil {
			Instances[i] = NewSpotAnimType()
		}
		Instances[i].Index = i
		Instances[i].Decode(var2)
	}
}

func (t *SpotAnimType) Decode(arg1 *io.Packet) {
	for {
		var3 := arg1.G1()
		switch var3 {
		case 0:
			return
		case 1:
			t.Model = arg1.G2()
		case 2:
			t.Anim = arg1.G2()
			if seqtype.Instances != nil {
				t.Seq = seqtype.Instances[t.Anim]
			}
		case 3:
			t.AnimHasAlpha = true
		case 4:
			t.ResizeH = arg1.G2()
		case 5:
			t.ResizeV = arg1.G2()
		case 6:
			t.Orientation = arg1.G2()
		case 7:
			t.Ambient = arg1.G1()
		case 8:
			t.Contrast = arg1.G1()
		case 40, 41, 42, 43, 44, 45, 46, 47, 48, 49:
			t.RecolS[var3-40] = arg1.G2()
		case 50, 51, 52, 53, 54, 55, 56, 57, 58, 59:
			t.RecolD[var3-50] = arg1.G2()
		default:
			fmt.Println("Error unrecognised spotanim config code:", var3)
		}
	}
}

func (t *SpotAnimType) GetModel() *model.Model {
	var1 := ModelCache.Get(int64(t.Index)).Value // TODO: .Value would cause panic
	if var1 != nil {
		return var1
	}
	var1 = model.NewModel1(t.Model)
	for i := range 6 {
		if t.RecolS[0] != 0 {
			var1.Recolor(t.RecolS[i], t.RecolD[i])
		}
	}
	//ModelCache.Put(int64(t.Index), var1) // TODO
	return var1
}

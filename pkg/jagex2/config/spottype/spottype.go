package spottype

import (
	"fmt"

	"github.com/zsrv/goscape-client/pkg/jagex2/config/seqtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

var (
	Count      int
	List       []*SpotType
	ModelCache = datastruct.NewLruCache[*model.Model](30)
)

// 254 deletes the animHasAlpha field + decode opcode 3 — alpha sharing is
// now derived per-frame via animframe.ShareAlpha (WS3).
type SpotType struct {
	Index       int
	Model       int
	Anim        int
	Seq         *seqtype.SeqType
	RecolS      []int
	RecolD      []int
	ResizeH     int
	ResizeV     int
	Orientation int
	Ambient     int
	Contrast    int
}

func NewSpotType() *SpotType {
	return &SpotType{
		Anim:    -1,
		RecolS:  make([]int, 6),
		RecolD:  make([]int, 6),
		ResizeH: 128,
		ResizeV: 128,
	}
}

func Init(arg0 *io.JagFile) {
	var2 := io.NewPacket(arg0.Read("spotanim.dat", nil))
	Count = var2.G2()
	if List == nil {
		List = make([]*SpotType, Count)
	}
	for i := range Count {
		if List[i] == nil {
			List[i] = NewSpotType()
		}
		List[i].Index = i
		List[i].Decode(var2)
	}
}

func (t *SpotType) Decode(arg1 *io.Packet) {
	for {
		var3 := arg1.G1()
		switch var3 {
		case 0:
			return
		case 1:
			t.Model = arg1.G2()
		case 2:
			t.Anim = arg1.G2()
			if seqtype.List != nil {
				t.Seq = seqtype.List[t.Anim]
			}
		// Java 254 drops opcode 3 (animHasAlpha) — an opcode-3 byte in the
		// data now falls through to the error println, as in Java.
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

// GetTempModel returns the (cached) recoloured spotanim model.
// Java: getTempModel (SpotAnimType.java:103-121 @2e62978; SpotType.java @32f3062; was getModel at
// 245.2).
func (t *SpotType) GetTempModel() *model.Model {
	var1 := ModelCache.Find(int64(t.Index))
	if var1 != nil {
		return var1
	}
	var1 = model.Load(t.Model)
	if var1 == nil {
		return nil
	}
	for i := range 6 {
		if t.RecolS[0] != 0 {
			var1.Recolor(t.RecolS[i], t.RecolD[i])
		}
	}
	ModelCache.Put(int64(t.Index), var1)
	return var1
}

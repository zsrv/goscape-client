package seqtype

import (
	"fmt"

	"goscape-client/pkg/jagex2/graphics/animframe"
	"goscape-client/pkg/jagex2/io"
)

var (
	Count     int
	Instances []*SeqType
)

type SeqType struct {
	FrameCount  int
	Frames      []int
	IFrames     []int
	Delay       []int
	ReplayOff   int
	WalkMerge   []int
	Stretches   bool
	Priority    int
	RightHand   int
	LeftHand    int
	ReplayCount int
}

func NewSeqType() *SeqType {
	return &SeqType{
		ReplayOff:   -1,
		Stretches:   false,
		Priority:    5,
		RightHand:   -1,
		LeftHand:    -1,
		ReplayCount: 99,
	}
}

func Unpack(arg0 *io.Jagfile) {
	var2 := io.NewPacket(arg0.Read("seq.dat", nil))
	Count = var2.G2()
	if Instances == nil {
		Instances = make([]*SeqType, Count)
	}
	for i := range Count {
		if Instances[i] == nil {
			Instances[i] = NewSeqType()
		}
		Instances[i].Decode(var2)
	}
}

func (t *SeqType) Decode(arg1 *io.Packet) {
	for {
		var3 := arg1.G1()
		switch var3 {
		case 0:
			if t.FrameCount == 0 {
				t.FrameCount = 1
				t.Frames = make([]int, 1)
				t.Frames[0] = -1
				t.IFrames = make([]int, 1)
				t.IFrames[0] = -1
				t.Delay = make([]int, 1)
				t.Delay[0] = -1
				return
			}
			return
		case 1:
			t.FrameCount = arg1.G1()
			t.Frames = make([]int, t.FrameCount)
			t.IFrames = make([]int, t.FrameCount)
			t.Delay = make([]int, t.FrameCount)
			for i := range t.FrameCount {
				t.Frames[i] = arg1.G2()
				t.IFrames[i] = arg1.G2()
				if t.IFrames[i] == 65535 {
					t.IFrames[i] = -1
				}
				t.Delay[i] = arg1.G2()
				if t.Delay[i] == 0 {
					t.Delay[i] = animframe.Instances[t.Frames[i]].Delay
				}
				if t.Delay[i] == 0 {
					t.Delay[i] = 1
				}
			}
		case 2:
			t.ReplayOff = arg1.G2()
		case 3:
			var4 := arg1.G1()
			t.WalkMerge = make([]int, var4+1)
			for i := range var4 {
				t.WalkMerge[i] = arg1.G1()
			}
			t.WalkMerge[var4] = 9999999
		case 4:
			t.Stretches = true
		case 5:
			t.Priority = arg1.G1()
		case 6:
			t.RightHand = arg1.G2()
		case 7:
			t.LeftHand = arg1.G2()
		case 8:
			t.ReplayCount = arg1.G1()
		default:
			fmt.Println("Error unrecognised seq config code:", var3)
		}
	}
}

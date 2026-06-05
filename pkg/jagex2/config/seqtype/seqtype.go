package seqtype

import (
	"fmt"

	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/animframe"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

var (
	Count int
	List  []*SeqType
)

type SeqType struct {
	NumFrames   int
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
	// Java: SeqType preanim_move/postanim_mode/duplicatebehavior (rev-244
	// opcodes 9/10/11). ReplayCount above is rev-244's maxloops (opcode 8,
	// same g1 read, default 99) under its rev-225 name.
	PreanimMove       int
	PostanimMode      int
	DuplicateBehavior int
}

// GetDelay returns the duration (in client cycles) of animation frame
// `frame`, lazily resolving a zero wire delay from the frame's AnimFrame
// (caching into Delay[]) and falling back to 1. The AnimFrame may not be
// loaded yet (frames arrive over OnDemand after seq.dat decodes), so the nil
// guard is load-bearing. Java 274: getDelay (SeqType.java:75 @32f3062; was
// getDuration at 254, getFrameDuration at 245.2).
func (t *SeqType) GetDelay(frame int) int {
	duration := t.Delay[frame]
	if duration == 0 {
		transform := animframe.Get(t.Frames[frame])
		if transform != nil {
			duration = transform.Delay
			t.Delay[frame] = duration
		}
	}
	if duration == 0 {
		duration = 1
	}
	return duration
}

func NewSeqType() *SeqType {
	return &SeqType{
		ReplayOff:   -1,
		Stretches:   false,
		Priority:    5,
		RightHand:   -1,
		LeftHand:    -1,
		ReplayCount: 99,
		// Java: SeqType field initializers (rev-244): preanim_move = -1,
		// postanim_mode = -1, duplicatebehavior = 0.
		PreanimMove:  -1,
		PostanimMode: -1,
	}
}

func Init(arg0 *io.JagFile) {
	var2 := io.NewPacket(arg0.Read("seq.dat", nil))
	Count = var2.G2()
	if List == nil {
		List = make([]*SeqType, Count)
	}
	for i := range Count {
		if List[i] == nil {
			List[i] = NewSeqType()
		}
		List[i].Decode(var2)
	}
}

func (t *SeqType) Decode(arg1 *io.Packet) {
	for {
		var3 := arg1.G1()
		switch var3 {
		case 0:
			if t.NumFrames == 0 {
				t.NumFrames = 1
				t.Frames = make([]int, 1)
				t.Frames[0] = -1
				t.IFrames = make([]int, 1)
				t.IFrames[0] = -1
				t.Delay = make([]int, 1)
				t.Delay[0] = -1
			}
			// Java: SeqType.decode post-loop defaulting (rev-244). preanim_move
			// and postanim_mode default by whether walkmerge is present.
			if t.PreanimMove == -1 {
				if t.WalkMerge == nil {
					t.PreanimMove = 0
				} else {
					t.PreanimMove = 2
				}
			}
			if t.PostanimMode == -1 {
				if t.WalkMerge == nil {
					t.PostanimMode = 0
				} else {
					t.PostanimMode = 2
				}
			}
			return
		case 1:
			t.NumFrames = arg1.G1()
			t.Frames = make([]int, t.NumFrames)
			t.IFrames = make([]int, t.NumFrames)
			t.Delay = make([]int, t.NumFrames)
			for i := range t.NumFrames {
				t.Frames[i] = arg1.G2()
				t.IFrames[i] = arg1.G2()
				if t.IFrames[i] == 0xFFFF {
					t.IFrames[i] = -1
				}
				// Java: rev-244 decode stops at the raw g2 read; the delay==0
				// fallback lives in the lazy getFrameDuration (AnimFrames are
				// NOT loaded yet at decode time — they arrive over OnDemand).
				t.Delay[i] = arg1.G2()
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
			t.ReplayCount = arg1.G1() // Java: maxloops (rev-244)
		// Java: SeqType.decode opcodes 9/10/11 (rev-244).
		case 9:
			t.PreanimMove = arg1.G1()
		case 10:
			t.PostanimMode = arg1.G1()
		case 11:
			t.DuplicateBehavior = arg1.G1()
		default:
			fmt.Println("Error unrecognised seq config code:", var3)
		}
	}
}

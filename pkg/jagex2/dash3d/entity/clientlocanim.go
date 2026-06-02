package entity

import (
	"math/rand"

	"github.com/zsrv/goscape-client/pkg/jagex2/config/seqtype"
)

type ClientLocAnim struct {
	Level    int
	Type     int
	X        int
	Z        int
	Index    int
	Seq      *seqtype.SeqType
	SeqFrame int
	SeqCycle int
}

func NewClientLocAnim(arg0 bool, index int, level int, typ int, seq *seqtype.SeqType, z int, x int) *ClientLocAnim {
	var e ClientLocAnim
	e.Level = level
	e.Type = typ
	e.X = x
	e.Z = z
	e.Index = index
	e.Seq = seq
	if arg0 && seq.ReplayOff != -1 {
		e.SeqFrame = int(rand.Float64() * float64(e.Seq.FrameCount))
		e.SeqCycle = int(rand.Float64() * float64(e.Seq.Delay[e.SeqFrame]))
	} else {
		e.SeqFrame = -1
		e.SeqCycle = 0
	}
	return &e
}

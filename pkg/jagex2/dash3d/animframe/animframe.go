package animframe

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/animbase"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
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

// Init allocates the per-id frame table. Java: AnimFrame.init(int).
func Init(capacity int) {
	Instances = make([]*AnimFrame, capacity+1)
}

// Get returns the frame for id, or nil when the table or the frame itself has
// not been loaded yet (frames arrive lazily over OnDemand archive 1).
// Java: AnimFrame.get (AnimFrame.java:153-159).
func Get(id int) *AnimFrame {
	if Instances == nil {
		return nil
	}
	return Instances[id]
}

// Unpack decodes a single per-id animation blob (rev-244). The 8-byte trailer
// gives the head/tran1/tran2/del section lengths; the AnimBase is embedded at
// the tail of the blob (there is no per-frame base-id lookup as in 225).
// Java: AnimFrame.unpack(byte[]).
func Unpack(data []byte) {
	buf := io.NewPacket(data)
	buf.Pos = len(data) - 8

	headLength := buf.G2()
	tran1Length := buf.G2()
	tran2Length := buf.G2()
	delLength := buf.G2()
	pos := 0

	head := io.NewPacket(data)
	head.Pos = pos
	pos += headLength + 2

	tran1 := io.NewPacket(data)
	tran1.Pos = pos
	pos += tran1Length

	tran2 := io.NewPacket(data)
	tran2.Pos = pos
	pos += tran2Length

	del := io.NewPacket(data)
	del.Pos = pos
	pos += delLength

	baseBuf := io.NewPacket(data)
	baseBuf.Pos = pos
	base := animbase.NewAnimBase(baseBuf)

	total := head.G2()
	tempTi := make([]int, 500)
	tempTx := make([]int, 500)
	tempTy := make([]int, 500)
	tempTz := make([]int, 500)

	for range total {
		id := head.G2()

		frame := NewAnimFrame()
		Instances[id] = frame
		frame.Delay = del.G1()
		frame.Base = base

		groupCount := head.G1()
		lastGroup := -1
		current := 0

		for j := range groupCount {
			flags := tran1.G1()
			if flags > 0 {
				if base.Types[j] != 0 {
					for group := j - 1; group > lastGroup; group-- {
						if base.Types[group] == 0 {
							tempTi[current] = group
							tempTx[current] = 0
							tempTy[current] = 0
							tempTz[current] = 0
							current++
							break
						}
					}
				}

				tempTi[current] = j

				defaultValue := 0
				if base.Types[tempTi[current]] == 3 {
					defaultValue = 128
				}

				if flags&0x1 == 0 {
					tempTx[current] = defaultValue
				} else {
					tempTx[current] = tran2.GSmart()
				}

				if flags&0x2 == 0 {
					tempTy[current] = defaultValue
				} else {
					tempTy[current] = tran2.GSmart()
				}

				if flags&0x4 == 0 {
					tempTz[current] = defaultValue
				} else {
					tempTz[current] = tran2.GSmart()
				}

				lastGroup = j
				current++
			}
		}

		frame.Length = current
		frame.Groups = make([]int, current)
		frame.X = make([]int, current)
		frame.Y = make([]int, current)
		frame.Z = make([]int, current)

		for j := range current {
			frame.Groups[j] = tempTi[j]
			frame.X[j] = tempTx[j]
			frame.Y[j] = tempTy[j]
			frame.Z[j] = tempTz[j]
		}
	}
}

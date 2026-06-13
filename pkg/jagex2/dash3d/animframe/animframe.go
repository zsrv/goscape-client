package animframe

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/animbase"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

var List []*AnimFrame

// Java: AnimFrame.opaque (h.j, AnimFrame.java:34 @2e62978) — new static
// boolean[] at 254. Intentionally not ported: it is a dead-write deob
// artifact (init() fills it all-true, unpack() clears entries on
// base-type-5 transforms, unload() never clears it) with zero readers
// anywhere at 2e62978 (re-verified by tree-wide grep 2026-06-04; the
// only other `opaque` is the unrelated Pix3D.opaque). Project policy
// excludes pure deob artifacts.

type AnimFrame struct {
	Delay int
	Base  *animbase.AnimBase
	Size  int   // Java: size
	Ti    []int // Java: ti
	Tx    []int // Java: tx
	Ty    []int // Java: ty
	Tz    []int // Java: tz
}

func NewAnimFrame() *AnimFrame {
	return &AnimFrame{}
}

// Init allocates the per-id frame table. Java: AnimFrame.init(int).
func Init(capacity int) {
	List = make([]*AnimFrame, capacity+1)
}

// Get returns the frame for id, or nil when the table or the frame itself has
// not been loaded yet (frames arrive lazily over OnDemand archive 1).
// Java: AnimFrame.get (AnimFrame.java:153-159).
func Get(id int) *AnimFrame {
	if List == nil {
		return nil
	}
	return List[id]
}

// ShareAlpha reports whether face alpha may be shared (not copied) when
// deriving a transformed model for the given frame id: only when no frame
// is applied (id == -1), since an applied animation may modify alpha.
// Replaces the per-config animHasAlpha flag wholesale at 254.
// Java: AnimFrame.shareAlpha (h.a(BI)Z, AnimFrame.java:144-147 @2e62978).
func ShareAlpha(id int) bool {
	return id == -1
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
		List[id] = frame
		frame.Delay = del.G1()
		frame.Base = base

		groupCount := head.G1()
		lastGroup := -1
		current := 0

		for j := range groupCount {
			flags := tran1.G1()
			if flags > 0 {
				if base.Type[j] != 0 {
					for group := j - 1; group > lastGroup; group-- {
						if base.Type[group] == 0 {
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
				if base.Type[tempTi[current]] == 3 {
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
				// Java 254 clears the dead-write opaque[id] here when
				// base.types[j] == 5 (AnimFrame.java:119-121 @2e62978).
				// Intentionally not ported — see the opaque note above.
			}
		}

		frame.Size = current
		frame.Ti = make([]int, current)
		frame.Tx = make([]int, current)
		frame.Ty = make([]int, current)
		frame.Tz = make([]int, current)

		for j := range current {
			frame.Ti[j] = tempTi[j]
			frame.Tx[j] = tempTx[j]
			frame.Ty[j] = tempTy[j]
			frame.Tz[j] = tempTz[j]
		}
	}
}

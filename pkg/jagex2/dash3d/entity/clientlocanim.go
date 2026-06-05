package entity

import (
	"math/rand"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/loctype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/seqtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
)

// ClientLocAnim is a self-animating scene loc. Java: rev-244 ClientLocAnim
// extends ModelSource. It is stored directly as a scene node's ModelSource
// (Wall.Model{A,B}, Sprite.Model, Decor.Model, GroundDecor.Model); each draw,
// GetTempModel advances the seq frame against Client's global loop cycle and
// returns the current-frame loc model. This replaces the rev-225 LocEntity-list
// + Client.PushLocs per-frame model-swap mechanism.
type ClientLocAnim struct {
	Index       int
	Shape       int
	Angle       int
	HeightmapSW int
	HeightmapSE int
	HeightmapNE int
	HeightmapNW int
	Seq         *seqtype.SeqType
	SeqFrame    int
	SeqCycle    int
}

// NewClientLocAnim mirrors rev-244 ClientLocAnim(int heightmapNW, int
// heightmapNE, int heightmapSW, int shape, int angle, boolean randomFrame, int
// heightmapSE, int index, int seq).
//
// Heightmap naming: the four corner heights are, by world position,
// SW=[x][z], SE=[x+1][z], NE=[x+1][z+1], NW=[x][z+1]. The rev-244 source calls
// this ctor with a permuted arg order whose labels (heightNE/heightNW/...) are
// swapped relative to Go's; callers here pass by the matching value, so a loc
// caller writes NewClientLocAnim(heightNW, heightNE, heightSW, ..., heightSE,
// ...). The stored fields then feed GetTempModel in the same positional order
// the static loc.GetModel call uses, keeping the two paths identical.
func NewClientLocAnim(heightmapNW, heightmapNE, heightmapSW, shape, angle int, randomFrame bool, heightmapSE, index, seq int) *ClientLocAnim {
	e := &ClientLocAnim{
		Index:       index,
		Shape:       shape,
		Angle:       angle,
		HeightmapSW: heightmapSW,
		HeightmapSE: heightmapSE,
		HeightmapNE: heightmapNE,
		HeightmapNW: heightmapNW,
		Seq:         seqtype.Instances[seq],
		SeqFrame:    0,
		SeqCycle:    clientextras.LoopCycle,
	}
	// Java: seq.loops (rev-225 name: ReplayOff). numFrames -> FrameCount.
	if randomFrame && e.Seq.ReplayOff != -1 {
		e.SeqFrame = int(rand.Float64() * float64(e.Seq.FrameCount))
		e.SeqCycle -= int(rand.Float64() * float64(e.Seq.GetFrameDuration(e.SeqFrame)))
	}
	return e
}

// GetTempModel advances the animation against Client.loopCycle and returns
// the current-frame model. Java: ClientLocAnim.getTempModel @2e62978 (was
// getModel in ≤245.2).
func (e *ClientLocAnim) GetTempModel() *model.Model {
	if e.Seq != nil {
		delta := clientextras.LoopCycle - e.SeqCycle
		if delta > 100 && e.Seq.ReplayOff > 0 {
			delta = 100
		}

		for delta > e.Seq.GetFrameDuration(e.SeqFrame) {
			delta -= e.Seq.GetFrameDuration(e.SeqFrame)
			e.SeqFrame++

			if e.SeqFrame < e.Seq.FrameCount {
				continue
			}

			e.SeqFrame -= e.Seq.ReplayOff

			if e.SeqFrame < 0 || e.SeqFrame >= e.Seq.FrameCount {
				e.Seq = nil
				break
			}
		}

		e.SeqCycle = clientextras.LoopCycle - delta
	}

	transformId := -1
	if e.Seq != nil {
		transformId = e.Seq.Frames[e.SeqFrame]
	}

	return loctype.Get(e.Index).GetModel(e.Shape, e.Angle, e.HeightmapSW, e.HeightmapSE, e.HeightmapNE, e.HeightmapNW, transformId)
}

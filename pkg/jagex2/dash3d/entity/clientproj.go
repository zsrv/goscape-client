package entity

import (
	"math"

	"github.com/zsrv/goscape-client/pkg/jagex2/config/spotanimtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/animframe"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
)

// ClientProj
type ClientProj struct {
	SpotAnim      *spotanimtype.SpotAnimType
	Level         int
	SrcX          int
	SrcZ          int
	SrcY          int
	OffsetY       int
	StartCycle    int
	LastCycle     int
	PeakPitch     int
	Arc           int
	Target        int
	Mobile        bool
	X             float64
	Z             float64
	Y             float64
	VelocityX     float64
	VelocityZ     float64
	Velocity      float64
	VelocityY     float64
	AccelerationY float64
	Yaw           int
	Pitch         int
	SeqFrame      int
	SeqCycle      int
}

func NewClientProj(offsetY, peakPitch, srcZ, lastCycle, level, target, startCycle, arc, srcY, arg10, srcX int) *ClientProj {
	return &ClientProj{
		SpotAnim:   spotanimtype.List[arg10],
		Level:      level,
		SrcX:       srcX,
		SrcZ:       srcZ,
		SrcY:       srcY,
		StartCycle: startCycle,
		LastCycle:  lastCycle,
		PeakPitch:  peakPitch,
		Arc:        arc,
		Target:     target,
		OffsetY:    offsetY,
		Mobile:     false,
	}
}

func (e *ClientProj) UpdateVelocity(arg0, arg1, arg2, arg4 int) {
	if !e.Mobile {
		// Java: double var6 = arg2 - srcX — a double, so var6*var6 is a double
		// multiply. Kept as float64 to match Java's arithmetic exactly.
		var6 := float64(arg2 - e.SrcX)
		var8 := float64(arg1 - e.SrcZ)
		var10 := math.Sqrt(var6*var6 + var8*var8)
		e.X = float64(e.SrcX) + var6*float64(e.Arc)/var10
		e.Z = float64(e.SrcZ) + var8*float64(e.Arc)/var10
		e.Y = float64(e.SrcY)
	}
	// Java: double var6 = lastCycle + 1 - arg4 (double, so var6*var6 below is a double multiply).
	var6 := float64(e.LastCycle + 1 - arg4)
	e.VelocityX = (float64(arg2) - e.X) / var6
	e.VelocityZ = (float64(arg1) - e.Z) / var6
	e.Velocity = math.Sqrt(e.VelocityX*e.VelocityX + e.VelocityZ*e.VelocityZ)
	if !e.Mobile {
		e.VelocityY = -e.Velocity * math.Tan(float64(e.PeakPitch)*0.02454369)
	}
	e.AccelerationY = (float64(arg0) - e.Y - e.VelocityY*var6) * 2.0 / (var6 * var6)
}

func (e *ClientProj) Update(arg1 int) {
	e.Mobile = true
	e.X += e.VelocityX * float64(arg1)
	e.Z += e.VelocityZ * float64(arg1)
	e.Y += e.VelocityY*float64(arg1) + e.AccelerationY*0.5*float64(arg1)*float64(arg1)
	e.VelocityY += e.AccelerationY * float64(arg1)
	e.Yaw = (int(math.Atan2(e.VelocityX, e.VelocityZ)*325.949) + 0x400) & 0x7FF
	e.Pitch = int(math.Atan2(e.VelocityY, e.Velocity)*325.949) & 0x7FF
	if e.SpotAnim.Seq == nil {
		return
	}
	e.SeqCycle += arg1
	for e.SeqCycle > e.SpotAnim.Seq.GetDuration(e.SeqFrame) {
		e.SeqCycle -= e.SpotAnim.Seq.GetDuration(e.SeqFrame) + 1
		e.SeqFrame++
		if e.SeqFrame >= e.SpotAnim.Seq.NumFrames {
			e.SeqFrame = 0
		}
	}
}

// Java: getTempModel (ClientProj.java:138-160 @2e62978; was getModel) — 254
// computes the resolved frame id up front (-1 when there is no seq) and
// gates prepareAnim/animate on it instead of on seq != null.
func (e *ClientProj) GetTempModel() *model.Model {
	var2 := e.SpotAnim.GetTempModel()
	// Java: ClientProj.java:140-142 @2e62978 — nil while the spotanim model
	// faults in.
	if var2 == nil {
		return nil
	}
	var3 := -1 // Java: var3
	if e.SpotAnim.Seq != nil {
		var3 = e.SpotAnim.Seq.Frames[e.SeqFrame]
	}
	var4 := model.NewModel4(var2, true, animframe.ShareAlpha(var3), false)
	if var3 != -1 {
		var4.PrepareAnim()
		var4.Animate(var3)
		var4.LabelFaces = nil
		var4.LabelVertices = nil
	}
	if e.SpotAnim.ResizeH != 128 || e.SpotAnim.ResizeV != 128 {
		var4.Scale(e.SpotAnim.ResizeH, e.SpotAnim.ResizeV, e.SpotAnim.ResizeH)
	}
	var4.RotateXAxis(e.Pitch)
	var4.CalculateNormals(e.SpotAnim.Ambient+64, e.SpotAnim.Contrast+850, -30, -50, -30, true)
	return var4
}

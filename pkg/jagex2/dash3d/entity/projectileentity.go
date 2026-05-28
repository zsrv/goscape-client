package entity

import (
	"math"

	"github.com/zsrv/goscape-client/pkg/jagex2/config/spotanimtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/model"
)

// ClientProj
type ProjectileEntity struct {
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

func NewProjectileEntity(offsetY, peakPitch, srcZ, lastCycle, level, target, startCycle, arc, srcY, arg10, srcX int) *ProjectileEntity {
	return &ProjectileEntity{
		SpotAnim:   spotanimtype.Instances[arg10],
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

func (e *ProjectileEntity) UpdateVelocity(arg0, arg1, arg2, arg4 int) {
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

func (e *ProjectileEntity) Update(arg1 int) {
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
	for e.SeqCycle > e.SpotAnim.Seq.Delay[e.SeqFrame] {
		e.SeqCycle -= e.SpotAnim.Seq.Delay[e.SeqFrame] + 1
		e.SeqFrame++
		if e.SeqFrame >= e.SpotAnim.Seq.FrameCount {
			e.SeqFrame = 0
		}
	}
}

func (e *ProjectileEntity) Draw() *model.Model {
	var2 := e.SpotAnim.GetModel()
	var3 := model.NewModel4(var2, true, !e.SpotAnim.AnimHasAlpha, false)
	if e.SpotAnim.Seq != nil {
		var3.CreateLabelReferences()
		var3.ApplyTransform(e.SpotAnim.Seq.Frames[e.SeqFrame])
		var3.LabelFaces = nil
		var3.LabelVertices = nil
	}
	if e.SpotAnim.ResizeH != 128 || e.SpotAnim.ResizeV != 128 {
		var3.Scale(e.SpotAnim.ResizeH, e.SpotAnim.ResizeV, e.SpotAnim.ResizeH)
	}
	var3.RotateX(e.Pitch)
	var3.CalculateNormals(e.SpotAnim.Ambient+64, e.SpotAnim.Contrast+850, -30, -50, -30, true)
	return var3
}

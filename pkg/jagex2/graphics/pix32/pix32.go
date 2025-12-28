package pix32

import (
	"math"

	"goscape-client/pkg/jagex2/graphics/pix2d"
	"goscape-client/pkg/jagex2/graphics/pix8"
	"goscape-client/pkg/jagex2/io"
)

type Pix32 struct {
	Pixels []int
	CropW  int
	Width  int
	CropH  int
	Height int
	CropY  int
	CropX  int
}

func NewPix321(width int, height int) *Pix32 {
	var p Pix32
	p.Pixels = make([]int, width*height)
	p.CropW = width
	p.Width = p.CropW
	p.CropH = height
	p.Height = p.CropH
	p.CropY = 0
	p.CropX = 0
	return &p
}

//func NewPix322(arg0 []byte, arg1 Component) *Pix32 {} // TODO: java.awt.Component

func NewPix323(jag *io.Jagfile, name string, sprite int) *Pix32 {
	var p Pix32

	dat := io.NewPacket(jag.Read(name+".dat", nil))
	idx := io.NewPacket(jag.Read("index.dat", nil))
	idx.Pos = dat.G2()
	p.CropW = idx.G2()
	p.CropH = idx.G2()
	palCount := idx.G1()
	bPal := make([]int, palCount) // base palette
	for i := range palCount - 1 {
		bPal[i+1] = idx.G3()
		if bPal[i+1] == 0 {
			bPal[i+1] = 1
		}
	}
	for range sprite {
		idx.Pos += 2
		dat.Pos += idx.G2() * idx.G2()
		idx.Pos++
	}
	p.CropX = idx.G1()
	p.CropY = idx.G1()
	p.Width = idx.G2()
	p.Height = idx.G2()
	pixelOrder := idx.G1()
	length := p.Width * p.Height
	p.Pixels = make([]int, length)
	switch pixelOrder {
	case 0:
		for i := range length {
			p.Pixels[i] = bPal[dat.G1()]
		}
	case 1:
		for x := range p.Width {
			for y := range p.Height {
				p.Pixels[x+y*p.Width] = bPal[dat.G1()]
			}
		}
	}
	return &p
}

func (p *Pix32) Bind() {
	pix2d.Bind(p.Width, p.Pixels, p.Height)
}

func (p *Pix32) Translate(arg0, arg1, arg2 int) { // RGBAdjust
	for i := range len(p.Pixels) {
		var6 := p.Pixels[i]
		if var6 != 0 {
			var7 := (var6 >> 16) & 0xFF
			var7 += arg0
			if var7 < 1 {
				var7 = 1
			} else if var7 > 255 {
				var7 = 255
			}
			var8 := (var6 >> 8) & 0xFF
			var8 += arg1
			if var8 < 1 {
				var8 = 1
			} else if var8 > 255 {
				var8 = 255
			}
			var9 := var6 & 0xFF
			var9 += arg2
			if var9 < 1 {
				var9 = 1
			} else if var9 > 255 {
				var9 = 255
			}
			p.Pixels[i] = (var7 << 16) + (var8 << 8) + var9
		}
	}
}

// QuickPlotSprite
func (p *Pix32) BlitOpaque(arg1, arg2 int) {
	arg1 += p.CropX
	arg2 += p.CropY
	var4 := arg1 + arg2*pix2d.Width2D
	var5 := 0
	var6 := p.Height
	var7 := p.Width
	var8 := pix2d.Width2D - var7
	var9 := 0
	if arg2 < pix2d.BoundTop {
		var10 := pix2d.BoundTop - arg2
		var6 -= var10
		arg2 = pix2d.BoundTop
		var5 += var10 * var7
		var4 += var10 * pix2d.Width2D
	}
	if arg2+var6 > pix2d.BoundBottom {
		var6 -= arg2 + var6 - pix2d.BoundBottom
	}
	if arg1 < pix2d.BoundLeft {
		var10 := pix2d.BoundLeft - arg1
		var7 -= var10
		arg1 = pix2d.BoundLeft
		var5 += var10
		var4 += var10
		var9 += var10
		var8 += var10
	}
	if arg1+var7 > pix2d.BoundRight {
		var10 := arg1 + var7 - pix2d.BoundRight
		var7 -= var10
		var9 += var10
		var8 += var10
	}
	if var7 > 0 && var6 > 0 {
		p.CopyPixels1(p.Pixels, var8, var6, var5, var9, var4, var7, pix2d.Data)
	}
}

// QuickPlot
func (p *Pix32) CopyPixels1(arg1 []int, arg2 int, arg3 int, arg4 int, arg5 int, arg6 int, arg7 int, arg8 []int) {
	var10 := -(arg7 >> 2)
	var14 := -(arg7 & 0x3)
	for i := -arg3; i < 0; i++ {
		for j := var10; j < 0; j++ {
			arg8[arg6] = arg1[arg4]
			arg6++
			arg4++
			arg8[arg6] = arg1[arg4]
			arg6++
			arg4++
			arg8[arg6] = arg1[arg4]
			arg6++
			arg4++
			arg8[arg6] = arg1[arg4]
			arg6++
			arg4++
		}
		for j := var14; j < 0; j++ {
			arg8[arg6] = arg1[arg4]
			arg6++
			arg4++
		}
		arg6 += arg2
		arg4 += arg5
	}
}

// PlotSprite
func (p *Pix32) Draw(y int, x int) {
	x += p.CropX
	y += p.CropY

	dstOff := x + y*pix2d.Width2D
	srcOff := 0

	h := p.Height
	w := p.Width

	dstStep := pix2d.Width2D - w
	srcStep := 0

	if y < pix2d.BoundTop {
		cutoff := pix2d.BoundTop - y
		h -= cutoff
		y = pix2d.BoundTop
		srcOff += cutoff * w
		dstOff += cutoff * pix2d.Width2D
	}

	if y+h > pix2d.BoundBottom {
		h -= y + h - pix2d.BoundBottom
	}

	if x < pix2d.BoundLeft {
		cutoff := pix2d.BoundLeft - x
		w -= cutoff
		x = pix2d.BoundLeft
		srcOff += cutoff
		dstOff += cutoff
		srcStep += cutoff
		dstStep += cutoff
	}

	if x+w > pix2d.BoundRight {
		cutoff := x + w - pix2d.BoundRight
		w -= cutoff
		srcStep += cutoff
		dstStep += cutoff
	}

	if w > 0 && h > 0 {
		p.CopyPixels2(pix2d.Data, p.Pixels, srcOff, dstOff, w, h, dstStep, srcStep)
	}
}

// Plot
func (p *Pix32) CopyPixels2(arg0 []int, arg1 []int, arg3, arg4, arg5, arg6, arg7, arg8 int) {
	var10 := -(arg5 >> 2)
	var15 := -(arg5 & 0x3)
	for i := -arg6; i < 0; i++ {
		for j := var10; j < 0; j++ {
			var14 := arg1[arg3]
			arg3++
			if var14 == 0 {
				arg4++
			} else {
				arg0[arg4] = var14
				arg4++
			}
			var14 = arg1[arg3]
			arg3++
			if var14 == 0 {
				arg4++
			} else {
				arg0[arg4] = var14
				arg4++
			}
			var14 = arg1[arg3]
			arg3++
			if var14 == 0 {
				arg4++
			} else {
				arg0[arg4] = var14
				arg4++
			}
			var14 = arg1[arg3]
			arg3++
			if var14 == 0 {
				arg4++
			} else {
				arg0[arg4] = var14
				arg4++
			}
		}
		for j := var15; j < 0; j-- {
			var14 := arg1[arg3]
			arg3++
			if var14 == 0 {
				arg4++
			} else {
				arg0[arg4] = var14
				arg4++
			}
		}
		arg4 += arg7
		arg3 += arg8
	}
}

func (p *Pix32) Crop(arg0, arg1, arg2, arg4 int) {
	var6 := p.Width
	var7 := p.Height
	var8 := 0
	var9 := 0
	_ = (var6 << 16) / arg2
	_ = (var7 << 16) / arg0
	var12 := p.CropW
	var13 := p.CropH
	var18 := (var12 << 16) / arg2
	var19 := (var13 << 16) / arg0
	arg4 += (p.CropX*arg2 + var12 - 1) / var12
	arg1 += (p.CropY*arg0 + var13 - 1) / var13
	if p.CropX*arg2%var12 != 0 {
		var8 = ((var12 - (p.CropX*arg2)%var12) << 16) / arg2
	}
	if p.CropY*arg0%var13 != 0 {
		var9 = ((var13 - (p.CropY*arg0)%var13) << 16) / arg0
	}
	arg2 = arg2 * (p.Width - (var8 >> 16)) / var12
	arg0 = arg0 * (p.Height - (var9 >> 16)) / var13
	var14 := arg4 + arg1*pix2d.Width2D
	var15 := pix2d.Width2D - arg2
	if arg1 < pix2d.BoundTop {
		var16 := pix2d.BoundTop - arg1
		arg0 -= var16
		arg1 = 0
		var14 += var16 * pix2d.Width2D
		var9 += var19 * var16
	}
	if arg1+arg0 > pix2d.BoundBottom {
		arg0 -= arg1 + arg0 - pix2d.BoundBottom
	}
	if arg4 < pix2d.BoundLeft {
		var16 := pix2d.BoundLeft - arg4
		arg2 -= var16
		arg4 = 0
		var14 += var16
		var8 += var18 * var16
		var15 += var16
	}
	if arg4+arg2 > pix2d.BoundRight {
		var16 := arg4 + arg2 - pix2d.BoundRight
		arg2 -= var16
		var15 += var16
	}
	p.Scale(var8, var18, pix2d.Data, var19, var19, p.Pixels, var15, var14, arg0, var6, arg2)
}

func (p *Pix32) Scale(arg0 int, arg1 int, arg2 []int, arg4 int, arg5 int, arg7 []int, arg8, arg9, arg10, arg11, arg12 int) {
	var14 := arg0
	for i := -arg10; i < 0; i++ {
		var16 := (arg5 >> 16) * arg11
		for j := -arg12; j < 0; j++ {
			var19 := arg7[(arg0>>16)+var16]
			if var19 == 0 {
				arg9++
			} else {
				arg2[arg9] = var19
				arg9++
			}
			arg0 += arg1
		}
		arg5 += arg4
		arg0 = var14
		arg9 += arg8
	}
}

func (p *Pix32) DrawAlpha(arg0, arg1, arg2 int) {
	arg1 += p.CropX
	arg2 += p.CropY
	var5 := arg1 + arg2*pix2d.Width2D
	var6 := 0
	var7 := p.Height
	var8 := p.Width
	var9 := pix2d.Width2D - var8
	var10 := 0
	if arg2 < pix2d.BoundTop {
		var11 := pix2d.BoundTop - arg2
		var7 -= var11
		arg2 = pix2d.BoundTop
		var6 += var11 * var8
		var5 += var11 * pix2d.Width2D
	}
	if arg2+var7 > pix2d.BoundBottom {
		var7 -= arg2 + var7 - pix2d.BoundBottom
	}
	if arg1 < pix2d.BoundLeft {
		var11 := pix2d.BoundLeft - arg1
		var8 -= var11
		arg1 = pix2d.BoundLeft
		var6 += var11
		var5 += var11
		var10 += var11
		var9 += var11
	}
	if arg1+var8 > pix2d.BoundRight {
		var11 := arg1 + var8 - pix2d.BoundRight
		var8 -= var11
		var10 += var11
		var9 += var11
	}
	if var8 > 0 && var7 > 0 {
		p.CopyPixelsAlpha(var5, p.Pixels, arg0, var7, pix2d.Data, var6, var8, var9, var10)
	}
}

func (p *Pix32) CopyPixelsAlpha(arg0 int, arg2 []int, arg3 int, arg4 int, arg5 []int, arg6, arg8, arg9, arg10 int) {
	var12 := 256 - arg3
	for i := -arg4; i < 0; i++ {
		for j := -arg8; j < 0; j++ {
			var16 := arg2[arg6]
			arg6++
			if var16 == 0 {
				arg0++
			} else {
				var15 := arg5[arg0]
				arg5[arg0] = ((((var16&0xFF00FF)*arg3 + (var15&0xFF00FF)*var12) & 0xFF00FF00) + (((var16&0xFF00)*arg3 + (var15&0xFF00)*var12) & 0xFF0000)) >> 8
				arg0++
			}
		}
		arg0 += arg9
		arg6 += arg10
	}
}

func (p *Pix32) DrawRotatedMasked(arg0 int, w int, lineStart []int, h int, anchorY int, arg5 int, anchorX int, x int, y int, lineWidth []int) {
	centerX := -w / 2
	centerY := -h / 2

	sin := int(math.Sin(float64(arg0)/326.11) * 65536.0)
	cos := int(math.Cos(float64(arg0)/326.11) * 65536.0)
	sinZoom := (sin * arg5) >> 8
	cosZoom := (cos * arg5) >> 8

	leftX := (anchorX << 16) + centerY*sinZoom + centerX*cosZoom
	leftY := (anchorY << 16) + (centerY*cosZoom - centerX*sinZoom)
	leftOff := x + y*pix2d.Width2D

	for i := range h {
		dstOff := lineStart[i]
		dstX := leftOff + dstOff

		srcX := leftX + cosZoom*dstOff
		srcY := leftY - sinZoom*dstOff

		for j := -lineWidth[i]; j < 0; j++ {
			pix2d.Data[dstX] = p.Pixels[(srcX>>16)+(srcY>>16)*p.Width]
			dstX++
			srcX += cosZoom
			srcY -= sinZoom
		}

		leftX += sinZoom
		leftY += cosZoom
		leftOff += pix2d.Width2D
	}
}

func (p *Pix32) DrawMasked(arg0 *pix8.Pix8, arg1 int, arg2 int) {
	arg2 += p.CropX
	arg1 += p.CropY
	var5 := arg2 + arg1*pix2d.Width2D
	var6 := 0
	var7 := p.Height
	var8 := p.Width
	var9 := pix2d.Width2D - var8
	var10 := 0
	if arg1 < pix2d.BoundTop {
		var11 := pix2d.BoundTop - arg1
		var7 -= var11
		arg1 = pix2d.BoundTop
		var6 += var11 * var8
		var5 += var11 * pix2d.Width2D
	}
	if arg1+var7 > pix2d.BoundBottom {
		var7 -= arg1 + var7 - pix2d.BoundBottom
	}
	if arg2 < pix2d.BoundLeft {
		var11 := pix2d.BoundLeft - arg2
		var8 -= var11
		arg2 = pix2d.BoundLeft
		var6 += var11
		var5 += var11
		var10 += var11
		var9 += var11
	}
	if arg2+var8 > pix2d.BoundRight {
		var11 := arg2 + var8 - pix2d.BoundRight
		var8 -= var11
		var10 += var11
		var9 += var11
	}
	if var8 > 0 && var7 > 0 {
		p.CopyPixelsMasked(var8, var10, var7, var6, pix2d.Data, p.Pixels, var5, arg0.Pixels, var9)
	}
}

func (p *Pix32) CopyPixelsMasked(arg0 int, arg1 int, arg4 int, arg5 int, arg6 []int, arg7 []int, arg8 int, arg9 []byte, arg10 int) {
	var12 := -(arg0 >> 2)
	var16 := -(arg0 & 0x3)
	for i := -arg4; i < 0; i++ {
		for j := var12; j < 0; j++ {
			var17 := arg7[arg5]
			arg5++
			if var17 != 0 && arg9[arg8] == 0 {
				arg6[arg8] = var17
				arg8++
			} else {
				arg8++
			}
			var17 = arg7[arg5]
			arg5++
			if var17 != 0 && arg9[arg8] == 0 {
				arg6[arg8] = var17
				arg8++
			} else {
				arg8++
			}
			var17 = arg7[arg5]
			arg5++
			if var17 != 0 && arg9[arg8] == 0 {
				arg6[arg8] = var17
				arg8++
			} else {
				arg8++
			}
			var17 = arg7[arg5]
			arg5++
			if var17 != 0 && arg9[arg8] == 0 {
				arg6[arg8] = var17
				arg8++
			} else {
				arg8++
			}
		}
		for j := var16; j < 0; j++ {
			var17 := arg7[arg5]
			arg5++
			if var17 != 0 && arg9[arg8] == 0 {
				arg6[arg8] = var17
				arg8++
			} else {
				arg8++
			}
		}
		arg8 += arg10
		arg5 += arg1
	}
}

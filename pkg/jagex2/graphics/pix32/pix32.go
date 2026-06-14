package pix32

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg" // registers the JPEG decoder for image.Decode (title.dat is a JPEG)
	"math"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix8"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

type Pix32 struct {
	Pixels []int
	OWi    int // original width - was CropW
	Wi     int // width - was Width
	OHi    int // original height - was CropH
	Hi     int // height - was Height
	YOf    int // y offset - was CropY
	XOf    int // x offset - was CropX
}

func NewPix321(width int, height int) *Pix32 {
	var p Pix32
	p.Pixels = make([]int, width*height)
	p.OWi = width
	p.Wi = p.OWi
	p.OHi = height
	p.Hi = p.OHi
	p.YOf = 0
	p.XOf = 0
	return &p
}

func NewPix322(imageData []byte) *Pix32 {
	// Java uses Toolkit.createImage + MediaTracker.waitForAll + PixelGrabber to load
	// a JPEG and grab raw ARGB pixels; those are applet-only AWT APIs. Go's stdlib
	// image.Decode is a direct replacement and is already synchronous, so no
	// MediaTracker wait is needed.
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		// Java catches the exception, prints "Error converting jpg" and returns a
		// zero-initialised Pix32; mirror that here (don't panic on a nil img).
		fmt.Println("Error converting jpg")
		return &Pix32{}
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	p := &Pix32{
		Pixels: make([]int, width*height),
		OWi:    width,
		Wi:     width,
		OHi:    height,
		Hi:     height,
		YOf:    0,
		XOf:    0,
	}

	// Pack ARGB into pixels[] in the same layout Java's PixelGrabber produces.
	// Java's int is 32-bit signed and Go's int is 64-bit, but the field is []int
	// throughout this package and downstream consumers are bitwise-only, so the
	// width difference is invisible to callers.
	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// Convert from 16-bit channels to 8-bit and pack as ARGB.
			// Java: (a << 24) | (r << 16) | (g << 8) | b
			p.Pixels[idx] = int(((a >> 8) << 24) | ((r >> 8) << 16) | ((g >> 8) << 8) | (b >> 8))
			idx++
		}
	}
	return p
}

func NewPix323(jag *io.Jagfile, name string, sprite int) *Pix32 {
	var p Pix32

	dat := io.NewPacket(jag.Read(name+".dat", nil))
	idx := io.NewPacket(jag.Read("index.dat", nil))
	idx.Pos = dat.G2()
	p.OWi = idx.G2()
	p.OHi = idx.G2()
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
	p.XOf = idx.G1()
	p.YOf = idx.G1()
	p.Wi = idx.G2()
	p.Hi = idx.G2()
	pixelOrder := idx.G1()
	length := p.Wi * p.Hi
	p.Pixels = make([]int, length)
	switch pixelOrder {
	case 0:
		for i := range length {
			p.Pixels[i] = bPal[dat.G1()]
		}
	case 1:
		for x := range p.Wi {
			for y := range p.Hi {
				p.Pixels[x+y*p.Wi] = bPal[dat.G1()]
			}
		}
	}
	return &p
}

func (p *Pix32) SetPixels() {
	pix2d.SetPixels(p.Wi, p.Pixels, p.Hi)
}

// was Translate
func (p *Pix32) RGBAdjust(arg0, arg1, arg2 int) {
	for i := range len(p.Pixels) {
		var6 := p.Pixels[i]
		if var6 != 0 {
			var7 := (var6 >> 16) & 0xFF
			var7 += arg0
			if var7 < 1 {
				var7 = 1
			} else if var7 > 0xFF {
				var7 = 0xFF
			}
			var8 := (var6 >> 8) & 0xFF
			var8 += arg1
			if var8 < 1 {
				var8 = 1
			} else if var8 > 0xFF {
				var8 = 0xFF
			}
			var9 := var6 & 0xFF
			var9 += arg2
			if var9 < 1 {
				var9 = 1
			} else if var9 > 0xFF {
				var9 = 0xFF
			}
			p.Pixels[i] = (var7 << 16) + (var8 << 8) + var9
		}
	}
}

// Old name: BlitOpaque
func (p *Pix32) QuickPlotSprite(arg1, arg2 int) {
	arg1 += p.XOf
	arg2 += p.YOf
	var4 := arg1 + arg2*pix2d.Width2D
	var5 := 0
	var6 := p.Hi
	var7 := p.Wi
	var8 := pix2d.Width2D - var7
	var9 := 0
	if arg2 < pix2d.ClipMinY {
		var10 := pix2d.ClipMinY - arg2
		var6 -= var10
		arg2 = pix2d.ClipMinY
		var5 += var10 * var7
		var4 += var10 * pix2d.Width2D
	}
	if arg2+var6 > pix2d.ClipMaxY {
		var6 -= arg2 + var6 - pix2d.ClipMaxY
	}
	if arg1 < pix2d.ClipMinX {
		var10 := pix2d.ClipMinX - arg1
		var7 -= var10
		arg1 = pix2d.ClipMinX
		var5 += var10
		var4 += var10
		var9 += var10
		var8 += var10
	}
	if arg1+var7 > pix2d.ClipMaxX {
		var10 := arg1 + var7 - pix2d.ClipMaxX
		var7 -= var10
		var9 += var10
		var8 += var10
	}
	if var7 > 0 && var6 > 0 {
		p.QuickPlot(p.Pixels, var8, var6, var5, var9, var4, var7, pix2d.Data)
	}
}

// was CopyPixels1 - copies pixels into pix2d.Data
func (p *Pix32) QuickPlot(arg1 []int, arg2 int, arg3 int, arg4 int, arg5 int, arg6 int, arg7 int, dest []int) {
	var10 := -(arg7 >> 2)
	var14 := -(arg7 & 0x3)
	for i := -arg3; i < 0; i++ {
		for j := var10; j < 0; j++ {
			dest[arg6] = arg1[arg4]
			arg6++
			arg4++
			dest[arg6] = arg1[arg4]
			arg6++
			arg4++
			dest[arg6] = arg1[arg4]
			arg6++
			arg4++
			dest[arg6] = arg1[arg4]
			arg6++
			arg4++
		}
		for j := var14; j < 0; j++ {
			dest[arg6] = arg1[arg4]
			arg6++
			arg4++
		}
		arg6 += arg2
		arg4 += arg5
	}
}

// was Draw
func (p *Pix32) PlotSprite(y int, x int) {
	x += p.XOf
	y += p.YOf

	dstOff := x + y*pix2d.Width2D
	srcOff := 0

	h := p.Hi
	w := p.Wi

	dstStep := pix2d.Width2D - w
	srcStep := 0

	if y < pix2d.ClipMinY {
		cutoff := pix2d.ClipMinY - y
		h -= cutoff
		y = pix2d.ClipMinY
		srcOff += cutoff * w
		dstOff += cutoff * pix2d.Width2D
	}

	if y+h > pix2d.ClipMaxY {
		h -= y + h - pix2d.ClipMaxY
	}

	if x < pix2d.ClipMinX {
		cutoff := pix2d.ClipMinX - x
		w -= cutoff
		x = pix2d.ClipMinX
		srcOff += cutoff
		dstOff += cutoff
		srcStep += cutoff
		dstStep += cutoff
	}

	if x+w > pix2d.ClipMaxX {
		cutoff := x + w - pix2d.ClipMaxX
		w -= cutoff
		srcStep += cutoff
		dstStep += cutoff
	}

	if w > 0 && h > 0 {
		p.Plot(pix2d.Data, p.Pixels, srcOff, dstOff, w, h, dstStep, srcStep)
	}
}

// was CopyPixels2 - copies src into pix2d.Data
func (p *Pix32) Plot(pix2dData []int, pix32PixelsSrc []int, srcOff, dstOff, w, h, dstStep, srcStep int) {
	var10 := -(w >> 2)
	var15 := -(w & 0x3)
	for i := -h; i < 0; i++ {
		for j := var10; j < 0; j++ {
			var14 := pix32PixelsSrc[srcOff]
			srcOff++
			if var14 == 0 {
				dstOff++
			} else {
				pix2dData[dstOff] = var14
				dstOff++
			}
			var14 = pix32PixelsSrc[srcOff]
			srcOff++
			if var14 == 0 {
				dstOff++
			} else {
				pix2dData[dstOff] = var14
				dstOff++
			}
			var14 = pix32PixelsSrc[srcOff]
			srcOff++
			if var14 == 0 {
				dstOff++
			} else {
				pix2dData[dstOff] = var14
				dstOff++
			}
			var14 = pix32PixelsSrc[srcOff]
			srcOff++
			if var14 == 0 {
				dstOff++
			} else {
				pix2dData[dstOff] = var14
				dstOff++
			}
		}
		for j := var15; j < 0; j++ {
			var14 := pix32PixelsSrc[srcOff]
			srcOff++
			if var14 == 0 {
				dstOff++
			} else {
				pix2dData[dstOff] = var14
				dstOff++
			}
		}
		dstOff += dstStep
		srcOff += srcStep
	}
}

func (p *Pix32) Crop(arg0, arg1, arg2, arg4 int) {
	// Java: crop() wraps its body in try { ... } catch (Exception var17) {
	// System.out.println("error in sprite clipping routine"); } (Pix32.java:302-353)
	// — an out-of-bounds index aborts this single draw, logs, and continues.
	defer func() {
		if recover() != nil {
			fmt.Println("error in sprite clipping routine")
		}
	}()
	var6 := p.Wi
	var7 := p.Hi
	var8 := 0
	var9 := 0
	_ = (var6 << 16) / arg2
	_ = (var7 << 16) / arg0
	var12 := p.OWi
	var13 := p.OHi
	var18 := (var12 << 16) / arg2
	var19 := (var13 << 16) / arg0
	arg4 += (p.XOf*arg2 + var12 - 1) / var12
	arg1 += (p.YOf*arg0 + var13 - 1) / var13
	if p.XOf*arg2%var12 != 0 {
		var8 = ((var12 - (p.XOf*arg2)%var12) << 16) / arg2
	}
	if p.YOf*arg0%var13 != 0 {
		var9 = ((var13 - (p.YOf*arg0)%var13) << 16) / arg0
	}
	arg2 = arg2 * (p.Wi - (var8 >> 16)) / var12
	arg0 = arg0 * (p.Hi - (var9 >> 16)) / var13
	var14 := arg4 + arg1*pix2d.Width2D
	var15 := pix2d.Width2D - arg2
	if arg1 < pix2d.ClipMinY {
		var16 := pix2d.ClipMinY - arg1
		arg0 -= var16
		arg1 = 0
		var14 += var16 * pix2d.Width2D
		var9 += var19 * var16
	}
	if arg1+arg0 > pix2d.ClipMaxY {
		arg0 -= arg1 + arg0 - pix2d.ClipMaxY
	}
	if arg4 < pix2d.ClipMinX {
		var16 := pix2d.ClipMinX - arg4
		arg2 -= var16
		arg4 = 0
		var14 += var16
		var8 += var18 * var16
		var15 += var16
	}
	if arg4+arg2 > pix2d.ClipMaxX {
		var16 := arg4 + arg2 - pix2d.ClipMaxX
		arg2 -= var16
		var15 += var16
	}
	p.Scale(var8, var18, pix2d.Data, var19, var9, p.Pixels, var15, var14, arg0, var6, arg2)
}

func (p *Pix32) Scale(arg0 int, arg1 int, arg2 []int, arg4 int, arg5 int, arg7 []int, arg8, arg9, arg10, arg11, arg12 int) {
	// Java: scale() wraps its body in try { ... } catch (Exception var18) {
	// System.out.println("error in plot_scale"); } (Pix32.java:357-378) — an
	// out-of-bounds index aborts this single draw, logs, and continues.
	defer func() {
		if recover() != nil {
			fmt.Println("error in plot_scale")
		}
	}()
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

func (p *Pix32) DrawAlpha(arg0, x, y int) {
	x += p.XOf
	y += p.YOf

	dstOff := x + y*pix2d.Width2D
	srcOff := 0

	h := p.Hi
	w := p.Wi

	dstStep := pix2d.Width2D - w
	srcStep := 0

	if y < pix2d.ClipMinY {
		cutoff := pix2d.ClipMinY - y
		h -= cutoff
		y = pix2d.ClipMinY
		srcOff += cutoff * w
		dstOff += cutoff * pix2d.Width2D
	}

	if y+h > pix2d.ClipMaxY {
		h -= y + h - pix2d.ClipMaxY
	}

	if x < pix2d.ClipMinX {
		cutoff := pix2d.ClipMinX - x
		w -= cutoff
		x = pix2d.ClipMinX
		srcOff += cutoff
		dstOff += cutoff
		srcStep += cutoff
		dstStep += cutoff
	}

	if x+w > pix2d.ClipMaxX {
		cutoff := x + w - pix2d.ClipMaxX
		w -= cutoff
		srcStep += cutoff
		dstStep += cutoff
	}
	if w > 0 && h > 0 {
		p.TransPlot(dstOff, p.Pixels, arg0, h, pix2d.Data, srcOff, w, dstStep, srcStep)
	}
}

// was CopyPixelsAlpha
func (p *Pix32) TransPlot(arg0 int, arg2 []int, arg3 int, arg4 int, arg5 []int, arg6, arg8, arg9, arg10 int) {
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
	// Java: drawRotatedMasked() wraps its body in try { ... } catch (Exception
	// var23) {} (Pix32.java:442-468) — silently swallow an out-of-bounds index
	// (e.g. a rotated source coord outside Pixels) and skip this draw.
	defer func() { _ = recover() }()
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
			pix2d.Data[dstX] = p.Pixels[(srcX>>16)+(srcY>>16)*p.Wi]
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
	arg2 += p.XOf
	arg1 += p.YOf
	var5 := arg2 + arg1*pix2d.Width2D
	var6 := 0
	var7 := p.Hi
	var8 := p.Wi
	var9 := pix2d.Width2D - var8
	var10 := 0
	if arg1 < pix2d.ClipMinY {
		var11 := pix2d.ClipMinY - arg1
		var7 -= var11
		arg1 = pix2d.ClipMinY
		var6 += var11 * var8
		var5 += var11 * pix2d.Width2D
	}
	if arg1+var7 > pix2d.ClipMaxY {
		var7 -= arg1 + var7 - pix2d.ClipMaxY
	}
	if arg2 < pix2d.ClipMinX {
		var11 := pix2d.ClipMinX - arg2
		var8 -= var11
		arg2 = pix2d.ClipMinX
		var6 += var11
		var5 += var11
		var10 += var11
		var9 += var11
	}
	if arg2+var8 > pix2d.ClipMaxX {
		var11 := arg2 + var8 - pix2d.ClipMaxX
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

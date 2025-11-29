package pix8

import (
	"goscape-client/pkg/jagex2/graphics/pix2d"
	"goscape-client/pkg/jagex2/io"
)

type Pix8 struct {
	Pixels  []byte
	CropW   int
	CropH   int
	Palette []int
	CropX   int
	CropY   int
	Width   int
	Height  int
}

func NewPix8(arg0 io.Jagfile, arg1 string, arg2 int) *Pix8 {
	p := new(Pix8)

	var4 := io.NewPacket(arg0.Read(arg1+".dat", nil))
	var5 := io.NewPacket(arg0.Read("index.dat", nil))
	var5.Pos = var4.G2()

	p.CropW = int(var5.G2())
	p.CropH = int(var5.G2())
	var6 := var5.G1()
	p.Palette = make([]int, var6)
	for i := range var6 - 1 {
		p.Palette[i+1] = int(var5.G3())
	}
	for range arg2 {
		var5.Pos += 2
		var4.Pos += var5.G2() * var5.G2()
		var5.Pos++
	}
	p.CropX = int(var5.G1())
	p.CropY = int(var5.G1())
	p.Width = int(var5.G2())
	p.Height = int(var5.G2())
	var9 := var5.G1()
	var10 := p.Width * p.Height
	p.Pixels = make([]byte, var10)
	if var9 == 0 {
		for i := range var10 {
			p.Pixels[i] = var4.G1B()
		}
	} else if var9 == 1 {
		for i := range p.Width {
			for j := range p.Height {
				p.Pixels[i+j*p.Width] = var4.G1B()
			}
		}
	}
	return p
}

func (p *Pix8) Shrink() {
	p.CropW /= 2
	p.CropH /= 2
	pixels := make([]byte, p.CropW*p.CropH)
	var3 := 0
	for i := range p.Height {
		for j := range p.Width {
			pixels[(j+p.CropX>>1)+(i+p.CropY>>1)*p.CropW] = p.Pixels[var3]
			var3++
		}
	}
	p.Pixels = pixels
	p.Width = p.CropW
	p.Height = p.CropH
	p.CropX = 0
	p.CropY = 0
}

func (p *Pix8) Crop() {
	if p.Width == p.CropW && p.Height == p.CropH {
		return
	}
	pixels := make([]byte, p.CropW*p.CropH)
	var3 := 0
	for i := range p.Height {
		for j := range p.Width {
			pixels[j+p.CropX+(i+p.CropY)*p.CropW] = p.Pixels[var3]
			var3++
		}
	}
	p.Pixels = pixels
	p.Width = p.CropW
	p.Height = p.CropH
	p.CropX = 0
	p.CropY = 0
}

func (p *Pix8) FlipHorizontally() {
	pixels := make([]byte, p.Width*p.Height)
	var3 := 0
	for i := range p.Height {
		for j := p.Width - 1; j >= 0; j-- {
			pixels[var3] = p.Pixels[j+i+p.Width]
			var3++
		}
	}
	p.Pixels = pixels
	p.CropX = p.CropW - p.Width - p.CropX
}

func (p *Pix8) FlipVertically() {
	pixels := make([]byte, p.Width*p.Height)
	var3 := 0
	for i := p.Height - 1; i >= 0; i-- {
		for j := range p.Width {
			pixels[var3] = p.Pixels[j+i*p.Width]
			var3++
		}
	}
	p.Pixels = pixels
	p.CropY = p.CropH - p.Height - p.CropY
}

func (p *Pix8) Translate(arg0 int, arg1 int, arg2 int) {
	var6 := 0
	for i := range len(p.Palette) {
		var6 = p.Palette[i] >> 16 & 0xFF
		var6 += arg0
		if var6 < 0 {
			var6 = 0
		} else if var6 > 255 {
			var6 = 255
		}
		var7 := p.Palette[i] >> 8 & 0xFF
		var7 += arg1
		if var7 < 0 {
			var7 = 0
		} else if var7 > 255 {
			var7 = 255
		}
		var8 := p.Palette[i] & 0xFF
		var8 += arg2
		if var8 < 0 {
			var8 = 0
		} else if var8 > 255 {
			var8 = 255
		}
		p.Palette[i] = (var6 << 16) + (var7 << 8) + var8
	}
}

func (p *Pix8) Draw(arg0 int, arg1 int) {
	arg1 += p.CropX
	arg0 += p.CropY
	var4 := arg1 + arg0*pix2d.Width2D
	var5 := 0
	var6 := p.Height
	var7 := p.Width
	var8 := pix2d.Width2D - var7
	var9 := 0
	var10 := 0
	if arg0 < pix2d.BoundTop {
		var10 = pix2d.BoundTop - arg0
		var6 -= var10
		arg0 = pix2d.BoundTop
		var5 += var10 * var7
		var4 += var10 * pix2d.Width2D
	}
	if arg0+var6 > pix2d.BoundBottom {
		var6 -= arg0 + var6 - pix2d.BoundBottom
	}
	if arg1 < pix2d.BoundLeft {
		var10 = pix2d.BoundLeft - arg1
		var7 -= var10
		arg1 = pix2d.BoundLeft
		var5 += var10
		var4 += var10
		var9 += var10
		var8 += var10
	}
	if arg1+var7 > pix2d.BoundRight {
		var10 = arg1 + var7 - pix2d.BoundRight
		var7 -= var10
		var9 += var10
		var8 += var10
	}
	if var7 > 0 && var6 > 0 {
		p.CopyPixels(pix2d.Data, var5, var9, p.Pixels, var6, 0, var7, var4, var8, p.Palette)
	}
}

func (p *Pix8) CopyPixels(arg0 []int, arg1 int, arg2 int, arg3 []byte, arg4 int, arg5 int, arg6 int, arg7 int, arg8 int, arg9 []int) {
	var11 := -(arg6 >> 2)
	var16 := -(arg6 & 0x3)
	if arg5 != 0 {
		return
	}
	for i := -arg4; i < 0; i++ {
		for j := var11; j < 0; j++ {
			var14 := arg3[arg1]
			arg1++
			if var14 == 0 {
				arg7++
			} else {
				arg0[arg7] = arg9[var14&0xFF]
				arg7++
			}
			var14 = arg3[arg1]
			arg1++
			if var14 == 0 {
				arg7++
			} else {
				arg0[arg7] = arg9[var14&0xFF]
				arg7++
			}
			var14 = arg3[arg1]
			arg1++
			if var14 == 0 {
				arg7++
			} else {
				arg0[arg7] = arg9[var14&0xFF]
				arg7++
			}
			var14 = arg3[arg1]
			arg1++
			if var14 == 0 {
				arg7++
			} else {
				arg0[arg7] = arg9[var14&0xFF]
				arg7++
			}
		}
		for j := var16; j < 0; j++ {
			var15 := arg3[arg1]
			arg1++
			if var15 == 0 {
				arg7++
			} else {
				arg0[arg7] = arg9[var15&0xFF]
				arg7++
			}
		}
		arg7 += arg8
		arg1 += arg2
	}
}

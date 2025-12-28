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

func NewPix8(jag *io.Jagfile, name string, sprite int) *Pix8 {
	p := new(Pix8)

	dat := io.NewPacket(jag.Read(name+".dat", nil))
	idx := io.NewPacket(jag.Read("index.dat", nil))
	idx.Pos = dat.G2()

	p.CropW = int(idx.G2())
	p.CropH = int(idx.G2())
	palCount := idx.G1()
	p.Palette = make([]int, palCount)
	for i := range palCount - 1 {
		p.Palette[i+1] = int(idx.G3())
	}
	for range sprite {
		idx.Pos += 2
		dat.Pos += idx.G2() * idx.G2()
		idx.Pos++
	}
	p.CropX = int(idx.G1())
	p.CropY = int(idx.G1())
	p.Width = int(idx.G2())
	p.Height = int(idx.G2())
	pixelOrder := idx.G1()
	length := p.Width * p.Height
	p.Pixels = make([]byte, length)
	if pixelOrder == 0 {
		for i := range length {
			p.Pixels[i] = dat.G1B()
		}
	} else if pixelOrder == 1 {
		for i := range p.Width {
			for j := range p.Height {
				p.Pixels[i+j*p.Width] = dat.G1B()
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
	for y := range p.Height {
		for x := range p.Width {
			pixels[((x+p.CropX)>>1)+((y+p.CropY)>>1)*p.CropW] = p.Pixels[var3]
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
	i := 0
	for y := range p.Height {
		for x := range p.Width {
			pixels[x+p.CropX+(y+p.CropY)*p.CropW] = p.Pixels[i]
			i++
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
	i := 0
	for y := range p.Height {
		for x := p.Width - 1; x >= 0; x-- {
			pixels[i] = p.Pixels[x+y+p.Width]
			i++
		}
	}
	p.Pixels = pixels
	p.CropX = p.CropW - p.Width - p.CropX
}

func (p *Pix8) FlipVertically() {
	pixels := make([]byte, p.Width*p.Height)
	i := 0
	for y := p.Height - 1; y >= 0; y-- {
		for x := range p.Width {
			pixels[i] = p.Pixels[x+y*p.Width]
			i++
		}
	}
	p.Pixels = pixels
	p.CropY = p.CropH - p.Height - p.CropY
}

func (p *Pix8) Translate(arg0 int, arg1 int, arg2 int) {
	var6 := 0
	for i := range len(p.Palette) {
		var6 = (p.Palette[i] >> 16) & 0xFF
		var6 += arg0
		if var6 < 0 {
			var6 = 0
		} else if var6 > 255 {
			var6 = 255
		}
		var7 := (p.Palette[i] >> 8) & 0xFF
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

func (p *Pix8) Draw(y int, x int) {
	x += p.CropX
	y += p.CropY
	var4 := x + y*pix2d.Width2D
	var5 := 0
	var6 := p.Height
	var7 := p.Width
	var8 := pix2d.Width2D - var7
	var9 := 0
	var10 := 0
	if y < pix2d.BoundTop {
		var10 = pix2d.BoundTop - y
		var6 -= var10
		y = pix2d.BoundTop
		var5 += var10 * var7
		var4 += var10 * pix2d.Width2D
	}
	if y+var6 > pix2d.BoundBottom {
		var6 -= y + var6 - pix2d.BoundBottom
	}
	if x < pix2d.BoundLeft {
		var10 = pix2d.BoundLeft - x
		var7 -= var10
		x = pix2d.BoundLeft
		var5 += var10
		var4 += var10
		var9 += var10
		var8 += var10
	}
	if x+var7 > pix2d.BoundRight {
		var10 = x + var7 - pix2d.BoundRight
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

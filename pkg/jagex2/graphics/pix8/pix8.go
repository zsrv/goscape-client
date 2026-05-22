package pix8

import (
	"goscape-client/pkg/jagex2/graphics/pix2d"
	"goscape-client/pkg/jagex2/io"
)

type Pix8 struct {
	// these short field names are authentic to native

	Pixels []byte
	OWi    int   // original width - was CropW
	OHi    int   // original height - was CropH
	BPal   []int // base palette - was Palette
	XOf    int   // x offset - was CropX
	YOf    int   // y offset - was CropY
	Wi     int   // width - was Width
	Hi     int   // height - was Height
}

func NewPix8(jag *io.Jagfile, name string, sprite int) *Pix8 {
	p := new(Pix8)

	dat := io.NewPacket(jag.Read(name+".dat", nil))
	idx := io.NewPacket(jag.Read("index.dat", nil))

	idx.Pos = dat.G2()
	p.OWi = int(idx.G2())
	p.OHi = int(idx.G2())

	palCount := idx.G1()
	p.BPal = make([]int, palCount)
	for i := range palCount - 1 {
		p.BPal[i+1] = int(idx.G3())
	}

	for range sprite {
		idx.Pos += 2
		dat.Pos += idx.G2() * idx.G2()
		idx.Pos++
	}

	p.XOf = int(idx.G1())
	p.YOf = int(idx.G1())
	p.Wi = int(idx.G2())
	p.Hi = int(idx.G2())
	pixelOrder := idx.G1()

	length := p.Wi * p.Hi
	p.Pixels = make([]byte, length)

	if pixelOrder == 0 {
		for i := range length {
			p.Pixels[i] = byte(dat.G1B())
		}
	} else if pixelOrder == 1 {
		for x := range p.Wi {
			for y := range p.Hi {
				p.Pixels[x+y*p.Wi] = byte(dat.G1B())
			}
		}
	}
	return p
}

// was Shrink
func (p *Pix8) HalveSize() {
	p.OWi /= 2
	p.OHi /= 2

	pixels := make([]byte, p.OWi*p.OHi)
	i := 0
	for y := range p.Hi {
		for x := range p.Wi {
			pixels[((x+p.XOf)>>1)+((y+p.YOf)>>1)*p.OWi] = p.Pixels[i]
			i++
		}
	}
	p.Pixels = pixels

	p.Wi = p.OWi
	p.Hi = p.OHi
	p.XOf = 0
	p.YOf = 0
}

// was Crop
func (p *Pix8) Trim() {
	if p.Wi == p.OWi && p.Hi == p.OHi {
		return
	}

	pixels := make([]byte, p.OWi*p.OHi)
	i := 0
	for y := range p.Hi {
		for x := range p.Wi {
			pixels[x+p.XOf+(y+p.YOf)*p.OWi] = p.Pixels[i]
			i++
		}
	}
	p.Pixels = pixels

	p.Wi = p.OWi
	p.Hi = p.OHi
	p.XOf = 0
	p.YOf = 0
}

// was FlipHorizontally
func (p *Pix8) HFlip() {
	pixels := make([]byte, p.Wi*p.Hi)
	i := 0
	for y := range p.Hi {
		for x := p.Wi - 1; x >= 0; x-- {
			pixels[i] = p.Pixels[x+y*p.Wi]
			i++
		}
	}
	p.Pixels = pixels

	p.XOf = p.OWi - p.Wi - p.XOf
}

// was FlipVertically
func (p *Pix8) VFlip() {
	pixels := make([]byte, p.Wi*p.Hi)
	i := 0
	for y := p.Hi - 1; y >= 0; y-- {
		for x := range p.Wi {
			pixels[i] = p.Pixels[x+y*p.Wi]
			i++
		}
	}
	p.Pixels = pixels

	p.YOf = p.OHi - p.Hi - p.YOf
}

// was RGBAdjust
func (p *Pix8) RGBAdjust(arg0 int, arg1 int, arg2 int) {
	var6 := 0
	for i := range len(p.BPal) {
		var6 = (p.BPal[i] >> 16) & 0xFF
		var6 += arg0
		if var6 < 0 {
			var6 = 0
		} else if var6 > 0xFF {
			var6 = 0xFF
		}
		var7 := (p.BPal[i] >> 8) & 0xFF
		var7 += arg1
		if var7 < 0 {
			var7 = 0
		} else if var7 > 0xFF {
			var7 = 0xFF
		}
		var8 := p.BPal[i] & 0xFF
		var8 += arg2
		if var8 < 0 {
			var8 = 0
		} else if var8 > 0xFF {
			var8 = 0xFF
		}
		p.BPal[i] = (var6 << 16) + (var7 << 8) + var8
	}
}

// was Draw
func (p *Pix8) PlotSprite(y int, x int) {
	x += p.XOf
	y += p.YOf
	var4 := x + y*pix2d.Width2D
	var5 := 0
	var6 := p.Hi
	var7 := p.Wi
	var8 := pix2d.Width2D - var7
	var9 := 0
	var10 := 0
	if y < pix2d.Top {
		var10 = pix2d.Top - y
		var6 -= var10
		y = pix2d.Top
		var5 += var10 * var7
		var4 += var10 * pix2d.Width2D
	}
	if y+var6 > pix2d.Bottom {
		var6 -= y + var6 - pix2d.Bottom
	}
	if x < pix2d.Left {
		var10 = pix2d.Left - x
		var7 -= var10
		x = pix2d.Left
		var5 += var10
		var4 += var10
		var9 += var10
		var8 += var10
	}
	if x+var7 > pix2d.Right {
		var10 = x + var7 - pix2d.Right
		var7 -= var10
		var9 += var10
		var8 += var10
	}
	if var7 > 0 && var6 > 0 {
		p.Plot(pix2d.Data, var5, var9, p.Pixels, var6, 0, var7, var4, var8, p.BPal)
	}
}

// was CopyPixels
func (p *Pix8) Plot(arg0 []int, arg1 int, arg2 int, arg3 []byte, arg4 int, arg5 int, arg6 int, arg7 int, arg8 int, arg9 []int) {
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

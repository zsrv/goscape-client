package pixfont

import (
	"math"
	"math/rand"
	"strings"

	"goscape-client/pkg/jagex2/graphics/pix2d"
	"goscape-client/pkg/jagex2/io"
)

var CHAR_LOOKUP []int = make([]int, 256)

type PixFont struct {
	CharMask       [][]byte
	CharMaskWidth  []int
	CharMaskHeight []int
	CharOffsetX    []int
	CharOffsetY    []int
	CharAdvance    []int
	DrawWidth      []int
	Random         *rand.Rand
	Height         int
}

func init() {
	var0 := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!\"£$%^&*()-_=+[{]};:'@#~,<.>/?\\| "
	for i := range 256 {
		var2 := strings.IndexByte(var0, byte(i))
		if var2 == -1 {
			var2 = 74
		}
		CHAR_LOOKUP[i] = var2
	}
}

func NewPixFont(arg0 *io.Jagfile, arg1 string) *PixFont {
	p := &PixFont{
		CharMask:       make([][]byte, 94),
		CharMaskWidth:  make([]int, 94),
		CharMaskHeight: make([]int, 94),
		CharOffsetX:    make([]int, 94),
		CharOffsetY:    make([]int, 94),
		CharAdvance:    make([]int, 95),
		DrawWidth:      make([]int, 256),
	}

	var4 := io.NewPacket(arg0.Read(arg1+".dat", nil))
	var5 := io.NewPacket(arg0.Read("index.dat", nil))
	var5.Pos = var4.G2() + 4
	var6 := var5.G1()
	if var6 > 0 {
		var5.Pos += (var6 - 1) * 3
	}
	var8 := 0
	for i := range 94 {
		p.CharOffsetX[i] = int(var5.G1())
		p.CharOffsetY[i] = int(var5.G1())
		p.CharMaskWidth[i] = int(var5.G2())
		var8 = p.CharMaskWidth[i]
		p.CharMaskHeight[i] = int(var5.G2())
		var9 := p.CharMaskHeight[i]
		var10 := var5.G1()
		var11 := var8 * var9
		p.CharMask[i] = make([]byte, var11)
		if var10 == 0 {
			for j := range var11 {
				p.CharMask[i][j] = var4.G1B()
			}
		} else if var10 == 1 {
			for j := range var8 {
				for k := range var9 {
					p.CharMask[i][j+k*var8] = var4.G1B()
				}
			}
		}
		if var9 > p.Height {
			p.Height = var9
		}
		p.CharOffsetX[i] = 1
		p.CharAdvance[i] = var8 + 2
		var12 := 0
		for j := var9 / 7; j < var9; j++ {
			var12 += int(p.CharMask[i][j*var8])
		}
		if var12 <= var9/7 {
			p.CharAdvance[i]--
			p.CharOffsetX[i] = 0
		}
		var12 = 0
		for j := var9 / 7; j < var9; j++ {
			var12 += int(p.CharMask[i][var8-1+j*var8])
		}
		if var12 <= var9/7 {
			p.CharAdvance[i]--
		}
	}
	p.CharAdvance[94] = p.CharAdvance[8]
	for i := range 256 {
		p.DrawWidth[i] = p.CharAdvance[CHAR_LOOKUP[i]]
	}
	return p
}

func (p *PixFont) DrawStringCenter(arg0 int, arg2 int, arg3 string, arg4 int) {
	p.DrawString(arg4-p.StringWidth(arg3)/2, arg0, arg2, arg3)
}

func (p *PixFont) DrawStringTaggableCenter(arg0 int, arg1 int, arg2 bool, arg3 int, arg4 string) {
	p.DrawStringTaggable(arg0-p.StringWidth(arg4)/2, arg3, arg4, arg2, arg1)
}

func (p *PixFont) StringWidth(arg1 string) int {
	if arg1 == "" {
		return 0
	}
	var3 := 0
	for i := range len(arg1) {
		if arg1[i] == '@' && i+4 < len(arg1) && arg1[i+4] == '@' {
			i += 4
		} else {
			var3 += p.DrawWidth[arg1[i]]
		}
	}
	return var3
}

func (p *PixFont) DrawString(arg0 int, arg1 int, arg3 int, arg4 string) {
	if arg4 == "" {
		return
	}
	var8 := arg1 - p.Height
	for i := range len(arg4) {
		var7 := CHAR_LOOKUP[arg4[i]]
		if var7 != 94 {
			p.DrawChar(p.CharMask[var7], arg0+p.CharOffsetX[var7], var8+p.CharOffsetY[var7], p.CharMaskWidth[var7], p.CharMaskHeight[var7], arg3)
		}
		arg0 += p.CharAdvance[var7]
	}
}

func (p *PixFont) DrawCenteredWave(arg0 int, arg2 int, arg3 int, arg4 int, arg5 string) {
	if arg5 == "" {
		return
	}
	arg2 -= p.StringWidth(arg5) / 2
	var9 := arg3 - p.Height
	for i := range len(arg5) {
		var8 := CHAR_LOOKUP[arg5[i]]
		if var8 != 94 {
			p.DrawChar(p.CharMask[var8], arg2+p.CharOffsetX[var8], var9+p.CharOffsetY[var8]+int(math.Sin(float64(i)/20+float64(arg0)/5.0)*5.0), p.CharMaskWidth[var8], p.CharMaskHeight[var8], arg4)
		}
		arg2 += p.CharAdvance[var8]
	}
}

func (p *PixFont) DrawStringTaggable(arg0 int, arg2 int, arg3 string, arg4 bool, arg5 int) {
	if arg3 == "" {
		return
	}
	var9 := arg2 - p.Height
	for i := range len(arg3) {
		if arg3[i] == '@' && i+4 < len(arg3) && arg3[i+4] == '@' {
			arg5 = p.EvaluateTag(arg3[i+1 : i+4])
			i += 4
		} else {
			var8 := CHAR_LOOKUP[arg3[i]]
			if var8 != 94 {
				if arg4 {
					p.DrawChar(p.CharMask[var8], arg0+p.CharOffsetX[var8]+1, var9+p.CharOffsetY[var8]+1, p.CharMaskWidth[var8], p.CharMaskHeight[var8], 0)
				}
				p.DrawChar(p.CharMask[var8], arg0+p.CharOffsetX[var8], var9+p.CharOffsetY[var8], p.CharMaskWidth[var8], p.CharMaskHeight[var8], arg5)
			}
			arg0 += p.CharAdvance[var8]
		}
	}
}

func (p *PixFont) DrawStringTooltip(arg0 int, arg1 bool, arg3 int, arg4 int, arg5 string, arg6 int) {
	if arg5 == "" {
		return
	}
	p.Random = rand.New(rand.NewSource(int64(arg0)))
	var8 := (p.Random.Int() & 0x1F) + 192
	var11 := arg3 - p.Height
	for i := range len(arg5) {
		if arg5[i] == '@' && i+4 < len(arg5) && arg5[i+4] == '@' {
			arg4 = p.EvaluateTag(arg5[i+1 : i+4])
			i += 4
		} else {
			var10 := CHAR_LOOKUP[arg5[i]]
			if var10 != 94 {
				if arg1 {
					p.DrawCharAlpha(p.CharMask[var10], arg6+p.CharOffsetX[var10]+1, p.CharMaskHeight[var10], 0, var11+p.CharOffsetY[var10]+1, 192, p.CharMaskWidth[var10])
				}
				p.DrawCharAlpha(p.CharMask[var10], arg6+p.CharOffsetX[var10], p.CharMaskHeight[var10], arg4, var11+p.CharOffsetY[var10], var8, p.CharMaskWidth[var10])
			}
			arg6 += p.CharAdvance[var10]
			if p.Random.Int()&0x3 == 0 {
				arg6++
			}
		}
	}
}

func (p *PixFont) EvaluateTag(arg1 string) int {
	switch arg1 {
	case "red":
		return 16711680
	case "gre":
		return 65280
	case "blu":
		return 255
	case "yel":
		return 16776960
	case "cya":
		return 65535
	case "mag":
		return 16711935
	case "whi":
		return 16777215
	case "bla":
		return 0
	case "lre":
		return 16748608
	case "dre":
		return 8388608
	case "dbl":
		return 128
	case "or1":
		return 16756736
	case "or2":
		return 16740352
	case "or3":
		return 16723968
	case "gr1":
		return 12648192
	case "gr2":
		return 8453888
	case "gr3":
		return 4259584
	default:
		return 0
	}
}

func (p *PixFont) DrawChar(arg0 []byte, arg1, arg2, arg3, arg4, arg5 int) {
	var7 := arg1 + arg2*pix2d.Width2D
	var8 := pix2d.Width2D - arg3
	var9 := 0
	var10 := 0
	var11 := 0
	if arg2 < pix2d.BoundTop {
		var11 = pix2d.BoundTop - arg2
		arg4 -= var11
		arg2 = pix2d.BoundTop
		var10 += var11 * arg3
		var7 += var11 * pix2d.Width2D
	}
	if arg2+arg4 >= pix2d.BoundBottom {
		arg4 -= arg2 + arg4 - pix2d.BoundBottom + 1
	}
	if arg1 < pix2d.BoundLeft {
		var11 = pix2d.BoundLeft - arg1
		arg3 -= var11
		arg1 = pix2d.BoundLeft
		var10 += var11
		var7 += var11
		var9 += var11
		var8 += var11
	}
	if arg1+arg3 >= pix2d.BoundRight {
		var11 = arg1 + arg3 - pix2d.BoundRight + 1
		arg3 -= var11
		var9 += var11
		var8 += var11
	}
	if arg3 > 0 && arg4 > 0 {
		p.DrawMask(pix2d.Data, arg0, arg5, var10, var7, arg3, arg4, var8, var9)
	}
}

func (p *PixFont) DrawMask(arg0 []int, arg1 []byte, arg2, arg3, arg4, arg5, arg6, arg7, arg8 int) {
	var10 := -(arg5 >> 2)
	var14 := -(arg5 & 0x3)
	for i := -arg6; i < 0; i++ {
		for j := var10; j < 0; j++ {
			if arg1[arg3] == 0 {
				arg4++
			} else {
				arg0[arg4] = arg2
				arg4++
			}
			arg3++
			if arg1[arg3] == 0 {
				arg4++
			} else {
				arg0[arg4] = arg2
				arg4++
			}
			arg3++
			if arg1[arg3] == 0 {
				arg4++
			} else {
				arg0[arg4] = arg2
				arg4++
			}
			arg3++
			if arg1[arg3] == 0 {
				arg4++
			} else {
				arg0[arg4] = arg2
				arg4++
			}
			arg3++
		}
		for j := var14; j < 0; j++ {
			if arg1[arg3] == 0 {
				arg4++
			} else {
				arg0[arg4] = arg2
				arg4++
			}
			arg3++
		}
		arg4 += arg7
		arg3 += arg8
	}
}

func (p *PixFont) DrawCharAlpha(arg0 []byte, arg2, arg3, arg4, arg5, arg6, arg7 int) {
	var9 := arg2 + arg5*pix2d.Width2D
	var10 := pix2d.Width2D - arg7
	var11 := 0
	var12 := 0
	var13 := 0
	if arg5 < pix2d.BoundTop {
		var13 = pix2d.BoundTop - arg5
		arg3 -= var13
		arg5 = pix2d.BoundTop
		var12 += var13 * arg7
		var9 += var13 * pix2d.Width2D
	}
	if arg5+arg3 >= pix2d.BoundBottom {
		arg3 -= arg5 + arg3 - pix2d.BoundBottom + 1
	}
	if arg2 < pix2d.BoundLeft {
		var13 = pix2d.BoundLeft - arg2
		arg7 -= var13
		arg2 = pix2d.BoundLeft
		var12 += var13
		var9 += var13
		var11 += var13
		var10 += var13
	}
	if arg2+arg7 >= pix2d.BoundRight {
		var13 = arg2 + arg7 - pix2d.BoundRight + 1
		arg7 -= var13
		var11 += var13
		var10 += var13
	}
	if arg7 > 0 && arg3 > 0 {
		p.DrawMaskAlpha(arg3, var9, arg7, pix2d.Data, arg0, arg6, var12, var10, var11, arg4)
	}
}

func (p *PixFont) DrawMaskAlpha(arg0 int, arg1 int, arg2 int, arg3 []int, arg4 []byte, arg5 int, arg6 int, arg7 int, arg8 int, arg10 int) {
	var17 := ((((arg10 & 0xFF00FF) * arg5) & 0xFF00FF00) + (((arg10 & 0xFF00) * arg5) & 0xFF0000)) >> 8
	var15 := 256 - arg5
	for i := -arg0; i < 0; i++ {
		for j := -arg2; j < 0; j++ {
			if arg4[arg6] == 0 {
				arg1++
			} else {
				var14 := arg3[arg1]
				arg3[arg1] = (((((var14 & 0xFF00FF) * var15) & 0xFF00FF00) + (((var14 & 0xFF00) * var15) & 0xFF0000)) >> 8) + var17
				arg1++
			}
			arg6++
		}
		arg1 += arg7
		arg6 += arg8
	}
}

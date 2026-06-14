package pixfont

import (
	"math"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// javaRandom is a faithful java.util.Random 48-bit LCG (setSeed/nextInt
// subset). The anti-macro tooltip jitter must reproduce the Java client's
// exact sequence for a given seed, and Go's math/rand is an unrelated
// generator. Java: java.util.Random (PixFont.java:34 `rand`).
type javaRandom struct {
	seed int64
}

func (r *javaRandom) SetSeed(seed int64) {
	r.seed = (seed ^ 0x5DEECE66D) & (1<<48 - 1)
}

// NextInt advances the LCG and returns the high 32 bits as a signed Java int
// (java.util.Random.next(32)). The seed is masked to 48 bits, so Java's
// logical >>> equals Go's >> here.
func (r *javaRandom) NextInt() int {
	r.seed = (r.seed*0x5DEECE66D + 0xB) & (1<<48 - 1)
	return int(int32(r.seed >> 16))
}

// CHAR_LOOKUP maps a Latin-1 codepoint (0..255) to a glyph table index.
// Index 94 is the "no glyph" sentinel; index 74 is the catch-all/default
// (the space glyph in the alphabet below). Mirrors Java PixFont.CHAR_LOOKUP
// (PixFont.java:39, init at :381-390).
//
// Java's `arg1.charAt(i)` walks UTF-16 code units, so a char like '£'
// (U+00A3) is one lookup against this table. The Go port previously
// byte-indexed CHAR_LOOKUP[s[i]], which only works when `s` is invalid UTF-8
// (one byte per char, our previous wire-decoded shape). After GJStr now
// returns valid UTF-8, callers must iterate runes — see GlyphIndex below.
var CHAR_LOOKUP []int = make([]int, 256)

// GlyphIndex returns CHAR_LOOKUP[r] for code points in 0..255, and the
// catch-all sentinel (74, same as any other unmapped char in Java) for
// code points outside Latin-1. Use this everywhere a Java caller wrote
// `CHAR_LOOKUP[arg1.charAt(i)]`.
func GlyphIndex(r rune) int {
	if r >= 0 && r < 256 {
		return CHAR_LOOKUP[r]
	}
	return 74
}

type PixFont struct {
	// Java: charMask is byte[][] (signed int8). Stored as int8 so the
	// advance-trim sums (DrawChar setup) sign-extend each mask byte exactly
	// as Java's byte->int promotion does. Pixel reads only test `== 0`, so
	// signedness is irrelevant there.
	CharMask       [][]int8
	CharMaskWidth  []int
	CharMaskHeight []int
	CharOffsetX    []int
	CharOffsetY    []int
	CharAdvance    []int
	DrawWidth      []int
	Random         javaRandom
	// Java: PixFont.strikeout (PixFont.java:37) — set by the @str@ tag while
	// drawing; drawStringTag emits a dark-red strikethrough when it ends true.
	Strikeout bool
	Height    int
}

func init() {
	var0 := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!\"£$%^&*()-_=+[{]};:'@#~,<.>/?\\| ")
	for i := range 256 {
		//var2 := strings.IndexByte(var0, byte(i))
		//var2 := strings.IndexRune(var0, rune(i))
		var2 := -1
		for r := range var0 {
			if var0[r] == rune(i) {
				var2 = r
				break
			}
		}
		if var2 == -1 {
			var2 = 74
		}
		CHAR_LOOKUP[i] = var2
	}
}

func NewPixFont(arg0 *io.Jagfile, arg1 string) *PixFont {
	p := &PixFont{
		CharMask:       make([][]int8, 94),
		CharMaskWidth:  make([]int, 94),
		CharMaskHeight: make([]int, 94),
		CharOffsetX:    make([]int, 94),
		CharOffsetY:    make([]int, 94),
		CharAdvance:    make([]int, 95),
		DrawWidth:      make([]int, 256),
	}

	var4 := io.NewPacket(arg0.Read(arg1+".dat", nil))
	var5 := io.NewPacket(arg0.Read("index.dat", nil))

	var5.Pos = var4.G2() + 4 // skip height and width

	// skip palette
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
		var10 := var5.G1() // pixel order
		var11 := var8 * var9
		p.CharMask[i] = make([]int8, var11)
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
		p.Height = max(var9, p.Height)
		p.CharOffsetX[i] = 1
		p.CharAdvance[i] = var8 + 2
		// Java: var12 += this.charMask[var7][...] (PixFont.java:78,87) where
		// charMask is byte[][] (signed) — each mask byte sign-extends into the
		// sum. CharMask is now []int8 (see field decl), so int(...) below
		// sign-extends to match; a high-bit-set mask byte contributes a
		// negative value, flipping CharAdvance/CharOffsetX exactly as in Java.
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

func (p *PixFont) CentreString(arg0 int, hexColour int, arg3 string, arg4 int) {
	p.DrawString(arg4-p.StringWidth(arg3)/2, arg0, hexColour, arg3)
}

func (p *PixFont) DrawStringTaggableCenter(arg0 int, arg1 int, arg2 bool, arg3 int, arg4 string) {
	p.DrawStringTaggable(arg0-p.StringWidth(arg4)/2, arg3, arg4, arg2, arg1)
}

// StringWidth returns the rendered pixel width of arg1. Java walks code
// units via `arg1.charAt(var4)` (PixFont.java:115-122), so we walk runes
// here — byte-indexing a Go (UTF-8) string would mis-handle '£' and any
// other Latin-1 char produced by GJStr after the wire→UTF-8 transcode.
func (p *PixFont) StringWidth(arg1 string) int {
	if arg1 == "" {
		return 0
	}
	runes := []rune(arg1)
	var3 := 0
	// `for i := 0; i < N; i++` (not `range len(...)`): the `i += 4` below must
	// advance the loop counter to skip a `@xxx@` tag; range-over-int rebinds
	// i each iteration and would silently drop the skip.
	for i := 0; i < len(runes); i++ {
		if runes[i] == '@' && i+4 < len(runes) && runes[i+4] == '@' {
			i += 4
		} else {
			// DrawWidth is 256 wide, keyed by Latin-1 codepoint. Map any
			// out-of-range rune to a safe fallback (same fallback CHAR_LOOKUP
			// uses for unmapped chars — codepoint 0, which lands on glyph 74,
			// the catch-all space-ish width).
			r := runes[i]
			if r < 0 || r >= 256 {
				r = 0
			}
			var3 += p.DrawWidth[r]
		}
	}
	return var3
}

// DrawString renders arg4 at (arg0, arg1) in arg3. Java walks code units via
// `arg4.charAt(var6)` (PixFont.java:131); the Go port walks runes for the
// same reason as StringWidth.
func (p *PixFont) DrawString(arg0 int, arg1 int, arg3 int, arg4 string) {
	if arg4 == "" {
		return
	}
	var8 := arg1 - p.Height
	for _, r := range arg4 {
		var7 := GlyphIndex(r)
		if var7 != 94 {
			p.DrawChar(p.CharMask[var7], arg0+p.CharOffsetX[var7], var8+p.CharOffsetY[var7], p.CharMaskWidth[var7], p.CharMaskHeight[var7], arg3)
		}
		arg0 += p.CharAdvance[var7]
	}
}

// DrawCenteredWave renders arg5 with a sinusoidal vertical offset. The
// `var7` index in Java (PixFont.java:148) is the char-position into the
// string; in Go we iterate rune-by-rune and use the rune ordinal so the
// wave phase matches Java even when the input contains multi-byte chars.
func (p *PixFont) DrawCenteredWave(arg0 int, arg2 int, arg3 int, arg4 int, arg5 string) {
	if arg5 == "" {
		return
	}
	arg2 -= p.StringWidth(arg5) / 2
	var9 := arg3 - p.Height
	i := 0
	for _, r := range arg5 {
		var8 := GlyphIndex(r)
		if var8 != 94 {
			p.DrawChar(p.CharMask[var8], arg2+p.CharOffsetX[var8], var9+p.CharOffsetY[var8]+int(math.Sin(float64(i)/2.0+float64(arg0)/5.0)*5.0), p.CharMaskWidth[var8], p.CharMaskHeight[var8], arg4)
		}
		arg2 += p.CharAdvance[var8]
		i++
	}
}

// DrawStringTaggable renders arg3, interpreting `@xxx@` 3-char tag sequences
// as color escapes (PixFont.java:158-178). Java walks code units; the Go port
// walks runes via a []rune so the i+4 lookahead matches Java exactly when
// the string contains non-ASCII chars like '£'.
func (p *PixFont) DrawStringTaggable(arg0 int, arg2 int, arg3 string, arg4 bool, arg5 int) {
	p.Strikeout = false // Java: PixFont.java:174 — reset before the null check
	var7 := arg0        // Java: var7 = start x for the strikethrough width
	if arg3 == "" {
		return
	}
	runes := []rune(arg3)
	var9 := arg2 - p.Height
	// C-style loop required: `i += 4` below must skip a `@xxx@` tag.
	for i := 0; i < len(runes); i++ {
		if runes[i] == '@' && i+4 < len(runes) && runes[i+4] == '@' {
			// Java: unknown tags return -1 and leave the colour unchanged.
			var10 := p.EvaluateTag(string(runes[i+1 : i+4]))
			if var10 != -1 {
				arg5 = var10
			}
			i += 4
		} else {
			var8 := GlyphIndex(runes[i])
			if var8 != 94 {
				if arg4 {
					p.DrawChar(p.CharMask[var8], arg0+p.CharOffsetX[var8]+1, var9+p.CharOffsetY[var8]+1, p.CharMaskWidth[var8], p.CharMaskHeight[var8], 0)
				}
				p.DrawChar(p.CharMask[var8], arg0+p.CharOffsetX[var8], var9+p.CharOffsetY[var8], p.CharMaskWidth[var8], p.CharMaskHeight[var8], arg5)
			}
			arg0 += p.CharAdvance[var8]
		}
	}
	// Java: PixFont.java:199-201 — dark-red strikethrough across the rendered
	// text when an @str@ tag was seen.
	if p.Strikeout {
		pix2d.HLine(8388608, int(float64(p.Height)*0.7)+var9, arg0-var7, var7)
	}
}

// DrawStringTooltip renders arg5 with jitter for tooltip popups, supporting
// `@xxx@` color tags (PixFont.java:181-206). Walks runes; see
// DrawStringTaggable for the rationale.
func (p *PixFont) DrawStringTooltip(arg0 int, arg1 bool, arg3 int, arg4 int, arg5 string, arg6 int) {
	if arg5 == "" {
		return
	}
	// Java: this.rand.setSeed((long) arg1) — java.util.Random LCG, ported
	// exactly so the per-call alpha and per-char jitter match the Java client.
	p.Random.SetSeed(int64(arg0))
	var8 := (p.Random.NextInt() & 0x1F) + 192
	var11 := arg3 - p.Height
	runes := []rune(arg5)
	// C-style loop required: `i += 4` below must skip a `@xxx@` tag.
	for i := 0; i < len(runes); i++ {
		if runes[i] == '@' && i+4 < len(runes) && runes[i+4] == '@' {
			// Java: unknown tags return -1 and leave the colour unchanged.
			var12 := p.EvaluateTag(string(runes[i+1 : i+4]))
			if var12 != -1 {
				arg4 = var12
			}
			i += 4
		} else {
			var10 := GlyphIndex(runes[i])
			if var10 != 94 {
				if arg1 {
					p.DrawCharAlpha(p.CharMask[var10], arg6+p.CharOffsetX[var10]+1, p.CharMaskHeight[var10], 0, var11+p.CharOffsetY[var10]+1, 192, p.CharMaskWidth[var10])
				}
				p.DrawCharAlpha(p.CharMask[var10], arg6+p.CharOffsetX[var10], p.CharMaskHeight[var10], arg4, var11+p.CharOffsetY[var10], var8, p.CharMaskWidth[var10])
			}
			arg6 += p.CharAdvance[var10]
			if p.Random.NextInt()&0x3 == 0 {
				arg6++
			}
		}
	}
}

func (p *PixFont) EvaluateTag(arg1 string) int {
	switch arg1 {
	case "red":
		return 0xFF0000
	case "gre":
		return 0xFF00
	case "blu":
		return 0xFF
	case "yel":
		return 0xFFFF00
	case "cya":
		return 0xFFFF
	case "mag":
		return 0xFF00FF
	case "whi":
		return 0xFFFFFF
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
		// Java: PixFont.java:272-276 — "str" sets the strikeout side effect;
		// every unrecognized tag (including "str") yields the -1 sentinel so
		// callers keep the current colour.
		if arg1 == "str" {
			p.Strikeout = true
		}
		return -1
	}
}

func (p *PixFont) DrawChar(arg0 []int8, arg1, arg2, arg3, arg4, arg5 int) {
	var7 := arg1 + arg2*pix2d.Width2D
	var8 := pix2d.Width2D - arg3
	var9 := 0
	var10 := 0
	var11 := 0
	if arg2 < pix2d.ClipMinY {
		var11 = pix2d.ClipMinY - arg2
		arg4 -= var11
		arg2 = pix2d.ClipMinY
		var10 += var11 * arg3
		var7 += var11 * pix2d.Width2D
	}
	if arg2+arg4 >= pix2d.ClipMaxY {
		arg4 -= arg2 + arg4 - pix2d.ClipMaxY + 1
	}
	if arg1 < pix2d.ClipMinX {
		var11 = pix2d.ClipMinX - arg1
		arg3 -= var11
		arg1 = pix2d.ClipMinX
		var10 += var11
		var7 += var11
		var9 += var11
		var8 += var11
	}
	if arg1+arg3 >= pix2d.ClipMaxX {
		var11 = arg1 + arg3 - pix2d.ClipMaxX + 1
		arg3 -= var11
		var9 += var11
		var8 += var11
	}
	if arg3 > 0 && arg4 > 0 {
		p.DrawMask(pix2d.Data, arg0, arg5, var10, var7, arg3, arg4, var8, var9)
	}
}

func (p *PixFont) DrawMask(arg0 []int, arg1 []int8, arg2, arg3, arg4, arg5, arg6, arg7, arg8 int) {
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

func (p *PixFont) DrawCharAlpha(arg0 []int8, arg2, arg3, arg4, arg5, arg6, arg7 int) {
	var9 := arg2 + arg5*pix2d.Width2D
	var10 := pix2d.Width2D - arg7
	var11 := 0
	var12 := 0
	var13 := 0
	if arg5 < pix2d.ClipMinY {
		var13 = pix2d.ClipMinY - arg5
		arg3 -= var13
		arg5 = pix2d.ClipMinY
		var12 += var13 * arg7
		var9 += var13 * pix2d.Width2D
	}
	if arg5+arg3 >= pix2d.ClipMaxY {
		arg3 -= arg5 + arg3 - pix2d.ClipMaxY + 1
	}
	if arg2 < pix2d.ClipMinX {
		var13 = pix2d.ClipMinX - arg2
		arg7 -= var13
		arg2 = pix2d.ClipMinX
		var12 += var13
		var9 += var13
		var11 += var13
		var10 += var13
	}
	if arg2+arg7 >= pix2d.ClipMaxX {
		var13 = arg2 + arg7 - pix2d.ClipMaxX + 1
		arg7 -= var13
		var11 += var13
		var10 += var13
	}
	if arg7 > 0 && arg3 > 0 {
		p.DrawMaskAlpha(arg3, var9, arg7, pix2d.Data, arg0, arg6, var12, var10, var11, arg4)
	}
}

func (p *PixFont) DrawMaskAlpha(arg0 int, arg1 int, arg2 int, arg3 []int, arg4 []int8, arg5 int, arg6 int, arg7 int, arg8 int, arg10 int) {
	// Java: PixFont.java:424 — 32-bit blend sum; arithmetic >>8 sign-extends the
	// top byte when bit 31 is set (audit pixfont-01)
	var17 := int(int32((((arg10&0xFF00FF)*arg5)&0xFF00FF00)+(((arg10&0xFF00)*arg5)&0xFF0000))) >> 8
	var15 := 256 - arg5
	for i := -arg0; i < 0; i++ {
		for j := -arg2; j < 0; j++ {
			if arg4[arg6] == 0 {
				arg1++
			} else {
				var14 := arg3[arg1]
				// Java: PixFont.java:437 — same 32-bit blend; final store wraps too
				arg3[arg1] = int(int32((int(int32((((var14&0xFF00FF)*var15)&0xFF00FF00)+(((var14&0xFF00)*var15)&0xFF0000))) >> 8) + var17))
				arg1++
			}
			arg6++
		}
		arg1 += arg7
		arg6 += arg8
	}
}

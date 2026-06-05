package pixfont

import (
	"math"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// javaRandom is a faithful java.util.Random 48-bit LCG (setSeed/nextInt
// subset). The anti-macro tooltip jitter must reproduce the Java client's
// exact sequence for a given seed, and Go's math/rand is an unrelated
// generator. Java: java.util.Random (PixFont.java:31 `rand` @32f3062).
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

// charIndex maps a rune to the 0..255 per-char table index. Java 274 indexes
// the [256] tables directly with the UTF-16 code unit (e.g. PixFont.java:115
// @32f3062, `charAdvance[arg1.charAt(var4)]`) and would AIOOBE on any char
// >= 256; game strings are Latin-1 (wire bytes -> UTF-8 transcode), so that
// never fires in practice. The Go port walks runes and clamps out-of-Latin-1
// code points to ' ' (the skip sentinel, advancing by the space width)
// instead of panicking — the same defensive deviation the <=254 port made
// via its GlyphIndex helper (274 deleted CHAR_LOOKUP and the 0..93 glyph
// indirection it served; per-char arrays are now keyed by raw char code).
func charIndex(r rune) int {
	if r >= 0 && r < 256 {
		return int(r)
	}
	return ' '
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
	Rand           javaRandom
	// Java: PixFont.strikeout (PixFont.java:34 @32f3062) — set by the @str@
	// tag while drawing; drawStringTag emits a dark-red strikethrough when it
	// ends true.
	Strikeout bool
	Height    int // Java: height (274 renames 254's height2d)
}

// NewPixFont loads the named font archive entry. Java 274 ctor
// PixFont(JagFile, boolean wide, String name, byte) (PixFont.java:39-93
// @32f3062) — the trailing byte is an unread obfuscation dummy, dropped here.
// 254's (String, JagFile) order and its 94-glyph table are gone: 274 reads
// 256 glyph entries and indexes them directly by char code. arg1 ("wide")
// selects the space advance: 'I' (73) when true (q8), else 'i' (105).
func NewPixFont(arg0 *io.JagFile, arg1 bool, arg2 string) *PixFont {
	p := &PixFont{
		CharMask:       make([][]int8, 256),
		CharMaskWidth:  make([]int, 256),
		CharMaskHeight: make([]int, 256),
		CharOffsetX:    make([]int, 256),
		CharOffsetY:    make([]int, 256),
		CharAdvance:    make([]int, 256),
	}

	var5 := io.NewPacket(arg0.Read(arg2+".dat", nil))
	var6 := io.NewPacket(arg0.Read("index.dat", nil))

	// Java: var6.data = var5.g2() + 4 (PixFont.java:42 @32f3062) — 274's
	// `data` is the CURSOR (254 `pos`); Go keeps Pos=cursor / Data=buffer
	// (trap: Packet data<->pos name swap, see io.Packet).
	var6.Pos = var5.G2() + 4 // skip height and width

	// skip palette
	var7 := var6.G1()
	if var7 > 0 {
		var6.Pos += (var7 - 1) * 3
	}

	// Java: 274 reads 256 glyph entries (PixFont.java:47, `var8 < 256`);
	// 254 read 94.
	for i := range 256 {
		p.CharOffsetX[i] = int(var6.G1())
		p.CharOffsetY[i] = int(var6.G1())
		p.CharMaskWidth[i] = int(var6.G2())
		var9 := p.CharMaskWidth[i]
		p.CharMaskHeight[i] = int(var6.G2())
		var10 := p.CharMaskHeight[i]
		var11 := var6.G1() // pixel order
		var12 := var9 * var10
		p.CharMask[i] = make([]int8, var12)
		if var11 == 0 {
			for j := range var12 {
				p.CharMask[i][j] = var5.G1B()
			}
		} else if var11 == 1 {
			for j := range var9 {
				for k := range var10 {
					p.CharMask[i][j+k*var9] = var5.G1B()
				}
			}
		}
		// Java: PixFont.java:66-68 @32f3062 — NEW in 274: only glyphs with
		// index < 128 contribute to the font height (254 let all 94 in).
		if var10 > p.Height && i < 128 {
			p.Height = var10
		}
		p.CharOffsetX[i] = 1
		p.CharAdvance[i] = var9 + 2
		// Java: var16 += charMask[var8][...] (PixFont.java:73,82 @32f3062)
		// where charMask is byte[][] (signed) — each mask byte sign-extends
		// into the sum. CharMask is []int8 (see field decl), so int(...)
		// below sign-extends to match; a high-bit-set mask byte contributes
		// a negative value, flipping CharAdvance/CharOffsetX exactly as in
		// Java.
		var16 := 0
		for j := var10 / 7; j < var10; j++ {
			var16 += int(p.CharMask[i][j*var9])
		}
		if var16 <= var10/7 {
			p.CharAdvance[i]--
			p.CharOffsetX[i] = 0
		}
		var18 := 0
		for j := var10 / 7; j < var10; j++ {
			var18 += int(p.CharMask[i][var9-1+j*var9])
		}
		if var18 <= var10/7 {
			p.CharAdvance[i]--
		}
	}
	// Java: PixFont.java:88-92 @32f3062 — space is ASCII 32 now (254 used
	// glyph slot 94); its advance copies from 'I' (73) when wide, else 'i'
	// (105). 254's drawWidth[256] precompute table is gone entirely.
	if arg1 {
		p.CharAdvance[32] = p.CharAdvance[73]
	} else {
		p.CharAdvance[32] = p.CharAdvance[105]
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
// units via `arg1.charAt(var4)` and sums `charAdvance[char]` directly
// (PixFont.java:106-120 @32f3062 stringWid — 254's drawWidth/CHAR_LOOKUP
// indirection is gone); we walk runes here — byte-indexing a Go (UTF-8)
// string would mis-handle '£' and any other Latin-1 char produced by GStr
// after the wire->UTF-8 transcode.
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
			var3 += p.CharAdvance[charIndex(runes[i])]
		}
	}
	return var3
}

// DrawString renders arg4 at (arg0, arg1) in arg3. Java walks code units via
// `arg1.charAt(var7)` (PixFont.java:128 @32f3062); the Go port walks runes
// for the same reason as StringWidth. 274 indexes the per-char arrays
// directly by char and skips ' ' (254 mapped through CHAR_LOOKUP with glyph
// 94 as the skip sentinel).
func (p *PixFont) DrawString(arg0 int, arg1 int, arg3 int, arg4 string) {
	if arg4 == "" {
		return
	}
	var6 := arg1 - p.Height
	for _, r := range arg4 {
		var8 := charIndex(r)
		if var8 != ' ' {
			p.DrawChar(p.CharMask[var8], arg0+p.CharOffsetX[var8], var6+p.CharOffsetY[var8], p.CharMaskWidth[var8], p.CharMaskHeight[var8], arg3)
		}
		arg0 += p.CharAdvance[var8]
	}
}

// DrawCenteredWave renders arg5 with a sinusoidal vertical offset. The
// `var9` index in Java (PixFont.java:146 @32f3062 centreStringWave) is the
// char-position into the string; in Go we iterate rune-by-rune and use the
// rune ordinal so the wave phase matches Java even when the input contains
// multi-byte chars.
func (p *PixFont) DrawCenteredWave(arg0 int, arg2 int, arg3 int, arg4 int, arg5 string) {
	if arg5 == "" {
		return
	}
	arg2 -= p.StringWidth(arg5) / 2
	var8 := arg3 - p.Height
	i := 0
	for _, r := range arg5 {
		var10 := charIndex(r)
		if var10 != ' ' {
			p.DrawChar(p.CharMask[var10], arg2+p.CharOffsetX[var10], var8+p.CharOffsetY[var10]+int(math.Sin(float64(i)/2.0+float64(arg0)/5.0)*5.0), p.CharMaskWidth[var10], p.CharMaskHeight[var10], arg4)
		}
		arg2 += p.CharAdvance[var10]
		i++
	}
}

// DrawStringTaggable renders arg3, interpreting `@xxx@` 3-char tag sequences
// as color escapes (PixFont.java:153-186 @32f3062 drawStringTag). Java walks
// code units; the Go port walks runes via a []rune so the i+4 lookahead
// matches Java exactly when the string contains non-ASCII chars like '£'.
func (p *PixFont) DrawStringTaggable(arg0 int, arg2 int, arg3 string, arg4 bool, arg5 int) {
	p.Strikeout = false // Java: PixFont.java:154 @32f3062 — reset before the null check
	var7 := arg0        // Java: var7 = start x for the strikethrough width
	if arg3 == "" {
		return
	}
	runes := []rune(arg3)
	var8 := arg2 - p.Height
	// C-style loop required: `i += 4` below must skip a `@xxx@` tag.
	for i := 0; i < len(runes); i++ {
		if runes[i] == '@' && i+4 < len(runes) && runes[i+4] == '@' {
			// Java: unknown tags return -1 and leave the colour unchanged.
			var10 := p.UpdateState(string(runes[i+1 : i+4]))
			if var10 != -1 {
				arg5 = var10
			}
			i += 4
		} else {
			var11 := charIndex(runes[i])
			if var11 != ' ' {
				if arg4 {
					p.DrawChar(p.CharMask[var11], arg0+p.CharOffsetX[var11]+1, var8+p.CharOffsetY[var11]+1, p.CharMaskWidth[var11], p.CharMaskHeight[var11], 0)
				}
				p.DrawChar(p.CharMask[var11], arg0+p.CharOffsetX[var11], var8+p.CharOffsetY[var11], p.CharMaskWidth[var11], p.CharMaskHeight[var11], arg5)
			}
			arg0 += p.CharAdvance[var11]
		}
	}
	// Java: PixFont.java:178-180 @32f3062 — dark-red strikethrough across the
	// rendered text when an @str@ tag was seen.
	if p.Strikeout {
		pix2d.HLine(8388608, int(float64(p.Height)*0.7)+var8, arg0-var7, var7)
	}
}

// DrawStringTooltip renders arg5 with jitter for tooltip popups, supporting
// `@xxx@` color tags (PixFont.java:184-212 @32f3062 drawStringAntiMacro).
// Walks runes; see DrawStringTaggable for the rationale.
func (p *PixFont) DrawStringTooltip(arg0 int, arg1 bool, arg3 int, arg4 int, arg5 string, arg6 int) {
	if arg5 == "" {
		return
	}
	// Java: rand.setSeed((long) arg5) — java.util.Random LCG, ported exactly
	// so the per-call alpha and per-char jitter match the Java client.
	p.Rand.SetSeed(int64(arg0))
	var8 := (p.Rand.NextInt() & 0x1F) + 192
	var9 := arg3 - p.Height
	runes := []rune(arg5)
	// C-style loop required: `i += 4` below must skip a `@xxx@` tag.
	for i := 0; i < len(runes); i++ {
		if runes[i] == '@' && i+4 < len(runes) && runes[i+4] == '@' {
			// Java: unknown tags return -1 and leave the colour unchanged.
			var11 := p.UpdateState(string(runes[i+1 : i+4]))
			if var11 != -1 {
				arg4 = var11
			}
			i += 4
		} else {
			var12 := charIndex(runes[i])
			if var12 != ' ' {
				if arg1 {
					p.DrawCharAlpha(p.CharMask[var12], arg6+p.CharOffsetX[var12]+1, p.CharMaskHeight[var12], 0, var9+p.CharOffsetY[var12]+1, 192, p.CharMaskWidth[var12])
				}
				p.DrawCharAlpha(p.CharMask[var12], arg6+p.CharOffsetX[var12], p.CharMaskHeight[var12], arg4, var9+p.CharOffsetY[var12], var8, p.CharMaskWidth[var12])
			}
			arg6 += p.CharAdvance[var12]
			if p.Rand.NextInt()&0x3 == 0 {
				arg6++
			}
		}
	}
}

// UpdateState resolves a 3-char `@xxx@` tag body to its RGB colour.
// Java: updateState (PixFont.java:215-256 @32f3062; 254 named it
// evaluateTag — rename only, body unchanged).
func (p *PixFont) UpdateState(arg0 string) int {
	switch arg0 {
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
		// Java: PixFont.java:251-254 @32f3062 — "str" sets the strikeout
		// side effect; every unrecognized tag (including "str") yields the
		// -1 sentinel so callers keep the current colour.
		if arg0 == "str" {
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
	if arg2 < pix2d.Top {
		var11 = pix2d.Top - arg2
		arg4 -= var11
		arg2 = pix2d.Top
		var10 += var11 * arg3
		var7 += var11 * pix2d.Width2D
	}
	if arg2+arg4 >= pix2d.Bottom {
		arg4 -= arg2 + arg4 - pix2d.Bottom + 1
	}
	if arg1 < pix2d.Left {
		var11 = pix2d.Left - arg1
		arg3 -= var11
		arg1 = pix2d.Left
		var10 += var11
		var7 += var11
		var9 += var11
		var8 += var11
	}
	if arg1+arg3 >= pix2d.Right {
		var11 = arg1 + arg3 - pix2d.Right + 1
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
	if arg5 < pix2d.Top {
		var13 = pix2d.Top - arg5
		arg3 -= var13
		arg5 = pix2d.Top
		var12 += var13 * arg7
		var9 += var13 * pix2d.Width2D
	}
	if arg5+arg3 >= pix2d.Bottom {
		arg3 -= arg5 + arg3 - pix2d.Bottom + 1
	}
	if arg2 < pix2d.Left {
		var13 = pix2d.Left - arg2
		arg7 -= var13
		arg2 = pix2d.Left
		var12 += var13
		var9 += var13
		var11 += var13
		var10 += var13
	}
	if arg2+arg7 >= pix2d.Right {
		var13 = arg2 + arg7 - pix2d.Right + 1
		arg7 -= var13
		var11 += var13
		var10 += var13
	}
	if arg7 > 0 && arg3 > 0 {
		p.DrawMaskAlpha(arg3, var9, arg7, pix2d.Data, arg0, arg6, var12, var10, var11, arg4)
	}
}

func (p *PixFont) DrawMaskAlpha(arg0 int, arg1 int, arg2 int, arg3 []int, arg4 []int8, arg5 int, arg6 int, arg7 int, arg8 int, arg10 int) {
	// Java: PixFont.java:371 @32f3062 — 32-bit blend sum; arithmetic >>8
	// sign-extends the top byte when bit 31 is set (audit pixfont-01)
	var12 := int(int32((((arg10&0xFF00FF)*arg5)&0xFF00FF00)+(((arg10&0xFF00)*arg5)&0xFF0000))) >> 8
	var13 := 256 - arg5
	for i := -arg0; i < 0; i++ {
		for j := -arg2; j < 0; j++ {
			if arg4[arg6] == 0 {
				arg1++
			} else {
				var16 := arg3[arg1]
				// Java: PixFont.java:379 @32f3062 — same 32-bit blend; final store wraps too
				arg3[arg1] = int(int32((int(int32((((var16&0xFF00FF)*var13)&0xFF00FF00)+(((var16&0xFF00)*var13)&0xFF0000))) >> 8) + var12))
				arg1++
			}
			arg6++
		}
		arg1 += arg7
		arg6 += arg8
	}
}

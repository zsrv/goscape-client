// Package bootfont renders text during the boot phase before c.JagTitle
// (and thus the RuneScape pixel fonts in pixfont) has been loaded. It
// wraps golang.org/x/image/font/basicfont.Face7x13, a monospace 7x13
// font shipped in x/image. Used exclusively by DrawProgressGameShell.
package bootfont

import (
	"image"
	"image/color"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"goscape-client/pkg/jagex2/graphics/pixmap"
)

// Height returns the font's inter-line height in pixels.
func Height() int {
	return basicfont.Face7x13.Height
}

// StringWidth returns the rendered pixel width of s, assuming
// basicfont.Face7x13's fixed 7-pixel advance per glyph.
func StringWidth(s string) int {
	return utf8.RuneCountInString(s) * basicfont.Face7x13.Advance
}

// DrawString rasterizes s onto p starting with the baseline at (x, y),
// in the given 0x00RRGGBB color. Matches AWT Graphics.drawString
// semantics: y is the glyph baseline, not the top of the box.
//
// The renderer rasterizes the glyphs into a temp image.NRGBA sized to
// the message bounding box, then copies set pixels into p.Data as
// 0x00RRGGBB ints. Runs at most a few times per boot frame, so the
// allocation cost is not in the hot path.
func DrawString(p *pixmap.PixMap, x, y, hexColor int, s string) {
	if s == "" {
		return
	}

	width := StringWidth(s)
	height := basicfont.Face7x13.Ascent + basicfont.Face7x13.Descent
	if width <= 0 || height <= 0 {
		return
	}

	src := image.NewNRGBA(image.Rect(0, 0, width, height))
	drawer := font.Drawer{
		Dst: src,
		Src: image.NewUniform(color.NRGBA{
			R: uint8(hexColor >> 16),
			G: uint8(hexColor >> 8),
			B: uint8(hexColor),
			A: 0xFF,
		}),
		Face: basicfont.Face7x13,
		Dot: fixed.Point26_6{
			X: fixed.I(0),
			Y: fixed.I(basicfont.Face7x13.Ascent),
		},
	}
	drawer.DrawString(s)

	// Copy non-transparent pixels into the PixMap, offset so that the
	// drawer-baseline (Ascent rows down in src) lands at y in p.
	topLeftY := y - basicfont.Face7x13.Ascent
	for srcY := range height {
		dstY := topLeftY + srcY
		if dstY < 0 || dstY >= p.Height {
			continue
		}
		for srcX := range width {
			dstX := x + srcX
			if dstX < 0 || dstX >= p.Width {
				continue
			}
			off := (srcY*width + srcX) * 4
			// basicfont.Face7x13 has a 1-bit mask, so every rasterized
			// pixel is either alpha 0 (skip) or alpha 255 (write). Treat
			// "below half" as transparent in case a future face is wired
			// through with antialiased glyphs.
			if src.Pix[off+3] < 128 {
				continue
			}
			p.Data[dstY*p.Width+dstX] = hexColor
		}
	}
}

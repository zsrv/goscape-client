// Package errorfont renders the text on DrawError's error screens using the
// "Go" typeface (golang.org/x/image/font/gofont/gobold, embedded in x/image)
// via the opentype rasterizer. It is a proportional bold face chosen to
// approximate the Helvetica BOLD the original Java client used for these
// screens (deob/client.java drawError; GameShell.java:541). The TTF is
// compiled in, so this font is always available even when an error fires
// before the RuneScape cache fonts (pixfont) have loaded — the situation that
// made DrawError dereference a nil *PixFont.
//
// Like bootfont (and unlike pixfont, which writes through pix2d package
// globals set by a prior Bind), DrawString takes an explicit *PixMap and
// blends glyph coverage straight into its int-packed (0x00RRGGBB) buffer, so
// it needs no pix2d binding lifecycle. It is deliberately separate from
// bootfont, which keeps the monospace basicfont for the boot/progress screen:
// only the error screens want the Helvetica-like face.
package errorfont

import (
	"image"
	"image/color"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pixmap"
)

// sizePt is the face size in points; rendered at 72 DPI so 1pt == 1px. ~16px
// approximates Java's Helvetica BOLD error text while keeping the long body
// lines within the 789px screen width.
const sizePt = 16

var (
	faceOnce sync.Once
	face     font.Face
	ascent   int
	descent  int
)

// loadFace lazily parses the embedded Go Bold TTF and builds the shared face.
// The data is compiled in, so a parse failure is a programmer error rather
// than a runtime condition — hence panic. The face is used only from the
// single game/draw goroutine, so the one-time init needs no further locking.
func loadFace() font.Face {
	faceOnce.Do(func() {
		f, err := opentype.Parse(gobold.TTF)
		if err != nil {
			panic("errorfont: parse gobold: " + err.Error())
		}
		face, err = opentype.NewFace(f, &opentype.FaceOptions{
			Size:    sizePt,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err != nil {
			panic("errorfont: new face: " + err.Error())
		}
		m := face.Metrics()
		ascent = m.Ascent.Ceil()
		descent = m.Descent.Ceil()
	})
	return face
}

// DrawString rasterizes s onto p with the baseline at (x, y) in the given
// 0x00RRGGBB color (AWT Graphics.drawString semantics: y is the baseline, not
// the top of the box). Antialiased glyph coverage is alpha-blended over the
// existing pixels so the proportional font's edges composite smoothly. Called
// at most a few times per error frame, so the per-call allocation is not hot.
func DrawString(p *pixmap.PixMap, x, y, hexColor int, s string) {
	if s == "" {
		return
	}
	f := loadFace()

	// Pad the bounding box a few px on the right so a bold glyph's overhang
	// (advance < inked width) is not clipped; trailing transparent columns are
	// skipped by the alpha==0 test below.
	width := font.MeasureString(f, s).Ceil() + 4
	height := ascent + descent
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
		Face: f,
		Dot:  fixed.Point26_6{X: fixed.I(0), Y: fixed.I(ascent)},
	}
	drawer.DrawString(s)

	sr := hexColor >> 16 & 0xFF
	sg := hexColor >> 8 & 0xFF
	sb := hexColor & 0xFF

	// Blend set pixels into p, offset so the drawer baseline (ascent rows down
	// in src) lands at y in p. image.NewNRGBA with a (0,0)-origin rect has
	// Stride == 4*width, so the per-pixel offset is (row*width + col)*4.
	topLeftY := y - ascent
	for srcY := range height {
		dstY := topLeftY + srcY
		if dstY < 0 || dstY >= p.Height {
			continue
		}
		row := srcY * width
		for srcX := range width {
			a := int(src.Pix[(row+srcX)*4+3])
			if a == 0 {
				continue
			}
			dstX := x + srcX
			if dstX < 0 || dstX >= p.Width {
				continue
			}
			idx := dstY*p.Width + dstX
			if a >= 0xFF {
				p.Data[idx] = hexColor & 0xFFFFFF
				continue
			}
			d := p.Data[idx]
			dr := d >> 16 & 0xFF
			dg := d >> 8 & 0xFF
			db := d & 0xFF
			r := (sr*a + dr*(255-a)) / 255
			g := (sg*a + dg*(255-a)) / 255
			b := (sb*a + db*(255-a)) / 255
			p.Data[idx] = r<<16 | g<<8 | b
		}
	}
}

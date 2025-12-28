package pixmap

import (
	"image"

	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// TODO

// PixMap is a CPU-side pixel buffer that can be efficiently uploaded to GPU.
type PixMap struct {
	Pixels []byte
	Width  int
	Height int
	Image  *image.RGBA
	Op     paint.ImageOp
	Ready  bool
}

// NewPixMap allocates a width*height pixel buffer.
func NewPixMap(width, height int) *PixMap {
	pix := image.NewRGBA(image.Rect(0, 0, width, height))
	return &PixMap{
		Width:  width,
		Height: height,
		Pixels: pix.Pix,
		Image:  pix,
	}
}

// Bind uploads the current pixel data to GPU.
// Call this once per frame before drawing.
func (p *PixMap) Bind() {
	p.Op = paint.NewImageOp(p.Image)
	p.Ready = true
}

// Draw adds the necessary operations to render the buffer at (x,y).
// Must be called between op.Ops{}.Reset() and window.Event().
func (p *PixMap) Draw(ops *op.Ops, x, y int) {
	if !p.Ready {
		p.Bind()
	}
	// Clip to the widget area
	clip.Rect{Min: image.Pt(x, y), Max: image.Pt(x+p.Width, y+p.Height)}.Push(ops)
	// Paint the image
	p.Op.Add(ops)
	paint.PaintOp{}.Add(ops)
}

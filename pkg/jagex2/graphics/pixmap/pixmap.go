package pixmap

import (
	"image"
	"sync"

	"gioui.org/op"
	"gioui.org/op/paint"

	"goscape-client/pkg/jagex2/graphics/pix2d"
)

var (
	// MINE
	DrawMu sync.Mutex
)

// TODO

// PixMap is a CPU-side pixel buffer that can be efficiently uploaded to GPU.
type PixMap struct {
	//Data []byte
	//Data []uint8
	Data    []int
	Width   int
	Height  int
	Image   *image.RGBA
	OpCache *op.Ops
}

// NewPixMap allocates a width*height pixel buffer.
func NewPixMap(width, height int) *PixMap {
	//pix := image.NewRGBA(image.Rect(0, 0, width, height))
	//return &PixMap{
	//	Wi:  width,
	//	Hi: height,
	//	Data: pix.Pix,
	//	Image:  pix,
	//}

	var m PixMap
	m.Width = width
	m.Height = height
	m.Data = make([]int, width*height)
	m.Image = image.NewRGBA(image.Rect(0, 0, width, height)) // TODO: unused
	m.OpCache = new(op.Ops)                                  // MINE
	m.Bind()
	//
	return &m
}

// Bind uploads the current pixel data to GPU.
// Call this once per frame before drawing.
func (p *PixMap) Bind() {
	//p.Op = paint.NewImageOp(p.Image)
	//p.Ready = true
	pix2d.Bind(p.Width, p.Data, p.Height)
}

// Draw adds the necessary operations to render the buffer at (x,y).
// Must be called between op.Ops{}.Reset() and window.Event().
// TODO: the source of problems?
// TODO: problem might be multiple goroutines using draw (and acting on ops.Ops) at the same time, causing bad stacks?
func (p *PixMap) Draw(ops *op.Ops, x, y int) {
	// MINE
	DrawMu.Lock()
	defer DrawMu.Unlock()

	//if !p.Ready {
	//	p.Bind()
	//}
	//// Clip to the widget area
	//clip.Rect{Min: image.Pt(x, y), Max: image.Pt(x+p.Wi, y+p.Hi)}.Push(ops)
	//// Paint the image
	//p.Op.Add(ops)
	//paint.PaintOp{}.Add(ops)

	// transofrmop translats the pos of the ops that come after it
	// example: offset the red rect 100 pixels to the right:
	// 	defer op.Offset(image.Pt(100, 0)).Push(ops).Pop()

	// Save the operations in an independent ops value (the cache)
	macro := op.Record(p.OpCache)
	//defer op.Offset(image.Point{x, y}).Push(p.OpCache).Pop()
	stack := op.Offset(image.Point{x, y}).Push(p.OpCache)
	//op.TransformOp{}
	img := convertPixmapPixels(p.Width, p.Height, p.Data)
	imageOp := paint.NewImageOp(img)
	imageOp.Filter = paint.FilterNearest
	imageOp.Add(p.OpCache)
	paint.PaintOp{}.Add(p.OpCache)
	stack.Pop()
	call := macro.Stop()
	// Draw the operations from the cache
	call.Add(ops)

	// The specified ColorModel object should be used to convert the pixels into their corresponding color and alpha components.
	//defer op.Offset(image.Point{x, y}).Push(ops).Pop()
	////op.TransformOp{}
	//img := convertPixmapPixels(p.Wi, p.Hi, p.Data)
	//imageOp := paint.NewImageOp(img)
	//imageOp.Filter = paint.FilterNearest
	//imageOp.Add(ops)
	//paint.PaintOp{}.Add(ops)

}

// convertPixmapPixels converts packed 0x00RRGGBB ints (Java pix2d format) to image.NRGBA.
// Java's DirectColorModel has no alpha mask, so all pixels are fully opaque.
// image.NRGBA (straight alpha) is also Gio's preferred optimized format for paint.NewImageOp.
func convertPixmapPixels(width, height int, javaPixels []int) *image.NRGBA {
	rgba := image.NewNRGBA(image.Rect(0, 0, width, height))

	for i, argb := range javaPixels {
		rgba.Pix[i*4] = uint8(argb >> 16)
		rgba.Pix[i*4+1] = uint8(argb >> 8)
		rgba.Pix[i*4+2] = uint8(argb)
		rgba.Pix[i*4+3] = 0xFF // always opaque, matching Java's DirectColorModel (no alpha mask)
	}

	return rgba
}

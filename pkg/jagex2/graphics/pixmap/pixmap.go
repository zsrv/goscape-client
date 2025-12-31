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
	//Pixels []byte
	//Pixels []uint8
	Pixels  []int
	Width   int
	Height  int
	Image   *image.RGBA
	OpCache *op.Ops
}

// NewPixMap allocates a width*height pixel buffer.
func NewPixMap(width, height int) *PixMap {
	//pix := image.NewRGBA(image.Rect(0, 0, width, height))
	//return &PixMap{
	//	Width:  width,
	//	Height: height,
	//	Pixels: pix.Pix,
	//	Image:  pix,
	//}

	var m PixMap
	m.Width = width
	m.Height = height
	m.Pixels = make([]int, width*height)
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
	pix2d.Bind(p.Width, p.Pixels, p.Height)
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
	//clip.Rect{Min: image.Pt(x, y), Max: image.Pt(x+p.Width, y+p.Height)}.Push(ops)
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
	img := convertPixmapPixels(p.Width, p.Height, p.Pixels)
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
	//img := convertPixmapPixels(p.Width, p.Height, p.Pixels)
	//imageOp := paint.NewImageOp(img)
	//imageOp.Filter = paint.FilterNearest
	//imageOp.Add(ops)
	//paint.PaintOp{}.Add(ops)

}

// The resulting image.RGBA can be directly used with paint.NewImageOp() for drawing
func convertPixmapPixels(width, height int, javaPixels []int) *image.RGBA { // changed javaPixels from int32 to int
	// Create RGBA image (Go's standard format)
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))

	// Convert each ARGB pixel to RGBA
	for i, argb := range javaPixels {
		// Extract components from packed ARGB
		a := uint8((argb >> 24) & 0xFF)
		r := uint8((argb >> 16) & 0xFF)
		g := uint8((argb >> 8) & 0xFF)
		b := uint8(argb & 0xFF)

		// Set in RGBA (R,G,B,A order)
		rgba.Pix[i*4] = r
		rgba.Pix[i*4+1] = g
		rgba.Pix[i*4+2] = b
		rgba.Pix[i*4+3] = a
	}

	return rgba
}

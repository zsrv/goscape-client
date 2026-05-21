package pixmap

import (
	"image"
	"sync"

	"gioui.org/op"
	"gioui.org/op/paint"

	"goscape-client/pkg/jagex2/graphics/pix2d"
)

// OpsMu serializes all access to the *op.Ops owned by Client. Both the game
// goroutine (via PixMap.Draw at ~25 call sites in client.go) and the Gio
// event goroutine (via event.Op, source.Event(...), and e.Frame inside the
// FrameEvent handler) touch that op list, so every touch must happen under
// this mutex. Java had no analogue — AWT's EDT and the game thread were
// serialized naturally through the repaint queue.
var OpsMu sync.Mutex

// PixMap is a CPU-side pixel buffer that can be efficiently uploaded to GPU.
type PixMap struct {
	Data    []int
	Width   int
	Height  int
	OpCache *op.Ops
}

// NewPixMap allocates a width*height pixel buffer.
func NewPixMap(width, height int) *PixMap {
	var m PixMap
	m.Width = width
	m.Height = height
	m.Data = make([]int, width*height)
	m.OpCache = new(op.Ops)
	m.Bind()
	return &m
}

// Bind uploads the current pixel data to GPU.
// Call this once per frame before drawing.
func (p *PixMap) Bind() {
	pix2d.Bind(p.Width, p.Data, p.Height)
}

// Draw splices a cached macro for this PixMap into the caller's op list.
// The caller (game goroutine) writes into the shared *op.Ops, so we hold
// OpsMu to serialize against the Gio goroutine's event.Op/e.Frame calls.
func (p *PixMap) Draw(ops *op.Ops, x, y int) {
	OpsMu.Lock()
	defer OpsMu.Unlock()

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

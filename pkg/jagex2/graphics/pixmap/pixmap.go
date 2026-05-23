package pixmap

import (
	"encoding/binary"
	"image"
	"sync"

	"gioui.org/op"
	"gioui.org/op/paint"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
)

// OpsMu serializes all access to the *op.Ops owned by Client. Both the
// game goroutine (transitively from c.Draw → PixMap.Draw at ~44 call
// sites in client.go) and the Gio event goroutine (via event.Op,
// source.Event(...), and e.Frame inside the FrameEvent handler) touch
// that op list, so every touch must happen under this mutex.
//
// Contract: callers of PixMap.Draw MUST already hold OpsMu. The lock
// is held at frame-granularity by c.Draw (which Resets the op list
// then issues all per-frame appends atomically) and by the FrameEvent
// handler (which appends event.Op + drains inputs + presents via
// e.Frame). This ensures the Gio goroutine never observes a partial
// frame — without that guarantee, partial PixMap.Draws would render
// as white-flashing artifacts on static elements (title screen,
// game frame chrome), since e.Frame replays whatever ops are
// currently in the buffer including a freshly-Reset empty list.
//
// Java had no analogue — AWT's EDT and the game thread were
// serialized naturally through the repaint queue.
var OpsMu sync.Mutex

// PixMap is a CPU-side pixel buffer that can be efficiently uploaded to GPU.
type PixMap struct {
	Data   []int
	Width  int
	Height int

	// imgBuf is the reusable RGBA buffer that Draw fills in place and
	// hands to paint.NewImageOp each frame, so the steady-state render
	// path performs no image allocation. See Draw for the safety
	// invariants that make in-place reuse correct.
	imgBuf *image.RGBA
}

// NewPixMap allocates a width*height pixel buffer.
func NewPixMap(width, height int) *PixMap {
	var m PixMap
	m.Width = width
	m.Height = height
	m.Data = make([]int, width*height)
	m.imgBuf = image.NewRGBA(image.Rect(0, 0, width, height))
	m.Bind()
	return &m
}

// Bind uploads the current pixel data to GPU.
// Call this once per frame before drawing.
func (p *PixMap) Bind() {
	pix2d.Bind(p.Width, p.Data, p.Height)
}

// Draw emits the GPU-upload ops for this PixMap directly into the
// caller's op list. Caller must hold OpsMu (see OpsMu comment above).
//
// The prior implementation recorded a macro into a per-PixMap
// `OpCache *op.Ops` field that was never Reset between calls; every
// Draw appended a fresh macro region (including a reference to the
// converted NRGBA image), and after a few minutes of play those
// OpCaches retained multiple gigabytes of stale image data. The
// macro indirection served no purpose — Draw recorded and
// immediately called, no replay. Emitting ops directly to `ops`
// is equivalent and lets GC collect the NRGBA after c.Ops.Reset
// each frame.
func (p *PixMap) Draw(ops *op.Ops, x, y int) {
	defer op.Offset(image.Point{X: x, Y: y}).Push(ops).Pop()
	// Fill the reused buffer in place instead of allocating a fresh image
	// every frame. This is safe because:
	//   1. OpsMu serializes this write against e.Frame's read (the caller
	//      holds OpsMu for the whole frame build; the FrameEvent handler
	//      holds the same OpsMu across e.Frame).
	//   2. The GL upload (TexSubImage2D, inside e.Frame) copies the bytes
	//      synchronously, so imgBuf is free to overwrite once e.Frame
	//      returns.
	//   3. paint.NewImageOp still mints a fresh handle, which forces Gio
	//      to re-read imgBuf every frame (a stable handle would show a
	//      frozen frame).
	// See docs/superpowers/specs/2026-05-23-pixmap-buffer-reuse-design.md.
	writePixmapPixels(p.imgBuf, p.Data)
	imageOp := paint.NewImageOp(p.imgBuf)
	imageOp.Filter = paint.FilterNearest
	imageOp.Add(ops)
	paint.PaintOp{}.Add(ops)
}

// writePixmapPixels fills dst in place from packed 0x00RRGGBB ints (Java
// pix2d format). dst must be a contiguous *image.RGBA (created by
// image.NewRGBA, not SubImage) whose pixel count is at least
// len(javaPixels). Java's DirectColorModel has no alpha mask, so all
// pixels are fully opaque; premultiplied RGBA then equals straight NRGBA
// byte-for-byte.
//
// 0x00RRGGBB -> 0xRRGGBBFF; a big-endian 32-bit store lays the bytes down
// as [R, G, B, 0xFF]. One wide store per pixel is ~2.2x faster than four
// byte writes (benchmarked 2026-05-23), and writing into a caller-owned
// buffer avoids a per-frame allocation.
func writePixmapPixels(dst *image.RGBA, javaPixels []int) {
	pix := dst.Pix
	for i, argb := range javaPixels {
		binary.BigEndian.PutUint32(pix[i*4:], uint32(argb)<<8|0xFF)
	}
}

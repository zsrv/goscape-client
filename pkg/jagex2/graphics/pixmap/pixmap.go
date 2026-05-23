package pixmap

import (
	"encoding/binary"
	"image"
	"sync"

	"gioui.org/op"
	"gioui.org/op/paint"

	"goscape-client/pkg/jagex2/graphics/pix2d"
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

	// imgBuf is a reusable RGBA buffer, sized to the PixMap and allocated
	// once here. It is scaffolding for the per-frame render path: a
	// follow-up change wires Draw to fill it in place instead of
	// allocating a fresh image every frame.
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
	img := convertPixmapPixels(p.Width, p.Height, p.Data)
	imageOp := paint.NewImageOp(img)
	imageOp.Filter = paint.FilterNearest
	imageOp.Add(ops)
	paint.PaintOp{}.Add(ops)
}

// writePixmapPixels fills dst in place from packed 0x00RRGGBB ints (Java
// pix2d format). dst must have at least len(javaPixels) pixels. Java's
// DirectColorModel has no alpha mask, so all pixels are fully opaque;
// premultiplied RGBA then equals straight NRGBA byte-for-byte.
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

// convertPixmapPixels converts packed 0x00RRGGBB ints (Java pix2d format) to image.RGBA.
// Java's DirectColorModel has no alpha mask, so all pixels are fully opaque.
//
// The type matters for performance: paint.NewImageOp uses an *image.RGBA
// as-is, but force-converts any other type (e.g. *image.NRGBA) via a
// per-frame draw.Draw — which the 2026-05-23 baseline profile showed
// costing ~25% of CPU and ~48% of all allocations. Because every pixel
// is fully opaque, premultiplied RGBA and straight NRGBA are byte-
// identical, so emitting RGBA is both faster and pixel-equivalent.
func convertPixmapPixels(width, height int, javaPixels []int) *image.RGBA {
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))

	dst := rgba.Pix
	for i, argb := range javaPixels {
		// 0x00RRGGBB -> 0xRRGGBBFF; a big-endian 32-bit store lays the
		// bytes down as [R, G, B, 0xFF] — opaque, matching Java's
		// DirectColorModel (no alpha mask). One wide store per pixel is
		// ~2.2x faster than four byte writes (benchmarked 2026-05-23).
		binary.BigEndian.PutUint32(dst[i*4:], uint32(argb)<<8|0xFF)
	}

	return rgba
}

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

	// imgBuf is the reusable RGBA buffer that Draw fills in place, backing a
	// stable mutable image op (imageOp). The steady-state render path performs
	// no image allocation and no GPU texture churn — the texture is created
	// once and re-uploaded in place only when the pixels change.
	imgBuf *image.RGBA

	// imageOp is one stable mutable op reused every frame. A fresh ImageOp per
	// frame (the old approach) made Gio create+delete a GL texture per frame,
	// which the WebGL backend never reclaimed (multi-GB wasm leak). See
	// docs/superpowers/specs/2026-05-24-wasm-texture-leak-patched-gio-design.md.
	imageOp  paint.MutableImageOp
	lastHash uint64
	uploaded bool
}

// NewPixMap allocates a width*height pixel buffer.
func NewPixMap(width, height int) *PixMap {
	var m PixMap
	m.Width = width
	m.Height = height
	m.Data = make([]int, width*height)
	m.imgBuf = image.NewRGBA(image.Rect(0, 0, width, height))
	m.imageOp = paint.NewMutableImageOp(m.imgBuf)
	m.imageOp.Filter = paint.FilterNearest
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
// The prior implementation minted a fresh paint.NewImageOp every frame,
// causing Gio to create+delete a GL texture per frame; the WebGL backend
// never reclaimed these, producing a multi-GB memory leak in wasm builds.
// This version reuses one stable paint.MutableImageOp and re-uploads (in
// place, via the patched Gio) only when an FNV hash of the pixel data
// detects a change since the last upload.
func (p *PixMap) Draw(ops *op.Ops, x, y int) {
	defer op.Offset(image.Point{X: x, Y: y}).Push(ops).Pop()
	// Re-upload only when the pixel content changed since the last upload. The
	// stable imageOp keeps one GPU texture alive across frames (no per-frame
	// texture create/delete churn), and hashPixels detects change without any
	// plumbing into the pix2d write paths.
	//
	// CONCURRENCY: callers hold OpsMu across the whole frame build, and the
	// FrameEvent handler holds the same OpsMu across e.Frame (inside which the
	// patched gpu.texHandle does the in-place UploadImage). So the write to
	// imgBuf, the Invalidate, and the upload are all serialized — same invariant
	// as before. See the OpsMu comment above.
	h := hashPixels(p.Data)
	if !p.uploaded || h != p.lastHash {
		writePixmapPixels(p.imgBuf, p.Data)
		p.imageOp.Invalidate()
		p.lastHash = h
		p.uploaded = true
	}
	// Always add the op so the texture stays "used" this frame and is not
	// evicted/deleted by Gio's texture cache.
	p.imageOp.Add(ops)
	paint.PaintOp{}.Add(ops)
}

// hashPixels is a fast FNV-1a-style 64-bit hash over the packed 0x00RRGGBB
// pixels, used to detect whether the buffer changed since the last GPU upload.
// Allocation-free and int-width-independent (each pixel fits in uint32). A
// collision would at worst skip one frame's re-upload of changed content (a
// 1-frame stale flicker), which is negligible for change detection.
func hashPixels(data []int) uint64 {
	const (
		offset uint64 = 1469598103934665603
		prime  uint64 = 1099511628211
	)
	h := offset
	for _, v := range data {
		h = (h ^ uint64(uint32(v))) * prime
	}
	return h
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

package pixmap

import (
	"encoding/binary"
	"image"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
)

// PixMap is a CPU-side pixel buffer blitted to the screen via the active
// platform Backend. Java: PixMap (drawImage per frame). The texture is created
// once and re-uploaded in place only when the pixels change (hashPixels), so
// there is no per-frame GPU texture churn.
type PixMap struct {
	Data   []int
	Width  int
	Height int

	// AlwaysUpload skips the per-frame hashPixels change-detection and uploads
	// unconditionally. Set true for pixmaps that change almost every frame (the
	// 3D viewport): hashing the whole buffer to detect a change that is
	// essentially always present is pure overhead, and the texture is reused in
	// place (texSubImage2D), so unconditional upload does not leak.
	AlwaysUpload bool

	imgBuf *image.RGBA // reusable RGBA staging buffer written before each UploadTexture

	// tex is the backend texture handle, allocated once in NewPixMap.
	tex      platform.Texture
	lastHash uint64
	uploaded bool
}

// NewPixMap allocates a width*height pixel buffer and its backend texture.
func NewPixMap(width, height int) *PixMap {
	var m PixMap
	m.Width = width
	m.Height = height
	m.Data = make([]int, width*height)
	m.imgBuf = image.NewRGBA(image.Rect(0, 0, width, height))
	m.tex = platform.Active.NewTexture(width, height)
	m.Bind()
	return &m
}

// Bind sets this PixMap as the active pix2d draw target.
func (p *PixMap) Bind() {
	pix2d.Bind(p.Width, p.Data, p.Height)
}

// Draw uploads the pixels (only if changed since last Draw) and blits the
// texture with its top-left at (x, y). Java: Graphics.drawImage(image, x, y).
func (p *PixMap) Draw(x, y int) {
	if p.AlwaysUpload {
		// Skip the full-buffer hash for always-changing pixmaps; upload in place.
		writePixmapPixels(p.imgBuf, p.Data)
		platform.Active.UploadTexture(p.tex, p.imgBuf.Pix)
	} else {
		h := hashPixels(p.Data)
		if !p.uploaded || h != p.lastHash {
			writePixmapPixels(p.imgBuf, p.Data)
			platform.Active.UploadTexture(p.tex, p.imgBuf.Pix)
			p.lastHash = h
			p.uploaded = true
		}
	}
	platform.Active.Blit(p.tex, x, y)
}

// hashPixels is a fast FNV-1a-style 64-bit hash over the packed 0x00RRGGBB
// pixels, used to detect whether the buffer changed since the last GPU upload.
// Allocation-free and int-width-independent (each pixel fits in uint32). A
// collision would at worst skip one frame's re-upload of changed content (a
// 1-frame stale flicker), which is negligible for change detection.
func hashPixels(data []int) uint64 {
	const (
		offset uint64 = 14695981039346656037 // FNV-1a 64-bit offset basis
		prime  uint64 = 1099511628211        // FNV-1a 64-bit prime
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

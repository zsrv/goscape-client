// Command icondump renders every item-inventory icon in a rev-245.2 game cache
// to a 32x32 PNG, headlessly (no window, no GPU). It drives the same client
// render path the running game uses for the inventory/sidebar — config +
// textures + deterministic HSL palette + per-id model blobs → ObjType.GetIcon →
// Pix32 — and is the end-to-end gate for the item-icon rasterizer port: its
// output is diffed byte-for-byte against the pinned LostCityRS/Client-Java
// goldens (commit 176a85f7; see cmd/icondump/icondump_test.go and
// tools/iconref/java/).
//
// Usage:
//
//	icondump -cache <dir> -out <dir> [-id N] [-v]
//
// -cache is a classic on-disk cache (main_file_cache.dat + main_file_cache.idx0..4).
// -out receives one <id>.png per obj with a renderable model, plus an index.tsv
// of "id<TAB>name" lines (name from ObjType.Name; may be empty) for the rendered
// ids only. -id N dumps a single obj instead of all. A final line
//
//	rendered=N skipped=M total=T
//
// is printed to stdout. Exit codes: 0 success (skipped ids are fine), 1 runtime
// error, 2 usage error.
//
// Icon pixel rule (mirrors the goldens): a Pix32 value of 0 is the client's
// "not drawn" sentinel → transparent (0,0,0,0); any other value → opaque
// (R,G,B,255) from its low 24 bits.
//
// Determinism: the palette is built with InitColourTableDeterministic(0.8) (the
// un-jittered brightness the reference pins), so two runs over the same cache
// produce byte-identical output.
package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zsrv/goscape-client/pkg/jagex2/config/objtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix32"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix3d"
	jio "github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// iconConfig is the whole input to the dump pipeline, so run() is testable
// in-process (the golden test calls it directly with a t.TempDir out dir).
type iconConfig struct {
	cacheDir string
	outDir   string
	id       int // >= 0 dumps only that obj; < 0 dumps every obj
	verbose  bool
}

// stats is the pipeline's tally, echoed as the final stdout line.
type stats struct {
	rendered int
	skipped  int
	total    int
}

// noopProvider is the model on-demand provider for headless dumping: every
// archive-1 blob present in the cache is pre-unpacked before any render, so
// Model.Load never faults in a model and this is never called (an obj whose
// model is absent from the cache simply skips). Java: OnDemandProvider.
type noopProvider struct{}

func (noopProvider) RequestModel(int) {}

func main() {
	cacheDir := flag.String("cache", "", "cache directory (main_file_cache.dat + main_file_cache.idx0..4)")
	outDir := flag.String("out", "", "output directory for <id>.png and index.tsv")
	id := flag.Int("id", -1, "dump only this obj id (default: all objs)")
	verbose := flag.Bool("v", false, "log skipped obj ids")
	flag.Parse()

	if *cacheDir == "" || *outDir == "" {
		fmt.Fprintln(os.Stderr, "usage: icondump -cache <dir> -out <dir> [-id N] [-v]")
		flag.PrintDefaults()
		os.Exit(2)
	}

	st, err := run(iconConfig{cacheDir: *cacheDir, outDir: *outDir, id: *id, verbose: *verbose})
	if err != nil {
		fmt.Fprintf(os.Stderr, "icondump: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("rendered=%d skipped=%d total=%d\n", st.rendered, st.skipped, st.total)
}

// run executes the full pipeline: load config + textures + palette, pre-unpack
// every model blob, then render each obj's 32x32 icon and write it. It mirrors
// the reference harness (tools/iconref/java/IconDump.java) call order exactly.
//
// SINGLE-SHOT per process: run must be called at most once. The render packages
// keep client state in package-level vars (CLAUDE.md "Global State Pattern"),
// and not all of it has a reset path — in particular objtype.IconCache and
// objtype.ModelCache are package-level LRUs with no objtype.Reset, so a second
// run in the same process would serve icons/models cached from the FIRST run's
// cache and palette (plus stale pix3d texture and model metadata state). A
// caller needing a second dump must exec a fresh process.
func run(cfg iconConfig) (stats, error) {
	if cfg.cacheDir == "" {
		return stats{}, errors.New("cache directory is required")
	}
	if cfg.outDir == "" {
		return stats{}, errors.New("output directory is required")
	}

	cache, closeCache, err := openCache(cfg.cacheDir)
	if err != nil {
		return stats{}, err
	}
	defer closeCache()

	// Config jag (archive 0, file 2) → ObjType definitions. archiveCacheLoad in
	// the client reads startup archives from cache index 0; the bytes are stored
	// raw (Jagfile decompresses internally).
	configRaw := cache.Read(0, 2)
	if configRaw == nil {
		return stats{}, errors.New("config jag (archive 0, file 2) not found in cache")
	}
	objtype.Unpack(jio.NewJagfile(configRaw))

	// Textures jag (archive 0, file 6) → deterministic palette → texel pool, in
	// the client's order (textures before the palette, which derives per-texture
	// palettes during InitColourTable). LowMem = false matches the reference
	// (Client-Java Pix3D.lowMem default), selecting the 128x128 texel path.
	texRaw := cache.Read(0, 6)
	if texRaw == nil {
		return stats{}, errors.New("textures jag (archive 0, file 6) not found in cache")
	}
	pix3d.Reset()
	pix3d.LowMem = false
	pix3d.UnpackTextures(jio.NewJagfile(texRaw))
	pix3d.InitColourTableDeterministic(0.8)
	pix3d.InitPool(20)

	// Models: size the metadata table from the archive-1 (idx1) entry count and
	// pre-unpack every present blob so Model.Load resolves without a provider.
	numModels, err := idxFileCount(cfg.cacheDir, 1)
	if err != nil {
		return stats{}, err
	}
	// model.Reset rebinds the model package's cached Sin/Cos/Palette/Reciprocal16
	// to pix3d's live tables and must run AFTER the palette is built: pix3d.Reset
	// reallocated ColourTable, and the flat-face path resolves Palette[idx] from
	// this cached reference (Java caches Pix3D.palette; the client never
	// reallocates it, so it stays valid there). Without this rebind, Palette would
	// point at the stale zero-filled table and every flat face would render black.
	// It also nils Metadata, so it must precede model.Init.
	//
	// Same species of cross-invocation alias, with NO reset path: the
	// objtype.IconCache / objtype.ModelCache package-level LRUs. They cache
	// rendered Pix32 icons and lit models keyed by obj id across GetIcon
	// calls; nothing clears them. That is fine here only because run() is
	// single-shot per process (see its doc comment).
	model.Reset()
	model.Init(numModels, noopProvider{})
	for mid := range numModels {
		blob := loadModelBlob(cache, mid)
		if blob == nil {
			continue
		}
		unpackModel(mid, blob, cfg.verbose)
	}

	if err := os.MkdirAll(cfg.outDir, 0o755); err != nil {
		return stats{}, fmt.Errorf("create out dir: %w", err)
	}

	lo, hi := 0, objtype.Count
	if cfg.id >= 0 {
		if cfg.id >= objtype.Count {
			return stats{}, fmt.Errorf("id %d out of range [0,%d)", cfg.id, objtype.Count)
		}
		lo, hi = cfg.id, cfg.id+1
	}

	st := stats{total: hi - lo}
	var index strings.Builder
	for oid := lo; oid < hi; oid++ {
		spr := renderIcon(oid, cfg.verbose)
		if spr == nil {
			st.skipped++
			if cfg.verbose {
				fmt.Fprintf(os.Stderr, "skip id=%d (no renderable model)\n", oid)
			}
			continue
		}
		if err := writePNG(filepath.Join(cfg.outDir, strconv.Itoa(oid)+".png"), spr); err != nil {
			return st, err
		}
		index.WriteString(strconv.Itoa(oid))
		index.WriteByte('\t')
		index.WriteString(objName(oid, cfg.verbose))
		index.WriteByte('\n')
		st.rendered++
	}

	if err := os.WriteFile(filepath.Join(cfg.outDir, "index.tsv"), []byte(index.String()), 0o644); err != nil {
		return st, fmt.Errorf("write index.tsv: %w", err)
	}
	return st, nil
}

// openCache opens the shared dat + five idx files read-only and wraps them in a
// FileStreamCache (the same store the client uses; index 0 backs the startup
// jags, indices 1..4 back the on-demand model/anim/midi/map archives).
func openCache(dir string) (*jio.FileStreamCache, func(), error) {
	dat, err := os.Open(filepath.Join(dir, "main_file_cache.dat"))
	if err != nil {
		return nil, nil, fmt.Errorf("open cache dat: %w", err)
	}
	var idx [5]*os.File
	for i := range 5 {
		f, err := os.Open(filepath.Join(dir, "main_file_cache.idx"+strconv.Itoa(i)))
		if err != nil {
			_ = dat.Close()
			for j := range i {
				_ = idx[j].Close()
			}
			return nil, nil, fmt.Errorf("open cache idx%d: %w", i, err)
		}
		idx[i] = f
	}
	closeAll := func() {
		_ = dat.Close()
		for i := range 5 {
			_ = idx[i].Close()
		}
	}
	return jio.NewFileStreamCache(dat, idx), closeAll, nil
}

// idxFileCount returns the number of file slots in an idx file (6 bytes each) —
// the count the reference passes to Model.init for archive 1.
func idxFileCount(dir string, archive int) (int, error) {
	fi, err := os.Stat(filepath.Join(dir, "main_file_cache.idx"+strconv.Itoa(archive)))
	if err != nil {
		return 0, fmt.Errorf("stat idx%d: %w", archive, err)
	}
	return int(fi.Size() / 6), nil
}

// loadModelBlob reads one archive-1 model blob from the cache and returns its
// decompressed bytes, or nil if absent/corrupt. This mirrors the OnDemand
// data path exactly: cache entries for archives 1..4 are stored as a gzip
// stream followed by a 2-byte big-endian version trailer, so the trailer is
// sliced off before gunzip (OnDemand.Cycle does the same). Feeding the whole
// buffer to Go's multistream gzip.Reader would error on the trailing 2 bytes.
func loadModelBlob(cache *jio.FileStreamCache, id int) []byte {
	raw := cache.Read(1, id)
	if len(raw) < 2 {
		return nil
	}
	gz, err := gzip.NewReader(bytes.NewReader(raw[:len(raw)-2]))
	if err != nil {
		return nil
	}
	defer func() { _ = gz.Close() }()
	blob, err := io.ReadAll(gz)
	if err != nil {
		return nil
	}
	return blob
}

// unpackModel decodes one model blob's metadata, guarding against a panic on
// malformed data (leaving the id's metadata unset → Model.Load returns nil →
// that obj skips). Under -v the recovered value is logged so a real decoder bug
// is distinguishable from a legitimately absent/corrupt blob.
func unpackModel(id int, blob []byte, verbose bool) {
	defer logRecover("model.Unpack", id, verbose)
	model.Unpack(id, blob)
}

// renderIcon renders obj id's plain 32x32 icon (outlineRgb 0, count 1), or nil
// if there is no renderable model. GetIcon can panic on hostile geometry
// (DrawSimple has its own recover in the client, but Get/decode could still
// fault), so the call is guarded and a panic counts as a skip. Under -v the
// recovered value is logged so a real render bug is distinguishable from a
// legitimate skip. Java 245.2 arg order: getIcon(outlineRgb, count, id).
func renderIcon(id int, verbose bool) (spr *pix32.Pix32) {
	defer func() {
		if r := recover(); r != nil {
			spr = nil
			if verbose {
				fmt.Fprintf(os.Stderr, "recovered in GetIcon: id=%d value=%v\n", id, r)
			}
		}
	}()
	return objtype.GetIcon(0, 1, id)
}

// objName returns obj id's display name (ObjType.Name), or "" if it cannot be
// read. Guarded because Get decodes on demand.
func objName(id int, verbose bool) (name string) {
	defer logRecover("objtype.Get", id, verbose)
	return objtype.Get(id).Name
}

// logRecover is a deferred recover guard that, under -v, logs the panicking
// call site, obj/model id, and recovered value to stderr. Non-verbose output is
// unchanged (the panic is still swallowed either way).
func logRecover(site string, id int, verbose bool) {
	if r := recover(); r != nil && verbose {
		fmt.Fprintf(os.Stderr, "recovered in %s: id=%d value=%v\n", site, id, r)
	}
}

// writePNG encodes a Pix32 icon to a 32x32 NRGBA PNG applying the client's
// 0-pixel-transparent rule: value 0 → (0,0,0,0); else → (R,G,B,255).
func writePNG(path string, spr *pix32.Pix32) error {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for i := range 32 * 32 {
		p := int32(spr.Pixels[i])
		if p == 0 {
			continue // NRGBA zero value is already (0,0,0,0)
		}
		img.Pix[i*4] = byte((p >> 16) & 0xff)
		img.Pix[i*4+1] = byte((p >> 8) & 0xff)
		img.Pix[i*4+2] = byte(p & 0xff)
		img.Pix[i*4+3] = 0xff
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create png: %w", err)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return fmt.Errorf("encode png: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close png: %w", err)
	}
	return nil
}

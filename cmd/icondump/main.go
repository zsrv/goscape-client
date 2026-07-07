// Command icondump renders every item-inventory icon in a rev-225 game's client
// jag archives to a 32x32 PNG, headlessly (no window, no GPU). It drives the same
// client render path the running game uses for the inventory/sidebar — config +
// textures + deterministic HSL palette + whole-archive model blobs →
// ObjType.GetIcon → Pix32 — and is the end-to-end gate for the item-icon
// rasterizer port: its output is diffed byte-for-byte against the pinned
// LostCityRS/Client-Java goldens (commit cc3781de, branch 225-clean; see
// cmd/icondump/icondump_test.go and tools/iconref/java/).
//
// Usage:
//
//	icondump -jag-dir <dir> -out <dir> [-id N] [-v]
//
// -jag-dir is a directory holding the three client jag archives named exactly
// "config", "models" and "textures" (the 225 pack pipeline's client-jag names;
// no ".jag" extension — the files ARE jag archives). Unlike the later revisions
// there is no classic .dat/.idx cache and no OnDemand: the whole models jag is
// unpacked once and every obj definition lives in the config jag.
//
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
// un-jittered brightness the reference pins), so two runs over the same jags
// produce byte-identical output.
package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zsrv/goscape-client/pkg/jagex2/config/objtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/model"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix32"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix3d"
	jio "github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// The three client jag archive basenames the 225 pack pipeline emits under the
// jag dir. They are raw jag archives (no ".jag" extension); io.NewJagfile reads
// their raw bytes directly (bzip2 handled internally by the Jagfile).
const (
	configJagName   = "config"
	modelsJagName   = "models"
	texturesJagName = "textures"
)

// iconConfig is the whole input to the dump pipeline, so run() is testable
// in-process (the golden test calls it directly with a t.TempDir out dir).
type iconConfig struct {
	jagDir  string
	outDir  string
	id      int // >= 0 dumps only that obj; < 0 dumps every obj
	verbose bool
}

// stats is the pipeline's tally, echoed as the final stdout line.
type stats struct {
	rendered int
	skipped  int
	total    int
}

func main() {
	jagDir := flag.String("jag-dir", "", "directory holding the client jag archives named config, models, textures")
	outDir := flag.String("out", "", "output directory for <id>.png and index.tsv")
	id := flag.Int("id", -1, "dump only this obj id (default: all objs)")
	verbose := flag.Bool("v", false, "log skipped obj ids")
	flag.Parse()

	if *jagDir == "" || *outDir == "" {
		fmt.Fprintln(os.Stderr, "usage: icondump -jag-dir <dir> -out <dir> [-id N] [-v]")
		flag.PrintDefaults()
		os.Exit(2)
	}

	st, err := run(iconConfig{jagDir: *jagDir, outDir: *outDir, id: *id, verbose: *verbose})
	if err != nil {
		fmt.Fprintf(os.Stderr, "icondump: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("rendered=%d skipped=%d total=%d\n", st.rendered, st.skipped, st.total)
}

// run executes the full pipeline: load textures + palette, unpack the whole
// models jag, load the config jag's obj definitions, then render each obj's
// 32x32 icon and write it. It mirrors the pinned client startup order exactly
// (Client-Java @cc3781de client.java: unpackTextures → setBrightness(0.8) →
// initPool(20) → Model.unpack → ObjType.unpack) and the reference harness
// (tools/iconref/java/IconDump.java).
//
// SINGLE-SHOT per process: run must be called at most once. The render packages
// keep client state in package-level vars (CLAUDE.md "Global State Pattern"),
// and not all of it has a reset path — in particular objtype.IconCache and
// objtype.ModelCache are package-level LRUs, so a second run in the same process
// would serve icons/models cached from the FIRST run. A caller needing a second
// dump must exec a fresh process.
func run(cfg iconConfig) (stats, error) {
	if cfg.jagDir == "" {
		return stats{}, errors.New("jag directory is required")
	}
	if cfg.outDir == "" {
		return stats{}, errors.New("output directory is required")
	}

	texturesJag, err := loadJag(cfg.jagDir, texturesJagName)
	if err != nil {
		return stats{}, err
	}
	modelsJag, err := loadJag(cfg.jagDir, modelsJagName)
	if err != nil {
		return stats{}, err
	}
	configJag, err := loadJag(cfg.jagDir, configJagName)
	if err != nil {
		return stats{}, err
	}

	// Textures → deterministic palette → texel pool, in the client's order
	// (textures before the palette, which derives per-texture palettes during
	// initColourTable). LowMem = false matches the reference: at cc3781de the
	// texture-detail flag is Java Pix3D.lowDetail (the Go port renamed it LowMem);
	// forcing it false selects the 128x128 texel path the goldens were dumped
	// with. pix3d.Reset zeroes the package state first (it sets LowMem = true, so
	// LowMem = false must come after it).
	pix3d.Reset()
	pix3d.LowMem = false
	pix3d.UnpackTextures(texturesJag)
	pix3d.InitColourTableDeterministic(0.8)
	pix3d.InitPool(20)

	// model.Reset allocates the render scratch buffers (the inner rows of
	// TmpDepthFaces/TmpPriorityFaces are only made in Reset) and rebinds the
	// model package's cached Sin/Cos/Palette/Reciprocal16 to pix3d's live tables.
	// It must run AFTER the palette is built: pix3d.Reset reallocated ColourTable,
	// and the flat-face path resolves Palette[idx] from this cached reference.
	// Without the rebind, Palette would point at the stale table and every flat
	// face would render black. model.Unpack then loads the whole models jag.
	model.Reset()
	model.Unpack(modelsJag)
	objtype.Unpack(configJag)

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

// loadJag reads one client jag archive file from the jag dir and wraps its raw
// bytes in a Jagfile (which decompresses internally). At 225 the archives are
// stored as jag files directly — there is no cache/OnDemand layer.
func loadJag(dir, name string) (*jio.Jagfile, error) {
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil, fmt.Errorf("read %s jag: %w", name, err)
	}
	return jio.NewJagfile(raw), nil
}

// renderIcon renders obj id's plain 32x32 icon (count 1), or nil if there is no
// renderable model. GetIcon can panic on hostile geometry (DrawSimple has its
// own recover in the client, but Get/decode could still fault), so the call is
// guarded and a panic counts as a skip. Under -v the recovered value is logged
// so a real render bug is distinguishable from a legitimate skip. Java 225
// signature: getIcon(id, count) — no outline param (that arrived at 244).
func renderIcon(id int, verbose bool) (spr *pix32.Pix32) {
	defer func() {
		if r := recover(); r != nil {
			spr = nil
			if verbose {
				fmt.Fprintf(os.Stderr, "recovered in GetIcon: id=%d value=%v\n", id, r)
			}
		}
	}()
	return objtype.GetIcon(id, 1)
}

// objName returns obj id's display name (ObjType.Name), or "" if it cannot be
// read. Guarded because Get decodes on demand.
func objName(id int, verbose bool) (name string) {
	defer func() {
		if r := recover(); r != nil && verbose {
			fmt.Fprintf(os.Stderr, "recovered in objtype.Get: id=%d value=%v\n", id, r)
		}
	}()
	return objtype.Get(id).Name
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

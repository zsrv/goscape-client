package pix3d

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// Golden tests that confront this package's rasterizer with the byte-for-byte
// reference goldens produced by tools/iconref/java/IconDump.java from the PINNED
// LostCityRS/Client-Java render code (commit cc3781de, branch 225-clean). Two
// artifacts under cmd/icondump/testdata:
//
//   - palette.bin: 65536 little-endian int32 == Pix3D.colourTable at brightness
//     0.8 with the Math.random() jitter pinned to zero. Compared against
//     InitColourTableDeterministic(0.8).
//   - tri/manifest.json + tri/*.rgba: 8 synthetic single-triangle cases (each a
//     32x32 RGBA image, 0-pixel => transparent).
//
// testdataDir is relative to this package directory (pkg/jagex2/graphics/pix3d);
// four levels up reaches the repo root.
const testdataDir = "../../../../cmd/icondump/testdata"

// referenceJagDirEnv names the env var pointing at a directory holding the three
// client jag archives (config, models, textures). Texture-dependent cases skip
// with a clear message when it is unset. (225 loads jag files directly — no
// classic .dat/.idx cache.)
const referenceJagDirEnv = "ICONDUMP_TEST_JAGDIR"

// texturesJagName is the client textures-jag basename inside the jag dir.
const texturesJagName = "textures"

// The reference pins brightness = 0.8 with zero jitter (IconDump.java patches
// setBrightness's Math.random jitter term out so 0.8 is used verbatim). The Go
// deterministic hook must reproduce the palette exactly at this brightness.
const goldenBrightness = 0.8

// TestPaletteGoldenDeterministic pins InitColourTableDeterministic(0.8) against
// palette.bin. ColourTable is []int (64-bit); each entry is a packed 0xRRGGBB
// (< 2^24, never negative), so its low 32 bits (int32) equal the golden LE
// int32 exactly. No jags are needed: the HSL colour table does not depend on
// any loaded texture.
func TestPaletteGoldenDeterministic(t *testing.T) {
	wantBytes := readGolden(t, "palette.bin")
	if len(wantBytes) != 65536*4 {
		t.Fatalf("palette.bin: got %d bytes, want %d", len(wantBytes), 65536*4)
	}

	Reset()
	InitColourTableDeterministic(goldenBrightness)

	if len(ColourTable) != 65536 {
		t.Fatalf("ColourTable len=%d, want 65536", len(ColourTable))
	}
	mismatches := 0
	for i := range 65536 {
		got := int32(ColourTable[i])
		want := int32(binary.LittleEndian.Uint32(wantBytes[i*4:]))
		if got != want {
			if mismatches < 8 {
				t.Errorf("ColourTable[%d]=%#06x (%d), want %#06x (%d)", i, uint32(got), got, uint32(want), want)
			}
			mismatches++
		}
	}
	if mismatches != 0 {
		t.Fatalf("palette mismatch: %d/%d entries differ", mismatches, 65536)
	}
}

// triCase mirrors one entry of tri/manifest.json. Args is a flat name->int map;
// the routine dispatch below maps those named args into the Go/Java parameter
// order explicitly (see runTriangle).
type triCase struct {
	Name             string         `json:"name"`
	Routine          string         `json:"routine"`
	Width            int            `json:"width"`
	Height           int            `json:"height"`
	Prefill          int            `json:"prefill"`
	LowDetail        bool           `json:"lowDetail"`
	HClip            bool           `json:"hclip"`
	Trans            int            `json:"trans"`
	Args             map[string]int `json:"args"`
	PaletteDependent bool           `json:"palette_dependent"`
	TextureDependent bool           `json:"texture_dependent"`
	NonZeroPixels    int            `json:"nonZeroPixels"`
}

// TestTriangleGoldens replays every manifest case through the matching Go
// rasterizer and diffs the 0-pixel-transparent RGBA against the golden.
//
// Gouraud/flat cases need only the deterministic palette (ColourTable indices).
// Texture cases additionally need the textures jag loaded from the reference
// jag dir; they skip (via ensureTextures) when ICONDUMP_TEST_JAGDIR is unset.
//
// The manifest orders all gouraud/flat cases before the textured ones, so the
// palette is set up first and reused, then the textured setup reloads it
// alongside the textures (deterministically, same values).
func TestTriangleGoldens(t *testing.T) {
	manifest := readManifest(t)

	// Deterministic palette for the gouraud/flat cases; no jags required.
	Reset()
	InitColourTableDeterministic(goldenBrightness)

	texturesReady := false
	for _, c := range manifest {
		t.Run(c.Name, func(t *testing.T) {
			if c.Routine == "texture" {
				ensureTextures(t, &texturesReady)
			}

			buf := setupBuffer(c)
			runTriangle(c)
			got := toRGBA(buf, c.Width, c.Height)

			want := readGolden(t, filepath.Join("tri", c.Name+".rgba"))
			if len(want) != c.Width*c.Height*4 {
				t.Fatalf("%s.rgba: got %d bytes, want %d", c.Name, len(want), c.Width*c.Height*4)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("%s: RGBA mismatch\n%s", c.Name, describeDiff(got, want, c.Width, c.Height))
			}
		})
	}
}

// setupBuffer binds a fresh prefilled 32x32 target and mirrors the exact state
// IconDump.java sets before each draw:
//
//	Java: Pix2D.bind(W, px, H); Pix3D.init2D();
//	    Pix3D.jagged = c.lowDetail; Pix3D.hclip = c.hclip; Pix3D.trans = c.trans
//
// Go equivalents (name mapping for 225-clean):
//   - pix2d.SetPixels(width, data, height)  == Java Pix2D.bind(W, px, H)
//   - pix3d.Init2D()                         == Java Pix3D.init2D() (builds
//     LineOffset and sets CenterW3D/CenterH3D = W/2, H/2 = 16 for 32x32, which
//     the textured raster reads as the screen origin)
//   - pix3d.LowDetail package var            == Java Pix3D.jagged (the raster
//     detail flag the manifest's "lowDetail" key toggles)
//   - HClip / Trans package vars             == Java Pix3D.hclip / trans
func setupBuffer(c triCase) []int {
	buf := make([]int, c.Width*c.Height)
	for i := range buf {
		buf[i] = c.Prefill
	}
	pix2d.SetPixels(c.Width, buf, c.Height)
	Init2D()
	LowDetail = c.LowDetail
	HClip = c.HClip
	Trans = c.Trans
	return buf
}

// runTriangle dispatches to the Go rasterizer with the manifest args mapped into
// the Go/Java parameter order. The Java signatures list the screen coords
// y-major-first (yA,yB,yC,xA,xB,xC,...) and group the textured view-space coords
// AXIS-major; the Go port mirrors Java positionally, so the same named args map
// into the same positions on both sides. The manifest records the args x-major
// (xA,xB,xC,yA,yB,yC,...) so the mapping below re-orders them.
func runTriangle(c triCase) {
	a := c.Args
	switch c.Routine {
	case "gouraud":
		GouraudTriangle(
			a["yA"], a["yB"], a["yC"],
			a["xA"], a["xB"], a["xC"],
			a["colourA"], a["colourB"], a["colourC"],
		)
	case "flat":
		FlatTriangle(
			a["yA"], a["yB"], a["yC"],
			a["xA"], a["xB"], a["xC"],
			a["colour"],
		)
	case "texture":
		TextureTriangle(
			a["yA"], a["yB"], a["yC"],
			a["xA"], a["xB"], a["xC"],
			a["shadeA"], a["shadeB"], a["shadeC"],
			a["originX"], a["txB"], a["txC"],
			a["originY"], a["tyB"], a["tyC"],
			a["originZ"], a["tzB"], a["tzC"],
			a["texture"],
		)
	default:
		panic("unknown routine: " + c.Routine)
	}
}

// ensureTextures loads the textures jag from the reference jag dir and rebuilds
// the deterministic palette + texel pool the way the icon path does, exactly
// once. It skips the whole texture path when the env var is unset.
//
// The reference ran with the texture-detail flag false (Go LowMem == Java
// lowDetail), which selects getTexels' 128x128 / 64->128-upsampling path and a
// 65536-entry texel pool, so we force LowMem = false here to match the goldens.
// Order mirrors IconDump.java: unpackTextures -> setBrightness(0.8) [determ] ->
// initPool(20).
func ensureTextures(t *testing.T, ready *bool) {
	t.Helper()
	jagDir := os.Getenv(referenceJagDirEnv)
	if jagDir == "" {
		t.Skipf("%s unset; skipping texture-dependent golden (set it to a dir holding the config, models and textures jag archives)", referenceJagDirEnv)
	}
	if *ready {
		return
	}

	jag := loadTexturesJag(t, jagDir)

	Reset()
	LowMem = false // match the reference (Go LowMem == Java Pix3D.lowDetail, forced false)
	UnpackTextures(jag)
	InitColourTableDeterministic(goldenBrightness)
	InitPool(20)
	*ready = true
}

// loadTexturesJag reads the textures jag file directly from the jag dir. At 225
// the archive is stored as a raw jag file (bzip2 handled inside Jagfile), so the
// bytes feed straight into NewJagfile — there is no cache/OnDemand layer.
func loadTexturesJag(t *testing.T, jagDir string) *io.Jagfile {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(jagDir, texturesJagName))
	if err != nil {
		t.Fatalf("read textures jag: %v", err)
	}
	return io.NewJagfile(raw)
}

// toRGBA applies the reference's 0-pixel-transparent rule: a pixel buffer holds
// 0x00RRGGBB ints where 0 is the "not drawn" sentinel. p==0 => (0,0,0,0);
// p!=0 => (R,G,B,255). Uses the low 32 bits (int32) since the buffer is []int.
func toRGBA(px []int, w, h int) []byte {
	out := make([]byte, w*h*4)
	for i := range w * h {
		p := int32(px[i])
		if p == 0 {
			continue
		}
		out[i*4] = byte((p >> 16) & 0xff)
		out[i*4+1] = byte((p >> 8) & 0xff)
		out[i*4+2] = byte(p & 0xff)
		out[i*4+3] = 0xff
	}
	return out
}

// describeDiff summarizes the first few differing pixels for a readable failure.
func describeDiff(got, want []byte, w, h int) string {
	var b bytes.Buffer
	diffs := 0
	for i := range w * h {
		g := got[i*4 : i*4+4]
		wa := want[i*4 : i*4+4]
		if !bytes.Equal(g, wa) {
			if diffs < 12 {
				fmt.Fprintf(&b, "  px (%d,%d): got %v want %v\n", i%w, i/w, g, wa)
			}
			diffs++
		}
	}
	fmt.Fprintf(&b, "  total differing pixels: %d/%d", diffs, w*h)
	return b.String()
}

func readManifest(t *testing.T) []triCase {
	t.Helper()
	raw := readGolden(t, filepath.Join("tri", "manifest.json"))
	var cases []triCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("parse manifest.json: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("manifest.json has no cases")
	}
	return cases
}

func readGolden(t *testing.T, rel string) []byte {
	t.Helper()
	p := filepath.Join(testdataDir, rel)
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read golden %s: %v", p, err)
	}
	return raw
}

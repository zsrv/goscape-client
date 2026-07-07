package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/config/objtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
)

// referenceCacheEnv names an env var pointing at a classic .dat/.idxN cache dir
// (main_file_cache.dat + main_file_cache.idx0..4). TestIcondumpMatchesGoldens
// skips with a clear message when it is unset.
const referenceCacheEnv = "ICONDUMP_TEST_CACHE"

// sampleEntry mirrors one record of testdata/icons274/sample.json: the obj id
// and its RuneScript debugname (the golden .rgba is keyed by debugname).
type sampleEntry struct {
	ID        int    `json:"id"`
	Debugname string `json:"debugname"`
}

// TestIcondumpMatchesGoldens drives the whole in-process dump pipeline (run)
// against the reference cache into a fresh temp dir, then byte-compares every
// sampled icon's PNG-decoded RGBA against the byte-for-byte reference golden
// produced by tools/iconref/dump.ts (the vendored LostCityRS/Client-TS render
// code). A match proves the Go item-icon path reproduces the client pixels.
//
// The comparison applies the reference's 0-pixel-transparent rule implicitly:
// run() encodes an NRGBA PNG where a Pix32 value of 0 maps to (0,0,0,0) and any
// other value to (R,G,B,255); decoding back to NRGBA recovers those exact bytes,
// which is what the .rgba goldens hold.
func TestIcondumpMatchesGoldens(t *testing.T) {
	cacheDir := os.Getenv(referenceCacheEnv)
	if cacheDir == "" {
		t.Skipf("%s unset; set it to a dir with main_file_cache.dat + main_file_cache.idx0..4", referenceCacheEnv)
	}

	samples := readSamples(t)

	out, st := runPipelineOnce(t, cacheDir)
	if st.rendered == 0 {
		t.Fatalf("run rendered 0 icons (rendered=%d skipped=%d total=%d)", st.rendered, st.skipped, st.total)
	}
	t.Logf("rendered=%d skipped=%d total=%d", st.rendered, st.skipped, st.total)

	for _, s := range samples {
		t.Run(s.Debugname, func(t *testing.T) {
			got := decodePNGToRGBA(t, filepath.Join(out, strconv.Itoa(s.ID)+".png"))
			want := readGolden(t, filepath.Join("testdata", "icons274", s.Debugname+".rgba"))
			if len(want) != 32*32*4 {
				t.Fatalf("%s.rgba: got %d bytes, want %d", s.Debugname, len(want), 32*32*4)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("%s (id %d): RGBA mismatch\n%s", s.Debugname, s.ID, describeDiff(got, want, 32, 32))
			}
		})
	}
}

// pipelineOnce memoizes the single permitted run() call for this test binary
// (run's doc comment: "run must be called at most once" per process, because
// objtype.ModelCache/SpriteCache and the pix3d/model package-level tables
// have no reset path once populated by a live render). TestIcondumpMatchesGoldens
// and TestLitModelGolden both need that populated state — the latter reads
// obj 1205's lit model straight out of objtype.ModelCache, which run()
// populates for every rendered id via GetSprite -> GetInterfaceModel — so they
// share this helper instead of each calling run() independently. Whichever
// test runs first (by source order, or the only one selected under -run)
// performs the real call; the other reuses the memoized result.
//
// The output dir is deliberately NOT a t.TempDir() of whichever test triggers
// the Once: that would tie its cleanup to that one test's lifetime, and if a
// future test file in this package happened to sort before this one (Go runs
// tests in file-then-declaration order) and triggered the pipeline first, its
// cleanup would delete out from under a later test still trying to read it —
// exactly the kind of ordering hazard this shared-Once design exists to
// avoid. Instead pipelineOut is a plain os.MkdirTemp removed once via
// TestMain, after every test in the binary has finished.
var (
	pipelineOnce sync.Once
	pipelineOut  string
	pipelineSt   stats
	pipelineErr  error
)

// TestMain removes pipelineOut (if runPipelineOnce ever created it) after all
// tests in this binary have run, decoupling its lifetime from whichever
// individual test happened to trigger the shared pipeline.
func TestMain(m *testing.M) {
	code := m.Run()
	if pipelineOut != "" {
		_ = os.RemoveAll(pipelineOut)
	}
	os.Exit(code)
}

// runPipelineOnce returns the memoized result of the single run() call for
// this process (see pipelineOnce's doc comment above), driving it against
// cacheDir if it has not run yet.
func runPipelineOnce(t *testing.T, cacheDir string) (string, stats) {
	t.Helper()
	pipelineOnce.Do(func() {
		out, err := os.MkdirTemp("", "icondump-golden-*")
		if err != nil {
			pipelineErr = fmt.Errorf("create pipeline out dir: %w", err)
			return
		}
		pipelineOut = out
		pipelineSt, pipelineErr = run(iconConfig{cacheDir: cacheDir, outDir: pipelineOut, id: -1})
	})
	if pipelineErr != nil {
		t.Fatalf("run: %v", pipelineErr)
	}
	return pipelineOut, pipelineSt
}

func readSamples(t *testing.T) []sampleEntry {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "icons274", "sample.json"))
	if err != nil {
		t.Fatalf("read sample.json: %v", err)
	}
	var samples []sampleEntry
	if err := json.Unmarshal(raw, &samples); err != nil {
		t.Fatalf("parse sample.json: %v", err)
	}
	if len(samples) == 0 {
		t.Fatal("sample.json has no entries")
	}
	return samples
}

// decodePNGToRGBA reads a 32x32 PNG and returns its pixels as row-major
// non-premultiplied RGBA bytes, matching the .rgba golden layout.
func decodePNGToRGBA(t *testing.T, path string) []byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	b := img.Bounds()
	if b.Dx() != 32 || b.Dy() != 32 {
		t.Fatalf("%s: size %dx%d, want 32x32", path, b.Dx(), b.Dy())
	}
	out := make([]byte, 32*32*4)
	for y := range 32 {
		for x := range 32 {
			c := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			i := (y*32 + x) * 4
			out[i] = c.R
			out[i+1] = c.G
			out[i+2] = c.B
			out[i+3] = c.A
		}
	}
	return out
}

func readGolden(t *testing.T, rel string) []byte {
	t.Helper()
	raw, err := os.ReadFile(rel)
	if err != nil {
		t.Fatalf("read golden %s: %v", rel, err)
	}
	return raw
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

// litGolden mirrors testdata/lit/dagger.json: the reference lit-model output
// for one obj (its FaceColourA/B/C after the recolour + CalculateNormals
// pass ObjType.GetInterfaceModel applies), produced by the pinned Client-TS
// getModelLit path (model load -> recolour -> calculateNormals(ambient+64,
// contrast+768, -50, -10, -50, true)) at deterministic brightness 0.8.
type litGolden struct {
	ID          int    `json:"id"`
	Debugname   string `json:"debugname"`
	ModelID     int    `json:"modelId"`
	NumFaces    int    `json:"numFaces"`
	NumPoints   int    `json:"numPoints"`
	Ambient     int    `json:"ambient"`
	Contrast    int    `json:"contrast"`
	FaceColourA []int  `json:"faceColourA"`
	FaceColourB []int  `json:"faceColourB"`
	FaceColourC []int  `json:"faceColourC"`
}

// TestLitModelGolden drives the same in-process pipeline as
// TestIcondumpMatchesGoldens (shared via runPipelineOnce — run() may only be
// called once per process) and then fetches the golden's obj id's lit model
// exactly the way GetSprite does: List(id).GetInterfaceModel(1). That call is
// a cache hit against objtype.ModelCache, already populated for every
// rendered obj (including this one) by the pipeline's full render pass, so
// it returns the very model GetSprite rasterized bronze_dagger's icon from.
// This wires testdata/lit/dagger.json (committed by the icon-rasterizer
// migration in 1d572c9 but never asserted against) into a real check of the
// per-face lit colour arrays it holds.
func TestLitModelGolden(t *testing.T) {
	cacheDir := os.Getenv(referenceCacheEnv)
	if cacheDir == "" {
		t.Skipf("%s unset; set it to a dir with main_file_cache.dat + main_file_cache.idx0..4", referenceCacheEnv)
	}

	want := readLitGolden(t, filepath.Join("testdata", "lit", "dagger.json"))
	if want.NumFaces != 33 {
		t.Fatalf("golden numFaces = %d, want 33 (testdata/lit/dagger.json is expected to pin 33 faces)", want.NumFaces)
	}

	if _, st := runPipelineOnce(t, cacheDir); st.rendered == 0 {
		t.Fatalf("run rendered 0 icons (rendered=%d skipped=%d total=%d)", st.rendered, st.skipped, st.total)
	}

	got := objtype.List(want.ID).GetInterfaceModel(1)
	if got == nil {
		t.Fatalf("GetInterfaceModel(1) for id %d (%s): nil model", want.ID, want.Debugname)
	}
	assertLitModel(t, want, got)
}

// assertLitModel checks a live lit model's per-face colour arrays against a
// litGolden, element-wise. Extracted from TestLitModelGolden so the mutation
// sensitivity check (see the task report) can drive it directly against a
// perturbed in-memory copy of the golden without touching the committed file.
func assertLitModel(t *testing.T, want litGolden, got *model.Model) {
	t.Helper()
	if len(want.FaceColourA) != want.NumFaces || len(want.FaceColourB) != want.NumFaces || len(want.FaceColourC) != want.NumFaces {
		t.Fatalf("golden array length mismatch: len(A)=%d len(B)=%d len(C)=%d, want numFaces=%d",
			len(want.FaceColourA), len(want.FaceColourB), len(want.FaceColourC), want.NumFaces)
	}
	if got.FaceCount != want.NumFaces {
		t.Fatalf("FaceCount = %d, want %d", got.FaceCount, want.NumFaces)
	}
	if diff := diffIntSlice(got.FaceColourA, want.FaceColourA); diff != "" {
		t.Errorf("FaceColourA mismatch:\n%s", diff)
	}
	if diff := diffIntSlice(got.FaceColourB, want.FaceColourB); diff != "" {
		t.Errorf("FaceColourB mismatch:\n%s", diff)
	}
	if diff := diffIntSlice(got.FaceColourC, want.FaceColourC); diff != "" {
		t.Errorf("FaceColourC mismatch:\n%s", diff)
	}
}

// diffIntSlice summarizes the first few differing elements for a readable
// failure, mirroring describeDiff's shape for the pixel goldens above.
func diffIntSlice(got, want []int) string {
	if len(got) != len(want) {
		return fmt.Sprintf("  length: got %d, want %d", len(got), len(want))
	}
	var b bytes.Buffer
	diffs := 0
	for i := range got {
		if got[i] != want[i] {
			if diffs < 12 {
				fmt.Fprintf(&b, "  [%d]: got %d want %d\n", i, got[i], want[i])
			}
			diffs++
		}
	}
	if diffs == 0 {
		return ""
	}
	fmt.Fprintf(&b, "  total differing: %d/%d", diffs, len(got))
	return b.String()
}

func readLitGolden(t *testing.T, path string) litGolden {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}
	var g litGolden
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("parse golden %s: %v", path, err)
	}
	return g
}

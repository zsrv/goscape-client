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
	"testing"
)

// referenceCacheEnv names an env var pointing at a classic .dat/.idxN cache dir
// (main_file_cache.dat + main_file_cache.idx0..4). TestIcondumpMatchesGoldens
// skips with a clear message when it is unset.
const referenceCacheEnv = "ICONDUMP_TEST_CACHE"

// sampleEntry mirrors one record of testdata/icons2452/sample.json: the obj id
// and its RuneScript debugname (the golden .rgba is keyed by debugname).
type sampleEntry struct {
	ID        int    `json:"id"`
	Debugname string `json:"debugname"`
}

// TestIcondumpMatchesGoldens drives the whole in-process dump pipeline (run)
// against the reference cache into a fresh temp dir, then byte-compares every
// sampled icon's PNG-decoded RGBA against the byte-for-byte reference golden
// produced by tools/iconref/java/IconDump.java (the pinned LostCityRS/Client-Java
// render code, commit 176a85f7). A match proves the Go item-icon path reproduces
// the client pixels.
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

	out := t.TempDir()
	st, err := run(iconConfig{cacheDir: cacheDir, outDir: out, id: -1})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if st.rendered == 0 {
		t.Fatalf("run rendered 0 icons (rendered=%d skipped=%d total=%d)", st.rendered, st.skipped, st.total)
	}
	t.Logf("rendered=%d skipped=%d total=%d", st.rendered, st.skipped, st.total)

	for _, s := range samples {
		t.Run(s.Debugname, func(t *testing.T) {
			got := decodePNGToRGBA(t, filepath.Join(out, strconv.Itoa(s.ID)+".png"))
			want := readGolden(t, filepath.Join("testdata", "icons2452", s.Debugname+".rgba"))
			if len(want) != 32*32*4 {
				t.Fatalf("%s.rgba: got %d bytes, want %d", s.Debugname, len(want), 32*32*4)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("%s (id %d): RGBA mismatch\n%s", s.Debugname, s.ID, describeDiff(got, want, 32, 32))
			}
		})
	}
}

func readSamples(t *testing.T) []sampleEntry {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "icons2452", "sample.json"))
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

//go:build !js

package audio

import (
	"testing"
)

// These tests cover the native midiSource's silence/clipping/buffer
// behavior — the io.Reader contract the persistent oto Player depends on.
// They run without an oto context and without a real meltysynth synth, so
// they're cheap and don't need audio hardware. The audio package as a
// whole still needs ALSA dev headers to compile on Linux; these tests run
// wherever that builds.
//
// The fade/sequencing logic that used to be pinned here (the gain smoother
// that prevented track-change overlap) now lives in audioLoop (audioloop.go,
// the faithful SignLink consumer) and is tested in audioloop_test.go. The
// source itself no longer applies gain: volume rides on the oto Player via
// the audioLoop's stepped setVolume, so the source's only contract is to
// render the current sequencer (or silence when swapped to nil).

func TestMidiSourceNilSeqEmitsSilence(t *testing.T) {
	s := newMidiSource(nil)
	buf := make([]byte, 64)
	n, err := s.Read(buf)
	if err != nil || n != 64 {
		t.Fatalf("Read: n=%d err=%v, want 64 <nil>", n, err)
	}
	for i, b := range buf {
		if b != 0 {
			t.Fatalf("byte %d = %#x, want silence", i, b)
		}
	}
}

func TestMidiSourceSwapToNilSilencesActiveStream(t *testing.T) {
	s := newMidiSource(nil)
	s.swap(nil) // stop(): the source must render silence from the next Read
	buf := make([]byte, 32)
	if _, err := s.Read(buf); err != nil {
		t.Fatal(err)
	}
	for i, b := range buf {
		if b != 0 {
			t.Fatalf("byte %d = %#x, want silence after swap(nil)", i, b)
		}
	}
}

func TestMidiSourceShortBufferReturnsZero(t *testing.T) {
	// oto promises buffers that are multiples of the frame size, but
	// we tolerate odd remainders by rounding down. A buffer too small
	// for a single frame (< 4 bytes) returns (0, nil) so oto can
	// re-request.
	s := newMidiSource(nil)
	buf := make([]byte, 3)
	n, err := s.Read(buf)
	if n != 0 {
		t.Fatalf("Read(<4) returned n=%d, want 0", n)
	}
	if err != nil {
		t.Fatalf("Read(<4) returned err=%v, want nil", err)
	}
}

func TestVolumeFromCentibels(t *testing.T) {
	// Matches the TS client's Math.pow(10, dB / 20) in tinymidipcm.js:300.
	// Centibels are 1/100 dB, so the exponent is cb / 100 / 20 = cb / 2000.
	cases := []struct {
		cb   int
		want float32
	}{
		{0, 1.0},
		{100, 1.0},     // positive clamped to unity (signlink range is 0..-1200)
		{-400, 0.6310}, // -4 dB ≈ 0.631
		{-1200, 0.251}, // -12 dB ≈ 0.251
	}
	for _, c := range cases {
		got := volumeFromCentibels(c.cb)
		// Tolerance is 1e-3 — we're not asserting bit-exact, just that
		// the dB→linear conversion is in the right ballpark.
		diff := got - c.want
		if diff < 0 {
			diff = -diff
		}
		if diff > 1e-3 {
			t.Errorf("volumeFromCentibels(%d) = %v, want ~%v", c.cb, got, c.want)
		}
	}
}

func TestClipInt16Saturates(t *testing.T) {
	// meltysynth's float32 output is nominally -1..1 but transient peaks
	// can exceed the rails. clipInt16 must hard-clip rather than
	// wrap (which produces audible buzz).
	cases := []struct {
		f    float32
		want int16
	}{
		{0, 0},
		{1.0, 32767},
		{-1.0, -32767},
		{2.0, 32767},
		{-2.0, -32768},
		{1e6, 32767},
		{-1e6, -32768},
	}
	for _, c := range cases {
		got := clipInt16(c.f)
		if got != c.want {
			t.Errorf("clipInt16(%v) = %v, want %v", c.f, got, c.want)
		}
	}
}

func TestWave8MonoToStereoInt16ConvertsMidpoint(t *testing.T) {
	// sound/wave.GetWave emits 22050 Hz mono 8-bit unsigned PCM with
	// 0x80 as the silent midpoint (sound/wave.go:138). The converter
	// must subtract 128 before promoting to 16-bit so silence maps to
	// 0 and the stereo channels match.
	wav := makeTestWAV([]byte{0x80, 0xFF, 0x00})
	out, ok := wave8MonoToStereoInt16(wav)
	if !ok {
		t.Fatalf("wave8MonoToStereoInt16 rejected a valid header")
	}
	// 3 input samples × 4 bytes per stereo int16 frame = 12 bytes out.
	if len(out) != 12 {
		t.Fatalf("output len = %d, want 12", len(out))
	}
	// Sample 0: 0x80 (midpoint) → 0
	if out[0] != 0 || out[1] != 0 {
		t.Errorf("midpoint sample: got L=%v R=%v, want 0 0", out[0:2], out[2:4])
	}
}

func TestWave8MonoToStereoInt16RejectsForeignFormat(t *testing.T) {
	// If someone passes a 16-bit or stereo or wrong-sample-rate WAV,
	// we'd produce garbage. Reject early.
	wav := makeTestWAV([]byte{0x80})
	// Corrupt the bits-per-sample field (offset 34) to 16.
	wav[34] = 16
	if _, ok := wave8MonoToStereoInt16(wav); ok {
		t.Errorf("accepted 16-bit input; want rejection")
	}
}

// makeTestWAV builds a minimal RIFF/WAV in the exact format sound.wave
// emits: 22050 Hz, 1 channel, 8-bit unsigned PCM. Header bytes are
// little-endian per the RIFF spec; layout matches sound/wave.go:99.
func makeTestWAV(samples []byte) []byte {
	const headerLen = 44
	buf := make([]byte, headerLen+len(samples))
	copy(buf[0:4], "RIFF")
	// ChunkSize = 36 + sample data length (little-endian uint32).
	putU32LE(buf[4:], uint32(36+len(samples)))
	copy(buf[8:12], "WAVE")
	copy(buf[12:16], "fmt ")
	putU32LE(buf[16:], 16) // fmt chunk size
	putU16LE(buf[20:], 1)  // audio format = PCM
	putU16LE(buf[22:], 1)  // channels
	putU32LE(buf[24:], 22050)
	putU32LE(buf[28:], 22050)
	putU16LE(buf[32:], 1) // block align
	putU16LE(buf[34:], 8) // bits per sample
	copy(buf[36:40], "data")
	putU32LE(buf[40:], uint32(len(samples)))
	copy(buf[44:], samples)
	return buf
}

func putU16LE(b []byte, v uint16) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}

func putU32LE(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

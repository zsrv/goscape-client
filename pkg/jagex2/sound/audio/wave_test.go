package audio

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// makeWAV builds the 22050 Hz / mono / 8-bit unsigned RIFF WAV that
// wave8MonoToStereoInt16 accepts (the format sound/wave.GetWave emits).
func makeWAV(samples []byte) []byte {
	buf := make([]byte, 44+len(samples))
	copy(buf[0:], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:], uint32(36+len(samples)))
	copy(buf[8:], "WAVE")
	copy(buf[12:], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:], 16) // subchunk1 size
	binary.LittleEndian.PutUint16(buf[20:], 1)  // PCM
	binary.LittleEndian.PutUint16(buf[22:], 1)  // mono
	binary.LittleEndian.PutUint32(buf[24:], SampleRate)
	binary.LittleEndian.PutUint32(buf[28:], SampleRate) // byte rate
	binary.LittleEndian.PutUint16(buf[32:], 1)          // block align
	binary.LittleEndian.PutUint16(buf[34:], 8)          // bits per sample
	copy(buf[36:], "data")
	binary.LittleEndian.PutUint32(buf[40:], uint32(len(samples)))
	copy(buf[44:], samples)
	return buf
}

func TestWave8MonoToStereoInt16(t *testing.T) {
	out, ok := wave8MonoToStereoInt16(makeWAV([]byte{0x80, 0xFF, 0x00}))
	if !ok {
		t.Fatal("valid WAV rejected")
	}
	if len(out) != 12 { // 3 samples * 2 channels * 2 bytes
		t.Fatalf("len = %d, want 12", len(out))
	}
	want := []int16{0, 0, 32512, 32512, -32768, -32768}
	for i, w := range want {
		if got := int16(binary.LittleEndian.Uint16(out[i*2:])); got != w {
			t.Errorf("int16 %d: got %d, want %d", i, got, w)
		}
	}
}

func TestWave8MonoToStereoInt16Rejects(t *testing.T) {
	if _, ok := wave8MonoToStereoInt16([]byte("short")); ok {
		t.Error("accepted too-short input")
	}
	wav := makeWAV([]byte{0x80})
	binary.LittleEndian.PutUint16(wav[34:], 16) // 16-bit, not our 8-bit format
	if _, ok := wave8MonoToStereoInt16(wav); ok {
		t.Error("accepted non-8-bit WAV")
	}
}

func TestPlayWaveCachesForReplay(t *testing.T) {
	// Start() is never called in tests, so readyCtx is nil: PlayWave drops
	// playback but must still cache a defensive copy for ReplayWave.
	in := makeWAV([]byte{0x10, 0x20, 0x30})
	PlayWave(in)

	cached := lastWaveForTest()
	if !bytes.Equal(cached, in) {
		t.Fatal("PlayWave did not cache the input bytes")
	}
	in[44] = 0x99
	if lastWaveForTest()[44] == 0x99 {
		t.Error("cache aliases the caller's slice (missing defensive copy)")
	}
}

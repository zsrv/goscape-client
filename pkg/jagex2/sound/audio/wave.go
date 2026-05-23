package audio

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"time"

	"github.com/ebitengine/oto/v3"

	"github.com/zsrv/goscape-client/pkg/sign/signlink"
)

// runWaveWatcher polls signlink.ConsumeWave for newly-saved SFX files.
// Each successful pickup spawns a one-shot Player that drains the
// converted PCM and self-finalizes when done. There's no centralized
// player table because oto v3 doesn't require explicit Close, and the
// audio goroutine inside oto handles cleanup when the reader hits EOF.
const wavePollInterval = 50 * time.Millisecond

func runWaveWatcher(ctx *oto.Context) {
	for {
		path := signlink.ConsumeWave()
		if path != "" && ctx != nil {
			playWaveFile(ctx, path)
		}
		time.Sleep(wavePollInterval)
	}
}

// playWaveFile loads a 22050 Hz mono 8-bit unsigned WAV from disk,
// converts it to 22050 Hz stereo 16-bit signed LE (the shared oto
// context's format), and spawns a Player to drain it. The conversion
// runs once up front rather than streaming — SFX clips top out at a
// few seconds and the wave package's buffer is bounded to ~441 KB
// (sound/wave.go:28), so the in-memory cost is negligible.
//
// CRITICAL: oto's Player relies on a finalizer for cleanup (see the
// note at player.go:93 in oto/v3 — "(*mux.Player).Close() is called
// by the finalizer. Let's rely on it"). If the Player goes
// GC-unreachable before playback finishes, the finalizer will Close
// it mid-stream and the SFX gets cut off. We anchor the Player in a
// goroutine's closure that polls IsPlaying() until the source is
// drained, then drops the reference. Without this anchor, even a
// 1-second sound effect can be silenced after just a few ms of
// audible playback.
func playWaveFile(ctx *oto.Context, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("audio/wave: read %q: %v", path, err)
		return
	}
	stereo, ok := wave8MonoToStereoInt16(data)
	if !ok {
		log.Printf("audio/wave: unsupported WAV format in %q", path)
		return
	}
	p := ctx.NewPlayer(&byteSliceReader{b: stereo})
	// SFX volume is per-Player rather than per-context because each
	// clip is short and the slider only takes effect for *new* sounds
	// — exactly oto.Player.SetVolume's contract. Reading WaveVol once
	// at spawn time matches what the slider's case 0..3 dispatch
	// publishes (client.go:3823-3833) for the four discrete options.
	p.SetVolume(float64(volumeFromCentibels(signlink.ReadWaveVol())))
	p.Play()
	go func() {
		for p.IsPlaying() {
			time.Sleep(50 * time.Millisecond)
		}
	}()
}

// wave8MonoToStereoInt16 parses a RIFF/WAV file emitted by sound.wave
// (22050 Hz, 1 ch, 8-bit unsigned PCM — see sound/wave.GetWave) and
// returns interleaved stereo 16-bit signed LE samples. Returns false
// if the header doesn't match what we expect: any deviation means the
// file wasn't produced by our own tone synthesizer and we'd rather skip
// than play garbage. Header layout matches GetWave in sound/wave.go:99.
func wave8MonoToStereoInt16(data []byte) ([]byte, bool) {
	if len(data) < 44 {
		return nil, false
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, false
	}
	if string(data[12:16]) != "fmt " {
		return nil, false
	}
	audioFormat := binary.LittleEndian.Uint16(data[20:22])
	channels := binary.LittleEndian.Uint16(data[22:24])
	sampleRate := binary.LittleEndian.Uint32(data[24:28])
	bitsPerSample := binary.LittleEndian.Uint16(data[34:36])
	if audioFormat != 1 || channels != 1 || sampleRate != SampleRate || bitsPerSample != 8 {
		return nil, false
	}
	if string(data[36:40]) != "data" {
		return nil, false
	}
	dataLen := int(binary.LittleEndian.Uint32(data[40:44]))
	if 44+dataLen > len(data) {
		dataLen = len(data) - 44
	}
	samples := data[44 : 44+dataLen]

	// 8-bit unsigned PCM uses 0x80 as the silent midpoint; subtract 128
	// to land in signed range, then scale to 16-bit. The shift is the
	// standard 8→16 promotion: (s - 128) << 8 maps 0x80→0, 0xFF→+32512,
	// 0x00→-32768, preserving zero-crossings correctly. Doubled to both
	// channels for the stereo oto context.
	out := make([]byte, len(samples)*4)
	for i, s := range samples {
		v := int16(int(s)-128) << 8
		off := i * 4
		u := uint16(v)
		binary.LittleEndian.PutUint16(out[off:], u)
		binary.LittleEndian.PutUint16(out[off+2:], u)
	}
	return out, true
}

// byteSliceReader is the one-shot io.Reader an SFX Player drains. It
// returns io.EOF after exhausting the slice, which signals oto's audio
// goroutine to release the player.
type byteSliceReader struct {
	b   []byte
	pos int
}

func (r *byteSliceReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.pos:])
	r.pos += n
	return n, nil
}

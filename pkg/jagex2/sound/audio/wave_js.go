//go:build js

package audio

import (
	"log"
	"sync"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/sign/signlink"
)

var (
	waveMu   sync.Mutex
	lastWave []byte
)

// PlayWave plays a one-shot SFX from 22050 Hz mono 8-bit WAV bytes and caches
// a copy for ReplayWave. Dropped silently if the context isn't ready yet.
func PlayWave(data []byte) {
	cp := make([]byte, len(data))
	copy(cp, data)
	waveMu.Lock()
	lastWave = cp
	waveMu.Unlock()
	playWaveBytes(cp)
}

// ReplayWave replays the most recent SFX (Java replaywave).
func ReplayWave() {
	waveMu.Lock()
	data := lastWave
	waveMu.Unlock()
	if data != nil {
		playWaveBytes(data)
	}
}

func playWaveBytes(data []byte) {
	if !ac.Truthy() {
		return // pre-gesture / disabled
	}
	samples, ok := parseWave8Mono(data)
	if !ok {
		log.Printf("audio/wave: unsupported WAV format")
		return
	}
	// 8-bit unsigned (0x80 = silence) -> float32 [-1,1): (s-128)/128.
	f := make([]float32, len(samples))
	for i, s := range samples {
		f[i] = float32(int(s)-128) / 128.0
	}
	buf := ac.Call("createBuffer", 1, len(f), SampleRate)
	buf.Call("copyToChannel", f32ToJSFloat32Array(f), 0)

	sfxGain.Get("gain").Set("value", float64(volumeFromCentibels(signlink.ReadWaveVol())))

	src := ac.Call("createBufferSource")
	src.Set("buffer", buf)
	src.Call("connect", sfxGain)
	src.Call("start")
}

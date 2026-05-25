package audio

import "math"

// Format constants for the audio pipeline. 22050 Hz stereo matches the TS
// reference client and the Wave/SFX pipeline (sound/wave.GetWave).
const (
	SampleRate   = 22050
	ChannelCount = 2
)

// volumeFromCentibels maps signlink's centibel scale (e.g. -400 for -4 dB,
// 0 for full) to a linear amplitude factor: dB = cb/100; linear = 10^(dB/20).
// Matches the TS client's Math.pow(10, dB/20) (tinymidipcm.js:300).
func volumeFromCentibels(cb int) float32 {
	if cb >= 0 {
		return 1.0
	}
	db := float64(cb) / 100.0
	return float32(math.Pow(10, db/20.0))
}

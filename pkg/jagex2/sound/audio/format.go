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

// linearVolume maps the 244 linear volume scale to an amplitude gain vol/256.
// The client sends 128/96/64/32 (Client.java:11372-11414); SignLink defaults
// midivol/wavevol to 96 (SignLink.java:59,71).
//
// Faithfulness proof: Java's MidiPlayer rescales each channel's 14-bit
// CC7/CC39 volume by sqrt(vol/256) before the synth sees it (getVolume:
// sqrt(((cc*vol)>>>8)*cc), MidiPlayer.java:123-126, applied to the file's
// own CC messages via check(), :134-160). meltysynth squares channel volume
// per the GM spec (voice.go:195-197: channelGain = ve*ve), so the audible
// composition is cc²·(vol/256) — meltysynth's native rendering times a
// linear vol/256 post-gain. Applying that gain at the player/gain node
// reproduces Java's volume curve exactly, and MidiPlayer's CC interception
// machinery does not need porting. Deviation: meltysynth handles the file's
// own CC121 per MIDI RP-015 (channel volume NOT reset), where Java's
// wrapper reset its tracked volume to the 12800 default.
func linearVolume(vol int) float64 {
	if vol <= 0 {
		return 0
	}
	if vol >= 256 {
		return 1
	}
	return float64(vol) / 256
}

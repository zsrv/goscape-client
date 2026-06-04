package audio

// Format constants for the audio pipeline. 22050 Hz stereo matches the TS
// reference client and the Wave/SFX pipeline (sound/wave.GetWave).
const (
	SampleRate   = 22050
	ChannelCount = 2
)

// linearVolume maps the 244 linear volume scale to an amplitude gain vol/128,
// clamped to [0,1]. The client sends 128/96/64/32 (Client.java:11372-11414) —
// gains 1/0.75/0.5/0.25, unity at the slider max; SignLink defaults
// midivol/wavevol to 96 (SignLink.java:59,71) — 0.75.
//
// Calibration: the two LostCityRS reconstructions of the wrapper-side
// consumer disagree on the unity point. The Java deob's MidiPlayer rescales
// each channel's 14-bit CC7/CC39 volume by sqrt(vol/256) before the synth
// sees it (getVolume: sqrt(((cc*vol)>>>8)*cc), MidiPlayer.java:123-126,
// applied to the file's own CC messages via check(), :134-160), which —
// through a GM-quadratic synth curve — composes to cc²·(vol/256): unity at
// 256, so the in-game slider max (128) could only ever reach 50% amplitude.
// The TS reference client instead applies vol/128 at its gain nodes
// (tinymidipcm.js:313, audio.js:64), reading the 128/96/64/32 ladder as
// fractions of full scale. We follow the TS model (decided 2026-06-04 after
// the /256 build played audibly half-loud): the shape of the Java algebra is
// preserved — meltysynth squares channel volume per the GM spec
// (voice.go:195-197: channelGain = ve*ve), so a linear post-gain at the
// player/gain node composes with the file's own CC volumes exactly like the
// wrapper's CC rescale, and MidiPlayer's CC interception machinery does not
// need porting — only the denominator is calibrated to the TS reference.
// Deviation: meltysynth handles the file's own CC121 per MIDI RP-015
// (channel volume NOT reset), where Java's wrapper reset its tracked volume
// to the 12800 default.
func linearVolume(vol int) float64 {
	if vol <= 0 {
		return 0
	}
	if vol >= 128 {
		return 1
	}
	return float64(vol) / 128
}

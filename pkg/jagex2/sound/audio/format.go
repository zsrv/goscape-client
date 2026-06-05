package audio

// Format constants for the audio pipeline. 22050 Hz stereo matches the TS
// reference client and the Wave/SFX pipeline (sound/jagfx.GetWave).
const (
	SampleRate   = 22050
	ChannelCount = 2
)

// linearVolume maps the seam's internal linear volume domain to an amplitude
// gain vol/128, clamped to [0,1]. This is the 244 wrapper's scale — the 244
// client sent 128/96/64/32 directly (gains 1/0.75/0.5/0.25, unity at the
// slider max); at 245.2 the client publishes centibels instead, restored to
// this domain at ingestion by centibelToVol128 below.
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

// centibelToVol128 maps the 245.2 publisher scale onto the seam's internal
// linear 0..128 domain. At 245.2 the client publishes music/SFX volume in
// centibels — 0/-400/-800/-1200 (Client.java:10726-10770 @176a85f) — which
// is an exact affine relabel of the 244 ladder: cb = (vol244-128)*12.5,
// i.e. 12.5 cB (1.25 dB) per linear unit. The integer-exact inverse
// 128 + cb*2/25 restores 128/96/64/32, so everything downstream (the ±8
// fade steps in audioloop.go and the vol/128 gain in linearVolume) keeps
// its audible 244-wrapper behaviour bit-for-bit.
//
// Calibration: the 245.2 deob removed the wrapper-side consumer entirely,
// so no Java reference exists for this conversion. The TS reference client
// on branch 245.2 (@bd29ce0) kept the linear ladder and vol/128 gain nodes
// unchanged (Client.ts:10327-10371, tinymidipcm.js:313, audio.js:64) —
// confirming the new constants are a unit change, not an audible one —
// consistent with the rev-244 vol/128 decision (TS as calibration arbiter;
// see linearVolume above). Out-of-ladder values clamp to [0,128]. Note the
// audible default DOES change at 245.2: midivol/wavevol lose their `= 96`
// initializers (signlink.java:51,63 @176a85f), and the zero default is
// 0 cB → 128 = full volume (244 defaulted to 96 = 75%).
func centibelToVol128(cb int) int {
	v := 128 + cb*2/25
	if v < 0 {
		return 0
	}
	if v > 128 {
		return 128
	}
	return v
}

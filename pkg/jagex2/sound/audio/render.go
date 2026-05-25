package audio

import (
	"bytes"
	"time"

	"github.com/sinshu/go-meltysynth/meltysynth"
)

// renderTailFrames is the extra silence rendered after a track's musical
// length so trailing note releases aren't cut off. Reverb/chorus are
// disabled, so 1s is ample.
const renderTailFrames = SampleRate

// renderFrameCount returns how many PCM frames to render for a track of the
// given musical length: length rounded to frames, plus the release tail.
func renderFrameCount(length time.Duration) int {
	frames := int(length.Seconds()*float64(SampleRate)) + renderTailFrames
	if frames < renderTailFrames {
		frames = renderTailFrames
	}
	return frames
}

// renderMidiToPCM synthesizes an entire MIDI track to left/right float32 PCM
// at SampleRate (one Render call covers any length; the sequencer renders the
// decay tail past the last event with loop=false). Reverb/chorus disabled to
// match the native path. Returns equal-length left/right slices.
func renderMidiToPCM(sf *meltysynth.SoundFont, midData []byte) (left, right []float32, err error) {
	midiFile, err := meltysynth.NewMidiFile(bytes.NewReader(midData))
	if err != nil {
		return nil, nil, err
	}
	settings := meltysynth.NewSynthesizerSettings(SampleRate)
	settings.EnableReverbAndChorus = false
	synth, err := meltysynth.NewSynthesizer(sf, settings)
	if err != nil {
		return nil, nil, err
	}
	seq := meltysynth.NewMidiFileSequencer(synth)
	seq.Play(midiFile, false) // no synth-side loop; game re-issues SetMidi
	frames := renderFrameCount(midiFile.GetLength())
	left = make([]float32, frames)
	right = make([]float32, frames)
	seq.Render(left, right)
	return left, right, nil
}

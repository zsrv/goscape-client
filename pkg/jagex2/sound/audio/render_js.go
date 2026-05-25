//go:build js

package audio

import (
	"bytes"

	"github.com/sinshu/go-meltysynth/meltysynth"
)

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

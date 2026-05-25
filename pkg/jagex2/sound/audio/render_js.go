//go:build js

package audio

import (
	"bytes"
	"time"

	"github.com/sinshu/go-meltysynth/meltysynth"
)

// renderChunkFrames is how many frames are synthesized between yields. The
// sequencer is stateful, so rendering consecutive sub-slices continues the
// track. ~250ms keeps each synthesis burst to roughly one frame's worth of CPU.
const renderChunkFrames = SampleRate / 4

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
	// Render in chunks, yielding to the JS event loop between them so the game
	// loop keeps drawing during the (100s-of-ms) synthesis instead of freezing.
	// Audio is unaffected: it plays from the previous static buffer until this
	// render swaps in. (Run on a background goroutine — see playFromBytes.)
	for off := 0; off < frames; off += renderChunkFrames {
		end := off + renderChunkFrames
		if end > frames {
			end = frames
		}
		seq.Render(left[off:end], right[off:end])
		time.Sleep(time.Millisecond)
	}
	return left, right, nil
}

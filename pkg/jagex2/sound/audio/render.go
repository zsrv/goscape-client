package audio

import (
	"time"
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

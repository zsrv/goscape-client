package audio

import (
	"testing"
	"time"
)

func TestRenderFrameCount(t *testing.T) {
	// 2.0s track at 22050 Hz = 44100 frames + 1s (22050) release tail.
	got := renderFrameCount(2 * time.Second)
	if got != 44100+SampleRate {
		t.Fatalf("renderFrameCount(2s) = %d; want %d", got, 44100+SampleRate)
	}
	// A zero-length track still renders at least the tail (never 0).
	if z := renderFrameCount(0); z != SampleRate {
		t.Fatalf("renderFrameCount(0) = %d; want %d (tail only)", z, SampleRate)
	}
}

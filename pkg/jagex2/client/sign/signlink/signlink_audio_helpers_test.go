package signlink

import (
	"bytes"
	"sync"
	"testing"
)

// These tests cover the audio-facing helpers added so the playback
// subsystem can read the Midi/MidiFade/MidiVol fields without
// racing the signlink polling loop (which holds mu while publishing).
//
// Before these helpers existed, client.StopMidi and client.SetMidiVolume
// wrote signlink.Midi/MidiFade/MidiVol directly without holding mu — a
// latent race against signlink.Run's same-field writes that only stayed
// silent because nothing read those fields. The new audio package is
// the first reader; without these helpers `go test -race` flagged the
// race in the live game. The test below is a unit-level pin: it doesn't
// reproduce the full live timing, just confirms the helpers do take the
// mutex (race detector catches it if they regress to bare access).

func TestMidiFadeAndVolRoundTrip(t *testing.T) {
	t.Cleanup(resetSignlinkAudioFields)

	SetMidiFade(1)
	if got := ReadMidiFade(); got != 1 {
		t.Fatalf("MidiFade round-trip: got %d, want 1", got)
	}
	SetMidiVol(-400)
	if got := ReadMidiVol(); got != -400 {
		t.Fatalf("MidiVol round-trip: got %d, want -400", got)
	}
}

func TestSetMidiCommandIsRaceFree(t *testing.T) {
	t.Cleanup(resetSignlinkAudioFields)

	// Hammer SetMidiCommand and PeekMidi/ClearMidi from multiple goroutines
	// so `go test -race` flags any unsynchronized access regression. We're
	// not asserting a specific final value — just that nothing trips the
	// race detector. The writer/reader split here approximates the live
	// runtime: client.go goroutines write commands; the audioLoop consumer
	// goroutine peeks and clears them.
	const writers, readers, iters = 4, 2, 500
	var wg sync.WaitGroup
	for range writers {
		wg.Go(func() {
			for range iters {
				SetMidiCommand("stop")
			}
		})
	}
	for range readers {
		wg.Go(func() {
			for range iters {
				if cmd, _ := PeekMidi(); cmd != "" {
					ClearMidi()
				}
			}
		})
	}
	wg.Wait()
}

// TestMidiSlotSingleSlotClobber pins the Java single-slot protocol: SignLink
// has ONE `midi` field (SignLink.java:45), so a "stop" issued while a track
// is pending replaces it (the track is lost), and vice versa. PeekMidi must
// NOT clear — the consumer clears separately (the fade-out latch,
// SignLink.java:422-424).
func TestMidiSlotSingleSlotClobber(t *testing.T) {
	t.Cleanup(ClearMidi)

	SetMidiTrack([]byte{1, 2})
	SetMidiCommand("stop")
	cmd, data := PeekMidi()
	if cmd != "stop" || data != nil {
		t.Fatalf("stop should clobber the pending track: got %q %v", cmd, data)
	}

	SetMidiTrack([]byte{3})
	cmd, data = PeekMidi()
	if cmd != "play" || !bytes.Equal(data, []byte{3}) {
		t.Fatalf("track should clobber the pending stop: got %q %v", cmd, data)
	}
	if again, _ := PeekMidi(); again != "play" {
		t.Fatalf("PeekMidi must not clear the slot: got %q", again)
	}

	ClearMidi()
	if cmd, data := PeekMidi(); cmd != "" || data != nil {
		t.Fatalf("ClearMidi should empty the slot: got %q %v", cmd, data)
	}
}

// resetSignlinkAudioFields wipes the audio-facing globals between tests
// so cross-test ordering is irrelevant. signlink's globals normally
// persist for process lifetime — fine in production, but a footgun for
// tests, hence this cleanup.
func resetSignlinkAudioFields() {
	mu.Lock()
	defer mu.Unlock()
	Midi = ""
	MidiData = nil
	MidiFade = 0
	MidiVol = 96
}

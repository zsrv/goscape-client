package signlink

import (
	"sync"
	"testing"
)

// These tests cover the audio-facing helpers added so the playback
// subsystem can read the Midi/Wave/MidiFade/MidiVol fields without
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

func TestConsumeMidiClearsAndReturns(t *testing.T) {
	t.Cleanup(resetSignlinkAudioFields)

	SetMidiCommand("scape_main.mid")
	got := ConsumeMidi()
	if got != "scape_main.mid" {
		t.Fatalf("ConsumeMidi: got %q, want %q", got, "scape_main.mid")
	}
	if again := ConsumeMidi(); again != "" {
		t.Fatalf("ConsumeMidi should clear: got %q, want \"\"", again)
	}
}

func TestConsumeWaveClearsAndReturns(t *testing.T) {
	t.Cleanup(resetSignlinkAudioFields)

	mu.Lock()
	Wave = "/tmp/sound0.wav"
	mu.Unlock()

	if got := ConsumeWave(); got != "/tmp/sound0.wav" {
		t.Fatalf("ConsumeWave: got %q, want %q", got, "/tmp/sound0.wav")
	}
	if again := ConsumeWave(); again != "" {
		t.Fatalf("ConsumeWave should clear: got %q, want \"\"", again)
	}
}

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

	// Hammer SetMidiCommand and ConsumeMidi from multiple goroutines so
	// `go test -race` flags any unsynchronized access regression. We're
	// not asserting a specific final value — just that nothing trips the
	// race detector. The writer/reader split here approximates the live
	// runtime: client.go goroutines write commands; the audio watcher
	// goroutine consumes them.
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
				_ = ConsumeMidi()
			}
		})
	}
	wg.Wait()
}

// resetSignlinkAudioFields wipes the audio-facing globals between tests
// so cross-test ordering is irrelevant. signlink's globals normally
// persist for process lifetime — fine in production, but a footgun for
// tests, hence this cleanup.
func resetSignlinkAudioFields() {
	mu.Lock()
	defer mu.Unlock()
	Midi = ""
	Wave = ""
	MidiFade = 0
	MidiVol = 0
}

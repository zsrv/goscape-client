//go:build js

package audio

import (
	"log"
	"sync"
	"syscall/js"
	"time"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/sign/signlink"
)

// One AudioContext for the process. Created in Start; resumed on the first
// user gesture (browsers block autoplay until then).
var (
	ac        js.Value // AudioContext
	musicGain js.Value // GainNode: user music volume (MidiVol)
	sfxGain   js.Value // GainNode: user SFX volume (WaveVol)
	gestures  []js.Func
)

// midiPollInterval matches the native watcher cadence (one game tick).
const midiPollInterval = 20 * time.Millisecond

// driver registry: PlayMIDI callers wait until Start has created the driver
// (or registered nil on disable), so the first SaveMidi can't race startup.
var (
	driverMu    sync.Mutex
	driver      *webMidiDriver
	driverReady = make(chan struct{})
)

func registerMidiDriver(d *webMidiDriver) {
	driverMu.Lock()
	driver = d
	driverMu.Unlock()
	close(driverReady)
}

func getMidiDriver() *webMidiDriver {
	<-driverReady
	driverMu.Lock()
	defer driverMu.Unlock()
	return driver
}

// Start brings up the Web Audio context and the MIDI driver + watcher.
// Called once from cmd/client/main.go on a goroutine.
func Start() {
	ctor := js.Global().Get("AudioContext")
	if !ctor.Truthy() {
		ctor = js.Global().Get("webkitAudioContext")
	}
	if !ctor.Truthy() {
		log.Printf("audio: Web Audio unavailable, game will run silently")
		registerMidiDriver(nil)
		return
	}
	ac = ctor.New(map[string]any{"sampleRate": SampleRate})

	musicGain = ac.Call("createGain")
	musicGain.Call("connect", ac.Get("destination"))
	sfxGain = ac.Call("createGain")
	sfxGain.Call("connect", ac.Get("destination"))

	installGestureResume()

	d := newWebMidiDriver()
	registerMidiDriver(d)
	go runMidiWatcher(d)
}

// DisableForLowMemory matches the native low-memory path: no context, no
// soundfont, nil driver so any stray PlayMIDI returns silently.
func DisableForLowMemory() {
	registerMidiDriver(nil)
}

// installGestureResume resumes the AudioContext on the first user gesture
// (touchend/keyup/mouseup), mirroring oto's autoplay handling.
func installGestureResume() {
	var resume js.Func
	resume = js.FuncOf(func(this js.Value, args []js.Value) any {
		ac.Call("resume")
		for _, e := range []string{"touchend", "keyup", "mouseup"} {
			js.Global().Call("removeEventListener", e, resume)
		}
		resume.Release()
		return nil
	})
	gestures = append(gestures, resume)
	for _, e := range []string{"touchend", "keyup", "mouseup"} {
		js.Global().Call("addEventListener", e, resume)
	}
}

func runMidiWatcher(d *webMidiDriver) {
	for {
		if cmd := signlink.ConsumeMidi(); cmd != "" {
			d.handle(cmd)
		}
		time.Sleep(midiPollInterval)
	}
}

//go:build js

package audio

import (
	"log"
	"syscall/js"
)

// One AudioContext for the process. Created in Start; resumed on the first
// user gesture (browsers block autoplay until then).
var (
	ac        js.Value // AudioContext
	musicGain js.Value // GainNode: MIDI volume — the audioLoop's stepped vol/128 lands here
	sfxGain   js.Value // GainNode: user SFX volume (WaveVol)
	gestures  []js.Func
)

// Start brings up the Web Audio context and spawns the shared audioLoop
// (the faithful SignLink consumer, audioloop.go) ticking the web MIDI sink.
// Called once from cmd/client/main.go on a goroutine.
func Start() {
	ctor := js.Global().Get("AudioContext")
	if !ctor.Truthy() {
		ctor = js.Global().Get("webkitAudioContext")
	}
	if !ctor.Truthy() {
		log.Printf("audio: Web Audio unavailable, game will run silently")
		go runAudioLoop(nullSink{})
		return
	}
	ac = ctor.New(map[string]any{"sampleRate": SampleRate})

	musicGain = ac.Call("createGain")
	musicGain.Call("connect", ac.Get("destination"))
	sfxGain = ac.Call("createGain")
	sfxGain.Call("connect", ac.Get("destination"))

	installGestureResume()

	go runAudioLoop(newWebMidiDriver())
}

// DisableForLowMemory matches the native low-memory path: no context, no
// soundfont; a nullSink loop drains the signlink slot (logout's stopMidi is
// not lowmem-gated).
func DisableForLowMemory() {
	go runAudioLoop(nullSink{})
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

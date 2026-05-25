//go:build js

package platform

import "time"

// Main builds the browser backend and runs loop in a goroutine. The loop yields
// to the JS event loop via time.Sleep (Go's wasm runtime parks the goroutine on
// timers), so the page composites and DOM input fires — the TS-client model, no
// requestAnimationFrame. main() never returns (select{} blocks the program).
func Main(width, height int, title string, loop func()) {
	Active = newJSBackend(width, height, title)
	go loop()
	select {}
}

// Yield briefly returns control to the JS event loop in the middle of a long
// synchronous burst on the single wasm thread. Go's wasm runtime parks the
// goroutine on a JS timer, runs other runnable goroutines (the streaming MIDI
// synth that feeds oto/Web Audio) and the browser event loop, then resumes.
// Call it at coarse boundaries in heavy work (e.g. BuildScene's per-region map
// decode) so the audio buffer cannot drain and skip. No-op on native, where
// audio runs on its own OS thread.
func Yield() { time.Sleep(time.Millisecond) }

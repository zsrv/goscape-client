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

// yieldInterval is how much wall time may pass between actual yields. Kept well
// under the oto audio buffer (~100ms, see sound/audio) so the streaming MIDI
// synth cannot fall behind and underrun between yields.
const yieldInterval = 50 * time.Millisecond

var lastYield time.Time

// Yield returns control to the JS event loop — but only if at least
// yieldInterval has elapsed since the last actual yield, so it is cheap to call
// inside tight loops (one time check per call; a real sleep only ~20×/sec).
// Go's wasm runtime parks the goroutine on a JS timer, runs other runnable
// goroutines (the streaming MIDI synth that feeds oto/Web Audio) and the browser
// event loop, then resumes. Call it inside any heavy synchronous loop reachable
// on wasm (BuildScene's per-region decode, world.LoadLocations' per-loc model
// loads) so the audio buffer cannot drain and skip. No-op on native, where audio
// runs on its own OS thread. Only the single wasm goroutine calls this, so the
// unsynchronized lastYield is race-free.
func Yield() {
	if time.Since(lastYield) >= yieldInterval {
		time.Sleep(time.Millisecond)
		lastYield = time.Now()
	}
}

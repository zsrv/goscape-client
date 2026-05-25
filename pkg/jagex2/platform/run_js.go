//go:build js

package platform

// Main builds the browser backend and runs loop in a goroutine. The loop yields
// to the JS event loop via time.Sleep (Go's wasm runtime parks the goroutine on
// timers), so the page composites and DOM input fires — the TS-client model, no
// requestAnimationFrame. main() never returns (select{} blocks the program).
func Main(width, height int, title string, loop func()) {
	Active = newJSBackend(width, height, title)
	go loop()
	select {}
}

//go:build !js

package platform

import "runtime"

// Main builds the native backend and runs loop on the main OS thread. GLFW and
// OpenGL are thread-affine, so the window, all GL calls, and the game loop must
// share the locked main goroutine. Blocks until loop returns.
func Main(width, height int, title string, loop func()) {
	runtime.LockOSThread()
	Active = newGLFWBackend(width, height, title)
	loop()
}

// Yield is a no-op on native: audio runs on its own OS thread, so a long CPU
// burst on the loop goroutine never starves it (unlike single-threaded wasm,
// where Yield returns control to the JS event loop so Web Audio stays fed).
func Yield() {}

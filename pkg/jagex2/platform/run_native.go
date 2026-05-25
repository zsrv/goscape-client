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

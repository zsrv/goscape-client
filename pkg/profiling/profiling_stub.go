//go:build !unix

package profiling

import (
	"log"
	"runtime"
)

// Start is a no-op on targets without POSIX signals (notably js/wasm
// and Windows), where syscall.SIGUSR1 does not exist. Signal-triggered
// profile capture is unavailable there; the call is kept so the rest
// of the program can wire profiling.Start unconditionally. The Unix
// implementation lives in profiling_signal.go.
func Start() {
	log.Printf("profiling: signal-triggered capture unavailable on %s/%s", runtime.GOOS, runtime.GOARCH)
}

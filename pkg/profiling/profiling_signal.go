//go:build unix

package profiling

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	// cpuWindow is the duration over which the CPU profile, runtime
	// trace, and mutex/block contention sampling all run. 30s is the
	// standard pprof window; long enough for stable sampling, short
	// enough to capture a single phase of gameplay.
	cpuWindow = 30 * time.Second

	// outputBaseDir is where session subdirectories are created,
	// relative to the process working directory at start.
	outputBaseDir = "profiles"
)

// Start registers a SIGUSR1 signal handler that triggers a full
// profile capture (CPU + heap + goroutine + mutex + block + trace).
// It returns immediately after registration. The listener goroutine
// runs for the lifetime of the process.
//
// To trigger a capture from another terminal:
//
//	kill -USR1 <pid>
//
// The PID is logged by this function at registration time. Sessions
// are written under ./profiles/<UTC-timestamp>/ relative to the
// working directory at process start.
//
// This implementation is Unix-only because SIGUSR1 has no equivalent
// on js/wasm or Windows; see profiling_stub.go for those targets.
func Start() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR1)
	go func() {
		for range ch {
			// Dispatch into a fresh goroutine so the listener never
			// blocks on capture I/O — if it did, subsequent signals
			// could be dropped by the runtime.
			go captureAll(outputBaseDir, cpuWindow)
		}
	}()
	log.Printf("profiling: signal listener ready, send SIGUSR1 to pid %d", os.Getpid())
}

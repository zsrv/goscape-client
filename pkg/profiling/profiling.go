// Package profiling provides in-process performance-profile capture
// triggered by SIGUSR1. See docs/superpowers/specs/2026-05-22-perf-profiling-design.md
// for the full design.
package profiling

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync/atomic"
	"time"
)

// sessionTimestamp formats t as a compact ISO 8601 basic-format UTC
// string ("YYYYMMDDTHHMMSSZ"). Lexicographically sortable, no colons,
// no whitespace — safe across all filesystems.
func sessionTimestamp(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// sessionDir creates <base>/<ts>/ (idempotent), resolves it to an
// absolute path, and returns that. The absolute path is what the
// caller will log so the user can navigate to it directly.
func sessionDir(base, ts string) (string, error) {
	dir := filepath.Join(base, ts)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		// Fall back to the relative path on the (very unlikely) Abs
		// failure — better to log something than to abort the whole
		// capture over a path-resolution edge case.
		return dir, nil
	}
	return abs, nil
}

// writeSnapshotProfile writes the named runtime/pprof profile to path.
// Use this for any profile that produces an instantaneous snapshot:
// "heap", "goroutine", "mutex", "block". CPU profiles use a different
// API and are not handled here.
//
// If the named profile is unknown, no file is created and an error
// is returned.
func writeSnapshotProfile(name, path string) error {
	p := pprof.Lookup(name)
	if p == nil {
		return fmt.Errorf("profiling: unknown profile %q", name)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("profiling: open %s: %w", path, err)
	}
	defer f.Close()
	if err := p.WriteTo(f, 0); err != nil {
		return fmt.Errorf("profiling: write %s: %w", path, err)
	}
	return nil
}

// writeCPUAndTrace runs runtime/pprof CPU profiling and runtime/trace
// concurrently for the given window, writing cpu.prof and trace.out
// into dir. The two profilers can run simultaneously; the stdlib
// supports it.
//
// If either profile fails to start, the other is unaffected: e.g., a
// failure to open trace.out still produces a usable cpu.prof. The
// returned error joins any per-profile errors so the caller can log
// them all at once.
func writeCPUAndTrace(dir string, window time.Duration) error {
	var (
		cpuErr   error
		traceErr error
	)

	cpuPath := filepath.Join(dir, "cpu.prof")
	if cpuFile, err := os.OpenFile(cpuPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err != nil {
		cpuErr = fmt.Errorf("profiling: open %s: %w", cpuPath, err)
	} else if err := pprof.StartCPUProfile(cpuFile); err != nil {
		cpuErr = fmt.Errorf("profiling: start cpu: %w", err)
		cpuFile.Close()
	} else {
		defer cpuFile.Close()
		defer pprof.StopCPUProfile()
	}

	tracePath := filepath.Join(dir, "trace.out")
	if traceFile, err := os.OpenFile(tracePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err != nil {
		traceErr = fmt.Errorf("profiling: open %s: %w", tracePath, err)
	} else if err := trace.Start(traceFile); err != nil {
		traceErr = fmt.Errorf("profiling: start trace: %w", err)
		traceFile.Close()
	} else {
		defer traceFile.Close()
		defer trace.Stop()
	}

	time.Sleep(window)

	// Deferred Stop / StopCPUProfile + Close calls fire here in LIFO
	// order: trace.Stop, traceFile.Close, pprof.StopCPUProfile, cpuFile.Close.

	if cpuErr != nil && traceErr != nil {
		return fmt.Errorf("%w; %w", cpuErr, traceErr)
	}
	if cpuErr != nil {
		return cpuErr
	}
	return traceErr
}

// enableContentionProfiles turns on mutex and block contention
// sampling at fraction/rate = 1 (every event recorded). The cost is
// small but non-zero; the caller is responsible for calling
// disableContentionProfiles when the capture window ends so the rest
// of the process runs with zero contention-profiling overhead.
func enableContentionProfiles() {
	runtime.SetMutexProfileFraction(1)
	runtime.SetBlockProfileRate(1)
}

// disableContentionProfiles turns off mutex and block contention
// sampling. Always paired with a prior enableContentionProfiles().
func disableContentionProfiles() {
	runtime.SetMutexProfileFraction(0)
	runtime.SetBlockProfileRate(0)
}

// inFlight prevents overlapping captureAll runs. If a second SIGUSR1
// arrives while a capture is active, the second invocation logs a
// skip message and returns immediately. We do not queue: queuing
// would surprise users who expect each signal to start an immediate
// capture.
var inFlight atomic.Bool

// captureAll runs one complete capture session: enables mutex/block
// sampling, runs CPU + trace concurrently for the window, then
// snapshots heap/goroutine/mutex/block, then disables contention
// sampling. All errors are logged but never returned — profiling is
// strictly best-effort and must not affect gameplay.
//
// outBase is the directory under which a new timestamped session
// subdirectory is created. cpuWindow is the CPU + trace + contention
// sampling duration.
func captureAll(outBase string, cpuWindow time.Duration) {
	if !inFlight.CompareAndSwap(false, true) {
		log.Printf("profiling: capture already in flight, ignoring SIGUSR1")
		return
	}
	defer inFlight.Store(false)

	defer func() {
		if r := recover(); r != nil {
			// A profiling-side panic must never crash the game.
			log.Printf("profiling: capture panicked: %v", r)
		}
	}()

	ts := sessionTimestamp(time.Now())
	dir, err := sessionDir(outBase, ts)
	if err != nil {
		log.Printf("profiling: mkdir %s/%s: %v", outBase, ts, err)
		return
	}

	enableContentionProfiles()
	defer disableContentionProfiles()

	if err := writeCPUAndTrace(dir, cpuWindow); err != nil {
		log.Printf("profiling: cpu/trace: %v", err)
		// Continue — we can still snapshot the four instant profiles.
	}

	// runtime.GC forces a mark phase so the heap profile reflects
	// post-GC live/garbage classification. The net/http/pprof heap
	// handler does this internally; programmatic capture must do it
	// explicitly.
	runtime.GC()

	for _, name := range []string{"heap", "goroutine", "mutex", "block"} {
		path := filepath.Join(dir, name+".prof")
		if err := writeSnapshotProfile(name, path); err != nil {
			log.Printf("profiling: %s: %v", name, err)
			// Continue to the next profile — one failure does not
			// abort the session.
		}
	}

	log.Printf("profiling: capture complete: %s", dir)
}

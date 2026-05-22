// Package profiling provides in-process performance-profile capture
// triggered by SIGUSR1. See docs/superpowers/specs/2026-05-22-perf-profiling-design.md
// for the full design.
package profiling

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
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

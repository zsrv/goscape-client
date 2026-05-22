// Package profiling provides in-process performance-profile capture
// triggered by SIGUSR1. See docs/superpowers/specs/2026-05-22-perf-profiling-design.md
// for the full design.
package profiling

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
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

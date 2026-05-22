# Performance profiling infrastructure — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a SIGUSR1-triggered, in-process performance-profile capture mechanism that emits CPU, heap, goroutine, mutex, block, and runtime/trace artifacts to `./profiles/<timestamp>/` per session, with best-effort error handling that never affects gameplay.

**Architecture:** New self-contained `pkg/profiling/` package with one exported function, `Start()`. A signal listener goroutine catches SIGUSR1 and dispatches an internal `captureAll(outBase, cpuWindow)` to do the I/O. An `atomic.Bool` named `inFlight` prevents overlapping captures; mutex/block sampling is enabled only during the capture window. `cmd/client/main.go` calls `profiling.Start()` once before the existing `wg.Go` launches.

**Tech Stack:** Go 1.26 stdlib only — `runtime`, `runtime/pprof`, `runtime/trace`, `os/signal`, `syscall`, `path/filepath`, `time`, `log`, `sync/atomic`. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-05-22-perf-profiling-design.md` (committed `cfe2ced`).

**Reference conventions** (from `CLAUDE.md` + `CLAUDE.local.md`):
- All `go` invocations prefix: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go`
- All commits use: `git commit --no-gpg-sign`
- Invoke `use-modern-go` skill before writing Go code
- `go.mod` / `go.sum` are intentionally untracked — do not stage them

---

## Task 1: Scaffold the `profiling` package with a session-timestamp helper

**Files:**
- Create: `pkg/profiling/profiling.go`
- Create: `pkg/profiling/profiling_test.go`

The smallest testable building block is the timestamp formatter that names session directories. We TDD it first, then everything else compounds on top.

- [ ] **Step 1: Write the failing test for `sessionTimestamp`**

Create `pkg/profiling/profiling_test.go`:

```go
package profiling

import (
	"testing"
	"time"
)

func TestSessionTimestamp_Format(t *testing.T) {
	got := sessionTimestamp(time.Date(2026, 5, 22, 14, 30, 15, 123_000_000, time.UTC))
	want := "20260522T143015Z"
	if got != want {
		t.Errorf("sessionTimestamp = %q; want %q", got, want)
	}
}

func TestSessionTimestamp_SortableByTime(t *testing.T) {
	early := sessionTimestamp(time.Date(2026, 5, 22, 14, 30, 15, 0, time.UTC))
	later := sessionTimestamp(time.Date(2026, 5, 22, 14, 30, 16, 0, time.UTC))
	if !(early < later) {
		t.Errorf("expected %q < %q lexicographically", early, later)
	}
}

func TestSessionTimestamp_AlwaysUTC(t *testing.T) {
	// Caller may pass a non-UTC time; the formatted string must still
	// reflect UTC so two captures from machines in different timezones
	// sort sensibly together.
	losAngeles, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skipf("tz data unavailable: %v", err)
	}
	got := sessionTimestamp(time.Date(2026, 5, 22, 7, 30, 15, 0, losAngeles))
	want := "20260522T143015Z"
	if got != want {
		t.Errorf("sessionTimestamp on LA-local 07:30 = %q; want %q (UTC)", got, want)
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/profiling/...`

Expected: build error mentioning `undefined: sessionTimestamp` (profiling.go does not exist yet).

- [ ] **Step 3: Create `profiling.go` with `sessionTimestamp`**

Create `pkg/profiling/profiling.go`:

```go
// Package profiling provides in-process performance-profile capture
// triggered by SIGUSR1. See docs/superpowers/specs/2026-05-22-perf-profiling-design.md
// for the full design.
package profiling

import "time"

// sessionTimestamp formats t as a compact ISO 8601 basic-format UTC
// string ("YYYYMMDDTHHMMSSZ"). Lexicographically sortable, no colons,
// no whitespace — safe across all filesystems.
func sessionTimestamp(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}
```

- [ ] **Step 4: Run the tests and verify they pass**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/profiling/... -v`

Expected: `PASS` for `TestSessionTimestamp_Format`, `TestSessionTimestamp_SortableByTime`, `TestSessionTimestamp_AlwaysUTC`.

- [ ] **Step 5: Commit**

```bash
git add pkg/profiling/profiling.go pkg/profiling/profiling_test.go
git commit --no-gpg-sign -m "feat(profiling): scaffold package with sessionTimestamp helper"
```

---

## Task 2: Session directory creation

`sessionDir(base, ts string) (string, error)` creates `<base>/<ts>/`, resolves its absolute path (the spec requires absolute paths in log output), and returns that absolute path.

**Files:**
- Modify: `pkg/profiling/profiling.go`
- Modify: `pkg/profiling/profiling_test.go`

- [ ] **Step 1: Add the failing tests**

Append to `pkg/profiling/profiling_test.go`:

```go
import (
	"os"               // add to existing import block
	"path/filepath"    // add to existing import block
	"strings"          // add to existing import block
)

func TestSessionDir_CreatesAndReturnsAbsolute(t *testing.T) {
	base := t.TempDir()
	ts := "20260522T143015Z"
	got, err := sessionDir(base, ts)
	if err != nil {
		t.Fatalf("sessionDir: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("returned path %q is not absolute", got)
	}
	if !strings.HasSuffix(got, ts) {
		t.Errorf("returned path %q does not end with timestamp %q", got, ts)
	}
	info, err := os.Stat(got)
	if err != nil {
		t.Fatalf("stat created dir: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("created path is not a directory")
	}
}

func TestSessionDir_IdempotentOnExisting(t *testing.T) {
	base := t.TempDir()
	ts := "20260522T143015Z"
	if _, err := sessionDir(base, ts); err != nil {
		t.Fatalf("first sessionDir: %v", err)
	}
	if _, err := sessionDir(base, ts); err != nil {
		t.Errorf("second sessionDir on existing dir should succeed; got %v", err)
	}
}
```

Update the existing import block at the top of the file. The final import block for this task:

```go
import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)
```

- [ ] **Step 2: Run the tests and verify they fail**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/profiling/... -v -run TestSessionDir`

Expected: build error mentioning `undefined: sessionDir`.

- [ ] **Step 3: Implement `sessionDir`**

Append to `pkg/profiling/profiling.go`:

```go
import (
	"os"            // add to existing import block
	"path/filepath" // add to existing import block
)

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
```

Final import block at top of `profiling.go`:

```go
import (
	"os"
	"path/filepath"
	"time"
)
```

- [ ] **Step 4: Run the tests and verify they pass**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/profiling/... -v`

Expected: PASS for both `TestSessionDir_*` tests plus the three existing timestamp tests.

- [ ] **Step 5: Commit**

```bash
git add pkg/profiling/profiling.go pkg/profiling/profiling_test.go
git commit --no-gpg-sign -m "feat(profiling): add sessionDir helper for output directory creation"
```

---

## Task 3: Snapshot profile writer (`writeSnapshotProfile`)

A single helper that handles the four "instant snapshot" profile types (heap, goroutine, mutex, block) uniformly. We test it with `goroutine` because that one always has non-empty output (every running goroutine appears), so the test is reliable regardless of GC timing or mutex contention.

**Files:**
- Modify: `pkg/profiling/profiling.go`
- Modify: `pkg/profiling/profiling_test.go`

- [ ] **Step 1: Add the failing test**

Append to `pkg/profiling/profiling_test.go`:

```go
func TestWriteSnapshotProfile_GoroutineNonEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "goroutine.prof")
	if err := writeSnapshotProfile("goroutine", path); err != nil {
		t.Fatalf("writeSnapshotProfile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if info.Size() == 0 {
		t.Errorf("goroutine profile is empty")
	}
}

func TestWriteSnapshotProfile_UnknownNameErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bogus.prof")
	err := writeSnapshotProfile("not-a-real-profile-name", path)
	if err == nil {
		t.Errorf("expected error for unknown profile name; got nil")
	}
	// File should not have been created.
	if _, statErr := os.Stat(path); statErr == nil {
		t.Errorf("expected file to NOT exist on error path")
	}
}
```

- [ ] **Step 2: Run the tests and verify they fail**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/profiling/... -v -run TestWriteSnapshotProfile`

Expected: build error mentioning `undefined: writeSnapshotProfile`.

- [ ] **Step 3: Implement `writeSnapshotProfile`**

Append to `pkg/profiling/profiling.go`:

```go
import (
	"fmt"           // add to existing import block
	"runtime/pprof" // add to existing import block
)

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
```

Final import block at top of `profiling.go`:

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"
)
```

- [ ] **Step 4: Run the tests and verify they pass**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/profiling/... -v`

Expected: PASS for all tests added so far.

- [ ] **Step 5: Commit**

```bash
git add pkg/profiling/profiling.go pkg/profiling/profiling_test.go
git commit --no-gpg-sign -m "feat(profiling): add writeSnapshotProfile for instant snapshots"
```

---

## Task 4: CPU + trace concurrent writer (`writeCPUAndTrace`)

CPU profile and runtime/trace both need a time window. Per the spec they run concurrently during the same window — the stdlib supports this. We isolate this into one helper that takes a `dir` and a `window time.Duration`.

**Files:**
- Modify: `pkg/profiling/profiling.go`
- Modify: `pkg/profiling/profiling_test.go`

- [ ] **Step 1: Add the failing test**

Append to `pkg/profiling/profiling_test.go`:

```go
func TestWriteCPUAndTrace_BothFilesNonEmpty(t *testing.T) {
	dir := t.TempDir()
	// 100ms is long enough to record at least one CPU sample at 100Hz
	// and a few trace events. Don't go shorter or the profiles can be
	// truly empty.
	if err := writeCPUAndTrace(dir, 100*time.Millisecond); err != nil {
		t.Fatalf("writeCPUAndTrace: %v", err)
	}
	for _, name := range []string{"cpu.prof", "trace.out"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("stat %s: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/profiling/... -v -run TestWriteCPUAndTrace`

Expected: build error mentioning `undefined: writeCPUAndTrace`.

- [ ] **Step 3: Implement `writeCPUAndTrace`**

Append to `pkg/profiling/profiling.go`:

```go
import (
	"runtime/trace" // add to existing import block
)

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
```

Final import block at top of `profiling.go`:

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"runtime/trace"
	"time"
)
```

- [ ] **Step 4: Run the tests and verify they pass**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/profiling/... -v`

Expected: PASS for all tests.

- [ ] **Step 5: Commit**

```bash
git add pkg/profiling/profiling.go pkg/profiling/profiling_test.go
git commit --no-gpg-sign -m "feat(profiling): add writeCPUAndTrace for windowed profile pair"
```

---

## Task 5: Mutex/block contention sampling enable/disable

Two tiny helpers that wrap `runtime.SetMutexProfileFraction` and `runtime.SetBlockProfileRate`. They exist as their own functions because the spec requires that contention sampling be enabled only during the capture window — never between sessions — and centralizing the on/off lets `captureAll` stay readable.

**Files:**
- Modify: `pkg/profiling/profiling.go`
- Modify: `pkg/profiling/profiling_test.go`

- [ ] **Step 1: Add the failing test**

Append to `pkg/profiling/profiling_test.go`:

```go
import (
	"runtime" // add to existing import block
)

func TestEnableDisableContentionProfiles_ResetsMutexFraction(t *testing.T) {
	// Snapshot the prior state so we leave the test process untouched.
	prior := runtime.SetMutexProfileFraction(-1)
	t.Cleanup(func() { runtime.SetMutexProfileFraction(prior) })

	enableContentionProfiles()
	if got := runtime.SetMutexProfileFraction(-1); got != 1 {
		t.Errorf("after enable: mutex fraction = %d; want 1", got)
	}

	disableContentionProfiles()
	if got := runtime.SetMutexProfileFraction(-1); got != 0 {
		t.Errorf("after disable: mutex fraction = %d; want 0", got)
	}
	// Block rate has no public getter; we trust the single line of
	// code that sets it. See spec §"Testing strategy" for rationale.
}
```

Final import block in `profiling_test.go`:

```go
import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/profiling/... -v -run TestEnableDisableContentionProfiles`

Expected: build error mentioning `undefined: enableContentionProfiles` and `undefined: disableContentionProfiles`.

- [ ] **Step 3: Implement the helpers**

Append to `pkg/profiling/profiling.go`:

```go
import (
	"runtime" // add to existing import block
)

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
```

Final import block in `profiling.go`:

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"time"
)
```

- [ ] **Step 4: Run the tests and verify they pass**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/profiling/... -v`

Expected: PASS for all tests.

- [ ] **Step 5: Commit**

```bash
git add pkg/profiling/profiling.go pkg/profiling/profiling_test.go
git commit --no-gpg-sign -m "feat(profiling): add contention-profile enable/disable helpers"
```

---

## Task 6: `captureAll` orchestrator with reentrancy guard and recover

This is the heart of the package — orchestrates all helpers, enforces single-capture-at-a-time via `atomic.Bool`, and catches any panic so a profiling crash never kills the game session.

**Files:**
- Modify: `pkg/profiling/profiling.go`
- Modify: `pkg/profiling/profiling_test.go`

- [ ] **Step 1: Add the failing tests**

Append to `pkg/profiling/profiling_test.go`:

```go
import (
	"sync" // add to existing import block
)

func TestCaptureAll_ProducesAllSixFiles(t *testing.T) {
	base := t.TempDir()
	captureAll(base, 100*time.Millisecond)

	// Find the one session directory under base.
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("read base: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 session dir, got %d", len(entries))
	}
	sessionPath := filepath.Join(base, entries[0].Name())

	want := []string{"cpu.prof", "heap.prof", "goroutine.prof", "mutex.prof", "block.prof", "trace.out"}
	for _, name := range want {
		info, err := os.Stat(filepath.Join(sessionPath, name))
		if err != nil {
			t.Errorf("stat %s: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestCaptureAll_ReentrancySecondCallSkips(t *testing.T) {
	base := t.TempDir()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); captureAll(base, 200*time.Millisecond) }()
	// Tiny delay so the first call wins the CAS reliably.
	time.Sleep(20 * time.Millisecond)
	go func() { defer wg.Done(); captureAll(base, 200*time.Millisecond) }()
	wg.Wait()

	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("read base: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected exactly 1 session dir after concurrent capture; got %d", len(entries))
	}
}

func TestCaptureAll_ResetsMutexFraction(t *testing.T) {
	// Confirm captureAll leaves the runtime in a clean state — mutex
	// fraction back at 0 — regardless of whether sampling was on
	// before the call.
	prior := runtime.SetMutexProfileFraction(-1)
	t.Cleanup(func() { runtime.SetMutexProfileFraction(prior) })
	runtime.SetMutexProfileFraction(0) // start clean

	captureAll(t.TempDir(), 50*time.Millisecond)

	if got := runtime.SetMutexProfileFraction(-1); got != 0 {
		t.Errorf("mutex fraction after captureAll = %d; want 0", got)
	}
}
```

Final import block in `profiling_test.go`:

```go
import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)
```

- [ ] **Step 2: Run the tests and verify they fail**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/profiling/... -v -run TestCaptureAll`

Expected: build error mentioning `undefined: captureAll`.

- [ ] **Step 3: Implement `captureAll`**

Append to `pkg/profiling/profiling.go`:

```go
import (
	"log"           // add to existing import block
	"sync/atomic"   // add to existing import block
)

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
```

Final import block in `profiling.go`:

```go
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
```

- [ ] **Step 4: Run the tests and verify they pass**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test ./pkg/profiling/... -v`

Expected: PASS for all tests. Note: the reentrancy test may take ~200ms because both `captureAll` calls hold for the full window before one CAS-fails-and-exits-fast. That is normal.

- [ ] **Step 5: Run race detector on the package**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test -race ./pkg/profiling/...`

Expected: PASS, no data race warnings.

- [ ] **Step 6: Commit**

```bash
git add pkg/profiling/profiling.go pkg/profiling/profiling_test.go
git commit --no-gpg-sign -m "feat(profiling): add captureAll orchestrator with reentrancy guard"
```

---

## Task 7: Public `Start()` with SIGUSR1 signal listener

The public entry point. Registers a signal handler, spawns the listener goroutine, logs the PID, returns.

**Files:**
- Modify: `pkg/profiling/profiling.go`

The signal handler itself is not unit-tested — sending a real signal to the test process is brittle and the four-line registration function provides minimal value beyond visual inspection. See spec §"Testing strategy" for rationale.

- [ ] **Step 1: Add the `Start` function**

Append to `pkg/profiling/profiling.go`:

```go
import (
	"os/signal" // add to existing import block
	"syscall"   // add to existing import block
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
```

Final import block at top of `profiling.go`:

```go
import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync/atomic"
	"syscall"
	"time"
)
```

- [ ] **Step 2: Run all package tests and verify they still pass**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test -race ./pkg/profiling/... -v`

Expected: PASS for all tests; no race warnings.

- [ ] **Step 3: Verify `go vet` is clean for the package**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go vet ./pkg/profiling/...`

Expected: no output (vet is silent on success).

- [ ] **Step 4: Commit**

```bash
git add pkg/profiling/profiling.go
git commit --no-gpg-sign -m "feat(profiling): add Start() public entry point + signal listener"
```

---

## Task 8: Wire `profiling.Start()` into `cmd/client/main.go`

A single import + a single call placed after argument parsing and before the existing three `wg.Go` launches.

**Files:**
- Modify: `cmd/client/main.go`

- [ ] **Step 1: Read the current state of `main.go` to confirm the insertion point**

Read `cmd/client/main.go`. The existing layout:

- Lines 1–15: package + imports
- Lines 17–54: `main()` opens, parses 4–5 positional args, validates each
- Line 56–57: TODO comment about WaitGroup
- Line 58: `var wg sync.WaitGroup`
- Lines 59–73: three `wg.Go(...)` launches (signlink, audio, client)
- Lines 74–80: `app.Main()` + `wg.Wait()`

Insertion point: between argument parsing and the `var wg sync.WaitGroup` line (currently line 58). After argument parsing means we won't run profiling if the user passes wrong args and we exit early — but before any real work begins.

- [ ] **Step 2: Add the import and the call**

Apply this change to `cmd/client/main.go`:

```go
import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"gioui.org/app"

	"goscape-client/pkg/jagex2/client"
	"goscape-client/pkg/jagex2/client/clientextras"
	"goscape-client/pkg/jagex2/sound/audio"
	"goscape-client/pkg/profiling"
	"goscape-client/pkg/sign/signlink"
)
```

(Insert `"goscape-client/pkg/profiling"` between the audio and signlink imports so they remain alphabetically sorted within the project-internal group.)

Then, immediately before `var wg sync.WaitGroup` (current line 58), insert:

```go
	// Register SIGUSR1 profile-capture handler. Non-blocking; returns
	// after signal listener goroutine is spawned. See
	// docs/superpowers/specs/2026-05-22-perf-profiling-design.md.
	profiling.Start()

```

- [ ] **Step 3: Build the whole project to confirm everything links**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go build ./...`

Expected: no output, no errors.

- [ ] **Step 4: Run `go vet` on the whole project**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go vet ./...`

Expected: no output (vet is silent on success). Pre-existing QF1003 diagnostic at `client.go:~1216` is intentionally left per CLAUDE.md conventions — if it surfaces, ignore.

- [ ] **Step 5: Run the full test suite under -race to confirm no regression**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test -race ./...`

Expected: PASS for all packages including `./pkg/profiling/...`.

- [ ] **Step 6: Commit**

```bash
git add cmd/client/main.go
git commit --no-gpg-sign -m "feat(profiling): wire profiling.Start() into main"
```

---

## Task 9: Document in README.md

Add a short "Performance profiling" section so a future user (or future-you) can discover the feature without reading the spec.

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Read the current state of `README.md` to identify where to insert**

Read `README.md`. Pick a sensible insertion point — typically after the "Build & Run" section or before "Translation notes," whichever exists. If you find no obvious section break, append the new section at the end of the file.

- [ ] **Step 2: Insert the profiling section**

Add this section at the insertion point chosen in Step 1:

````markdown
## Performance profiling

The client compiles in a SIGUSR1-triggered profile-capture mechanism. On
startup, it logs its PID:

```
profiling: signal listener ready, send SIGUSR1 to pid 12345
```

From another terminal, send SIGUSR1 to capture a full profile session:

```bash
kill -USR1 12345
```

After ~30 seconds the client writes a session directory under
`./profiles/<UTC-timestamp>/` (relative to the working directory at
process start) containing six files:

| File | Tool |
|---|---|
| `cpu.prof` | `go tool pprof cpu.prof` |
| `heap.prof` | `go tool pprof heap.prof` |
| `goroutine.prof` | `go tool pprof goroutine.prof` |
| `mutex.prof` | `go tool pprof mutex.prof` |
| `block.prof` | `go tool pprof block.prof` |
| `trace.out` | `go tool trace trace.out` |

Each session is ~100 MB on disk (mostly `trace.out`). Clean up
`./profiles/` periodically. A second SIGUSR1 received during an active
capture is logged and ignored, not queued.
````

- [ ] **Step 3: Verify the file renders cleanly**

Open `README.md` and scroll through to confirm the new section sits in a sensible place and the surrounding markdown is unbroken (no orphaned heading levels, no missing blank line between sections).

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit --no-gpg-sign -m "docs: document SIGUSR1 profile capture in README"
```

---

## Task 10: End-to-end verification

The plan is functionally complete. This task is the final verification per spec §"Verification plan."

**Files:** none modified — verification only.

- [ ] **Step 1: Final build + vet + race-tested test run**

Run all three sequentially:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go build ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go vet ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go go test -race ./...
```

Expected: all three clean. The `go test -race ./...` line is the gate — if anything fails here, fix it before declaring done.

- [ ] **Step 2: Hand off to the user for live verification**

The smoke test is hard-blocked in-sandbox per the `project_smoke_test_blocked` memory. The user runs:

1. `go run ./cmd/client 10 0 highmem members` and notes the PID from the `profiling: signal listener ready, send SIGUSR1 to pid <N>` log line.
2. From another terminal, `kill -USR1 <N>`.
3. Confirms `./profiles/<timestamp>/` appears with all six files after ~30s.
4. Spot-checks via `go tool pprof ./profiles/<ts>/cpu.prof` — should open the interactive pprof prompt cleanly, accept `top10`, and show the goscape-client packages in the result.
5. Sends a second `kill -USR1 <N>` during the 30s window of a fresh capture; confirms the client logs `profiling: capture already in flight, ignoring SIGUSR1` and that no second directory appears.

If any of these fail, return to the relevant task and iterate.

- [ ] **Step 3: Update `PORTING.md` if profiling is mentioned anywhere**

Run: `grep -in "profil" PORTING.md || echo "no mentions"`

If profiling appears in any TODO/planned section there, mark it complete. Otherwise no-op.

- [ ] **Step 4: Optional final commit if Step 3 produced a PORTING.md edit**

Only if Step 3 actually changed PORTING.md:

```bash
git add PORTING.md
git commit --no-gpg-sign -m "docs(porting): mark profiling infrastructure complete"
```

---

## Notes for the implementer

- **Run `use-modern-go` first.** The skill is project-mandated for any Go work. Modern Go syntax (Go 1.22+ `for range int`, Go 1.22+ loop-var-scoped, `slices` / `maps` packages, etc.) should be used where applicable — though this plan is intentionally written in a style that already conforms.
- **Don't stage `go.mod` / `go.sum`.** They are intentionally untracked in this repo. The profiling package uses only stdlib, so they should not change anyway — but if they do (e.g., because the build cache updates them), do not include them in commits.
- **The `-race` test run in Task 6 step 5 and Task 8 step 5 is non-negotiable.** Profiling code introduces new goroutines and an atomic — both are race-detector territory.
- **If a test in Task 6 is flaky** (the 100ms window can occasionally produce a zero-byte `cpu.prof` on a heavily loaded CI machine because no samples land), bump the window in `TestCaptureAll_ProducesAllSixFiles` to 250ms. Do not change the production `cpuWindow` constant.
- **Order matters in `captureAll`.** Specifically: `runtime.GC()` must come immediately before the heap snapshot, and the deferred `disableContentionProfiles()` must execute after the snapshot loop completes. The function body as written respects this; don't refactor it into "more idiomatic" cleanup-first order without thinking through the GC and contention-sampling-window semantics.

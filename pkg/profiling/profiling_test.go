package profiling

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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
	if early >= later {
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

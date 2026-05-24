package signlink

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestCacheLoadRace forces two goroutines to call CacheLoad("a") and
// CacheLoad("b") concurrently. The signlink polling protocol uses a
// package-global LoadReq/LoadBuf slot; without serialization the second
// writer to LoadReq wins and both readers receive the second file's
// bytes — the exact failure that made RunMidi receive `config` bytes
// when it asked for `scape_main.mid`.
func TestCacheLoadRace(t *testing.T) {
	dir := t.TempDir()
	// Files keyed by plain name: CacheLoad now keys by the name verbatim.
	fileA := []byte("AAAAAAAAAA")
	fileB := []byte("BBBBBBBBBB")
	if err := os.WriteFile(filepath.Join(dir, "a"), fileA, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b"), fileB, 0o644); err != nil {
		t.Fatal(err)
	}

	// Start a minimal polling loop that mirrors signlink.Run's Load branch.
	// We don't call StartPriv because that does a full filesystem search
	// and never returns. Drive only LoadReq processing here.
	var stop atomic.Bool
	pollerDone := make(chan struct{})
	go func() {
		defer close(pollerDone)
		for !stop.Load() {
			mu.Lock()
			req := LoadReq
			mu.Unlock()
			if req != "" {
				p := filepath.Join(dir, req)
				var buf []byte
				if _, err := os.Stat(p); err == nil {
					buf, _ = os.ReadFile(p)
				}
				mu.Lock()
				LoadBuf = buf
				LoadReq = ""
				cond.Broadcast()
				mu.Unlock()
			}
			time.Sleep(100 * time.Microsecond)
		}
	}()
	defer func() {
		stop.Store(true)
		<-pollerDone
	}()

	const iters = 500
	var wg sync.WaitGroup
	var mismatches int
	var mu sync.Mutex

	wg.Go(func() {
		for range iters {
			got := CacheLoad("a")
			if string(got) != string(fileA) {
				mu.Lock()
				mismatches++
				mu.Unlock()
			}
		}
	})
	wg.Go(func() {
		for range iters {
			got := CacheLoad("b")
			if string(got) != string(fileB) {
				mu.Lock()
				mismatches++
				mu.Unlock()
			}
		}
	})
	wg.Wait()

	if mismatches > 0 {
		t.Fatalf("CacheLoad returned the wrong file %d/%d times (race on LoadReq/LoadBuf)", mismatches, 2*iters)
	}
}

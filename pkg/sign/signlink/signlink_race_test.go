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
	// Files keyed by hash, since CacheLoad strconv-formats GetHash(name).
	fileA := []byte("AAAAAAAAAA")
	fileB := []byte("BBBBBBBBBB")
	hashA := GetHash("a")
	hashB := GetHash("b")
	if err := os.WriteFile(filepath.Join(dir, itoa(hashA)), fileA, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, itoa(hashB)), fileB, 0o644); err != nil {
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
			// Snapshot LoadReq into a local to avoid torn reads of the
			// 16-byte string under tight polling; production's 50ms sleep
			// makes this snapshotting incidental, but the test loops
			// fast.
			req := LoadReq
			if req != "" {
				p := filepath.Join(dir, req)
				if _, err := os.Stat(p); err == nil {
					LoadBuf, _ = os.ReadFile(p)
				} else {
					LoadBuf = nil
				}
				LoadReq = ""
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

// itoa is a tiny inlined int64-to-string to avoid pulling in strconv from
// a test that's already adjacent to the implementation.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

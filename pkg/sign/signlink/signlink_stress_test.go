package signlink

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

// TestSignlinkConcurrentStress exercises CacheLoad, CacheSave, and
// OpenURL from multiple goroutines simultaneously to validate the
// mu/cond/slotMu pattern under contention. Each operation has its own
// reference value; any cross-talk between callers (e.g. a CacheLoad
// returning another goroutine's bytes, an OpenURL returning a cached
// file's contents) shows up as a mismatch.
//
// The test drives a stripped-down version of Run's I/O against a temp
// dir and httptest server rather than calling StartPriv (which would
// scan the host filesystem for cache directories and never return).
// `go test -race` must be clean: any unsynchronized access to the
// protocol fields will be flagged.
func TestSignlinkConcurrentStress(t *testing.T) {
	dir := t.TempDir()

	// Pre-populate cache files that CacheLoad will read. CacheLoad keys by the
	// plain name, so the file names are the keys verbatim.
	cachedKeys := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	cachedBodies := map[string][]byte{}
	for _, k := range cachedKeys {
		body := []byte(strings.Repeat(k+":", 32))
		cachedBodies[k] = body
		if err := os.WriteFile(filepath.Join(dir, k), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// HTTP server that echoes the request path. OpenURL builds a URL
	// from clientextras.PortOffset, so we override PortOffset to point
	// at our test server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip leading slash. The path is the URL fragment OpenURL
		// passed in, so the echoed body identifies the caller's request.
		fmt.Fprintf(w, "URL:%s", strings.TrimPrefix(r.URL.Path, "/"))
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	prev := clientextras.PortOffset
	clientextras.PortOffset = port - 8888
	t.Cleanup(func() { clientextras.PortOffset = prev })

	// Polling goroutine. Mirrors Run's logic but bounded by stop and
	// rooted at our temp dir.
	var stop atomic.Bool
	pollerDone := make(chan struct{})
	go func() {
		defer close(pollerDone)
		for !stop.Load() {
			mu.Lock()
			loadReq := LoadReq
			saveReq := SaveReq
			saveBuf := SaveBuf
			saveLen := SaveLen
			urlReq := URLReq
			mu.Unlock()

			switch {
			case loadReq != "":
				var buf []byte
				p := filepath.Join(dir, loadReq)
				if _, err := os.Stat(p); err == nil {
					buf, _ = os.ReadFile(p)
				}
				mu.Lock()
				LoadBuf = buf
				LoadReq = ""
				cond.Broadcast()
				mu.Unlock()
			case saveReq != "":
				if saveBuf != nil {
					_ = os.WriteFile(filepath.Join(dir, saveReq), saveBuf[:saveLen], 0o644)
				}
				mu.Lock()
				SaveReq = ""
				cond.Broadcast()
				mu.Unlock()
			case urlReq != "":
				resp, err := http.Get(srv.URL + "/" + urlReq)
				var body []byte
				if err == nil {
					body, _ = io.ReadAll(resp.Body)
					resp.Body.Close()
				}
				mu.Lock()
				URLStream = body
				URLReq = ""
				cond.Broadcast()
				mu.Unlock()
			}

			time.Sleep(50 * time.Microsecond)
		}
	}()
	t.Cleanup(func() {
		stop.Store(true)
		<-pollerDone
	})

	const itersPerWorker = 100
	var (
		wg         sync.WaitGroup
		mismatchMu sync.Mutex
		mismatches []string
	)

	recordMismatch := func(format string, args ...any) {
		mismatchMu.Lock()
		mismatches = append(mismatches, fmt.Sprintf(format, args...))
		mismatchMu.Unlock()
	}

	// CacheLoad workers — three goroutines, each cycling through the
	// known keys.
	for w := range 3 {
		wg.Go(func() {
			for i := range itersPerWorker {
				k := cachedKeys[(w+i)%len(cachedKeys)]
				got := CacheLoad(k)
				want := cachedBodies[k]
				if string(got) != string(want) {
					recordMismatch("CacheLoad(%q): got %q want %q", k, got, want)
				}
			}
		})
	}

	// CacheSave workers — two goroutines, writing distinct keys.
	for w := range 2 {
		wg.Go(func() {
			for i := range itersPerWorker {
				k := fmt.Sprintf("save_%d_%d", w, i)
				body := []byte(fmt.Sprintf("body_%d_%d", w, i))
				CacheSave(k, body)
				// Read it back through CacheLoad to verify it round-tripped.
				got := CacheLoad(k)
				if string(got) != string(body) {
					recordMismatch("CacheSave/CacheLoad(%q): got %q want %q", k, got, body)
				}
			}
		})
	}

	// OpenURL workers — two goroutines, each requesting unique paths.
	for w := range 2 {
		wg.Go(func() {
			for i := range itersPerWorker {
				p := fmt.Sprintf("path_%d_%d", w, i)
				body, err := OpenURL(p)
				if err != nil {
					recordMismatch("OpenURL(%q) error: %v", p, err)
					continue
				}
				want := "URL:" + p
				if string(body) != want {
					recordMismatch("OpenURL(%q): got %q want %q", p, body, want)
				}
			}
		})
	}

	wg.Wait()

	if len(mismatches) > 0 {
		// Cap output to keep test logs readable.
		limit := 10
		if len(mismatches) < limit {
			limit = len(mismatches)
		}
		t.Fatalf("%d mismatches across concurrent signlink calls; first %d:\n%s",
			len(mismatches), limit, strings.Join(mismatches[:limit], "\n"))
	}
}

// Command wasmserve serves the gogio js/wasm build for local browser testing
// and reverse-proxies every other request — the game's cache-data fetches
// (crc, jag archives, MIDI) — to a backend data server. This presents the
// browser with a SINGLE origin shared by the wasm page and the cache data, so
// the client's same-origin fetches succeed without CORS. It mirrors the
// one-origin model the browser build assumes: signlink.ConfigureTransport
// derives the WebSocket target from window.location, and Client.GetCodeBase
// returns the page origin (codebase_js.go).
//
// Go's net/http maps .wasm to application/wasm, which
// WebAssembly.instantiateStreaming requires (a server returning
// application/octet-stream fails to stream-instantiate).
package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
)

// servesFromBundle reports whether reqPath should be served from the local
// gogio bundle dir (index.html, main.wasm, wasm.js) rather than proxied to the
// data backend. "/" maps to index.html; any other path is served locally only
// when it names a real file in dir — otherwise it is a cache-data request to
// proxy. filepath.Clean collapses any traversal under the bundle dir, so a
// "/../x" request can only ever resolve to a (non-existent) file inside dir.
func servesFromBundle(dir, reqPath string) bool {
	if reqPath == "/" {
		return true
	}
	p := filepath.Join(dir, filepath.Clean("/"+reqPath))
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func main() {
	dir := flag.String("dir", "gio/client", "directory with the gogio js output")
	addr := flag.String("addr", ":8080", "listen address")
	backend := flag.String("backend", "http://localhost:8888", "cache-data backend to proxy non-bundle requests to")
	flag.Parse()

	bu, err := url.Parse(*backend)
	if err != nil {
		log.Fatalf("wasmserve: invalid -backend %q: %v", *backend, err)
	}
	files := http.FileServer(http.Dir(*dir))
	proxy := httputil.NewSingleHostReverseProxy(bu)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if servesFromBundle(*dir, r.URL.Path) {
			files.ServeHTTP(w, r)
			return
		}
		proxy.ServeHTTP(w, r)
	})

	log.Printf("wasmserve: serving %s on http://localhost%s (cache data proxied to %s)", *dir, *addr, *backend)
	log.Printf("wasmserve: launch with e.g. http://localhost%s/?argv=10 0 highmem members", *addr)
	log.Fatal(http.ListenAndServe(*addr, handler))
}

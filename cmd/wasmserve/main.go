// Command wasmserve serves the gogio js/wasm build for local browser testing.
// Go's net/http maps .wasm to application/wasm, which
// WebAssembly.instantiateStreaming requires (a plain file:// open or a server
// that returns application/octet-stream fails to stream-instantiate).
package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	dir := flag.String("dir", "gio/client", "directory to serve (gogio js output)")
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	log.Printf("wasmserve: serving %s on http://localhost%s", *dir, *addr)
	log.Printf("wasmserve: launch with e.g. http://localhost%s/?argv=10 0 highmem members", *addr)
	log.Fatal(http.ListenAndServe(*addr, http.FileServer(http.Dir(*dir))))
}

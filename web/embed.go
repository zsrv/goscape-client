// Package web exposes the browser (js/wasm) host assets for the goscape client
// as embedded files, so that other Go modules can build the client to
// WebAssembly and obtain the reference host page programmatically — without
// copying files out of this repository.
//
// A consuming module (for example, a server that serves its own client build)
// produces the three browser artifacts like so:
//
//	# 1. Compile the client to wasm (no cgo/C toolchain required — the native
//	#    GLFW/OpenGL backend is excluded on GOOS=js).
//	GOOS=js GOARCH=wasm go build -o main.wasm github.com/zsrv/goscape-client/cmd/client
//
//	# 2. Copy the JS runtime glue from the SAME toolchain that built main.wasm.
//	cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" .
//
//	# 3. Serve web.IndexHTML (below) alongside main.wasm and wasm_exec.js.
//
// The host page uses relative asset paths and derives the WebSocket/cache
// origin from window.location, so it is same-origin by construction and can be
// served unmodified from any route.
package web

import "embed"

// Assets holds the embedded browser host assets (currently just index.html).
//
// wasm_exec.js is intentionally NOT embedded: it is specific to the Go toolchain
// that compiles main.wasm and must be copied from
// "$(go env GOROOT)/lib/wasm/wasm_exec.js" at build time so the two stay in
// sync. Embedding a copy here would risk a version mismatch with the consumer's
// compiler.
//
//go:embed index.html
var Assets embed.FS

// IndexHTML is the reference host page. It loads wasm_exec.js and instantiates
// main.wasm using relative paths. Consumers may serve it verbatim or use it as
// a template for their own page.
//
//go:embed index.html
var IndexHTML []byte

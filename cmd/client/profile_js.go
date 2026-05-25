//go:build js && goscapedebug

package main

import (
	"bytes"
	"fmt"
	"runtime"
	"runtime/pprof"
	"syscall/js"
)

// init installs browser-console debug hooks for the wasm build. Gated behind the
// `goscapedebug` build tag so they are absent from the normal/release wasm
// artifact; build with `-tags goscapedebug` to include them. They exist only in
// the js build (Go's pprof cannot write a file from wasm, and a browser heap
// snapshot sees the module as one opaque ArrayBuffer, so neither attributes
// allocations to Go call sites). Call from DevTools:
//
//	goscapeMemStats()    // one-line heap summary to the console
//	goscapeDumpAllocs()  // downloads allocs.pb.gz; analyze with
//	                     //   go tool pprof -alloc_space allocs.pb.gz
//
// Harmless if never called.
func init() {
	js.Global().Set("goscapeDumpAllocs", js.FuncOf(func(js.Value, []js.Value) any {
		runtime.GC() // settle the heap so the profile reflects live allocations
		var buf bytes.Buffer
		if err := pprof.Lookup("allocs").WriteTo(&buf, 0); err != nil {
			js.Global().Get("console").Call("error", "goscapeDumpAllocs: "+err.Error())
			return nil
		}
		downloadBytes("allocs.pb.gz", buf.Bytes())
		return nil
	}))

	js.Global().Set("goscapeMemStats", js.FuncOf(func(js.Value, []js.Value) any {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		const mib = 1024 * 1024
		js.Global().Get("console").Call("log", fmt.Sprintf(
			"mem: HeapAlloc=%dMiB HeapSys=%dMiB HeapInuse=%dMiB TotalAlloc=%dMiB Mallocs=%d NumGC=%d",
			m.HeapAlloc/mib, m.HeapSys/mib, m.HeapInuse/mib, m.TotalAlloc/mib, m.Mallocs, m.NumGC))
		return nil
	}))

	js.Global().Get("console").Call("log",
		"goscape debug hooks installed: goscapeMemStats(), goscapeDumpAllocs()")
}

// downloadBytes triggers a browser download of b under the given filename via a
// Blob object URL. The URL is intentionally not revoked: this is a rarely-used
// debug path, and revoking synchronously can cancel the download in some
// browsers. One leaked object URL per dump clears on page reload.
func downloadBytes(name string, b []byte) {
	u8 := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(u8, b)
	blob := js.Global().Get("Blob").New([]any{u8}, map[string]any{"type": "application/octet-stream"})
	url := js.Global().Get("URL").Call("createObjectURL", blob)
	a := js.Global().Get("document").Call("createElement", "a")
	a.Set("href", url)
	a.Set("download", name)
	a.Call("click")
}

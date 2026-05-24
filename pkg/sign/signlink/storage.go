package signlink

// cacheStore is signlink's persistence backend. The disk implementation
// (storage_disk.go, //go:build !js) preserves the Java file-store behavior;
// the in-memory implementation (storage_mem.go) backs the browser build.
//
// Methods are synchronous and are intended to be called only from
// signlink.Run()'s goroutine (wired in a later change), so a future IndexedDB
// implementation (sub-project 2) may block awaiting a JS promise — safe
// because a blocked goroutine yields to the browser event loop under js/wasm.
type cacheStore interface {
	// load returns the bytes stored under name, or nil on a miss (mirrors the
	// current os.Stat-then-ReadFile behavior, which returns nil for an absent
	// file rather than an error).
	load(name string) []byte
	// save stores data under name. Best-effort: failures are logged, never
	// returned, matching the current os.WriteFile error handling.
	save(name string, data []byte)
	// uid returns the persistent client id (Java: GetUID). The browser
	// implementation returns a session-stable constant.
	uid() int
	// cacheDir returns the on-disk base used to build wave/MIDI scratch paths
	// in Run(). "" in the browser (no filesystem).
	cacheDir() string
}

// store is the active backend, selected at build time by newCacheStore
// (storage_disk.go / storage_js.go), mirroring the profiling Start() split.
var store cacheStore = newCacheStore()

var _ cacheStore = (*memStore)(nil)

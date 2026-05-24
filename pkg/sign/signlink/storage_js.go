//go:build js

package signlink

// newCacheStore returns the IndexedDB-backed store for the browser build, so
// cache survives reloads. It degrades to an in-memory store when IndexedDB is
// unavailable (see idbStore).
func newCacheStore() cacheStore { return newIDBStore() }

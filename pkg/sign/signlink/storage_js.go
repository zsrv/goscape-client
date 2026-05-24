//go:build js

package signlink

// newCacheStore returns the volatile in-memory store for the browser build.
// Durable IndexedDB-backed storage is sub-project 2, behind the same interface.
func newCacheStore() cacheStore { return newMemStore() }

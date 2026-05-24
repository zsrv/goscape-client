package signlink

import "sync"

// browserUID is the session client id reported by the in-memory store. The
// browser has no persistent uid.dat; the TypeScript reference client
// (Client-TS/src/client/Client.ts:1729) likewise sends a fixed value and
// persists nothing, so a constant is parity, not a shortcut.
const browserUID = 1337

// memStore is a volatile, in-RAM cacheStore: a string-keyed blob map under a
// mutex. It has no syscall/js dependency, so it compiles and is unit-tested on
// native. Used by the browser build (storage_js.go); replaced by an
// IndexedDB-backed store in sub-project 2 behind this same interface.
type memStore struct {
	mu sync.Mutex
	m  map[string][]byte
}

func newMemStore() *memStore {
	return &memStore{m: make(map[string][]byte)}
}

func (s *memStore) load(name string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.m[name]
	if !ok {
		return nil
	}
	// Copy out so callers can't mutate stored bytes (the disk path decouples
	// by serializing to disk; this preserves that contract).
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp
}

func (s *memStore) save(name string, data []byte) {
	cp := make([]byte, len(data))
	copy(cp, data)
	s.mu.Lock()
	s.m[name] = cp
	s.mu.Unlock()
}

func (s *memStore) uid() int { return browserUID }

func (s *memStore) cacheDir() string { return "" }

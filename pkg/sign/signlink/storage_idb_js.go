//go:build js

package signlink

import (
	"errors"
	"log"
	"sync"
	"syscall/js"
)

var _ cacheStore = (*idbStore)(nil)

const (
	idbName      = "goscape"
	idbStoreName = "cache"
)

// idbStore is the browser cacheStore backed by IndexedDB, so cached game data
// survives page reloads. IndexedDB is asynchronous; each operation blocks the
// calling goroutine (signlink.Run's) on a channel until the JS callback fires —
// safe because Run is not the goroutine pumping the JS event loop, so the
// scheduler yields to the loop while it waits. The database is opened lazily on
// first use (NOT at package init, which runs before the event loop and would
// deadlock). When IndexedDB is unavailable (e.g. private browsing), idbStore
// delegates to an in-memory memStore so the session still works, just without
// cross-reload persistence — matching the Client-TS reference.
type idbStore struct {
	once      sync.Once
	db        js.Value
	available bool
	fallback  *memStore
}

func newIDBStore() *idbStore { return &idbStore{fallback: newMemStore()} }

// await attaches success/error handlers to an IDBRequest and blocks until one
// fires, returning req.result on success. Both js.Funcs are released before
// returning. Must be called only from a goroutine that does not pump the JS
// event loop (signlink.Run's goroutine qualifies).
func await(req js.Value) (js.Value, error) {
	type result struct {
		val js.Value
		err error
	}
	ch := make(chan result, 1)
	var onOK, onErr js.Func
	onOK = js.FuncOf(func(this js.Value, args []js.Value) any {
		ch <- result{val: req.Get("result")}
		return nil
	})
	onErr = js.FuncOf(func(this js.Value, args []js.Value) any {
		ch <- result{err: errors.New("indexeddb request failed")}
		return nil
	})
	req.Set("onsuccess", onOK)
	req.Set("onerror", onErr)
	r := <-ch
	onOK.Release()
	onErr.Release()
	return r.val, r.err
}

// ensure opens the IndexedDB database exactly once, creating the object store on
// first run. Any failure (no indexedDB, a thrown SecurityError in private
// browsing, or an open error) leaves available=false so every op uses the
// in-memory fallback. The recover guards against synchronous JS exceptions,
// which syscall/js surfaces as panics.
func (s *idbStore) ensure() {
	s.once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("signlink: indexedDB unavailable (%v); cache will not persist across reloads", r)
				s.available = false
			}
		}()

		idb := js.Global().Get("indexedDB")
		if !idb.Truthy() {
			log.Printf("signlink: indexedDB unavailable; cache will not persist across reloads")
			return
		}

		req := idb.Call("open", idbName, 1)
		// onupgradeneeded fires (before onsuccess) when the DB is created or
		// version-bumped; create the object store there.
		onUpgrade := js.FuncOf(func(this js.Value, args []js.Value) any {
			db := req.Get("result")
			if !db.Get("objectStoreNames").Call("contains", idbStoreName).Bool() {
				db.Call("createObjectStore", idbStoreName)
			}
			return nil
		})
		req.Set("onupgradeneeded", onUpgrade)
		db, err := await(req)
		onUpgrade.Release()
		if err != nil || !db.Truthy() {
			log.Printf("signlink: indexedDB open failed; cache will not persist: %v", err)
			return
		}
		s.db = db
		s.available = true
	})
}

// load returns the bytes stored under name, or nil on a miss — mirroring the
// disk store's os.Stat-then-ReadFile (nil for absent). A get that resolves to
// undefined is a miss; otherwise the stored Uint8Array is copied to a []byte.
func (s *idbStore) load(name string) (out []byte) {
	s.ensure()
	if !s.available {
		return s.fallback.load(name)
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("signlink: indexedDB load %q panicked: %v", name, r)
			out = nil
		}
	}()
	tx := s.db.Call("transaction", idbStoreName, "readonly")
	req := tx.Call("objectStore", idbStoreName).Call("get", name)
	res, err := await(req)
	if err != nil {
		log.Printf("signlink: indexedDB get %q: %v", name, err)
		return nil
	}
	if !res.Truthy() {
		return nil // miss: get resolved to undefined
	}
	n := res.Get("length").Int()
	buf := make([]byte, n)
	js.CopyBytesToGo(buf, res)
	return buf
}

// save stores data under name as a Uint8Array. Best-effort: errors are logged,
// never returned (matching the cacheStore contract). It blocks until the
// readwrite request resolves, mirroring os.WriteFile's synchronous semantics so
// a subsequent load observes the write.
func (s *idbStore) save(name string, data []byte) {
	s.ensure()
	if !s.available {
		s.fallback.save(name, data)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("signlink: indexedDB save %q panicked: %v", name, r)
		}
	}()
	arr := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(arr, data)
	tx := s.db.Call("transaction", idbStoreName, "readwrite")
	req := tx.Call("objectStore", idbStoreName).Call("put", arr, name)
	if _, err := await(req); err != nil {
		log.Printf("signlink: indexedDB put %q: %v", name, err)
	}
}

// uid returns the constant browser client id (browserUID). The browser has no
// persistent uid.dat and Client-TS sends a fixed value, so a constant is parity.
func (s *idbStore) uid() int { return browserUID }

// cacheDir returns "" — there is no on-disk scratch directory in the browser.
func (s *idbStore) cacheDir() string { return "" }

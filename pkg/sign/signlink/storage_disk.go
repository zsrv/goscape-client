//go:build !js

package signlink

import (
	"encoding/binary"
	"errors"
	"log"
	"math/rand"
	"os"
	"path"
	"sync"
)

func newCacheStore() cacheStore { return &diskStore{} }

// diskStore is the native cacheStore: the original Java-parity file store under
// FindCacheDir()/.file_store_32. dir and id are resolved once, lazily, on first
// use — matching the historical timing where Run() called FindCacheDir/GetUID
// at startup (not at package init), so importing signlink without running it
// has no filesystem side effects.
type diskStore struct {
	once sync.Once
	dir  string
	id   int
}

func (d *diskStore) ensure() {
	d.once.Do(func() {
		d.dir = FindCacheDir()
		d.id = GetUID(d.dir)
	})
}

func (d *diskStore) cacheDir() string { d.ensure(); return d.dir }

func (d *diskStore) uid() int { d.ensure(); return d.id }

func (d *diskStore) load(name string) []byte {
	d.ensure()
	p := path.Join(d.dir, name)
	if _, err := os.Stat(p); err != nil {
		return nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		log.Printf("signlink: failed to read file %s: %v", p, err)
		return nil
	}
	return b
}

func (d *diskStore) save(name string, data []byte) {
	d.ensure()
	p := path.Join(d.dir, name)
	if err := os.WriteFile(p, data, 0644); err != nil {
		log.Printf("signlink: failed to write file %s: %v", p, err)
	}
}

func FindCacheDir() string {
	var0 := []string{"c:/windows/", "c:/winnt/", "d:/windows/", "d:/winnt/", "e:/windows/", "e:/winnt/", "f:/windows/", "f:/winnt/", "c:/", "~/", "/tmp/", ""}
	var1 := ".file_store_32"
	for i := range len(var0) {
		var3 := var0[i]
		if len(var3) > 0 {
			if _, err := os.Stat(var3); err != nil {
				log.Printf("signlink: couldn't find cache at %s: %v", var3, err)
				continue
			}
		}
		var4 := path.Join(var3, var1)
		_, err := os.Stat(var4)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				// Java: File.exists() swallows permission errors and the
				// outer try/catch continues. Mirror that: any non-NotExist
				// stat error skips this candidate rather than returning a
				// path we can't access.
				log.Printf("signlink: couldn't stat cache at %s: %v", var4, err)
				continue
			}
			err2 := os.Mkdir(var4, 0755)
			if err2 != nil {
				log.Printf("signlink: couldn't create cache at %s: %v", var4, err2)
				continue
			}
		}
		return path.Join(var3, var1, "/")
	}
	return ""
}

func GetUID(arg0 string) int {
	var1 := path.Join(arg0, "uid.dat")
	stat, err := os.Stat(var1)
	if err != nil || stat.Size() < 4 {
		bs := make([]byte, 4)
		// Java: DataOutputStream.writeInt — big-endian. Stay byte-compatible
		// with the Java client's uid.dat format so shared caches work.
		binary.BigEndian.PutUint32(bs, uint32(rand.Float64()*9.9999999e7))
		os.WriteFile(var1, bs, 0644)
	}

	var5, err := os.ReadFile(var1)
	if err != nil {
		log.Println("signlink: couldn't read uid.dat")
		return 0
	}
	var6 := binary.BigEndian.Uint32(var5)
	return int(var6 + 1)
}

//go:build !js

package signlink

import (
	"encoding/binary"
	"errors"
	"log"
	"math/rand"
	"os"
	"path"
	"strconv"
	"sync"
)

func newCacheStore() cacheStore { return &diskStore{} }

var _ cacheStore = (*diskStore)(nil)

// diskStore is the native cacheStore: the original Java-parity file store at the
// .file_store_32/ directory returned by FindCacheDir(). dir and id are resolved
// once, lazily, on first use — matching the historical timing where Run()
// called FindCacheDir/GetUID at startup (not at package init), so importing
// signlink without running it has no filesystem side effects.
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

// storeDirName clamps StoreID to the valid window and returns the cache
// directory name. Java: findcachedir's clamp writes back to storeid before
// building the target (SignLink.java:206-210).
func storeDirName() string {
	if StoreID < 32 || StoreID > 34 {
		StoreID = 32
	}
	return ".file_store_" + strconv.Itoa(StoreID)
}

// Java: findcachedir (signlink.java:172-195 @32f3062) — 274 appends
// "c:/rscache" and "/rscache" (12→14 candidates). Those two lack the
// trailing slash the rest carry, and Java builds the target by raw string
// concat (var3 + var1), so they resolve to SIBLING directories
// ("c:/rscache.file_store_32", "/rscache.file_store_32"), not
// subdirectories — Go concatenates the same way to stay faithful.
func FindCacheDir() string {
	var0 := []string{"c:/windows/", "c:/winnt/", "d:/windows/", "d:/winnt/", "e:/windows/", "e:/winnt/", "f:/windows/", "f:/winnt/", "c:/", "~/", "/tmp/", "", "c:/rscache", "/rscache"}
	var1 := storeDirName()
	for i := range len(var0) {
		var3 := var0[i]
		if len(var3) > 0 {
			if _, err := os.Stat(var3); err != nil {
				log.Printf("signlink: couldn't find cache at %s: %v", var3, err)
				continue
			}
		}
		var4 := var3 + var1 // Java: new File(var3 + var1) — see doc comment
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
		// Java returns var3 + var1 + "/"; Go callers path.Join, so the
		// trailing slash is omitted here.
		return var4
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
		if err := os.WriteFile(var1, bs, 0644); err != nil {
			log.Println("signlink: couldn't write uid.dat")
		}
	}

	var5, err := os.ReadFile(var1)
	// Java: getuid reads via DataInputStream.readInt() inside try/catch; a short or
	// corrupt uid.dat throws EOFException, which is caught and returns 0 (a benign
	// fresh uid) — sign/signlink.java:213-220. Without the len guard,
	// binary.BigEndian.Uint32 panics on fewer than 4 bytes (e.g. when the rewrite
	// above failed on a read-only/full disk and was only logged).
	if err != nil || len(var5) < 4 {
		log.Println("signlink: couldn't read uid.dat")
		return 0
	}
	var6 := binary.BigEndian.Uint32(var5)
	// Java: getuid does signed int32 +1 (sign/signlink.java:209-211); int32()
	// reinterprets the wrapped uint32 so the sign matches (audit signlink-02)
	return int(int32(var6 + 1))
}

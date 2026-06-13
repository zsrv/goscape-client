package client

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/zsrv/goscape-client/pkg/jagex2/io"
	"github.com/zsrv/goscape-client/pkg/jagex2/io/ondemand"
)

// fileStreamCacheMaxBytes is the dat-file size past which the cache is dropped
// and rebuilt. Java: signlink.run deletes main_file_cache.dat when it exceeds
// 50 MB (signlink.java:106-109 @32f3062).
const fileStreamCacheMaxBytes = 52428800

// openFileStreamCache opens (creating if absent) the five-archive
// main_file_cache.dat / .idx0-4 random-access store under dir and returns it as
// an ondemand.Cache. It returns a true-nil interface — never a typed nil — when
// the files cannot be opened, so OnDemand's `cache == nil` gate degrades exactly
// like Java's `signlink.cache_dat == null` (no prefetch, no disk persistence).
//
// Java: maininit's `if (signlink.cache_dat != null) { for i in 0..5 fileStreams[i]
// = new FileStream(...) }` (Client.java:5131-5135), with the file handles opened
// by signlink.run (signlink.java:104-115).
func openFileStreamCache(dir string) ondemand.Cache {
	if dir == "" {
		return nil
	}

	datPath := filepath.Join(dir, "main_file_cache.dat")
	if fi, err := os.Stat(datPath); err == nil && fi.Size() > fileStreamCacheMaxBytes {
		_ = os.Remove(datPath)
	}

	dat, err := os.OpenFile(datPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		log.Printf("ondemand cache: cannot open %s: %v", datPath, err)
		return nil
	}

	var idx [5]*os.File
	for i := range 5 {
		f, err := os.OpenFile(filepath.Join(dir, "main_file_cache.idx"+strconv.Itoa(i)), os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			log.Printf("ondemand cache: cannot open main_file_cache.idx%d: %v", i, err)
			dat.Close()
			for j := range i {
				idx[j].Close()
			}
			return nil
		}
		idx[i] = f
	}

	return io.NewFileStreamCache(dat, idx)
}

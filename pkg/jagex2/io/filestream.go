package io

import (
	"fmt"
	"os"
	"sync"
)

// FileStream is a faithful port of Java jagex2.io.FileStream (vb): the
// random-access "main_file_cache.dat / main_file_cache.idxN" block store that
// backs the OnDemand cache. Each FileStream is one archive view over a shared
// dat file plus its own idx file. The on-disk layout matches the original
// client byte-for-byte:
//
//   - idx: 6 bytes per file — size (3 bytes, big-endian) + first data-block
//     number (3 bytes).
//   - dat: 520-byte blocks — an 8-byte header (file:2, part:2, nextBlock:3,
//     archive:1) followed by up to 512 payload bytes, chained via nextBlock.
//
// Java drives all access from a single loader thread under a static scratch
// buffer + synchronized methods; this port keeps a per-stream scratch buffer
// and leaves cross-stream serialization (the shared dat) to FileStreamCache.
type FileStream struct {
	dat         *os.File
	idx         *os.File
	archive     int
	maxFileSize int
	temp        [fileStreamBlockSize]byte // Java's static vb.temp (per-stream here)
}

const (
	// fileStreamBlockSize is the dat block stride: 8-byte header + 512 payload.
	fileStreamBlockSize = 520
	// fileStreamPayload is the max payload bytes carried by one block.
	fileStreamPayload = 512
	// fileStreamMaxSeek is Java's seek() clamp ceiling (vb.a(IZLjava/.../RandomAccessFile;)V).
	fileStreamMaxSeek = 62914560
)

// NewFileStream wraps an already-open dat/idx pair. Java's constructor also
// takes a 29615 sentinel (ignored) and the maxFileSize (500000 in the client);
// only the archive tag and maxFileSize affect behavior.
// Java: FileStream(RandomAccessFile,int,RandomAccessFile,int,int).
func NewFileStream(dat, idx *os.File, archive, maxFileSize int) *FileStream {
	return &FileStream{dat: dat, idx: idx, archive: archive, maxFileSize: maxFileSize}
}

// datLen returns the current length of the dat file. Java calls dat.length()
// repeatedly; the value grows as blocks are appended via WriteAt.
func (fs *FileStream) datLen() int64 {
	fi, err := fs.dat.Stat()
	if err != nil {
		return 0
	}
	return fi.Size()
}

// clampSeek mirrors Java's seek() guard: an out-of-range offset (a sign of a
// corrupt cache) is clamped to the ceiling. Java additionally sleeps 1s here;
// that is omitted because it would stall the game-loop goroutine and the branch
// is unreachable for the bounded offsets used here (file*6 and block*520 under a
// 50 MB dat cap).
func (fs *FileStream) clampSeek(pos int64) int64 {
	if pos < 0 || pos > fileStreamMaxSeek {
		fmt.Printf("Badseek - pos:%d len:%d\n", pos, fs.datLen())
		return fileStreamMaxSeek
	}
	return pos
}

// readAt reads exactly len(p) bytes from f at pos, returning false if fewer are
// available — matching Java's "dat.read() == -1 → bail" reads.
func (fs *FileStream) readAt(f *os.File, pos int64, p []byte) bool {
	n, _ := f.ReadAt(p, fs.clampSeek(pos))
	return n == len(p)
}

// writeAt writes all of p to f at pos, returning false on a short/failed write.
func (fs *FileStream) writeAt(f *os.File, pos int64, p []byte) bool {
	n, err := f.WriteAt(p, fs.clampSeek(pos))
	return err == nil && n == len(p)
}

// ReadFromFile returns the bytes stored for file, or nil if the entry is
// absent, truncated, or fails the per-block (file, part, archive) header check.
// Java: FileStream.readFromFile (vb.a(BI)[B).
func (fs *FileStream) ReadFromFile(file int) []byte {
	if !fs.readAt(fs.idx, int64(file)*6, fs.temp[:6]) {
		return nil
	}
	size := int(fs.temp[0])<<16 + int(fs.temp[1])<<8 + int(fs.temp[2])
	block := int(fs.temp[3])<<16 + int(fs.temp[4])<<8 + int(fs.temp[5])
	if size < 0 || size > fs.maxFileSize {
		return nil
	}
	datLen := fs.datLen()
	if block <= 0 || int64(block) > datLen/fileStreamBlockSize {
		return nil
	}

	out := make([]byte, size)
	read := 0
	part := 0
	for read < size {
		if block == 0 {
			return nil
		}
		chunk := min(size-read, fileStreamPayload)
		// One block = 8-byte header + chunk payload bytes.
		if !fs.readAt(fs.dat, int64(block)*fileStreamBlockSize, fs.temp[:chunk+8]) {
			return nil
		}
		hdrFile := int(fs.temp[0])<<8 + int(fs.temp[1])
		hdrPart := int(fs.temp[2])<<8 + int(fs.temp[3])
		nextBlock := int(fs.temp[4])<<16 + int(fs.temp[5])<<8 + int(fs.temp[6])
		hdrArchive := int(fs.temp[7])
		if hdrFile != file || hdrPart != part || hdrArchive != fs.archive {
			return nil
		}
		if nextBlock < 0 || int64(nextBlock) > datLen/fileStreamBlockSize {
			return nil
		}
		copy(out[read:read+chunk], fs.temp[8:8+chunk])
		read += chunk
		block = nextBlock
		part++
	}
	return out
}

// WriteToFile stores data (length bytes) under file. It first attempts an
// in-place overwrite of the existing block chain; if that chain is absent or
// inconsistent it falls back to a fresh append. Java: writeToFile(II[BB)Z.
func (fs *FileStream) WriteToFile(length, file int, data []byte) bool {
	if fs.write(true, data, file, length) {
		return true
	}
	return fs.write(false, data, file, length)
}

// write is the inner overwrite/append worker. Java: writeToFile(Z[BIIB)Z, where
// overwrite=arg0, data=arg1, file=arg2, length=arg3.
func (fs *FileStream) write(overwrite bool, data []byte, file, length int) bool {
	var block int
	if overwrite {
		// Reuse the first block recorded in the existing idx entry.
		if !fs.readAt(fs.idx, int64(file)*6, fs.temp[:6]) {
			return false
		}
		block = int(fs.temp[3])<<16 + int(fs.temp[4])<<8 + int(fs.temp[5])
		if block <= 0 || int64(block) > fs.datLen()/fileStreamBlockSize {
			return false
		}
	} else {
		// Append: a brand-new block one past the current end of dat.
		block = int((fs.datLen() + fileStreamBlockSize - 1) / fileStreamBlockSize)
		if block == 0 {
			block = 1
		}
	}

	// idx entry: size (3 bytes) + first data block (3 bytes).
	fs.temp[0] = byte(length >> 16)
	fs.temp[1] = byte(length >> 8)
	fs.temp[2] = byte(length)
	fs.temp[3] = byte(block >> 16)
	fs.temp[4] = byte(block >> 8)
	fs.temp[5] = byte(block)
	if !fs.writeAt(fs.idx, int64(file)*6, fs.temp[:6]) {
		return false
	}

	written := 0
	part := 0
	for written < length {
		nextBlock := 0
		if overwrite {
			// Read the existing block header to recover its successor; a short
			// read (chain ran out) leaves nextBlock==0 → switch to appending.
			if fs.readAt(fs.dat, int64(block)*fileStreamBlockSize, fs.temp[:8]) {
				hdrFile := int(fs.temp[0])<<8 + int(fs.temp[1])
				hdrPart := int(fs.temp[2])<<8 + int(fs.temp[3])
				nextBlock = int(fs.temp[4])<<16 + int(fs.temp[5])<<8 + int(fs.temp[6])
				hdrArchive := int(fs.temp[7])
				if hdrFile != file || hdrPart != part || hdrArchive != fs.archive {
					return false
				}
				if nextBlock < 0 || int64(nextBlock) > fs.datLen()/fileStreamBlockSize {
					return false
				}
			}
		}
		if nextBlock == 0 {
			overwrite = false
			nextBlock = int((fs.datLen() + fileStreamBlockSize - 1) / fileStreamBlockSize)
			if nextBlock == 0 {
				nextBlock++
			}
			if nextBlock == block {
				nextBlock++
			}
		}
		if length-written <= fileStreamPayload {
			nextBlock = 0 // last part: no successor
		}

		// Block header: file (2), part (2), nextBlock (3), archive (1).
		fs.temp[0] = byte(file >> 8)
		fs.temp[1] = byte(file)
		fs.temp[2] = byte(part >> 8)
		fs.temp[3] = byte(part)
		fs.temp[4] = byte(nextBlock >> 16)
		fs.temp[5] = byte(nextBlock >> 8)
		fs.temp[6] = byte(nextBlock)
		fs.temp[7] = byte(fs.archive)
		base := int64(block) * fileStreamBlockSize
		if !fs.writeAt(fs.dat, base, fs.temp[:8]) {
			return false
		}
		chunk := min(length-written, fileStreamPayload)
		if !fs.writeAt(fs.dat, base+8, data[written:written+chunk]) {
			return false
		}
		written += chunk
		block = nextBlock
		part++
	}
	return true
}

// FileStreamCache aggregates the five archive FileStreams over one shared dat,
// satisfying the ondemand.Cache contract (Read/Write keyed by archive index).
// The mutex serializes access to the shared dat file, replacing Java's
// synchronized methods + single-threaded loader access.
// Java: Client.fileStreams[5] (Client.java:577), built in maininit (5131-5135).
type FileStreamCache struct {
	mu      sync.Mutex
	streams [5]*FileStream
}

// NewFileStreamCache builds the five archive views over a shared dat file and
// five per-archive idx files. Java:
//
//	for i in 0..5: fileStreams[i] = new FileStream(cache_dat, 29615, cache_idx[i], i+1, 500000)
func NewFileStreamCache(dat *os.File, idx [5]*os.File) *FileStreamCache {
	c := &FileStreamCache{}
	for i := range 5 {
		c.streams[i] = NewFileStream(dat, idx[i], i+1, 500000)
	}
	return c
}

// Read returns the cached bytes for (archive, file), or nil. archive is the
// fileStreams index — OnDemand calls Read(onDemandArchive+1, file), so the
// in-game archives map to streams 1..4. Java: fileStreams[archive].readFromFile.
func (c *FileStreamCache) Read(archive, file int) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.streams[archive].ReadFromFile(file)
}

// Write stores data for (archive, file).
// Java: fileStreams[archive].writeToFile(data.length, file, data).
func (c *FileStreamCache) Write(archive, file int, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.streams[archive].WriteToFile(len(data), file, data)
}

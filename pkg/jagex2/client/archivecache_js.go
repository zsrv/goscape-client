//go:build js

package client

import (
	"strconv"

	"github.com/zsrv/goscape-client/pkg/sign/signlink"
)

// Browser startup-archive cache: the IndexedDB storage seam keyed by
// "<archive>.<file>", matching the TS client's db.read/write(0, fileId) (keys
// "0.1".."0.8"). See Client-TS src/io/Database.ts read/write — the FileStore is
// just an IndexedDB object store whose key is `${archive}.${file}`. There is no
// native main_file_cache on the browser (Client.Cache is nil here), so archives
// persist through signlink's IndexedDB store instead, the same way the rest of
// the wasm cache seam works.
//
// The archive index is always 0 for startup archives, so the key reduces to
// "0." + fileId.
func archiveCacheKey(fileId int) string {
	return "0." + strconv.Itoa(fileId)
}

// archiveCacheLoad returns the cached bytes for startup archive fileId, or nil on
// a miss.
func (c *Client) archiveCacheLoad(fileId int) []byte {
	return signlink.CacheLoad(archiveCacheKey(fileId))
}

// archiveCacheSave persists data for startup archive fileId under the TS-style
// "0.<fileId>" IndexedDB key.
func (c *Client) archiveCacheSave(fileId int, data []byte) {
	signlink.CacheSave(archiveCacheKey(fileId), data)
}

//go:build !js

package client

// Native startup-archive cache: the main_file_cache.dat/.idx store at index 0,
// faithful to Java's getJagFile (fileStreams[0].readFromFile/writeToFile).
//
// Java: getJagFile reads `if (fileStreams[0] != null) fileStreams[0].readFromFile(arg1)`
// and writes `if (fileStreams[0] != null) fileStreams[0].writeToFile(len, arg1, data)`
// (Client.java:4821-4823, 4873-4875 @32f3062). A nil Cache (signlink.cache_dat ==
// null, i.e. the store failed to open) skips caching, exactly as Java does.

// archiveCacheLoad returns the cached bytes for startup archive fileId, or nil on
// a miss / when the store is unavailable.
func (c *Client) archiveCacheLoad(fileId int) []byte {
	if c.Cache == nil {
		return nil
	}
	return c.Cache.Read(0, fileId)
}

// archiveCacheSave persists data for startup archive fileId at FileStore index 0.
func (c *Client) archiveCacheSave(fileId int, data []byte) {
	if c.Cache == nil {
		return
	}
	c.Cache.Write(0, fileId, data)
}

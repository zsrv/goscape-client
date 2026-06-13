//go:build !js

package client

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/io/ondemand"
	"github.com/zsrv/goscape-client/pkg/sign/signlink"
)

// newOnDemandCache opens the on-disk main_file_cache store in the native cache
// directory (signlink.FindCacheDir, e.g. /tmp/.file_store_32). A nil return
// (open failure) leaves OnDemand cache-less, exactly as Java tolerates a null
// signlink.cache_dat. See GetCodeBase / codeBaseURL for the platform split.
func newOnDemandCache() ondemand.Cache {
	return openFileStreamCache(signlink.FindCacheDir())
}

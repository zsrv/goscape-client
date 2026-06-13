//go:build js

package client

import "github.com/zsrv/goscape-client/pkg/jagex2/io/ondemand"

// newOnDemandCache returns nil in the browser: the FileStream store needs
// random-access disk files, which the wasm build has no access to. OnDemand
// runs cache-less (no background prefetch, no disk persistence), exactly as Java
// behaves when signlink.cache_dat is null. A future IndexedDB-backed
// ondemand.Cache could replace this (see the wasm parity roadmap).
func newOnDemandCache() ondemand.Cache {
	return nil
}

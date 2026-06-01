//go:build !js

package client

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

// codeBaseURL returns the native cache-server base URL. It is the -ondemand-server
// value (clientextras.OndemandBaseURL; default http://127.0.0.1:8888). Java's
// frame!=null STANDALONE branch used the literal 127.0.0.1:8888 (client.java:7624)
// plus portOffset; neither the offset nor the host-derivation is ported — the
// endpoint is now configured directly. See GetCodeBase for the platform split.
func codeBaseURL() string {
	return clientextras.OndemandBaseURL
}

//go:build !js

package signlink

import "github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"

// urlBase returns the scheme://host:port that signlink.OpenURL fetches against
// on the native standalone build: the -ondemand-server value
// (clientextras.OndemandBaseURL; default http://127.0.0.1:8888, mirroring Java's
// literal at deob/client.java:7624). The js build derives the origin from
// window.location instead — see signlink_url_js.go.
func urlBase() string {
	return clientextras.OndemandBaseURL
}

//go:build !js

package signlink

import (
	"strconv"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

// urlBase is the scheme://host[:port] that signlink.OpenURL fetches against.
// Native standalone uses the loopback data server (Java's literal
// http://127.0.0.1:<portOffset+8888>); see signlink_url_js.go for the browser
// origin-derived variant.
func urlBase() string {
	return "http://127.0.0.1:" + strconv.Itoa(clientextras.PortOffset+8888)
}

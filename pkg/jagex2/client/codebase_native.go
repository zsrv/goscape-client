//go:build !js

package client

import (
	"strconv"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

// codeBaseURL synthesizes the standalone cache-server URL,
// http://<host>:<portOffset+8888>. Java's frame!=null STANDALONE branch used
// the literal 127.0.0.1; we use the configured host so an operator can point
// the binary at a non-loopback server. See GetCodeBase for the platform split.
func codeBaseURL() string {
	return "http://" + clientextras.Host + ":" + strconv.Itoa(clientextras.PortOffset+8888)
}

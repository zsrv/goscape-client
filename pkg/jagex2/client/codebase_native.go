//go:build !js

package client

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
)

// codeBaseURL synthesizes the standalone cache-server URL, http://<host>:8888.
// Java's frame!=null STANDALONE branch used the literal 127.0.0.1; we use the
// configured host so an operator can point the binary at a non-loopback server.
// See GetCodeBase for the platform split. The Java port offset (portOffset+8888)
// is intentionally not ported — see cmd/client/main.go for the rationale.
func codeBaseURL() string {
	return "http://" + clientextras.Host + ":8888"
}

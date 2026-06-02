package build

import (
	"cmp"
	"fmt"
	"runtime"
)

var (
	Version   string
	Revision  string
	Branch    string
	BuildUser string
	BuildDate string
	GoVersion string
)

func init() {
	GoVersion = runtime.Version()
}

// Info returns the build metadata as a multi-line, human-readable string,
// suitable for printing behind a -version flag. The Version/Revision/Branch/
// BuildUser/BuildDate fields are injected at link time by the Makefile's
// -ldflags -X (see the GO_LDFLAGS block); a plain `go build`/`go run` leaves
// them empty, in which case they render as "unknown". GoVersion always
// reflects the toolchain that compiled the binary (set in init).
func Info() string {
	return fmt.Sprintf(`goscape-client
  version:     %s
  revision:    %s
  branch:      %s
  go version:  %s
  build user:  %s
  build date:  %s`,
		cmp.Or(Version, "unknown"),
		cmp.Or(Revision, "unknown"),
		cmp.Or(Branch, "unknown"),
		GoVersion,
		cmp.Or(BuildUser, "unknown"),
		cmp.Or(BuildDate, "unknown"),
	)
}

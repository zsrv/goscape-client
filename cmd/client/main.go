package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zsrv/goscape-client/pkg/jagex2/launch"
	"github.com/zsrv/goscape-client/pkg/util/build"
)

func main() {
	// Startup configuration comes from flags. This is a Go-original standalone
	// interface: the Java applet read positional args plus a getCodeBase() host.
	// The Java `port-offset` arg (arg0[1] -> portOffset, deob/client.java:10601),
	// which it added to BOTH the data-server port (portOffset + 8888;
	// client.java:7624) and the game socket port (portOffset + 43594;
	// client.java:6786), is not ported: instead of one offset over fixed base
	// ports, -world-server and -ondemand-server take the full scheme://host:port
	// for each endpoint.
	nodeID := flag.Int("node-id", 10, "server node id")
	mem := flag.String("mem", "high", "memory mode: high|low")
	worldType := flag.String("world-type", "members", "world type: free|members")
	worldServer := flag.String("world-server", "tcp://127.0.0.1:43594",
		"game server as [tcp|ws|wss]://host:port")
	ondemandServer := flag.String("ondemand-server", "http://127.0.0.1:8080",
		"on-demand/cache server as [http|https]://host:port")
	showVersion := flag.Bool("version", false, "print build version information and exit")
	flag.Parse()

	// -version prints the build metadata stamped in by the Makefile's -ldflags
	// (see pkg/util/build) and exits before any window/network/audio setup, so
	// it works headlessly. Handled before the startup banner so the output is
	// clean and machine-parseable.
	if *showVersion {
		fmt.Println(build.Info())
		return
	}

	opts := launch.Options{
		NodeID: *nodeID,
	}

	switch *mem {
	case "high":
		opts.LowMemory = false
	case "low":
		opts.LowMemory = true
	default:
		fmt.Printf("invalid -mem %q (want high|low)\n", *mem)
		os.Exit(1)
	}

	switch *worldType {
	case "free":
		opts.Members = false
	case "members":
		opts.Members = true
	default:
		fmt.Printf("invalid -world-type %q (want free|members)\n", *worldType)
		os.Exit(1)
	}

	kind, host, port, path, err := parseWorldServer(*worldServer)
	if err != nil {
		fmt.Printf("invalid -world-server: %v\n", err)
		os.Exit(1)
	}
	opts.Transport = kind
	opts.Host = host
	opts.WorldPort = port
	opts.WSPath = path

	base, err := parseOndemandServer(*ondemandServer)
	if err != nil {
		fmt.Printf("invalid -ondemand-server: %v\n", err)
		os.Exit(1)
	}
	opts.OndemandBaseURL = base

	// launch.Run prints the release banner, applies opts to the process-global
	// client state, starts signlink/audio, and blocks in the game loop on this
	// (main) goroutine. It leaves via clientextras.ExitFunc.
	launch.Run(opts)
}

package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/zsrv/goscape-client/pkg/jagex2/client"
	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
	"github.com/zsrv/goscape-client/pkg/jagex2/client/sign/signlink"
	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
	"github.com/zsrv/goscape-client/pkg/jagex2/sound/audio"
	"github.com/zsrv/goscape-client/pkg/profiling"
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
	ondemandServer := flag.String("ondemand-server", "http://127.0.0.1:8888",
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

	fmt.Println("RS2 user client - release #" + strconv.Itoa(244)) // Java: Client.java:1281

	client.NodeID = *nodeID

	switch *mem {
	case "high":
		client.SetHighMem()
	case "low":
		client.SetLowMem()
	default:
		fmt.Printf("invalid -mem %q (want high|low)\n", *mem)
		os.Exit(1)
	}

	switch *worldType {
	case "free":
		client.MembersWorld = false
	case "members":
		client.MembersWorld = true
	default:
		fmt.Printf("invalid -world-type %q (want free|members)\n", *worldType)
		os.Exit(1)
	}

	// -world-server selects the game-server transport, host, port, and (for
	// ws/wss) path. The parsed bare hostname is stored in clientextras.Host so
	// GetHost/GetCodeBase stay valid; WorldPort/WSPath/Transport drive OpenSocket.
	kind, host, port, path, err := parseWorldServer(*worldServer)
	if err != nil {
		fmt.Printf("invalid -world-server: %v\n", err)
		os.Exit(1)
	}
	clientextras.Host = host
	clientextras.Transport = kind
	clientextras.WorldPort = port
	clientextras.WSPath = path

	// -ondemand-server selects the cache/asset server base URL that
	// signlink.OpenURL and client.GetCodeBase fetch against (native build).
	base, err := parseOndemandServer(*ondemandServer)
	if err != nil {
		fmt.Printf("invalid -ondemand-server: %v\n", err)
		os.Exit(1)
	}
	clientextras.OndemandBaseURL = base

	// Browser builds derive the WebSocket target from window.location here;
	// no-op on native, where the transport comes from the flags above.
	signlink.ConfigureTransport()

	// Register SIGUSR1 profile-capture handler. Non-blocking; returns
	// after signal listener goroutine is spawned. See
	// docs/superpowers/specs/2026-05-22-perf-profiling-design.md.
	profiling.Start()

	// These three subsystems run for the lifetime of the process. There is no
	// explicit shutdown handshake: platform.Main blocks (native: the game loop
	// on the main OS thread; browser: select{}). When RunShell exits the loop
	// closure calls os.Exit(0), which tears down the background signlink and
	// audio goroutines — so no wg.Wait() or cancellation dance is needed here.
	var wg sync.WaitGroup
	wg.Go(func() {
		signlink.StartPriv()
	})
	wg.Go(func() {
		// audio.Start spawns its MIDI watcher goroutine and returns
		// after the oto context is ready (or has failed). The watcher
		// polls signlink.ConsumeMidi for the lifetime of the process;
		// SFX play synchronously via audio.PlayWave (no watcher). Started
		// after signlink so the soundfont fetch (via signlink.OpenURL)
		// doesn't race the protocol coming up.
		//
		// In low-memory mode we bring up no audio at all, matching the
		// Java client: it never starts the MIDI thread, never unpacks
		// sounds.dat, and gates every playback path behind !lowMemory
		// (deob/client.java:5949/6163/7374/...). Initializing oto there
		// would open an audio device and spawn watchers for a queue
		// nothing ever fills. client.LowMemory is set synchronously by
		// SetLowMem above, well before this goroutine reads it.
		if client.LowMemory {
			audio.DisableForLowMemory()
			return
		}
		audio.Start()
	})

	// platform.Main owns the threading model: native locks the OS thread,
	// builds the GLFW backend, and runs the loop on the main goroutine; the
	// browser build builds the WebGL backend and runs the loop in a goroutine,
	// blocking on select{}. The client is created INSIDE the loop closure so it
	// exists only once a backend is Active (NewClient / RunShell allocate
	// PixMaps, which create backend textures via platform.Active). RunShell
	// returns when the loop exits (window close / State == -1), then os.Exit(0)
	// tears down the background signlink + audio goroutines.
	platform.Main(789, 532, "Jagex", func() {
		c := client.NewClient()
		c.RunShell()
		os.Exit(0)
	})
}

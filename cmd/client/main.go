package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/zsrv/goscape-client/pkg/jagex2/client"
	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
	"github.com/zsrv/goscape-client/pkg/jagex2/sound/audio"
	"github.com/zsrv/goscape-client/pkg/profiling"
	"github.com/zsrv/goscape-client/pkg/sign/signlink"
)

func main() {
	fmt.Println("RS2 user client - release #" + strconv.Itoa(225))
	if len(os.Args) < 5 || len(os.Args) > 6 {
		fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members], [host|ws://host[:port][/path]|wss://host[:port][/path]]")
		os.Exit(1)
	}
	var err error
	client.NodeID, err = strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Printf("invalid node-id: %v\n", err)
		os.Exit(1)
	}
	clientextras.PortOffset, err = strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Printf("invalid port-offset: %v\n", err)
		os.Exit(1)
	}
	switch os.Args[3] {
	case "lowmem":
		client.SetLowMem()
	case "highmem":
		client.SetHighMem()
	default:
		fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members], [host|ws://host[:port][/path]|wss://host[:port][/path]]")
		os.Exit(1)
	}
	switch os.Args[4] {
	case "free":
		client.MembersWorld = false
	case "members":
		client.MembersWorld = true
	default:
		fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members], [host|ws://host[:port][/path]|wss://host[:port][/path]]")
		os.Exit(1)
	}
	// Java main accepts exactly 4 args (deob/client.java:10599); the applet host
	// came from getCodeBase().getHost(). This optional 5th `host` arg is a
	// Go-original standalone extension (no browser codebase exists here) that
	// lets the operator point the binary at a non-localhost server. A ws:// or
	// wss:// scheme additionally selects the WebSocket transport (for a future
	// js/wasm build); a bare host keeps the TCP default. The parsed bare
	// hostname is stored in clientextras.Host so GetHost/GetCodeBase stay valid.
	if len(os.Args) == 6 {
		tk, host, wsPort, wsPath, err := parseHostArg(os.Args[5])
		if err != nil {
			fmt.Printf("invalid host: %v\n", err)
			os.Exit(1)
		}
		clientextras.Host = host
		clientextras.Transport = tk
		clientextras.WSPort = wsPort
		clientextras.WSPath = wsPath
	}

	// Browser builds derive the WebSocket target from window.location here;
	// no-op on native, where the transport comes from the host arg above.
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
	platform.Main(532, 789, "Jagex", func() {
		c := client.NewClient()
		c.RunShell()
		os.Exit(0)
	})
}

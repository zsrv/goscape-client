// Package launch wires up and runs the standalone game client: the startup
// sequence extracted from cmd/client so other binaries (goscape-singleplayer's
// combined server+client) can embed the client with programmatic options
// instead of flags. Run blocks for the life of the process and leaves through
// clientextras.ExitFunc.
package launch

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/zsrv/goscape-client/pkg/jagex2/client"
	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
	"github.com/zsrv/goscape-client/pkg/jagex2/sound/audio"
	"github.com/zsrv/goscape-client/pkg/profiling"
	"github.com/zsrv/goscape-client/pkg/sign/signlink"
)

// Options carries the startup configuration for Run. cmd/client builds it
// from flags (already validated); embedders build it directly. The zero value
// is not meaningful — every field mirrors a previously-mandatory flag.
//
// rev-225 predates the -store-id flag; StoreID is not part of this branch's Options.
type Options struct {
	NodeID          int                        // -node-id; server node id
	LowMemory       bool                       // -mem low
	Members         bool                       // -world-type members
	Host            string                     // bare hostname for GetHost/GetCodeBase/TCP dial
	Transport       clientextras.TransportKind // tcp/ws/wss
	WorldPort       int                        // game-server port (every transport)
	WSPath          string                     // ws/wss path; "" means "/"
	OndemandBaseURL string                     // scheme://host:port for cache/asset HTTP fetches
}

// configure applies Options to the package-global client/signlink/clientextras
// state. Split from Run so the assignment wiring is unit-testable without
// opening a window.
func configure(opts Options) {
	client.NodeID = opts.NodeID

	if opts.LowMemory {
		client.SetLowMem()
	} else {
		client.SetHighMem()
	}

	client.MembersWorld = opts.Members

	// -world-server selects the game-server transport, host, port, and (for
	// ws/wss) path. The parsed bare hostname is stored in clientextras.Host so
	// GetHost/GetCodeBase stay valid; WorldPort/WSPath/Transport drive OpenSocket.
	clientextras.Host = opts.Host
	clientextras.Transport = opts.Transport
	clientextras.WorldPort = opts.WorldPort
	clientextras.WSPath = opts.WSPath

	// -ondemand-server selects the cache/asset server base URL that
	// signlink.OpenURL and client.GetCodeBase fetch against (native build).
	clientextras.OndemandBaseURL = opts.OndemandBaseURL
}

// Run configures the process-global client state, starts the background
// subsystems, and runs the game loop on the calling goroutine (which must be
// the main goroutine — platform.Main locks the OS thread on native builds).
// It never returns: the loop closure leaves through clientextras.ExitFunc.
func Run(opts Options) {
	fmt.Println("RS2 user client - release #" + strconv.Itoa(225))

	configure(opts)

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
	// closure calls clientextras.ExitFunc(0), which tears down the background signlink and
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
	// returns when the loop exits (window close / State == -1), then clientextras.ExitFunc(0)
	// tears down the background signlink + audio goroutines.
	platform.Main(789, 532, "Jagex", func() {
		c := client.NewClient()
		c.RunShell()
		clientextras.ExitFunc(0)
	})
}

package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"gioui.org/app"

	"goscape-client/pkg/jagex2/client"
	"goscape-client/pkg/jagex2/client/clientextras"
	"goscape-client/pkg/jagex2/sound/audio"
	"goscape-client/pkg/sign/signlink"
)

func main() {
	fmt.Println("RS2 user client - release #" + strconv.Itoa(225))
	if len(os.Args) < 5 || len(os.Args) > 6 {
		fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members], [host]")
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
		fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members]")
		os.Exit(1)
	}
	switch os.Args[4] {
	case "free":
		client.MembersWorld = false
	case "members":
		client.MembersWorld = true
	default:
		fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members], [host]")
		os.Exit(1)
	}
	if len(os.Args) == 6 {
		clientextras.Host = os.Args[5]
	}

	// TODO: if initApplication shuts down, shut the network thread down and exit?
	//  use select{}?
	var wg sync.WaitGroup
	wg.Go(func() {
		signlink.StartPriv()
	})
	wg.Go(func() {
		// audio.Start spawns its own watcher goroutines and returns
		// after the oto context is ready (or has failed). The watchers
		// poll signlink.ConsumeMidi / ConsumeWave for the lifetime of
		// the process. Started after signlink so the soundfont fetch
		// (via signlink.OpenURL) doesn't race the protocol coming up.
		audio.Start()
	})
	wg.Go(func() {
		c := client.NewClient()
		c.InitApplication(532, 789)
	})
	// Gio's documented pattern (https://gioui.org/app) requires app.Main()
	// to run on the OS main thread — mandatory on macOS, looser elsewhere.
	// It blocks until the last window closes. In practice the window
	// goroutine inside InitApplication calls os.Exit(0) on DestroyEvent,
	// so wg.Wait() below is only reached in degenerate paths.
	app.Main()
	wg.Wait()
}

package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"goscape-client/pkg/jagex2/client"
	"goscape-client/pkg/jagex2/client/clientextras"
	"goscape-client/pkg/sign/signlink"
)

func main() {
	fmt.Println("RS2 user client - release #" + strconv.Itoa(225))
	if len(os.Args) != 5 {
		fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members]")
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
		client.SetLowMemory()
	case "highmem":
		client.SetHighMemory()
	default:
		fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members]")
		os.Exit(1)
	}
	switch os.Args[4] {
	case "free":
		client.Members = false
	case "members":
		client.Members = true
	default:
		fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members]")
		os.Exit(1)
	}

	// TODO: if initApplication shuts down, shut the network thread down and exit?
	//  use select{}?
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		signlink.StartPriv()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.NewClient().InitApplication(532, 789)
	}()
	wg.Wait()
}

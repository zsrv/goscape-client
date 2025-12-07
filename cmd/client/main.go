package main

import (
	"fmt"
	"os"
	"strconv"

	"goscape-client/pkg/deob/client"
)

func main() {
	fmt.Println("RS2 user client - release #", 225)
	if len(os.Args) != 4 {
		fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members]")
		os.Exit(1)
	}
	var err error
	client.NodeID, err = strconv.Atoi(os.Args[0])
	if err != nil {
		fmt.Printf("invalid node-id: %v\n", err)
		os.Exit(1)
	}
	client.PortOffset, err = strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Printf("invalid port-offset: %v\n", err)
		os.Exit(1)
	}
	switch os.Args[2] {
	case "lowmem":
	// TODO
	case "highmem":
	// TODO
	default:
		fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members]")
		os.Exit(1)
	}
	switch os.Args[3] {
	case "free":
		client.Members = false
	case "members":
		client.Members = true
	default:
		fmt.Println("Usage: node-id, port-offset, [lowmem/highmem], [free/members]")
		os.Exit(1)
	}
	// TODO: signlink.startpriv
	// TODO: new client(), initApplication
}

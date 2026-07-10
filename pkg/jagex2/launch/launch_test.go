package launch

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/client"
	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
	"github.com/zsrv/goscape-client/pkg/jagex2/client/sign/signlink"
)

// configure writes package globals across client/signlink/clientextras; save
// and restore them so this test cannot leak state into other tests.
func TestConfigureAppliesOptions(t *testing.T) {
	savedStoreID := signlink.StoreID
	savedNodeID := client.NodeID
	savedLowMem := client.LowMemory
	savedMembers := client.MembersWorld
	savedHost := clientextras.Host
	savedTransport := clientextras.Transport
	savedPort := clientextras.WorldPort
	savedWSPath := clientextras.WSPath
	savedBase := clientextras.OndemandBaseURL
	t.Cleanup(func() {
		signlink.StoreID = savedStoreID
		client.NodeID = savedNodeID
		if savedLowMem {
			client.SetLowMem()
		} else {
			client.SetHighMem()
		}
		client.MembersWorld = savedMembers
		clientextras.Host = savedHost
		clientextras.Transport = savedTransport
		clientextras.WorldPort = savedPort
		clientextras.WSPath = savedWSPath
		clientextras.OndemandBaseURL = savedBase
	})

	configure(Options{
		NodeID:          12,
		StoreID:         33,
		LowMemory:       true,
		Members:         true,
		Host:            "127.0.0.1",
		Transport:       clientextras.TransportTCP,
		WorldPort:       40594,
		WSPath:          "",
		OndemandBaseURL: "http://127.0.0.1:40080",
	})

	if signlink.StoreID != 33 {
		t.Errorf("signlink.StoreID = %d, want 33", signlink.StoreID)
	}
	if client.NodeID != 12 {
		t.Errorf("client.NodeID = %d, want 12", client.NodeID)
	}
	if !client.LowMemory {
		t.Error("client.LowMemory = false, want true (SetLowMem not called)")
	}
	if !client.MembersWorld {
		t.Error("client.MembersWorld = false, want true")
	}
	if clientextras.Host != "127.0.0.1" {
		t.Errorf("clientextras.Host = %q", clientextras.Host)
	}
	if clientextras.Transport != clientextras.TransportTCP {
		t.Errorf("clientextras.Transport = %v", clientextras.Transport)
	}
	if clientextras.WorldPort != 40594 {
		t.Errorf("clientextras.WorldPort = %d", clientextras.WorldPort)
	}
	if clientextras.WSPath != "" {
		t.Errorf("clientextras.WSPath = %q", clientextras.WSPath)
	}
	if clientextras.OndemandBaseURL != "http://127.0.0.1:40080" {
		t.Errorf("clientextras.OndemandBaseURL = %q", clientextras.OndemandBaseURL)
	}

	// HighMem path too — SetHighMem must be called when LowMemory is false.
	configure(Options{Host: "127.0.0.1", OndemandBaseURL: "http://127.0.0.1:40080"})
	if client.LowMemory {
		t.Error("client.LowMemory = true after configure with LowMemory:false")
	}
}

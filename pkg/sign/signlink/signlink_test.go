package signlink

import (
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"goscape-client/pkg/jagex2/client/clientextras"
)

func TestFindCacheDir(t *testing.T) {
	t.Run("test", func(t *testing.T) {
		got := GetUID(FindCacheDir())
		fmt.Printf("got %+v\n", got)
	})
}

func TestOpenSocket(t *testing.T) {
	t.Run("connects to a listening port and round-trips one byte", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		t.Cleanup(func() { ln.Close() })

		_, portStr, err := net.SplitHostPort(ln.Addr().String())
		if err != nil {
			t.Fatalf("split host/port: %v", err)
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			t.Fatalf("parse port: %v", err)
		}

		accepted := make(chan net.Conn, 1)
		go func() {
			c, err := ln.Accept()
			if err != nil {
				t.Errorf("accept: %v", err)
				close(accepted)
				return
			}
			accepted <- c
		}()

		prev := clientextras.Host
		clientextras.Host = "127.0.0.1"
		t.Cleanup(func() { clientextras.Host = prev })

		conn, err := OpenSocket(port)
		if err != nil {
			t.Fatalf("OpenSocket: %v", err)
		}
		t.Cleanup(func() { conn.Close() })

		var server net.Conn
		select {
		case server = <-accepted:
			if server == nil {
				t.Fatal("accept failed")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("server never accepted")
		}
		t.Cleanup(func() { server.Close() })

		if _, err := conn.Write([]byte{0x42}); err != nil {
			t.Fatalf("client write: %v", err)
		}
		buf := make([]byte, 1)
		if err := server.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatalf("set deadline: %v", err)
		}
		if _, err := server.Read(buf); err != nil {
			t.Fatalf("server read: %v", err)
		}
		if buf[0] != 0x42 {
			t.Fatalf("round-trip byte = %#x, want 0x42", buf[0])
		}
	})

	t.Run("returns an error when dialing a port that isn't listening", func(t *testing.T) {
		// Grab a port, then close the listener so the port is (almost certainly)
		// free. Any subsequent dial should be refused.
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		_, portStr, err := net.SplitHostPort(ln.Addr().String())
		if err != nil {
			ln.Close()
			t.Fatalf("split host/port: %v", err)
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			ln.Close()
			t.Fatalf("parse port: %v", err)
		}
		ln.Close()

		prev := clientextras.Host
		clientextras.Host = "127.0.0.1"
		t.Cleanup(func() { clientextras.Host = prev })

		conn, err := OpenSocket(port)
		if err == nil {
			conn.Close()
			t.Fatal("OpenSocket succeeded against a closed port; expected error")
		}
	})
}

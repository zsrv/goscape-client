package clientstream

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

// readAll on the peer side of net.Pipe; returns once n bytes have been read
// or an error / timeout occurs.
func readN(t *testing.T, conn net.Conn, n int) []byte {
	t.Helper()
	out := make([]byte, n)
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	if _, err := io.ReadFull(conn, out); err != nil {
		t.Fatalf("peer read: %v", err)
	}
	return out
}

func TestWriteRoundTrip(t *testing.T) {
	a, b := net.Pipe()
	defer func() { _ = b.Close() }()
	cs := NewClientStream(a)
	defer cs.Close()

	payload := []byte{0x10, 0x20, 0x30, 0x40, 0x50}
	if err := cs.Write(payload, len(payload), 0); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got := readN(t, b, len(payload))
	if string(got) != string(payload) {
		t.Fatalf("got %v, want %v", got, payload)
	}
}

func TestWriteOffsetAndLength(t *testing.T) {
	a, b := net.Pipe()
	defer func() { _ = b.Close() }()
	cs := NewClientStream(a)
	defer cs.Close()

	// Java arg order: (buf, length, offset). Send only bytes 2..5 of src.
	src := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	if err := cs.Write(src, 3, 2); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got := readN(t, b, 3)
	want := []byte{0xCC, 0xDD, 0xEE}
	if string(got) != string(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestReadAndReadFully(t *testing.T) {
	a, b := net.Pipe()
	defer func() { _ = a.Close() }()
	cs := NewClientStream(a)
	defer cs.Close()

	go func() {
		_, _ = b.Write([]byte{0x42, 0x10, 0x20, 0x30})
		_ = b.Close()
	}()

	v, err := cs.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if v != 0x42 {
		t.Fatalf("Read got %d, want 0x42", v)
	}

	buf := make([]byte, 5)
	buf[0] = 0xFF // sentinel to verify off
	if err := cs.ReadFully(buf, 1, 3); err != nil {
		t.Fatalf("ReadFully: %v", err)
	}
	if buf[0] != 0xFF {
		t.Fatalf("ReadFully clobbered offset; buf[0]=%d", buf[0])
	}
	if buf[1] != 0x10 || buf[2] != 0x20 || buf[3] != 0x30 {
		t.Fatalf("ReadFully wrong payload: %v", buf)
	}
}

func TestReadEOFReturnsMinusOne(t *testing.T) {
	a, b := net.Pipe()
	cs := NewClientStream(a)
	defer cs.Close()

	_ = b.Close()
	v, err := cs.Read()
	if err != nil {
		t.Fatalf("Read at EOF returned err: %v", err)
	}
	if v != -1 {
		t.Fatalf("Read at EOF got %d, want -1", v)
	}
}

func TestCloseUnblocksReader(t *testing.T) {
	a, b := net.Pipe()
	defer func() { _ = b.Close() }()
	cs := NewClientStream(a)

	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 4)
		done <- cs.ReadFully(buf, 0, 4)
	}()

	// Give the reader a moment to actually block in conn.Read.
	time.Sleep(50 * time.Millisecond)
	cs.Close()

	select {
	case err := <-done:
		// Closing the underlying conn while bufio.Reader is blocked
		// surfaces as a non-nil error (e.g. "use of closed network
		// connection" or io.ErrClosedPipe). We just need to confirm the
		// reader actually unblocked.
		if err == nil {
			t.Fatal("expected non-nil error from ReadFully after Close")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ReadFully did not unblock after Close")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	a, b := net.Pipe()
	defer func() { _ = b.Close() }()
	cs := NewClientStream(a)

	cs.Close()
	cs.Close() // must not panic or double-close
	cs.Close()
}

func TestWriteAfterCloseIsNoOp(t *testing.T) {
	a, b := net.Pipe()
	defer func() { _ = b.Close() }()
	cs := NewClientStream(a)

	cs.Close()
	if err := cs.Write([]byte{1, 2, 3}, 3, 0); err != nil {
		t.Fatalf("Write after Close returned err: %v", err)
	}
}

func TestReadFullyEOFBeforeComplete(t *testing.T) {
	a, b := net.Pipe()
	cs := NewClientStream(a)
	defer cs.Close()

	go func() {
		_, _ = b.Write([]byte{0x01, 0x02})
		_ = b.Close()
	}()

	buf := make([]byte, 4)
	err := cs.ReadFully(buf, 0, 4)
	if err == nil {
		t.Fatal("expected error from ReadFully on truncated stream")
	}
	// Either io.ErrUnexpectedEOF (short read with n>0 then EOF) or io.EOF
	// (immediate EOF) is acceptable.
	if !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMultipleWritesDrain(t *testing.T) {
	a, b := net.Pipe()
	defer func() { _ = b.Close() }()
	cs := NewClientStream(a)
	defer cs.Close()

	// Issue several writes; the writer goroutine has to drain each one
	// against the synchronous net.Pipe. This exercises the cond.Wait
	// path: the writer drains, sleeps (bufPos == bufLen), then wakes on
	// the next Write.
	for i := range 5 {
		if err := cs.Write([]byte{byte(i)}, 1, 0); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
	}
	got := readN(t, b, 5)
	for i, v := range got {
		if int(v) != i {
			t.Fatalf("byte %d: got %d, want %d", i, v, i)
		}
	}
}

// loopbackPair returns a connected TCP pair on 127.0.0.1, giving us a real
// kernel send/recv buffer. net.Pipe is synchronous (Write blocks until the
// peer Reads), so it can't model "data queued at the OS but not yet pulled
// into bufio" — exactly the case Available() must handle.
func loopbackPair(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()
	type res struct {
		c   net.Conn
		err error
	}
	accepted := make(chan res, 1)
	go func() {
		c, err := ln.Accept()
		accepted <- res{c, err}
	}()
	dialed, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	r := <-accepted
	if r.err != nil {
		t.Fatalf("accept: %v", r.err)
	}
	return dialed, r.c
}

// Regression for the lazy-bufio.Reader bug: Available() must reflect data
// the peer wrote even when no prior Read/ReadFully has happened. The Java
// dispatcher uses Available()>0 as a non-blocking "is data ready" gate at
// the top of read(), so if it returns 0 here the dispatcher never advances
// and PacketCycle times out into "Connection lost".
func TestAvailableSeesUnreadKernelData(t *testing.T) {
	a, b := loopbackPair(t)
	cs := NewClientStream(a)
	defer cs.Close()
	defer func() { _ = b.Close() }()

	if _, err := b.Write([]byte{1, 2, 3, 4, 5}); err != nil {
		t.Fatalf("peer Write: %v", err)
	}

	// Give the kernel a moment to deliver the bytes to a's recv buffer.
	deadline := time.Now().Add(2 * time.Second)
	for {
		n, err := cs.Available()
		if err != nil {
			t.Fatalf("Available: %v", err)
		}
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Available stayed at 0 even though peer wrote 5 bytes")
		}
		time.Sleep(5 * time.Millisecond)
	}

	// After Available() reports data, ReadFully must succeed normally —
	// the deadline probe inside Available() must not poison bufio's error
	// state.
	buf := make([]byte, 5)
	if err := cs.ReadFully(buf, 0, 5); err != nil {
		t.Fatalf("ReadFully after Available probe: %v", err)
	}
	if string(buf) != string([]byte{1, 2, 3, 4, 5}) {
		t.Fatalf("ReadFully got %v, want [1 2 3 4 5]", buf)
	}
}

// Available() must not poison subsequent blocking reads when it probes an
// empty conn (timeout). The probe clears bufio's cached err; if it didn't,
// the next ReadFully would surface a spurious ErrDeadlineExceeded.
func TestAvailableProbeDoesNotPoisonReadFully(t *testing.T) {
	a, b := loopbackPair(t)
	cs := NewClientStream(a)
	defer cs.Close()
	defer func() { _ = b.Close() }()

	// Probe an empty conn — must return 0 without error.
	n, err := cs.Available()
	if err != nil {
		t.Fatalf("Available on empty: %v", err)
	}
	if n != 0 {
		t.Fatalf("Available on empty got %d, want 0", n)
	}

	// Now write from the peer after the probe.
	go func() {
		time.Sleep(20 * time.Millisecond)
		_, _ = b.Write([]byte{0xAB, 0xCD})
	}()

	buf := make([]byte, 2)
	if err := cs.ReadFully(buf, 0, 2); err != nil {
		t.Fatalf("ReadFully after empty probe: %v", err)
	}
	if buf[0] != 0xAB || buf[1] != 0xCD {
		t.Fatalf("ReadFully got %v, want [0xAB 0xCD]", buf)
	}
}

func TestAvailableReportsBuffered(t *testing.T) {
	a, b := net.Pipe()
	cs := NewClientStream(a)
	defer cs.Close()

	go func() {
		_, _ = b.Write([]byte{1, 2, 3, 4, 5})
	}()

	// Trigger a buffered read by reading one byte. bufio.Reader will
	// pre-fetch more, which Available() should then report > 0.
	if _, err := cs.Read(); err != nil {
		t.Fatalf("Read: %v", err)
	}
	n, err := cs.Available()
	if err != nil {
		t.Fatalf("Available: %v", err)
	}
	if n <= 0 {
		t.Fatalf("Available reported %d, want > 0 after pre-fetched read", n)
	}

	cs.Close()
	n, _ = cs.Available()
	if n != 0 {
		t.Fatalf("Available after Close = %d, want 0", n)
	}
}

// TestReadTimesOutWhenNoData verifies the SO_TIMEOUT port (Java
// socket.setSoTimeout(30000)): a blocking Read with no data and no error must
// give up after the timeout window with ErrReadTimeout, not hang forever.
func TestReadTimesOutWhenNoData(t *testing.T) {
	a, b := net.Pipe()
	defer func() { _ = b.Close() }() // peer kept open but silent — forces a stall, not EOF
	cs := NewClientStream(a)
	cs.readTimeout = 50 * time.Millisecond
	defer cs.Close()

	start := time.Now()
	n, err := cs.Read()
	if !errors.Is(err, ErrReadTimeout) {
		t.Fatalf("Read: want ErrReadTimeout, got n=%d err=%v", n, err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("Read blocked %v, want ~50ms", elapsed)
	}
}

// TestReadFullyTimesOutWhenNoData is the ReadFully counterpart.
func TestReadFullyTimesOutWhenNoData(t *testing.T) {
	a, b := net.Pipe()
	defer func() { _ = b.Close() }()
	cs := NewClientStream(a)
	cs.readTimeout = 50 * time.Millisecond
	defer cs.Close()

	dst := make([]byte, 8)
	start := time.Now()
	err := cs.ReadFully(dst, 0, 8)
	if !errors.Is(err, ErrReadTimeout) {
		t.Fatalf("ReadFully: want ErrReadTimeout, got err=%v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("ReadFully blocked %v, want ~50ms", elapsed)
	}
}

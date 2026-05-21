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
	defer b.Close()
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
	defer b.Close()
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
	defer a.Close()
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
	defer b.Close()
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
	defer b.Close()
	cs := NewClientStream(a)

	cs.Close()
	cs.Close() // must not panic or double-close
	cs.Close()
}

func TestWriteAfterCloseIsNoOp(t *testing.T) {
	a, b := net.Pipe()
	defer b.Close()
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
	defer b.Close()
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

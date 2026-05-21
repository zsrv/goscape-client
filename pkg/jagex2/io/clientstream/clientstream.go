// Package clientstream is a Go port of jagex2/io/ClientStream.java.
//
// It wraps a net.Conn with a bufio.Reader for input and a fixed-size ring
// buffer + writer goroutine for output. The Java source's synchronized +
// wait/notify pattern is preserved 1:1 with sync.Cond — each Wait/Signal
// matches a Java wait()/notify() — rather than collapsing the ring buffer
// into a channel, which would diverge structurally from the reference.
package clientstream

import (
	"bufio"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

const (
	bufSize = 5000
	// guardGap mirrors Java's "(bufLen + 4900) % 5000" overflow check: the
	// ring reserves a 100-byte gap rather than the usual 1-byte sentinel.
	guardGap = 4900
)

// ClientStream is the Go equivalent of jagex2.io.ClientStream.
type ClientStream struct {
	conn net.Conn
	in   *bufio.Reader

	// closed is read without the lock by Read/Available, so it's atomic.
	closed atomic.Bool

	mu   sync.Mutex
	cond *sync.Cond
	// Fields below are guarded by mu (matching Java's `synchronized (this)`).
	buf     []byte
	bufLen  int // drain tail
	bufPos  int // writer head
	writer  bool
	ioerror bool
}

// NewClientStream wraps conn. It does not dial — the caller is responsible
// for establishing the connection. Mirrors `ClientStream(GameShell, Socket)`
// in Java; the GameShell argument is dropped because the Java side used it
// only to spawn the writer thread via shell.startThread(this, 2), which the
// Go side does directly with `go cs.run()` (PORTING.md §2.3).
func NewClientStream(conn net.Conn) *ClientStream {
	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(true)
	}
	// TODO: Java calls socket.setSoTimeout(30000). Go has no native
	// per-read timeout; rolling SetReadDeadline before each read would
	// achieve the same effect but is left for a later pass.
	cs := &ClientStream{
		conn: conn,
		in:   bufio.NewReader(conn),
	}
	cs.cond = sync.NewCond(&cs.mu)
	return cs
}

// Close terminates the writer goroutine and closes the underlying
// connection. Idempotent — subsequent calls are no-ops.
func (cs *ClientStream) Close() {
	if !cs.closed.CompareAndSwap(false, true) {
		return
	}
	cs.mu.Lock()
	cs.writer = false
	cs.buf = nil
	cs.cond.Broadcast()
	cs.mu.Unlock()

	// Closing conn unblocks any goroutine blocked in Read on the bufio
	// reader; Java closes in/out/socket separately, but a single
	// net.Conn.Close is the Go equivalent.
	_ = cs.conn.Close()
}

// Read mirrors Java's `int read()`. Returns the next byte as a non-negative
// int, or -1 on EOF (matching Java InputStream.read semantics).
func (cs *ClientStream) Read() (int, error) {
	if cs.closed.Load() {
		return 0, nil
	}
	b, err := cs.in.ReadByte()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return -1, nil
		}
		return 0, err
	}
	return int(b), nil
}

// Available mirrors Java's `int available()`.
//
// Note the semantic gap: Java's InputStream.available() reports bytes the
// OS has buffered for this socket, while bufio.Reader.Buffered() reports
// only what bufio has already pre-fetched. Callers using it as a
// "can-I-read-without-blocking" hint will see equivalent behavior; callers
// that depend on the exact OS-buffered count will not.
func (cs *ClientStream) Available() (int, error) {
	if cs.closed.Load() {
		return 0, nil
	}
	return cs.in.Buffered(), nil
}

// ReadFully mirrors Java's `read(byte[] arg0, int arg1, int arg2)`. Reads
// exactly arg2 bytes into arg0 starting at offset arg1, blocking as needed.
// Returns io.ErrUnexpectedEOF if the stream ends before arg2 bytes arrive
// (Java throws IOException("EOF")).
func (cs *ClientStream) ReadFully(arg0 []byte, arg1, arg2 int) error {
	if cs.closed.Load() {
		return nil
	}
	for arg2 > 0 {
		n, err := cs.in.Read(arg0[arg1 : arg1+arg2])
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrUnexpectedEOF
		}
		arg1 += n
		arg2 -= n
	}
	return nil
}

// Write mirrors Java's `write(byte[] arg0, int arg1, int arg3)`.
//
// Parameter order matches the Java source verbatim, including the unusual
// (buf, length, offset) layout: an `arg2` boolean was stripped during
// deobfuscation, leaving the index gap. arg1 is the byte count, arg3 is
// the source offset.
func (cs *ClientStream) Write(arg0 []byte, arg1, arg3 int) error {
	if cs.closed.Load() {
		return nil
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.ioerror {
		cs.ioerror = false
		return errors.New("Error in writer thread")
	}
	if cs.buf == nil {
		cs.buf = make([]byte, bufSize)
	}
	for var7 := range arg1 {
		cs.buf[cs.bufPos] = arg0[var7+arg3]
		cs.bufPos = (cs.bufPos + 1) % bufSize
		if cs.bufPos == (cs.bufLen+guardGap)%bufSize {
			return errors.New("buffer overflow")
		}
	}
	if !cs.writer {
		cs.writer = true
		go cs.run()
	}
	cs.cond.Signal()
	return nil
}

// run is the writer goroutine. Drains the ring buffer to the underlying
// connection, then waits on cond when the buffer is empty. Started lazily
// by Write on first call; exits when Close flips writer=false and
// broadcasts.
func (cs *ClientStream) run() {
	for {
		cs.mu.Lock()
		if !cs.writer {
			cs.mu.Unlock()
			return
		}
		var var1, var2 int
		if cs.bufPos == cs.bufLen {
			cs.cond.Wait()
		}
		if !cs.writer {
			cs.mu.Unlock()
			return
		}
		var2 = cs.bufLen
		if cs.bufPos >= cs.bufLen {
			var1 = cs.bufPos - cs.bufLen
		} else {
			var1 = bufSize - cs.bufLen
		}
		var chunk []byte
		if var1 > 0 {
			// Snapshot under the lock; Close() may nil cs.buf afterwards
			// but the underlying array stays alive via this slice.
			chunk = cs.buf[var2 : var2+var1]
		}
		cs.mu.Unlock()

		if var1 > 0 {
			_, err := cs.conn.Write(chunk)
			cs.mu.Lock()
			if err != nil {
				cs.ioerror = true
			}
			cs.bufLen = (cs.bufLen + var1) % bufSize
			// Java calls out.flush() when bufPos == bufLen here; net.Conn
			// is unbuffered so there's nothing to flush. Site preserved
			// in shape for parity with the Java source.
			cs.mu.Unlock()
		}
	}
}

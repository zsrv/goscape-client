// Package clientstream is a Go port of jagex2/io/ClientStream.java.
//
// Both the read and write sides use a fixed-size ring buffer fed/drained by
// a dedicated goroutine. The writer goroutine drains the outbound ring to
// conn; the reader goroutine fills the inbound ring from conn. The Java
// source's synchronized + wait/notify pattern is preserved 1:1 with
// sync.Cond — each Wait/Broadcast matches a Java wait()/notifyAll() —
// rather than collapsing the rings into channels, which would diverge
// structurally from the reference.
//
// The reader goroutine exists because Java's InputStream.available() is
// expected to report the count of bytes the OS has queued for this socket,
// and the game dispatcher uses `available() > 0` as a non-blocking gate at
// the top of read(). A naive port to bufio.Reader.Buffered() leaves the
// gate closed forever because bufio is lazy (only pulls when a Read/Peek
// is invoked). Eagerly draining conn into our own ring buffer restores the
// expected semantics.
package clientstream

import (
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	bufSize = 5000
	// guardGap mirrors Java's "(bufLen + 4900) % 5000" overflow check: the
	// outbound ring reserves a 100-byte gap rather than the usual 1-byte
	// sentinel.
	guardGap = 4900
	// readBufSize is the inbound ring's capacity. Large enough to hold a
	// burst of opcodes including a full REBUILD_NORMAL payload (~5500 bytes)
	// without forcing the reader goroutine to block.
	readBufSize = 8192
	// defaultReadTimeout reproduces Java's `socket.setSoTimeout(30000)`
	// (ClientStream.java:46): a blocking read that goes this long without the
	// data it needs gives up instead of hanging forever. Java applied this at
	// the socket so each blocking in.read() threw SocketTimeoutException; the
	// Go port has no socket-level blocking read (the reader goroutine fills a
	// ring), so the equivalent bound is applied to the consumer-side wait in
	// Read/ReadFully — the only place that actually blocks. This matches Java's
	// behavior exactly: the dispatcher gates gameplay reads on Available(), so
	// only the login handshake's direct blocking reads are bounded; an idle
	// (but healthy) in-game connection never blocks here, so it is never timed
	// out, just as Java never times out an idle connection it isn't reading.
	defaultReadTimeout = 30 * time.Second
)

// ErrReadTimeout is returned by Read/ReadFully when the SO_TIMEOUT window
// elapses with insufficient data. It is the Go analog of Java's
// SocketTimeoutException (an IOException), so callers route it to the same
// reconnect/login-error path as any other read error.
var ErrReadTimeout = errors.New("clientstream: read timed out (SO_TIMEOUT)")

// ClientStream is the Go equivalent of jagex2.io.ClientStream.
type ClientStream struct {
	conn net.Conn

	// closed is read without the lock by callers, so it's atomic.
	closed atomic.Bool

	mu   sync.Mutex
	cond *sync.Cond
	// Fields below are guarded by mu (matching Java's `synchronized (this)`).

	// Outbound ring buffer (drained by run()).
	buf     []byte
	bufLen  int // drain tail
	bufPos  int // writer head
	writer  bool
	ioerror bool

	// Inbound ring buffer (filled by readRun(), drained by Read/ReadFully).
	// Initialized once in NewClientStream — never nil during the stream's
	// lifetime, which lets callers index it without re-checking under lock.
	rbuf []byte
	rPos int   // write head (advanced by readRun)
	rLen int   // read tail (advanced by Read/ReadFully)
	rErr error // sticky read-side error (EOF, conn closed, …)

	// readTimeout is the SO_TIMEOUT window for a blocking consumer read.
	// Set once at construction; tests may shorten it before issuing reads.
	readTimeout time.Duration
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
	cs := &ClientStream{
		conn:        conn,
		rbuf:        make([]byte, readBufSize),
		readTimeout: defaultReadTimeout,
	}
	cs.cond = sync.NewCond(&cs.mu)
	go cs.readRun()
	return cs
}

// broadcast wakes every goroutine blocked on cond. Used as the time.AfterFunc
// callback that ends a blocking read's SO_TIMEOUT window.
func (cs *ClientStream) broadcast() {
	cs.mu.Lock()
	cs.cond.Broadcast()
	cs.mu.Unlock()
}

// Close terminates both goroutines and closes the underlying connection.
// Idempotent — subsequent calls are no-ops.
func (cs *ClientStream) Close() {
	if !cs.closed.CompareAndSwap(false, true) {
		return
	}
	cs.mu.Lock()
	if cs.rErr == nil {
		cs.rErr = net.ErrClosed
	}
	cs.writer = false
	cs.buf = nil
	cs.cond.Broadcast()
	cs.mu.Unlock()

	// Closing conn unblocks the reader goroutine's blocked conn.Read.
	_ = cs.conn.Close()
}

// availableLocked returns the count of bytes ready in rbuf. Caller holds mu.
func (cs *ClientStream) availableLocked() int {
	return (cs.rPos - cs.rLen + readBufSize) % readBufSize
}

// readRun is the reader goroutine. Eagerly drains conn into rbuf so that
// Available() can return an accurate count without blocking. Exits when
// conn.Read errors (EOF, close, network failure) or when Close fires.
func (cs *ClientStream) readRun() {
	for {
		cs.mu.Lock()
		// Block while the ring is full. The sentinel — one empty slot
		// between head and tail — distinguishes "full" from "empty".
		for cs.availableLocked() == readBufSize-1 && !cs.closed.Load() {
			cs.cond.Wait()
		}
		if cs.closed.Load() {
			cs.mu.Unlock()
			return
		}
		// Compute the contiguous free region starting at rPos. Reserve a
		// sentinel slot so rPos == rLen unambiguously means empty.
		var maxN int
		if cs.rPos >= cs.rLen {
			maxN = readBufSize - cs.rPos
			if cs.rLen == 0 {
				maxN-- // sentinel reserved at index 0
			}
		} else {
			maxN = cs.rLen - cs.rPos - 1
		}
		head := cs.rPos
		cs.mu.Unlock()

		// conn.Read outside the lock — the syscall blocks until data
		// arrives or conn is closed; holding the mutex would deadlock any
		// concurrent Close or consumer.
		n, err := cs.conn.Read(cs.rbuf[head : head+maxN])

		cs.mu.Lock()
		if n > 0 {
			cs.rPos = (cs.rPos + n) % readBufSize
			cs.cond.Broadcast()
		}
		if err != nil {
			if cs.rErr == nil {
				cs.rErr = err
			}
			cs.cond.Broadcast()
			cs.mu.Unlock()
			return
		}
		cs.mu.Unlock()
	}
}

// Read mirrors Java's `int read()`. Returns the next byte as a non-negative
// int, or -1 on EOF (matching Java InputStream.read semantics).
func (cs *ClientStream) Read() (int, error) {
	if cs.closed.Load() {
		return 0, nil
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	var deadline time.Time
	armed := false
	for cs.availableLocked() == 0 {
		if cs.rErr != nil {
			if errors.Is(cs.rErr, io.EOF) {
				return -1, nil
			}
			return 0, cs.rErr
		}
		if !armed {
			// Java: socket.setSoTimeout(30000) — bound this blocking read to
			// the SO_TIMEOUT window. Arm a single timer that broadcasts on
			// expiry to break the cond.Wait; the deadline re-check below turns
			// that wakeup into ErrReadTimeout.
			deadline = time.Now().Add(cs.readTimeout)
			timer := time.AfterFunc(cs.readTimeout, cs.broadcast)
			defer timer.Stop()
			armed = true
		}
		if !time.Now().Before(deadline) {
			return 0, ErrReadTimeout
		}
		cs.cond.Wait()
	}
	b := cs.rbuf[cs.rLen]
	cs.rLen = (cs.rLen + 1) % readBufSize
	cs.cond.Broadcast() // unblock the reader if it was waiting on a full ring
	return int(b), nil
}

// Available mirrors Java's `int available()` — the number of bytes that can
// be read without blocking. With the reader goroutine eagerly draining conn,
// this is just the count of bytes ready in rbuf.
func (cs *ClientStream) Available() (int, error) {
	if cs.closed.Load() {
		return 0, nil
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.availableLocked(), nil
}

// ReadFully mirrors Java's `read(byte[] arg0, int arg1, int arg2)`. Reads
// exactly arg2 bytes into arg0 starting at offset arg1, blocking as needed.
// Returns io.ErrUnexpectedEOF if the stream ends after some bytes have been
// read; surfaces the underlying error if it errored before any bytes flowed.
func (cs *ClientStream) ReadFully(arg0 []byte, arg1, arg2 int) error {
	if arg2 == 0 {
		return nil
	}
	if cs.closed.Load() {
		return nil
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()

	readSome := false
	// Java: socket.setSoTimeout(30000) bounds each individual in.read() inside
	// readFully. The timer is armed only while stalled (ring empty) and
	// disarmed as soon as a chunk flows, so a packet that trickles in keeps
	// refreshing the window — only a genuine 30s stall yields ErrReadTimeout.
	var deadline time.Time
	var timer *time.Timer
	disarm := func() {
		if timer != nil {
			timer.Stop()
			timer = nil
		}
	}
	defer disarm()
	for arg2 > 0 {
		if cs.availableLocked() == 0 {
			if cs.rErr != nil {
				if readSome {
					return io.ErrUnexpectedEOF
				}
				return cs.rErr
			}
			if timer == nil {
				deadline = time.Now().Add(cs.readTimeout)
				timer = time.AfterFunc(cs.readTimeout, cs.broadcast)
			}
			if !time.Now().Before(deadline) {
				return ErrReadTimeout
			}
			cs.cond.Wait()
			continue
		}
		disarm() // progress is imminent — reset the per-read SO_TIMEOUT window
		// Copy a contiguous run from rbuf[rLen:] up to rPos or the end of
		// the ring, capped at arg2.
		var chunk int
		if cs.rPos > cs.rLen {
			chunk = cs.rPos - cs.rLen
		} else {
			chunk = readBufSize - cs.rLen
		}
		chunk = min(chunk, arg2)
		copy(arg0[arg1:arg1+chunk], cs.rbuf[cs.rLen:cs.rLen+chunk])
		cs.rLen = (cs.rLen + chunk) % readBufSize
		arg1 += chunk
		arg2 -= chunk
		readSome = true
		cs.cond.Broadcast() // unblock the reader if it was waiting on a full ring
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
	// Broadcast — not Signal — because cs.cond is shared with the reader
	// goroutine and consumers. Signal could wake the wrong sleeper, leaving
	// the writer blocked while data sits unsent.
	cs.cond.Broadcast()
	return nil
}

// run is the writer goroutine. Drains the outbound ring buffer to the
// underlying connection, then waits on cond when the buffer is empty.
// Started lazily by Write on first call; exits when Close flips
// writer=false and broadcasts.
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

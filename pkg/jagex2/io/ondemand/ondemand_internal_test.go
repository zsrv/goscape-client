package ondemand

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"net"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/zsrv/goscape-client/pkg/sign/signlink"
)

// fakeArchiveInternal satisfies the Archive interface for internal tests.
type fakeArchiveInternal map[string][]byte

func (f fakeArchiveInternal) Read(name string, _ []byte) []byte {
	return f[name]
}

// buildMinimalVersionlist returns a fakeArchiveInternal with one model entry
// for version/crc at index 0, and minimal valid tables for the remaining
// archives so that Unpack does not panic.
func buildMinimalVersionlist(version, wantCRC int) fakeArchiveInternal {
	p2 := func(vals ...int) []byte {
		buf := make([]byte, len(vals)*2)
		for i, v := range vals {
			binary.BigEndian.PutUint16(buf[i*2:], uint16(v))
		}
		return buf
	}
	p4 := func(vals ...int) []byte {
		buf := make([]byte, len(vals)*4)
		for i, v := range vals {
			binary.BigEndian.PutUint32(buf[i*4:], uint32(v))
		}
		return buf
	}
	p1 := func(vals ...int) []byte {
		buf := make([]byte, len(vals))
		for i, v := range vals {
			buf[i] = byte(v)
		}
		return buf
	}

	a := fakeArchiveInternal{}

	// model archive (index 0): one file
	a["model_version"] = p2(version)
	a["model_crc"] = p4(wantCRC)
	a["model_index"] = p1(0)

	// anim archive (index 1): one file, minimal
	a["anim_version"] = p2(0)
	a["anim_crc"] = p4(0)

	// midi archive (index 2): one file, minimal
	a["midi_version"] = p2(0)
	a["midi_crc"] = p4(0)

	// map archive (index 3): one file, minimal
	a["map_version"] = p2(0)
	a["map_crc"] = p4(0)

	// map_index: 0 records (empty)
	a["map_index"] = []byte{}

	// anim_index: 0 entries (empty)
	a["anim_index"] = []byte{}

	// midi_index: 0 entries (empty)
	a["midi_index"] = []byte{}

	return a
}

// TestUnpack_ParsedCRCValidates checks that Unpack stores the parsed version and
// CRC correctly, and that Validate accepts a payload whose trailer and CRC match
// the stored values — exercising the parsed-CRC path end to end.
func TestUnpack_ParsedCRCValidates(t *testing.T) {
	payload := []byte{1, 2, 3, 4, 5}
	version := 7
	wantCRC := int(int32(crc32.ChecksumIEEE(payload)))

	fake := buildMinimalVersionlist(version, wantCRC)
	od := New(fake, nil, nil)

	if got := od.versions[0][0]; got != version {
		t.Fatalf("od.versions[0][0] = %d, want %d", got, version)
	}
	if got := od.crcs[0][0]; got != wantCRC {
		t.Fatalf("od.crcs[0][0] = %d, want %d", got, wantCRC)
	}

	src := append(append([]byte{}, payload...), byte(version>>8), byte(version))
	if !Validate(src, od.crcs[0][0], od.versions[0][0]) {
		t.Fatal("Validate returned false for payload whose CRC and version match the parsed tables")
	}
}

// ---- state-machine tests ----------------------------------------------------

// buildModelVersionlist returns a versionlist with `count` model files, all
// with the given version and crc (so files 0..count-1 are valid for Request),
// plus minimal empty tables for the other archives.
func buildModelVersionlist(count, version, crc int) fakeArchiveInternal {
	p2 := func(vals ...int) []byte {
		buf := make([]byte, len(vals)*2)
		for i, v := range vals {
			binary.BigEndian.PutUint16(buf[i*2:], uint16(v))
		}
		return buf
	}
	p4 := func(vals ...int) []byte {
		buf := make([]byte, len(vals)*4)
		for i, v := range vals {
			binary.BigEndian.PutUint32(buf[i*4:], uint32(v))
		}
		return buf
	}

	versions := make([]int, count)
	crcs := make([]int, count)
	for i := range count {
		versions[i] = version
		crcs[i] = crc
	}

	a := fakeArchiveInternal{}
	a["model_version"] = p2(versions...)
	a["model_crc"] = p4(crcs...)
	a["model_index"] = make([]byte, count)

	a["anim_version"] = p2(0)
	a["anim_crc"] = p4(0)
	a["midi_version"] = p2(0)
	a["midi_crc"] = p4(0)
	a["map_version"] = p2(0)
	a["map_crc"] = p4(0)
	a["map_index"] = []byte{}
	a["anim_index"] = []byte{}
	a["midi_index"] = []byte{}
	return a
}

// gzipTrailer gzips payload and appends a 2-byte version trailer, mirroring the
// on-the-wire layout that Cycle()/loop() strips and decodes.
func gzipTrailer(t *testing.T, payload []byte, version int) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(payload); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	out := buf.Bytes()
	return append(out, byte(version>>8), byte(version))
}

// TestRequest_Dedup verifies that requesting the same (archive, file) twice
// only enqueues a single in-flight request.
func TestRequest_Dedup(t *testing.T) {
	od := New(buildModelVersionlist(10, 7, 0), nil, nil)

	od.Request(0, 5)
	od.Request(0, 5)

	if got := od.Remaining(); got != 1 {
		t.Fatalf("Remaining() = %d after duplicate Request(0,5), want 1", got)
	}
}

// TestCycle_GunzipTrailer verifies that Cycle() strips the 2-byte trailer and
// gunzips a completed request's data back to the original payload.
func TestCycle_GunzipTrailer(t *testing.T) {
	od := New(buildModelVersionlist(10, 7, 0), nil, nil)

	payload := []byte("hello on-demand payload")
	r := newRequest()
	r.Archive = 0
	r.File = 1
	r.Data = gzipTrailer(t, payload, 7)
	// Place the request in both lists the way the real pipeline would: it is in
	// requests (LinkList2) and completed (LinkList).
	od.requests.Push(r.node)
	od.completed.Push(r.node.Linkable)

	got := od.Cycle()
	if got == nil {
		t.Fatal("Cycle() returned nil, want the completed request")
	}
	if !bytes.Equal(got.Data, payload) {
		t.Fatalf("Cycle() decoded Data = %q, want %q", got.Data, payload)
	}
	// The request must have been unlinked from the requests list.
	if got := od.Remaining(); got != 0 {
		t.Fatalf("Remaining() = %d after Cycle(), want 0", got)
	}
}

// ---- socket transport test harness ------------------------------------------

// fakeApp satisfies App over a pre-seeded queue of net.Pipe client ends.
// An empty queue makes OpenSocket fail, like an unreachable world server.
type fakeApp struct {
	mu     sync.Mutex
	conns  []net.Conn
	ingame bool
}

func (a *fakeApp) OpenSocket() (net.Conn, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.conns) == 0 {
		return nil, errors.New("fakeApp: no conn available")
	}
	c := a.conns[0]
	a.conns = a.conns[1:]
	return c, nil
}

func (a *fakeApp) InGame() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ingame
}

// odServer drives the server side of one ondemand connection: it validates
// the 15 handshake byte, replies with 8 zero bytes, then records every
// 4-byte request frame it receives (including keepalives).
type odServer struct {
	conn net.Conn
	mu   sync.Mutex
	reqs [][4]byte
}

// startODServer returns a running server and the client end of the pipe.
func startODServer(t *testing.T) (*odServer, net.Conn) {
	t.Helper()
	server, client := net.Pipe()
	t.Cleanup(func() {
		_ = server.Close()
		_ = client.Close()
	})
	s := &odServer{conn: server}
	go s.run()
	return s, client
}

func (s *odServer) run() {
	one := make([]byte, 1)
	if _, err := io.ReadFull(s.conn, one); err != nil || one[0] != 15 {
		return
	}
	if _, err := s.conn.Write(make([]byte, 8)); err != nil {
		return
	}
	for {
		var req [4]byte
		if _, err := io.ReadFull(s.conn, req[:]); err != nil {
			return
		}
		s.mu.Lock()
		s.reqs = append(s.reqs, req)
		s.mu.Unlock()
	}
}

func (s *odServer) requests() [][4]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.reqs)
}

// respond writes one response part: the 6-byte header followed by the chunk.
// size is the TOTAL file size; part selects the 500-byte window.
func (s *odServer) respond(t *testing.T, archive, file, size, part int, chunk []byte) {
	t.Helper()
	hdr := []byte{byte(archive), byte(file >> 8), byte(file), byte(size >> 8), byte(size), byte(part)}
	if _, err := s.conn.Write(hdr); err != nil {
		t.Fatalf("respond header: %v", err)
	}
	if len(chunk) > 0 {
		if _, err := s.conn.Write(chunk); err != nil {
			t.Fatalf("respond chunk: %v", err)
		}
	}
}

// waitFor polls cond (driving step each iteration) until it holds or 5s pass.
func waitFor(t *testing.T, step func(), cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("waitFor: condition not met within 5s")
		}
		step()
		time.Sleep(time.Millisecond)
	}
}

// connectOD drives od.send(probe) until the async dial+handshake completes
// and the stream is attached. The probe request's bytes reach the server
// (possibly more than once); tests must account for them or use a distinct
// file id for assertions.
func connectOD(t *testing.T, od *OnDemand, probe *OnDemandRequest) {
	t.Helper()
	waitFor(t, func() { od.send(probe) }, func() bool { return od.stream != nil })
}

// ---- send() ------------------------------------------------------------------

func TestSend_HandshakeAndUrgentRequestBytes(t *testing.T) {
	server, conn := startODServer(t)
	app := &fakeApp{conns: []net.Conn{conn}}
	od := New(buildMinimalVersionlist(1, 0), app, nil)

	r := newRequest()
	r.Archive = 0
	r.File = 1
	r.Urgent = true
	connectOD(t, od, r)

	// The server's run() already validated the 15 handshake byte (it records
	// nothing otherwise). Now the request frame: [archive, file>>8, file, 2].
	waitFor(t, func() { od.send(r) }, func() bool { return len(server.requests()) >= 1 })
	want := [4]byte{0, 0, 1, 2}
	if got := server.requests()[0]; got != want {
		t.Fatalf("urgent request bytes = %v, want %v", got, want)
	}
	if od.FailCount != -10000 {
		t.Fatalf("FailCount = %d, want -10000 after successful send", od.FailCount)
	}
}

func TestSend_PriorityByteNotUrgent(t *testing.T) {
	server, conn := startODServer(t)
	app := &fakeApp{conns: []net.Conn{conn}} // ingame=false → priority 1
	od := New(buildMinimalVersionlist(1, 0), app, nil)

	r := newRequest()
	r.Archive = 2
	r.File = 3
	r.Urgent = false
	connectOD(t, od, r)
	waitFor(t, func() { od.send(r) }, func() bool { return len(server.requests()) >= 1 })

	want := [4]byte{2, 0, 3, 1} // Java: !urgent && !app.ingame → buf[3] = 1
	if got := server.requests()[0]; got != want {
		t.Fatalf("pre-game request bytes = %v, want %v", got, want)
	}
}

func TestSend_DialFailureIncrementsFailCount(t *testing.T) {
	app := &fakeApp{} // no conns → OpenSocket errors
	od := New(buildMinimalVersionlist(1, 0), app, nil)

	r := newRequest()
	r.Archive = 0
	r.File = 1
	// First send kicks the connector; subsequent sends poll its failure.
	waitFor(t, func() { od.send(r) }, func() bool { return od.FailCount >= 1 })
	if od.stream != nil {
		t.Fatal("stream attached despite dial failure")
	}
}

// ---- read() ------------------------------------------------------------------

// pushPending crafts a request directly onto the pending list, as
// handlePending would after a cache miss.
func pushPending(od *OnDemand, archive, file int, urgent bool) *OnDemandRequest {
	r := newRequest()
	r.Archive = archive
	r.File = file
	r.Urgent = urgent
	od.pending.Push(r.node.Linkable)
	return r
}

// newConnectedOD returns an OnDemand with an attached stream and its server.
func newConnectedOD(t *testing.T) (*OnDemand, *odServer) {
	t.Helper()
	server, conn := startODServer(t)
	app := &fakeApp{conns: []net.Conn{conn}}
	od := New(buildMinimalVersionlist(1, 0), app, nil)
	probe := newRequest()
	probe.Archive = 0
	probe.File = 0
	connectOD(t, od, probe)
	return od, server
}

func TestRead_SinglePartCompletion(t *testing.T) {
	od, server := newConnectedOD(t)
	r := pushPending(od, 1, 7, true)

	payload := bytes.Repeat([]byte{0xAB}, 300)
	server.respond(t, 1, 7, 300, 0, payload)

	waitFor(t, func() { od.Run() }, func() bool { return od.completed.Head() != nil })
	if !bytes.Equal(r.Data, payload) {
		t.Fatalf("reassembled %d bytes, want 300 matching payload", len(r.Data))
	}
	if got := od.completed.Head().Value; got != r {
		t.Fatalf("completed head = %+v, want the pending request", got)
	}
}

func TestRead_MultiPartReassembly(t *testing.T) {
	od, server := newConnectedOD(t)
	r := pushPending(od, 0, 9, true)

	payload := make([]byte, 700)
	for i := range payload {
		payload[i] = byte(i * 31)
	}
	server.respond(t, 0, 9, 700, 0, payload[:500])
	server.respond(t, 0, 9, 700, 1, payload[500:])

	waitFor(t, func() { od.Run() }, func() bool { return od.completed.Head() != nil })
	if !bytes.Equal(r.Data, payload) {
		t.Fatal("multi-part reassembly mismatch")
	}
}

func TestRead_RejectionDeliversNilData(t *testing.T) {
	// The rejection path calls signlink.ReportErrorFunc, which defaults to
	// enabled (signlink.go:79) and would block forever on OpenURL's
	// cond.Wait — the signlink polling goroutine (StartPriv) is not running
	// in unit tests. Disable it for this test.
	old := signlink.ReportError
	signlink.ReportError = false
	t.Cleanup(func() { signlink.ReportError = old })

	od, server := newConnectedOD(t)
	r := pushPending(od, 2, 4, true)

	server.respond(t, 2, 4, 0, 0, nil) // size 0 = server rejection

	waitFor(t, func() { od.Run() }, func() bool { return od.completed.Head() != nil })
	if r.Data != nil {
		t.Fatalf("rejected request carries %d bytes, want nil", len(r.Data))
	}
}

func TestRead_MissingStartOfFileTearsDownStream(t *testing.T) {
	od, server := newConnectedOD(t)
	pushPending(od, 0, 5, true)

	// part 1 with no prior part 0 → Java throws IOException("missing start
	// of file") → its catch closes the socket.
	server.respond(t, 0, 5, 700, 1, bytes.Repeat([]byte{1}, 200))

	waitFor(t, func() { od.Run() }, func() bool { return od.stream == nil })
}

func TestRead_Archive3PromotedTo93(t *testing.T) {
	od, server := newConnectedOD(t)
	r := pushPending(od, 3, 6, false) // non-urgent map fetch

	server.respond(t, 3, 6, 100, 0, bytes.Repeat([]byte{7}, 100))

	waitFor(t, func() { od.Run() }, func() bool { return od.completed.Head() != nil })
	if r.Archive != 93 || !r.Urgent {
		t.Fatalf("archive=%d urgent=%v, want 93/true promotion", r.Archive, r.Urgent)
	}
}

func TestRead_OrphanResponseDrainsToScratch(t *testing.T) {
	od, server := newConnectedOD(t)

	// Response for a (archive, file) that is not pending: header parsed,
	// part drained into the scratch buffer, nothing completed, stream alive.
	server.respond(t, 1, 99, 50, 0, bytes.Repeat([]byte{3}, 50))

	// Then a real request still completes over the same connection.
	r := pushPending(od, 1, 7, true)
	payload := bytes.Repeat([]byte{0xCD}, 80)
	server.respond(t, 1, 7, 80, 0, payload)

	waitFor(t, func() { od.Run() }, func() bool { return od.completed.Head() != nil })
	if od.stream == nil {
		t.Fatal("stream torn down by orphan response")
	}
	if !bytes.Equal(r.Data, payload) {
		t.Fatal("real request corrupted by orphan response")
	}
}

func TestRead_GrowingSizeTearsDownStream(t *testing.T) {
	od, server := newConnectedOD(t)
	pushPending(od, 0, 5, true)

	// part 0 allocates Data at size 700; a malformed part 1 then claims
	// size 1300, putting partOffset+partAvailable past len(Data). Java's
	// AIOOBE killed only its worker thread; the Go port must tear down the
	// connection instead of panicking.
	server.respond(t, 0, 5, 700, 0, bytes.Repeat([]byte{1}, 500))
	server.respond(t, 0, 5, 1300, 1, bytes.Repeat([]byte{2}, 500))

	waitFor(t, func() { od.Run() }, func() bool { return od.stream == nil })
}

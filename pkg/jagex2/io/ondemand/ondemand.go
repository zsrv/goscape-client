// Package ondemand ports Java's jagex2.io.OnDemand.
// Types, versionlist parse, Validate, getters, the request/cycle/prefetch
// state machine, and the socket transport (send / read part-reassembly) all
// live here.
//
// Transport: the genuine Java 274 socket protocol (OnDemand.java @32f3062) —
// handshake byte 15 on the world port, 4-byte requests, 6-byte response
// headers with 500-byte part reassembly. The pre-274 "modernized"
// /ondemand.zip bundle shim (a Client-TS-era convention served by Engine-TS
// ≤254) was removed when Engine-TS 274 dropped the route in favour of the
// real protocol.
//
// Threading: Java runs OnDemand on a worker thread (startThread(this, 2),
// OnDemand.java:216) sleeping 20|50 ms between pump iterations; this port
// drives Run() once per game frame instead (established WS1 decision), so
// Java's synchronized blocks are dropped — all state lives on the game-loop
// goroutine. Two exceptions: clientstream's internal reader/writer
// goroutines (own only stream internals), and the one-shot dial+handshake
// connector goroutine (see send), which would otherwise stall the frame.
package ondemand

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"hash/crc32"
	"io"
	"net"
	"time"

	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct"
	jio "github.com/zsrv/goscape-client/pkg/jagex2/io"
	"github.com/zsrv/goscape-client/pkg/jagex2/io/clientstream"
)

// OnDemandRequest mirrors Java's jagex2.io.OnDemandRequest (extends Linkable2).
// Java: nb (obfuscated class name).
//
// In Java the class inherits two link-field pairs (next/prev from Linkable,
// next2/prev2 from Linkable2) which allow a single request to sit in
// both a LinkList2 (requests) and a LinkList (queue/missing/pending/
// completed/prefetches) simultaneously.
//
// In Go we hold one *datastruct.Linkable2[*OnDemandRequest] whose
// embedded *Linkable[*OnDemandRequest] provides the second link pair:
//   - r.node        → LinkList2 slot (next2/prev2 via Uncache/Push/Pop)
//   - r.node.Linkable → LinkList slot     (next/prev  via Unlink/PushFront/…)
type OnDemandRequest struct {
	// Java: nb.i
	Archive int
	// Java: nb.j
	File int
	// Java: nb.k
	Data []byte
	// Java: nb.l — frames since this pending request was last (re)sent; the
	// Run() resend walk re-sends and zeroes it past 50 (OnDemand.java:406-425
	// @32f3062), and read() zeroes it on any response at-or-after this
	// request in the pending list (OnDemand.java:599-606).
	Cycle int
	// Java: nb.m
	Urgent bool

	// node provides the two link-field pairs that Java inherits from
	// Linkable2 -> Linkable:
	//   - r.node             → LinkList2 slot (requests; next2/prev2 via
	//                          Push/Pop/Uncache)
	//   - r.node.Linkable    → LinkList slot (queue/missing/pending/completed/
	//                          prefetches; next/prev via Push/PopFront/Unlink)
	node *datastruct.Linkable2[*OnDemandRequest]
}

// newRequest allocates a request with the Java default Urgent=true.
func newRequest() *OnDemandRequest {
	r := &OnDemandRequest{Urgent: true}
	r.node = datastruct.NewLinkable2(r)
	return r
}

// ---- seam interfaces -------------------------------------------------------

// App is the surface OnDemand needs from the client (Java: ub.q app —
// init() receives the Client itself; only these two members are read by
// the transport).
type App interface {
	// OpenSocket dials the world server for the ondemand service.
	// Java: app.openSocket(Client.portOff + 43594) (OnDemand.java:700
	// @32f3062) — portOff+43594 is the game port.
	OpenSocket() (net.Conn, error)
	// InGame mirrors Java app.ingame: read by send()'s priority byte and
	// the Run() keepalive gate.
	InGame() bool
}

// Cache is the per-file persistent store.
// Client wires the storage seam here; nil = in-memory only.
type Cache interface {
	Read(archive, file int) []byte
	Write(archive, file int, data []byte)
}

// Archive is the versionlist source.
// *io.JagFile satisfies it structurally.
// Java: OnDemand.unpack(JagFile, Client); taken as a reader so the parse is unit-testable.
type Archive interface {
	Read(name string, dst []byte) []byte
}

// *OnDemand is the model loader's on-demand provider.
// Java: OnDemand extends OnDemandProvider.
var _ jio.OnDemandProvider = (*OnDemand)(nil)

// ---- OnDemand struct -------------------------------------------------------

// OnDemand ports Java jagex2.io.OnDemand (obfuscated: vb).
// Fields, the request/cycle/prefetch/run/read state machine, and the modernized
// read path all live in this file.
type OnDemand struct {
	// Java: vb.h — per-archive file version tables (4 archives)
	versions [4][]int
	// Java: vb.i — per-archive file CRC tables (4 archives)
	crcs [4][]int
	// Java: vb.j — per-archive per-file download priority (byte[][], signed)
	// — written in Unpack, read by the prefetch logic
	priorities [4][]int8
	// Java: vb.k
	topPriority int

	// Java: vb.l — model index flags (byte[], masked & 0xFF on read)
	models []byte

	// Java: vb.m/n/o/p — map coordinate and member flag tables
	mapIndex   []int
	mapLand    []int
	mapLoc     []int
	mapMembers []int

	// Java: vb.q — anim (seq) file index
	animIndex []int
	// Java: vb.r — midi prefetch flag (1 = prefetch)
	midiIndex []int

	// Java: vb.s
	running bool

	// active mirrors Client-TS OnDemand.active — set whenever the pump did
	// real work in a Run() iteration so the loop keeps spinning.
	active bool
	// cycle mirrors Client-TS OnDemand.cycle (incremented once per Run()).
	// Client-TS also only increments this; intentionally write-only in the modernized path (no reader).
	cycle int
	// importantCount/requestCount mirror Client-TS — urgent vs non-urgent
	// pending counters recomputed each handlePending().
	importantCount int
	requestCount   int
	// Java: ub.I — the pending request read() is currently reassembling.
	current *OnDemandRequest

	// ---- request queues (Java: vb.x/y/z/A/B/C) ----------------------------
	// requests holds all in-flight OnDemandRequests in a doubly-linked list
	// so they can be found by (archive, file) and removed in O(1).
	requests *datastruct.LinkList2[*OnDemandRequest]
	// queue: incoming urgent requests waiting to be dispatched.
	queue *datastruct.LinkList[*OnDemandRequest]
	// missing: requests not satisfied from local cache; need network fetch.
	missing *datastruct.LinkList[*OnDemandRequest]
	// pending: requests sent to the server, awaiting reply.
	pending *datastruct.LinkList[*OnDemandRequest]
	// completed: requests whose data has arrived; ready for the client.
	completed *datastruct.LinkList[*OnDemandRequest]
	// prefetches: low-priority background prefetch requests.
	prefetches *datastruct.LinkList[*OnDemandRequest]

	// Java: vb.E/F
	loadedPrefetchFiles int
	totalPrefetchFiles  int

	// Java: vb.D
	message string

	// FailCount counts consecutive failed transport attempts; Client.load's
	// first two on-demand wait loops bail to showLoadError("ondemand") when
	// it exceeds 3. Java: failCount (OnDemand.java:105 @32f3062, NEW in 274)
	// — set to -10000 after a successful request write (OnDemand.java:721)
	// and incremented in send()'s IOException catch (:731), which here also
	// covers a failed dial or handshake.
	FailCount int

	// ---- socket transport state (Java: ub.E/F/G/J/K/L/N/O/P) ---------------

	// Java: ub.L — 500-byte wire scratch (request frames, response headers,
	// orphaned parts whose request is no longer pending)
	buf [500]byte
	// Java: ub.J — byte offset of the current part within current.Data
	partOffset int
	// Java: ub.K — bytes of the current part still unread (0 = expecting a
	// 6-byte header next)
	partAvailable int
	// Java: ub.N — frames the pending list has gone unanswered; > 750 tears
	// the connection down so send() redials
	packetCycle int
	// Java: ub.O — frames since the last request write; > 500 emits the
	// keepalive frame while in game
	noTimeoutCycle int
	// Java: ub.P — wall-clock ms of the last dial attempt; enforces the 4 s
	// redial backoff (274 tightened 5000→4000, OnDemand.java:697)
	socketOpenTime int64
	// stream wraps the ondemand socket. Java holds the raw Socket plus its
	// two streams (ub.E/F/G); the Go port reuses clientstream.ClientStream
	// because it reproduces InputStream.available()'s eager-buffering
	// semantics over a net.Conn (see that package's doc) with the same
	// read/readFully/write surface. nil = not connected (Java: socket==null).
	stream *clientstream.ClientStream
	// connecting is non-nil while the one-shot dial+handshake goroutine is
	// in flight; it delivers exactly one result, polled by send(). Java
	// performs the open synchronously on its worker thread
	// (OnDemand.java:696-707); this port is frame-driven on the game-loop
	// goroutine, so the blocking dial + 8-byte drain must not run inline.
	connecting chan connectResult

	// ---- injected seams (no client/* imports) ------------------------------
	app   App
	cache Cache // may be nil
}

// ---- constructor -----------------------------------------------------------

// New allocates an OnDemand, wires seams, and calls Unpack.
// Java: OnDemand constructor + init() (OnDemand.java:133-217 @32f3062).
// init()'s trailing startThread call is not ported — Run() is driven once
// per frame instead (see the package doc).
func New(versionlist Archive, app App, cache Cache) *OnDemand {
	od := &OnDemand{
		requests:   datastruct.NewLinkList2[*OnDemandRequest](),
		queue:      datastruct.NewLinkList[*OnDemandRequest](),
		missing:    datastruct.NewLinkList[*OnDemandRequest](),
		pending:    datastruct.NewLinkList[*OnDemandRequest](),
		completed:  datastruct.NewLinkList[*OnDemandRequest](),
		prefetches: datastruct.NewLinkList[*OnDemandRequest](),
		running:    true,
		app:        app,
		cache:      cache,
	}
	od.Unpack(versionlist)
	return od
}

// ---- versionlist parse -----------------------------------------------------

// Unpack reads all version/crc/index tables from the versionlist JagFile.
// Java: OnDemand.unpack(JagFile, Client) (vb.a(Lyb;Lclient;)V), lines 135–216.
// The trailing startThread call is not ported — Run() is driven once per frame instead.
func (od *OnDemand) Unpack(versionlist Archive) {
	// model_version / anim_version / midi_version / map_version
	// Java: for i 0..3 — count = data.length/2; g2 each
	vnames := [4]string{"model_version", "anim_version", "midi_version", "map_version"}
	for i := range 4 {
		data := versionlist.Read(vnames[i], nil)
		count := len(data) / 2
		buf := jio.NewPacket(data)

		od.versions[i] = make([]int, count)
		od.priorities[i] = make([]int8, count)

		for j := range count {
			od.versions[i][j] = buf.G2()
		}
	}

	// model_crc / anim_crc / midi_crc / map_crc
	// Java: for i 0..3 — count = data.length/4; g4 each
	cnames := [4]string{"model_crc", "anim_crc", "midi_crc", "map_crc"}
	for i := range 4 {
		data := versionlist.Read(cnames[i], nil)
		count := len(data) / 4
		buf := jio.NewPacket(data)

		od.crcs[i] = make([]int, count)

		for j := range count {
			// Java: g4() returns signed int32 — keeps the table comparable with
			// Validate's (int) CRC32.getValue() narrowing (audit ondemand-07)
			od.crcs[i][j] = int(int32(buf.G4()))
		}
	}

	// model_index: count = len(versions[0]); pad with 0 if data shorter
	// Java: models is byte[]; GetModelFlags masks & 0xFF
	{
		data := versionlist.Read("model_index", nil)
		count := len(od.versions[0])
		od.models = make([]byte, count)
		for i := range count {
			if i < len(data) {
				od.models[i] = data[i]
			} else {
				od.models[i] = 0
			}
		}
	}

	// map_index: count = len/7; per record g2 mapIndex, g2 mapLand, g2 mapLoc, g1 mapMembers
	{
		data := versionlist.Read("map_index", nil)
		buf := jio.NewPacket(data)
		count := len(data) / 7

		od.mapIndex = make([]int, count)
		od.mapLand = make([]int, count)
		od.mapLoc = make([]int, count)
		od.mapMembers = make([]int, count)

		for i := range count {
			od.mapIndex[i] = buf.G2()
			od.mapLand[i] = buf.G2()
			od.mapLoc[i] = buf.G2()
			od.mapMembers[i] = buf.G1()
		}
	}

	// anim_index: count = len/2; g2 each
	{
		data := versionlist.Read("anim_index", nil)
		buf := jio.NewPacket(data)
		count := len(data) / 2

		od.animIndex = make([]int, count)
		for i := range count {
			od.animIndex[i] = buf.G2()
		}
	}

	// midi_index: count = len; g1 each
	{
		data := versionlist.Read("midi_index", nil)
		buf := jio.NewPacket(data)
		count := len(data)

		od.midiIndex = make([]int, count)
		for i := range count {
			od.midiIndex[i] = buf.G1()
		}
	}
}

// ---- getters ---------------------------------------------------------------

// HasCache reports whether a persistent cache is available.
// Java: fileStreams[0] != null cache-presence gate (Client.java:~1662).
func (od *OnDemand) HasCache() bool {
	return od.cache != nil
}

// GetFileCount returns the number of files in archive.
// Java: OnDemand.getFileCount(int) (vb.a(II)I), ~line 223.
func (od *OnDemand) GetFileCount(archive int) int {
	return len(od.versions[archive])
}

// GetAnimCount returns the number of animation (seq) files.
// Java: OnDemand.getAnimCount() (vb.a(B)I), ~line 227.
func (od *OnDemand) GetAnimCount() int {
	return len(od.animIndex)
}

// GetMapFile returns the land (type==0) or loc (type!=0) file id for tile (z,x),
// or -1 if not found.
// Java: OnDemand.getMapFile(z,x,type) (vb.a(IIII)I), ~line 231.
// Java: getMapFile(z,x,type) — formula (x<<8)+z; callers pass values, not labels.
func (od *OnDemand) GetMapFile(z, x, type_ int) int {
	// Java: int map = (x << 8) + z;
	mapID := (x << 8) + z
	for i := range len(od.mapIndex) {
		if od.mapIndex[i] == mapID {
			if type_ == 0 {
				return od.mapLand[i]
			}
			return od.mapLoc[i]
		}
	}
	return -1
}

// HasMapLocFile reports whether any entry in the map table has the given loc file id.
// Java: OnDemand.hasMapLocFile(int) (vb.b(II)Z), ~line 248.
func (od *OnDemand) HasMapLocFile(file int) bool {
	for i := range len(od.mapIndex) {
		if od.mapLoc[i] == file {
			return true
		}
	}
	return false
}

// GetModelFlags returns the model index byte for model id, masked to unsigned.
// Java: OnDemand.getModelFlags(int) (vb.c(II)I), ~line 258.
// Java: return this.models[id] & 0xFF;
func (od *OnDemand) GetModelFlags(id int) int {
	return int(od.models[id]) & 0xFF
}

// ShouldPrefetchMidi reports whether the midi at id should be prefetched.
// Java: OnDemand.shouldPrefetchMidi(int) (vb.d(II)Z), ~line 263.
func (od *OnDemand) ShouldPrefetchMidi(id int) bool {
	return od.midiIndex[id] == 1
}

// RequestModel satisfies the io.OnDemandProvider interface.
// Java: OnDemand.requestModel(int) (vb.a(I)V), ~line 267.
// Client-TS: requestModel(id) { this.request(0, id); }
func (od *OnDemand) RequestModel(id int) {
	od.Request(0, id)
}

// ---- Validate --------------------------------------------------------------

// Validate checks whether src carries the expected CRC and version trailer.
// Java: OnDemand.validate(byte[],int,int) (vb.a([BIII)Z), ~lines 775–789.
//
// The last two bytes of src encode the version as big-endian uint16.
// The CRC is computed over src[:len(src)-2] using the IEEE polynomial
// (== java.util.zip.CRC32), then narrowed to int32 to match Java's
// (int) crc32.getValue() cast.
func Validate(src []byte, expectedCrc, expectedVersion int) bool {
	if len(src) < 2 { // nil slice has len 0; Java: src == null || src.length < 2
		return false
	}
	tp := len(src) - 2
	version := (int(src[tp])&0xFF)<<8 + int(src[tp+1])&0xFF
	crc := int(int32(crc32.ChecksumIEEE(src[:tp]))) // Java: (int) CRC32.getValue()
	return expectedVersion == version && expectedCrc == crc
}

// ---- request / cycle / prefetch state machine ------------------------------

// Request enqueues an urgent fetch for (archive, file), deduping against
// requests already in flight.
// Java: OnDemand.request(int,int) (vb.e(II)V), lines 287–314.
// Client-TS: request(archive, file).
//
// The bounds use Java's `>` (not `>=`) comparisons verbatim; the off-by-one is
// a faithful carry-over from the original and is never hit by valid callers
// (archive 0–3, file < len). The synchronized blocks are dropped: OnDemand runs
// on the single game-loop goroutine in this port.
func (od *OnDemand) Request(archive, file int) {
	if archive < 0 || archive > len(od.versions) || file < 0 || file > len(od.versions[archive]) || od.versions[archive][file] == 0 {
		return
	}

	for n := od.requests.Head(); n != nil; n = od.requests.Next() {
		if n.Value.Archive == archive && n.Value.File == file {
			return
		}
	}

	r := newRequest()
	r.Archive = archive
	r.File = file
	r.Urgent = true

	od.queue.Push(r.node.Linkable)
	od.requests.Push(r.node)
}

// Remaining returns the number of in-flight requests.
// Java: OnDemand.remaining() (vb.b()I), lines 316–321.
func (od *OnDemand) Remaining() int {
	return od.requests.Size()
}

// Message returns the loader status line shown on the welcome title screen
// ("Loading extra files - N%"). Java reads the OnDemand.message field
// directly (OnDemand.java:90, Client.java:5485); the Go field is unexported.
func (od *OnDemand) Message() string {
	return od.message
}

// Cycle pops the next completed request, unlinks it from the requests list, and
// (if it carries data) strips the 2-byte version trailer and gunzips it.
// Java: OnDemand.cycle() (vb.c()Lnb;), lines 323–369.
// Client-TS: loop().
//
// Client-TS slices off the last 2 bytes before gunzipSync; we do the same. Java
// gzip-decodes the whole buffer (GZIPInputStream stops at the gzip trailer and
// ignores the extra 2 bytes), which yields the same payload.
func (od *OnDemand) Cycle() *OnDemandRequest {
	n := od.completed.PopFront()
	if n == nil {
		return nil
	}

	r := n.Value
	r.node.Uncache() // Java: req.unlink2() — drop from the requests LinkList2

	// len(nil) == 0, so this also covers the no-data case; a present-but-
	// truncated bundle entry (len 0/1) would panic the slice below — treat it
	// like the corrupt-entry path (audit ondemand-01).
	if len(r.Data) < 2 {
		r.Data = nil
		return r
	}

	gz, err := gzip.NewReader(bytes.NewReader(r.Data[:len(r.Data)-2]))
	if err != nil {
		// Java threw RuntimeException("error unzipping"); the modernized path
		// drops the corrupt entry instead of crashing the game loop.
		r.Data = nil
		return r
	}
	defer func() { _ = gz.Close() }()

	decoded, err := io.ReadAll(gz)
	if err != nil {
		r.Data = nil
		return r
	}
	r.Data = decoded
	return r
}

// PrefetchPriority marks (archive, file) for background prefetch at the given
// priority, unless it is already present and valid in the local cache.
// Java: OnDemand.prefetchPriority(int,int,byte) (vb.a(IZIB)V), lines 371–389.
// Client-TS: prefetchPriority(archive, file, priority).
//
// A nil cache (the bundle-only default) makes this a no-op, matching Java's
// app.fileStreams[0] == null guard.
func (od *OnDemand) PrefetchPriority(archive, file int, priority int8) {
	if od.cache == nil || od.versions[archive][file] == 0 {
		return
	}

	data := od.cache.Read(archive+1, file)
	if Validate(data, od.crcs[archive][file], od.versions[archive][file]) {
		return
	}

	od.priorities[archive][file] = priority
	if int(priority) > od.topPriority {
		od.topPriority = int(priority)
	}

	od.totalPrefetchFiles++
}

// ClearPrefetches empties the prefetch queue.
// Java: OnDemand.clearPrefetches() (vb.b(I)V), lines 391–396.
func (od *OnDemand) ClearPrefetches() {
	od.prefetches.Clear()
}

// Prefetch enqueues a non-urgent fetch for (archive, file).
// Java: OnDemand.prefetch(int,int) (vb.a(III)V), lines 398–413.
// Client-TS: prefetch(archive, file).
func (od *OnDemand) Prefetch(archive, file int) {
	if od.cache == nil || od.versions[archive][file] == 0 || od.priorities[archive][file] == 0 || od.topPriority == 0 {
		return
	}

	r := newRequest()
	r.Archive = archive
	r.File = file
	r.Urgent = false

	od.prefetches.Push(r.node.Linkable)
}

// PrefetchMaps queues prefetch priorities for every map land/loc file.
// Java: OnDemand.prefetchMaps(boolean) (vb.a(ZI)V), lines 250–259.
// Client-TS: prefetchMaps(members).
func (od *OnDemand) PrefetchMaps(members bool) {
	count := len(od.mapIndex)
	for i := range count {
		if members || od.mapMembers[i] != 0 {
			od.PrefetchPriority(3, od.mapLoc[i], 2)
			od.PrefetchPriority(3, od.mapLand[i], 2)
		}
	}
}

// Stop halts the Run() pump.
// Java: OnDemand.stop() / Client-TS: stop().
func (od *OnDemand) Stop() {
	od.running = false
}

// Run pumps the request state machine once. It is called once per game frame
// (≈20 ms); Java ran the same body on a worker thread sleeping 20|50 ms per
// iteration (OnDemand.java:380-460 @32f3062). The resend walk, packetCycle
// teardown, and keepalive tail (Java :406-456) land with a following commit;
// until then only the idle message clear from that tail is ported.
func (od *OnDemand) Run() {
	if !od.running {
		return
	}

	od.cycle++

	od.active = true

	for i := 0; i < 100 && od.active; i++ {
		od.active = false

		od.handleQueue()
		od.handlePending()

		if od.importantCount == 0 && i >= 5 {
			break
		}

		od.handleExtras()
		od.read()
	}

	// Java: OnDemand.java:405-441 — the resend walk sets var3 iff the pending
	// list is non-empty; the walk and its if half (packetCycle/socket reset)
	// land with a following commit, only the else half's message clear is here.
	if od.pending.Head() == nil {
		od.message = ""
	}
}

// handleQueue drains incoming requests, satisfying them from the local cache
// when possible (→ completed) or routing them to the missing list otherwise.
// Client-TS: handleQueue(). 274 drops 254's dead dummy-arg guard
// (handleQueue(int) `if (arg0 != 2) return` → handleQueue(),
// OnDemand.java:464 @32f3062) — this port never carried it, so the Go
// shape is already exact.
func (od *OnDemand) handleQueue() {
	for n := od.queue.PopFront(); n != nil; n = od.queue.PopFront() {
		od.active = true
		r := n.Value

		var data []byte
		if od.cache != nil {
			data = od.cache.Read(r.Archive+1, r.File)
		}

		if !Validate(data, od.crcs[r.Archive][r.File], od.versions[r.Archive][r.File]) {
			data = nil
		}

		if data == nil {
			od.missing.Push(r.node.Linkable)
		} else {
			r.Data = data
			od.completed.Push(r.node.Linkable)
		}
	}
}

// handlePending recomputes the urgent/non-urgent pending counts, then promotes
// missing requests into pending (sending each) until 10 urgent are in flight.
// Client-TS: handlePending().
func (od *OnDemand) handlePending() {
	od.importantCount = 0
	od.requestCount = 0

	for n := od.pending.Head(); n != nil; n = od.pending.Next() {
		if n.Value.Urgent {
			od.importantCount++
		} else {
			od.requestCount++
		}
	}

	for od.importantCount < 10 {
		n := od.missing.PopFront()
		if n == nil {
			break
		}
		r := n.Value

		if od.priorities[r.Archive][r.File] != 0 {
			od.loadedPrefetchFiles++
		}

		od.priorities[r.Archive][r.File] = 0
		od.pending.Push(r.node.Linkable)
		od.importantCount++
		od.send(r)
		od.active = true
	}
}

// handleExtras drains prefetch requests (and then a top-priority scan over the
// four archives) into pending while no urgent requests are in flight and fewer
// than 10 non-urgent requests are queued. It updates the progress message.
// Client-TS: handleExtras().
func (od *OnDemand) handleExtras() {
	for od.importantCount == 0 && od.requestCount < 10 {
		if od.topPriority == 0 {
			return
		}

		for n := od.prefetches.PopFront(); n != nil; n = od.prefetches.PopFront() {
			extra := n.Value
			if od.priorities[extra.Archive][extra.File] != 0 {
				od.priorities[extra.Archive][extra.File] = 0
				od.pending.Push(extra.node.Linkable)
				od.send(extra)
				od.active = true

				if od.loadedPrefetchFiles < od.totalPrefetchFiles {
					od.loadedPrefetchFiles++
				}

				od.message = fmt.Sprintf("Loading extra files - %d%%", od.loadedPrefetchFiles*100/od.totalPrefetchFiles)
				od.requestCount++

				if od.requestCount == 10 {
					return
				}
			}
		}

		for archive := range 4 {
			priorities := od.priorities[archive]
			count := len(priorities)

			for i := range count {
				if int(priorities[i]) == od.topPriority {
					priorities[i] = 0

					r := newRequest()
					r.Archive = archive
					r.File = i
					r.Urgent = false
					od.pending.Push(r.node.Linkable)
					od.send(r)
					od.active = true

					if od.loadedPrefetchFiles < od.totalPrefetchFiles {
						od.loadedPrefetchFiles++
					}

					od.message = fmt.Sprintf("Loading extra files - %d%%", od.loadedPrefetchFiles*100/od.totalPrefetchFiles)
					od.requestCount++

					if od.requestCount == 10 {
						return
					}
				}
			}
		}

		od.topPriority--
	}
}

// read consumes ondemand socket responses; the full part-reassembly port of
// Java read() (OnDemand.java:583-670 @32f3062) lands with the next commit.
func (od *OnDemand) read() {}

// connectResult carries the outcome of one async dial+handshake attempt.
type connectResult struct {
	stream *clientstream.ClientStream
	err    error
}

// openStream dials the world server and performs the ondemand handshake:
// write service byte 15, drain the 8 response bytes.
// Java: OnDemand.java:700-707 @32f3062 — synchronous on the worker thread
// there; here it runs on the connector goroutine (see the connecting field).
// Java ignores the drained bytes' values (including a -1 EOF), so only
// stream errors abort; clientstream's 30 s read bound turns the half-open
// hang Java could suffer into a counted failure.
func openStream(app App) connectResult {
	conn, err := app.OpenSocket()
	if err != nil {
		return connectResult{err: err}
	}
	cs := clientstream.NewClientStream(conn)
	hello := []byte{15}
	if err := cs.Write(hello, 1, 0); err != nil {
		cs.Close()
		return connectResult{err: err}
	}
	for range 8 {
		if _, err := cs.Read(); err != nil {
			cs.Close()
			return connectResult{err: err}
		}
	}
	return connectResult{stream: cs}
}

// closeStream tears down the ondemand connection. Java inlines this
// socket.close() + socket/in/out=null + partAvailable=0 sequence at all
// three failure sites (OnDemand.java:428-435 run, :661-668 read,
// :723-730 send).
func (od *OnDemand) closeStream() {
	if od.stream != nil {
		od.stream.Close()
		od.stream = nil
	}
	od.partAvailable = 0
}

// send transmits one 4-byte request frame, lazily (re)opening the ondemand
// socket. Java: send(OnDemandRequest) (OnDemand.java:692-733 @32f3062).
// Java's single IOException catch (close + failCount++) maps onto the
// per-call error paths below; the dial+handshake runs on a one-shot
// goroutine instead of inline (see the connecting field), during which
// send() returns early — the request stays pending and the Run() resend
// walk retries it once the connection lands.
//
// Note clientstream.Write is asynchronous (ring buffer + writer goroutine):
// a wire failure surfaces on a LATER Write call, exactly like Java's
// buffered OutputStream surfacing the IOException on a later flush.
func (od *OnDemand) send(r *OnDemandRequest) {
	if od.stream == nil {
		if od.connecting != nil {
			select {
			case res := <-od.connecting:
				od.connecting = nil
				if res.err != nil {
					od.FailCount++ // Java: failCount++ in the catch (OnDemand.java:731)
					return
				}
				od.stream = res.stream
				od.packetCycle = 0 // Java: OnDemand.java:707
			default:
				return // dial+handshake still in flight
			}
		} else {
			// Java: 4 s redial backoff (OnDemand.java:696-699).
			now := time.Now().UnixMilli() // Java: System.currentTimeMillis()
			if now-od.socketOpenTime < 4000 {
				return
			}
			od.socketOpenTime = now
			ch := make(chan connectResult, 1)
			od.connecting = ch
			app := od.app
			go func() { ch <- openStream(app) }()
			return
		}
	}

	// Java: OnDemand.java:709-719 — [archive, file>>8, file, priority].
	od.buf[0] = byte(r.Archive)
	od.buf[1] = byte(r.File >> 8)
	od.buf[2] = byte(r.File)
	if r.Urgent {
		od.buf[3] = 2
	} else if od.app.InGame() {
		od.buf[3] = 0
	} else {
		od.buf[3] = 1
	}
	if err := od.stream.Write(od.buf[:], 4, 0); err != nil {
		od.closeStream()
		od.FailCount++
		return
	}
	od.noTimeoutCycle = 0
	od.FailCount = -10000 // Java: OnDemand.java:720-721
}

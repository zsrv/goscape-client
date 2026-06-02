// Package ondemand ports Java's jagex2.io.OnDemand (rev-244).
// Inc 1 covers: types, versionlist parse, Validate, and getters.
// The request/cycle/prefetch/run/read state machine is Inc 2.
package ondemand

import (
	"hash/crc32"

	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct"
	jio "github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// OnDemandRequest mirrors Java's jagex2.io.OnDemandRequest (extends DoublyLinkable).
// Java: nb (obfuscated class name).
//
// In Java the class inherits two link-field pairs (next/prev from Linkable,
// next2/prev2 from DoublyLinkable) which allow a single request to sit in
// both a DoublyLinkList (requests) and a LinkList (queue/missing/pending/
// completed/prefetches) simultaneously.
//
// In Go we hold one *datastruct.DoublyLinkable[*OnDemandRequest] whose
// embedded *Linkable[*OnDemandRequest] provides the second link pair:
//   - r.node        → DoublyLinkList slot (next2/prev2 via Uncache/Push/Pop)
//   - r.node.Linkable → LinkList slot     (next/prev  via Unlink/AddHead/…)
type OnDemandRequest struct {
	// Java: nb.i
	Archive int
	// Java: nb.j
	File int
	// Java: nb.k
	Data []byte
	// Java: nb.l
	Cycle int
	// Java: nb.m
	Urgent bool

	// node provides the two link-field pairs that Java inherits from
	// DoublyLinkable -> Linkable. Inc 2 uses r.node / r.node.Linkable /
	// r.node.Uncache() / r.node.Unlink().
	node *datastruct.DoublyLinkable[*OnDemandRequest] //nolint:unused // Inc 2 state machine
}

// newRequest allocates a request with the Java default Urgent=true.
//
//nolint:unused // Inc 2 state machine
func newRequest() *OnDemandRequest {
	r := &OnDemandRequest{Urgent: true}
	r.node = datastruct.NewDoublyLinkable(r)
	return r
}

// ---- seam interfaces -------------------------------------------------------

// Downloader fetches an absolute path from the on-demand server.
// Client wires signlink.OpenURL to this in Inc 2.
type Downloader interface {
	Get(path string) ([]byte, error)
}

// Cache is the per-file persistent store.
// Client wires the storage seam here; nil = in-memory only.
type Cache interface {
	Read(archive, file int) []byte
	Write(archive, file int, data []byte)
}

// Archive is the versionlist source.
// *io.Jagfile satisfies it structurally.
// Java: OnDemand.unpack(Jagfile, Client); taken as a reader so the parse is unit-testable.
type Archive interface {
	Read(name string, dst []byte) []byte
}

// ---- OnDemand struct -------------------------------------------------------

// OnDemand ports Java jagex2.io.OnDemand (obfuscated: vb).
// Inc 1 contains fields, versionlist parsing, Validate, and getters only.
// The request/cycle/prefetch/run/read state machine is implemented in Inc 2.
type OnDemand struct {
	// Java: vb.g — per-archive file version tables (4 archives)
	versions [4][]int
	// Java: vb.h — per-archive file CRC tables (4 archives)
	crcs [4][]int
	// Java: vb.i — per-archive per-file download priority — written now; read in Inc 2 prefetch logic
	priorities [4][]byte
	// Java: vb.j
	topPriority int //nolint:unused // Inc 2 state machine

	// Java: vb.k — model index flags (byte[], masked & 0xFF on read)
	models []byte

	// Java: vb.l/m/n/o — map coordinate and member flag tables
	mapIndex   []int
	mapLand    []int
	mapLoc     []int
	mapMembers []int

	// Java: vb.p — anim (seq) file index
	animIndex []int
	// Java: vb.q — midi prefetch flag (1 = prefetch)
	midiIndex []int

	// Java: vb.r
	running bool

	// ---- request queues (Java: vb.x/y/z/A/B/C) ----------------------------
	// requests holds all in-flight OnDemandRequests in a doubly-linked list
	// so they can be found by (archive, file) and removed in O(1).
	requests *datastruct.DoublyLinkList[*OnDemandRequest]
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

	// zip holds the unzipped ondemand bundle (nil until Inc 2 downloads it).
	// Java: not present (Java used a TCP socket; TS client uses /ondemand.zip).
	zip map[string][]byte //nolint:unused // Inc 2 state machine

	// Java: vb.E/F
	loadedPrefetchFiles int //nolint:unused // Inc 2 state machine
	totalPrefetchFiles  int //nolint:unused // Inc 2 state machine

	// Java: vb.D
	message string //nolint:unused // Inc 2 state machine

	// ---- injected seams (no client/* imports) ------------------------------
	dl     Downloader
	cache  Cache // may be nil
	ingame func() bool
}

// ---- constructor -----------------------------------------------------------

// New allocates an OnDemand, wires seams, and calls Unpack.
// Java: OnDemand constructor + unpack() (vb.a(Lyb;Lclient;)V).
func New(versionlist Archive, dl Downloader, cache Cache, ingame func() bool) *OnDemand {
	od := &OnDemand{
		requests:   datastruct.NewDoublyLinkList[*OnDemandRequest](),
		queue:      datastruct.NewLinkList[*OnDemandRequest](),
		missing:    datastruct.NewLinkList[*OnDemandRequest](),
		pending:    datastruct.NewLinkList[*OnDemandRequest](),
		completed:  datastruct.NewLinkList[*OnDemandRequest](),
		prefetches: datastruct.NewLinkList[*OnDemandRequest](),
		running:    true,
		dl:         dl,
		cache:      cache,
		ingame:     ingame,
	}
	od.Unpack(versionlist)
	return od
}

// ---- versionlist parse -----------------------------------------------------

// Unpack reads all version/crc/index tables from the versionlist Jagfile.
// Java: OnDemand.unpack(Jagfile, Client) (vb.a(Lyb;Lclient;)V), lines 135–216.
// The trailing startThread call is not ported (Inc 2 starts the worker).
func (od *OnDemand) Unpack(versionlist Archive) {
	// model_version / anim_version / midi_version / map_version
	// Java: for i 0..3 — count = data.length/2; g2 each
	vnames := [4]string{"model_version", "anim_version", "midi_version", "map_version"}
	for i := range 4 {
		data := versionlist.Read(vnames[i], nil)
		count := len(data) / 2
		buf := jio.NewPacket(data)

		od.versions[i] = make([]int, count)
		od.priorities[i] = make([]byte, count)

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
			od.crcs[i][j] = buf.G4()
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

// RequestModel satisfies the OnDemandProvider interface.
// Java: OnDemand.requestModel(int) (vb.a(I)V), ~line 267.
// Full implementation (calling request()) is deferred to Inc 2.
func (od *OnDemand) RequestModel(id int) {
	// Inc 2: od.request(0, id)
	_ = id
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

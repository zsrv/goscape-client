package ondemand

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"hash/crc32"
	"testing"
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

// fakeDownloader returns fixed bytes for any path.
type fakeDownloader struct {
	data []byte
	err  error
}

func (d fakeDownloader) Get(_ string) ([]byte, error) {
	return d.data, d.err
}

// buildMapVersionlist returns a versionlist where the map archive (index 3) has
// `mapCount` files, each with the given version and crc, plus minimal empty
// tables for the other archives. This lets tests place requests against archive 3.
func buildMapVersionlist(mapCount, version, crc int) fakeArchiveInternal {
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

	mapVersions := make([]int, mapCount)
	mapCRCs := make([]int, mapCount)
	for i := range mapCount {
		mapVersions[i] = version
		mapCRCs[i] = crc
	}

	a := fakeArchiveInternal{}
	a["model_version"] = p2(0)
	a["model_crc"] = p4(0)
	a["model_index"] = []byte{0}

	a["anim_version"] = p2(0)
	a["anim_crc"] = p4(0)

	a["midi_version"] = p2(0)
	a["midi_crc"] = p4(0)

	a["map_version"] = p2(mapVersions...)
	a["map_crc"] = p4(mapCRCs...)

	a["map_index"] = []byte{}
	a["anim_index"] = []byte{}
	a["midi_index"] = []byte{}
	return a
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

// TestRun_BundleReadEndToEnd drives the full modernized path: Request → Run
// (handleQueue → missing → handlePending → pending → read → /ondemand.zip) →
// Cycle, with a nil cache so the file is fetched from the bundle.
func TestRun_BundleReadEndToEnd(t *testing.T) {
	payload := []byte("end-to-end model bytes")
	const version = 7
	entry := gzipTrailer(t, payload, version)

	// CRC over the gzip+payload bytes (everything but the 2-byte trailer),
	// narrowed to int32 like Validate.
	crc := int(int32(crc32.ChecksumIEEE(entry[:len(entry)-2])))

	// Build a real in-memory zip with entry "1.5" (archive 0 → 1, file 5).
	var zbuf bytes.Buffer
	zw := zip.NewWriter(&zbuf)
	w, err := zw.Create("1.5")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := w.Write(entry); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}

	dl := fakeDownloader{data: zbuf.Bytes()}
	od := New(buildModelVersionlist(10, version, crc), dl, nil)

	od.Request(0, 5)

	// Pump a few frames; one is enough but a few guards against ordering.
	for range 3 {
		od.Run()
	}

	got := od.Cycle()
	if got == nil {
		t.Fatal("Cycle() returned nil after Run(); expected the completed request")
	}
	if !bytes.Equal(got.Data, payload) {
		t.Fatalf("end-to-end decoded Data = %q, want %q", got.Data, payload)
	}
	if rem := od.Remaining(); rem != 0 {
		t.Fatalf("Remaining() = %d after Cycle(), want 0", rem)
	}
}

// TestRead_Archive3PromotedTo93 verifies the archive-3 → 93 promotion in read():
// a non-urgent map-archive fetch must emerge from completed with Archive==93 and
// Urgent==true, matching the Client-TS read() promotion path.
func TestRead_Archive3PromotedTo93(t *testing.T) {
	const (
		mapFile = 4
		version = 11
	)

	payload := []byte("map tile data")
	entry := gzipTrailer(t, payload, version)
	crc := int(int32(crc32.ChecksumIEEE(entry[:len(entry)-2])))

	// Build an in-memory zip containing "4.4" (archive 3+1=4, file 4).
	var zbuf bytes.Buffer
	zw := zip.NewWriter(&zbuf)
	w, err := zw.Create("4.4")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := w.Write(entry); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}

	dl := fakeDownloader{data: zbuf.Bytes()}
	// Map archive (index 3) needs at least mapFile+1 entries with non-zero versions.
	od := New(buildMapVersionlist(mapFile+1, version, crc), dl, nil)

	// Place a non-urgent request for archive 3, file 4 directly onto pending,
	// mirroring how handlePending promotes a missing request (Urgent defaults to
	// true from newRequest; we override it to false to exercise the promotion path).
	r := newRequest()
	r.Archive = 3
	r.File = mapFile
	r.Urgent = false
	od.requests.Push(r.node)
	od.pending.Push(r.node.Linkable)

	// read() services the head of pending; it calls downloadZip() internally.
	od.read()

	// The request must have landed on completed with the promoted fields.
	n := od.completed.PopFront()
	if n == nil {
		t.Fatal("completed list is empty after read(); expected the promoted request")
	}
	got := n.Value
	if got.Archive != 93 {
		t.Errorf("Archive = %d after promotion, want 93", got.Archive)
	}
	if !got.Urgent {
		t.Error("Urgent = false after promotion, want true")
	}

	// Cycle() decodes the gzip+trailer payload.
	od.requests.Push(got.node)           // re-link so Cycle's Uncache works
	od.completed.Push(got.node.Linkable) // put it back for Cycle
	result := od.Cycle()
	if result == nil {
		t.Fatal("Cycle() returned nil")
	}
	if !bytes.Equal(result.Data, payload) {
		t.Fatalf("Cycle() decoded Data = %q, want %q", result.Data, payload)
	}
}

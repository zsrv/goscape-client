package ondemand_test

import (
	"encoding/binary"
	"hash/crc32"
	"testing"

	jio "github.com/zsrv/goscape-client/pkg/jagex2/io"
	"github.com/zsrv/goscape-client/pkg/jagex2/io/ondemand"
)

// ---- fake Archive for unit tests -------------------------------------------

// fakeArchive satisfies the ondemand.Archive interface using a map of name→data.
type fakeArchive map[string][]byte

func (f fakeArchive) Read(name string, _ []byte) []byte {
	return f[name]
}

// buildVersionlist returns a fakeArchive whose members encode small, predictable
// tables that the test can verify after Unpack.
//
// Encoded tables:
//   - model_version: 3 files, versions [10, 20, 30]
//   - anim_version:  2 files, versions [100, 200]
//   - midi_version:  1 file,  version  [5]
//   - map_version:   1 file,  version  [7]
//   - model_crc:     3 files, crcs [0x11111111, 0x22222222, 0x33333333]
//   - anim_crc:      2 files, crcs [0xAAAAAAAA, 0xBBBBBBBB]
//   - midi_crc:      1 file,  crc  [0xCCCCCCCC]
//   - map_crc:       1 file,  crc  [0xDDDDDDDD]
//   - model_index:   flags    [0x01, 0x02, 0x03]
//   - map_index:     1 record: mapID=(5<<8)+3, land=42, loc=43, members=1
//   - anim_index:    2 entries: [7, 9]
//   - midi_index:    3 entries: [0, 1, 0]
func buildVersionlist() fakeArchive {
	a := fakeArchive{}

	// helper: write n as big-endian 2-byte int via Packet.P2
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

	a["model_version"] = p2(10, 20, 30)
	a["anim_version"] = p2(100, 200)
	a["midi_version"] = p2(5)
	a["map_version"] = p2(7)

	a["model_crc"] = p4(0x11111111, 0x22222222, 0x33333333)
	a["anim_crc"] = p4(0xAAAAAAAA, 0xBBBBBBBB)
	a["midi_crc"] = p4(0xCCCCCCCC)
	a["map_crc"] = p4(0xDDDDDDDD)

	a["model_index"] = p1(0x01, 0x02, 0x03)

	// map_index: 1 record of 7 bytes: g2 mapID, g2 land, g2 loc, g1 members
	// mapID = (x<<8)+z; we encode x=5, z=3 → mapID=(5<<8)+3=1283
	mapRec := make([]byte, 7)
	binary.BigEndian.PutUint16(mapRec[0:], uint16((5<<8)+3)) // mapIndex
	binary.BigEndian.PutUint16(mapRec[2:], uint16(42))       // mapLand
	binary.BigEndian.PutUint16(mapRec[4:], uint16(43))       // mapLoc
	mapRec[6] = 1                                            // mapMembers
	a["map_index"] = mapRec

	a["anim_index"] = p2(7, 9)
	a["midi_index"] = p1(0, 1, 0)

	return a
}

// ---- Validate tests --------------------------------------------------------

func TestValidate_Valid(t *testing.T) {
	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	version := 42
	versionBytes := []byte{byte(version >> 8), byte(version)}
	src := append(payload, versionBytes...)

	expectedCrc := int(int32(crc32.ChecksumIEEE(payload)))
	if !ondemand.Validate(src, expectedCrc, version) {
		t.Fatal("expected Validate to return true for a well-formed src")
	}
}

func TestValidate_WrongVersion(t *testing.T) {
	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	version := 42
	versionBytes := []byte{byte(version >> 8), byte(version)}
	src := append(payload, versionBytes...)
	expectedCrc := int(int32(crc32.ChecksumIEEE(payload)))

	if ondemand.Validate(src, expectedCrc, version+1) {
		t.Fatal("expected Validate to return false when version is wrong")
	}
}

func TestValidate_CorruptPayload(t *testing.T) {
	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	version := 42
	versionBytes := []byte{byte(version >> 8), byte(version)}
	src := append(payload, versionBytes...)
	expectedCrc := int(int32(crc32.ChecksumIEEE(payload)))

	// corrupt a payload byte before passing
	corrupt := make([]byte, len(src))
	copy(corrupt, src)
	corrupt[0] ^= 0xFF

	if ondemand.Validate(corrupt, expectedCrc, version) {
		t.Fatal("expected Validate to return false when payload is corrupted")
	}
}

func TestValidate_TooShort(t *testing.T) {
	if ondemand.Validate([]byte{0x01}, 0, 0) {
		t.Fatal("expected Validate to return false for len(src)<2")
	}
}

func TestValidate_Nil(t *testing.T) {
	if ondemand.Validate(nil, 0, 0) {
		t.Fatal("expected Validate to return false for nil src")
	}
}

// ---- Unpack / getter tests --------------------------------------------------

// TestUnpack_GetFileCount verifies that Unpack reads model_version correctly
// and GetFileCount returns the right count.
func TestUnpack_GetFileCount(t *testing.T) {
	od := ondemand.New(buildVersionlist(), nil, nil)
	if got := od.GetFileCount(0); got != 3 {
		t.Fatalf("GetFileCount(0) = %d, want 3", got)
	}
	if got := od.GetFileCount(1); got != 2 {
		t.Fatalf("GetFileCount(1) = %d, want 2", got)
	}
}

// TestUnpack_GetAnimCount verifies anim_index parse.
func TestUnpack_GetAnimCount(t *testing.T) {
	od := ondemand.New(buildVersionlist(), nil, nil)
	if got := od.GetAnimCount(); got != 2 {
		t.Fatalf("GetAnimCount() = %d, want 2", got)
	}
}

// TestUnpack_GetMapFile verifies map_index parse and GetMapFile lookup.
// Encoded: x=5, z=3 → mapID=1283, land=42, loc=43.
func TestUnpack_GetMapFile(t *testing.T) {
	od := ondemand.New(buildVersionlist(), nil, nil)

	// type==0 → land file
	if got := od.GetMapFile(3, 5, 0); got != 42 {
		t.Fatalf("GetMapFile(z=3,x=5,type=0) = %d, want 42", got)
	}
	// type!=0 → loc file
	if got := od.GetMapFile(3, 5, 1); got != 43 {
		t.Fatalf("GetMapFile(z=3,x=5,type=1) = %d, want 43", got)
	}
	// not found
	if got := od.GetMapFile(0, 0, 0); got != -1 {
		t.Fatalf("GetMapFile(z=0,x=0,type=0) = %d, want -1", got)
	}
}

// TestUnpack_GetModelFlags verifies model_index parse and the & 0xFF mask.
func TestUnpack_GetModelFlags(t *testing.T) {
	od := ondemand.New(buildVersionlist(), nil, nil)
	for i, want := range []int{0x01, 0x02, 0x03} {
		if got := od.GetModelFlags(i); got != want {
			t.Fatalf("GetModelFlags(%d) = %d, want %d", i, got, want)
		}
	}
}

// TestUnpack_GetModelFlags_SignExtension verifies that a byte value ≥ 0x80 is
// masked correctly (Java byte sign-extension must NOT leak through).
func TestUnpack_GetModelFlags_SignExtension(t *testing.T) {
	// Build a custom versionlist with a 0xFF model_index flag.
	a := buildVersionlist()
	a["model_index"] = []byte{0xFF, 0x00, 0x00}
	od := ondemand.New(a, nil, nil)
	if got := od.GetModelFlags(0); got != 0xFF {
		t.Fatalf("GetModelFlags(0) = %d (0x%02X), want 255 (0xFF)", got, got)
	}
}

// TestUnpack_ShouldPrefetchMidi verifies midi_index parse.
// Encoded: [0, 1, 0] → only index 1 should prefetch.
func TestUnpack_ShouldPrefetchMidi(t *testing.T) {
	od := ondemand.New(buildVersionlist(), nil, nil)
	cases := []struct {
		id   int
		want bool
	}{
		{0, false},
		{1, true},
		{2, false},
	}
	for _, tc := range cases {
		if got := od.ShouldPrefetchMidi(tc.id); got != tc.want {
			t.Fatalf("ShouldPrefetchMidi(%d) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

// TestUnpack_HasMapLocFile verifies HasMapLocFile using the encoded loc=43.
func TestUnpack_HasMapLocFile(t *testing.T) {
	od := ondemand.New(buildVersionlist(), nil, nil)
	if !od.HasMapLocFile(43) {
		t.Fatal("HasMapLocFile(43) = false, want true")
	}
	if od.HasMapLocFile(99) {
		t.Fatal("HasMapLocFile(99) = true, want false")
	}
}

// TestPacket_P2G2RoundTrip checks that data written with io.Packet P2
// and read back by G2 produces the expected values — exercising the Packet
// writers the same way the Unpack tests rely on them.
func TestPacket_P2G2RoundTrip(t *testing.T) {
	// Build a 2-byte big-endian version via Packet.P2 and verify G2 reads it back.
	buf := make([]byte, 2)
	p := jio.NewPacket(buf)
	p.P2(0x1234)
	p.Pos = 0
	if got := p.G2(); got != 0x1234 {
		t.Fatalf("Packet P2/G2 round-trip: got 0x%04X, want 0x1234", got)
	}
}

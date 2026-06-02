package ondemand

import (
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
	od := New(fake, nil, nil, nil)

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

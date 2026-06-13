package animframe

import "testing"

// TestUnpackOneFrameBlob synthesizes a minimal rev-244 per-id animation blob
// with an embedded AnimBase (size 1, type 0) and a single frame that transforms
// the one group on all three axes, then asserts Unpack decodes it correctly.
//
// GSmart encodes a value v in [-64,63] as the single byte v+64 (decode is
// G1()-64), so each transform delta below consumes one byte.
func TestUnpackOneFrameBlob(t *testing.T) {
	gs := func(v int) byte { return byte(v + 64) } // GSmart single-byte encoding

	// Section layout (front to back), then the 8-byte trailer.
	data := []byte{
		// head (headLength+2 = 5 bytes): total=1, frame id=0, groupCount=1
		0x00, 0x01, // total = 1 (g2)
		0x00, 0x00, // frame id = 0 (g2)
		0x01, // groupCount = 1 (g1)
		// tran1 (tran1Length = 1): flags = 0x07 (x, y, z all present)
		0x07,
		// tran2 (tran2Length = 3): tx=5, ty=-3, tz=10
		gs(5), gs(-3), gs(10),
		// del (delLength = 1): delay = 7
		0x07,
		// base (AnimBase): size=1, type[0]=0, group0 label count=1, label[0][0]=0
		0x01, 0x00, 0x01, 0x00,
		// ---- 8-byte trailer ----
		0x00, 0x03, // headLength = 3 (g2)
		0x00, 0x01, // tran1Length = 1 (g2)
		0x00, 0x03, // tran2Length = 3 (g2)
		0x00, 0x01, // delLength = 1 (g2)
	}

	List = nil
	Init(1)
	Unpack(data)

	f := List[0]
	if f == nil {
		t.Fatal("List[0] is nil after Unpack")
	}
	if f.Base == nil || f.Base.Size != 1 || f.Base.Type[0] != 0 {
		t.Fatalf("base = %+v, want size 1 type[0]=0", f.Base)
	}
	if f.Delay != 7 {
		t.Errorf("Delay = %d, want 7", f.Delay)
	}
	if f.Size != 1 {
		t.Fatalf("Size = %d, want 1", f.Size)
	}
	if f.Ti[0] != 0 {
		t.Errorf("Ti[0] = %d, want 0", f.Ti[0])
	}
	if f.Tx[0] != 5 || f.Ty[0] != -3 || f.Tz[0] != 10 {
		t.Errorf("transform = (%d,%d,%d), want (5,-3,10)", f.Tx[0], f.Ty[0], f.Tz[0])
	}
}

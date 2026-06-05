package varptype

import (
	"fmt"

	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// VarpType decoder. Java 274 VarpType.java has opcodes 0-8 and 10-13
// that write to per-instance fields and a package-level array, but
// only opcode 5 (ClientCode) is ever read anywhere in the Java or Go
// codebase. The remaining fields (274 deob names field1181-field1191,
// incl. the new op12 field1191 and the op8 field1189 boolean→int
// promotion) are pure deobfuscator residue per the project's
// deob-artifact exclusion policy. Decode preserves the wire reads as
// discards so packet-position alignment matches Java byte-for-byte.

var (
	Count int
	List  []*VarpType
)

type VarpType struct {
	ClientCode int
}

func NewVarpType() *VarpType {
	return &VarpType{}
}

func Unpack(arg0 *io.JagFile) {
	var2 := io.NewPacket(arg0.Read("varp.dat", nil))
	Count = var2.G2()
	if List == nil {
		List = make([]*VarpType, Count)
	}
	for i := range Count {
		if List[i] == nil {
			List[i] = NewVarpType()
		}
		List[i].Decode(var2, i)
	}
	if var2.Pos != len(var2.Data) {
		fmt.Println("varptype load mismatch") // Java: VarpType.java:70-72 @32f3062
	}
}

// Decode reads one definition. Java 274 swaps the param order to
// decode(Packet, int) (VarpType.java:78 @32f3062; was decode(int, Packet)
// at 254). arg2 (the definition id) is only consumed by dead op3 in Java;
// kept for signature fidelity.
func (t *VarpType) Decode(arg0 *io.Packet, arg2 int) {
	for {
		var5 := arg0.G1()
		switch var5 {
		case 0:
			return
		case 1:
			arg0.G1() // Java: field1182 = arg0.g1() — dead-write field omitted
		case 2:
			arg0.G1() // Java: field1183 = arg0.g1() — dead-write field omitted
		case 3:
			// Java: field1184 = true; field1180[field1179++] = arg2 — dead
		case 4:
			// Java: field1185 = false — dead-write field omitted
		case 5:
			t.ClientCode = arg0.G2()
		case 6:
			// Java: field1187 = true — dead-write field omitted
		case 7:
			arg0.G4() // Java: field1188 = arg0.g4() — dead-write field omitted
		case 8:
			// Java 274: field1189 = 1 (int; was boolean code8 = true at 254) +
			// field1190 = true — both dead-write fields, omitted. No wire
			// payload.
		case 10:
			arg0.GStr() // Java: field1181 = arg0.gjstr() (VarpType.java:113 @32f3062; debugname @2e62978) — dead-write field omitted
		case 11:
			// Java: field1190 = true — dead-write field omitted. No wire
			// payload; handled so the default error branch is not hit.
		case 12:
			// Java 274 NEW op: field1191 = arg0.g4() (default -1,
			// VarpType.java:117 @32f3062) — dead-write field (no readers),
			// omitted; wire read preserved as a discard.
			arg0.G4()
		case 13:
			// Java 274 NEW op: field1189 = 2 (VarpType.java:119 @32f3062) —
			// dead-write field, omitted. No wire payload.
		default:
			fmt.Println("Error unrecognised config code:", var5)
		}
	}
}

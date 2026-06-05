package varptype

import (
	"fmt"

	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// VarpType decoder. Java VarpType.java has 10 opcodes (0-8, 10) that
// write to per-instance fields and a package-level Code3 array, but
// only opcode 5 (ClientCode) is ever read anywhere in the Java or Go
// codebase. The remaining fields (code1, code2, hasCode3, code3,
// code3Count, code4, code6, code7, code8, code10) are pure deobfuscator
// residue per the project's deob-artifact exclusion policy. Decode
// preserves the wire reads as discards so packet-position alignment
// matches Java byte-for-byte.

var (
	Count     int
	Instances []*VarpType
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
	if Instances == nil {
		Instances = make([]*VarpType, Count)
	}
	for i := range Count {
		if Instances[i] == nil {
			Instances[i] = NewVarpType()
		}
		Instances[i].Decode(i, var2)
	}
	if var2.Pos != len(var2.Data) {
		fmt.Println("varptype load mismatch") // Java: VarpType.java:76-78
	}
}

func (t *VarpType) Decode(arg1 int, arg2 *io.Packet) {
	for {
		var4 := arg2.G1()
		switch var4 {
		case 0:
			return
		case 1:
			arg2.G1() // Java: this.code1 = arg2.g1() — dead-write field omitted
		case 2:
			arg2.G1() // Java: this.code2 = arg2.g1() — dead-write field omitted
		case 3:
			// Java: this.hasCode3 = true; code3[code3Count++] = arg1 — dead
		case 4:
			// Java: this.code4 = false — dead-write field omitted
		case 5:
			t.ClientCode = arg2.G2()
		case 6:
			// Java: this.code6 = true — dead-write field omitted
		case 7:
			arg2.G4() // Java: this.code7 = arg2.g4() — dead-write field omitted
		case 8:
			// Java: this.code8 = true; this.code11 = true (rev-244 sets both) —
			// both dead-write fields, omitted. No wire payload.
		case 10:
			arg2.GStr() // Java: this.debugname = arg2.gstr() (VarpType.java:100 @2e62978) — dead-write field omitted
		case 11:
			// Java: this.code11 = true (rev-244) — dead-write field omitted. No
			// wire payload; handled so the default error branch is not hit.
		default:
			fmt.Println("Error unrecognised config code:", var4)
		}
	}
}

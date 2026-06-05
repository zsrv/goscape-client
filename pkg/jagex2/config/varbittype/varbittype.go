package varbittype

import (
	"fmt"

	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// VarbitType decoder. Java: VarBitType.java @2e62978 (254 name); VarbitType.java @32f3062 — body at 254 parity, P4 pending. A varbit
// addresses a bit-slice of a varp: getIfVar opcode 14 reads
// `varps[basevar] >> startbit & BITMASK[endbit-startbit]`. Java's `debugname`
// field (opcode 10) is written but never read anywhere at 2e62978 — per the
// project's dead-write exclusion policy (and matching VarpType/FloType) the
// field is omitted; Decode preserves the wire read as a discard so packet
// position alignment matches Java byte-for-byte.

var (
	Count int           // Java: VarBitType.count
	List  []*VarbitType // Java: VarBitType.list (VarBitType.java:14 @2e62978)
)

type VarbitType struct {
	BaseVar  int // Java: basevar
	StartBit int // Java: startbit
	EndBit   int // Java: endbit
}

func NewVarbitType() *VarbitType {
	return &VarbitType{}
}

func Unpack(arg1 *io.JagFile) {
	var2 := io.NewPacket(arg1.Read("varbit.dat", nil))
	Count = var2.G2()
	if List == nil {
		List = make([]*VarbitType, Count)
	}
	for i := range Count {
		if List[i] == nil {
			List[i] = NewVarbitType()
		}
		List[i].Decode(i, var2)
	}
	if var2.Pos != len(var2.Data) {
		fmt.Println("varbit load mismatch") // Java: VarBitType.java:42-44
	}
}

func (t *VarbitType) Decode(arg0 int, arg2 *io.Packet) {
	for {
		var5 := arg2.G1()
		switch var5 {
		case 0:
			return
		case 1:
			t.BaseVar = arg2.G2()
			t.StartBit = arg2.G1()
			t.EndBit = arg2.G1()
		case 10:
			arg2.GStr() // Java: this.debugname = arg2.gstr() — dead-write field omitted
		default:
			fmt.Println("Error unrecognised config code:", var5)
		}
	}
}

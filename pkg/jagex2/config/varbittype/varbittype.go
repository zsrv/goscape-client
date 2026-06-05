package varbittype

import (
	"fmt"

	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// VarbitType decoder. Java: VarBitType.java @2e62978 (254 name); VarbitType.java @32f3062 (274 parity: decode(Packet,int) swap). A varbit
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

func Init(arg1 *io.JagFile) {
	var2 := io.NewPacket(arg1.Read("varbit.dat", nil))
	Count = var2.G2()
	if List == nil {
		List = make([]*VarbitType, Count)
	}
	for i := range Count {
		if List[i] == nil {
			List[i] = NewVarbitType()
		}
		List[i].Decode(var2, i)
	}
	if var2.Pos != len(var2.Data) {
		fmt.Println("varbit load mismatch") // Java: VarbitType.java:41-43 @32f3062 (VarBitType.java:42-44 @2e62978)
	}
}

// Decode reads one definition. Java 274 swaps the param order to
// decode(Packet, int) (VarbitType.java:47 @32f3062; was decode(int, Packet)
// at 254's VarBitType). arg2 (the definition id) is unused in both revs;
// kept for signature fidelity.
func (t *VarbitType) Decode(arg0 *io.Packet, arg2 int) {
	for {
		var5 := arg0.G1()
		switch var5 {
		case 0:
			return
		case 1:
			t.BaseVar = arg0.G2()
			t.StartBit = arg0.G1()
			t.EndBit = arg0.G1()
		case 10:
			arg0.GStr() // Java: debugname = arg0.gjstr() — dead-write field omitted (still dead at @32f3062)
		default:
			fmt.Println("Error unrecognised config code:", var5)
		}
	}
}

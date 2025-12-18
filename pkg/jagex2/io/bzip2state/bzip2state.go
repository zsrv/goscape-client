package bzip2state

const (
	MTFA_SIZE         = 4096
	MTFL_SIZE         = 16
	BZ_MAX_ALPHA_SIZE = 258
	BZ_MAX_CODE_LEN   = 23
	anInt732          = 1
	BZ_N_GROUPS       = 6
	BZ_G_SIZE         = 50
	anInt735          = 4
	BZ_MAX_SELECTORS  = 18002
)

var (
	TT []int
)

type BZip2State struct {
	Stream          []byte
	NextIn          int
	AvailIn         int
	TotalInLo32     int
	TotalInHi32     int
	Decompressed    []byte
	NextOut         int
	AvailOut        int
	TotalOutLo32    int
	TotalOutHi32    int
	StateOutCh      byte
	StateOutLen     int
	BlockRandomized bool
	BsBuff          int
	BsLive          int
	BlockSize100k   int
	CurrBlockNo     int
	UnZFTab         [256]int
	CFTab           [257]int
	InUse           [256]bool
	InUse16         [16]bool
	SeqToUnseq      [256]byte
	MTFA            [4096]byte
	MTFBase         [16]int
	Selector        [18002]byte
	SelectorMTF     [18002]byte
	Len             [6][258]byte
	Limit           [6][258]int
	Base            [6][258]int
	Perm            [6][258]int
	MinLens         [6]int
	OrigPtr         int
	TPos            int
	K0              int
	CNBlockUsed     int
	NInUse          int
	SaveNBlock      int
}

func NewBZip2State() *BZip2State {
	return new(BZip2State)
}

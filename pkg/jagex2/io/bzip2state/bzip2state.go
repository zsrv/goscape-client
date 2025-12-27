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
	UnZFTab         []int
	CFTab           []int
	InUse           []bool
	InUse16         []bool
	SeqToUnseq      []byte
	MTFA            []byte
	MTFBase         []int
	Selector        []byte
	SelectorMTF     []byte
	Len             [][]byte
	Limit           [][]int
	Base            [][]int
	Perm            [][]int
	MinLens         []int
	OrigPtr         int
	TPos            int
	K0              int
	CNBlockUsed     int
	NInUse          int
	SaveNBlock      int
}

func NewBZip2State() *BZip2State {
	var s BZip2State
	s.UnZFTab = make([]int, 256)
	s.CFTab = make([]int, 257)
	s.InUse = make([]bool, 256)
	s.InUse16 = make([]bool, 16)
	s.SeqToUnseq = make([]byte, 256)
	s.MTFA = make([]byte, 4096)
	s.MTFBase = make([]int, 16)
	s.Selector = make([]byte, 18002)
	s.SelectorMTF = make([]byte, 18002)
	s.Len = make([][]byte, 6)
	for i := range s.Len {
		s.Len[i] = make([]byte, 258)
	}
	s.Limit = make([][]int, 6)
	for i := range s.Limit {
		s.Limit[i] = make([]int, 258)
	}
	s.Base = make([][]int, 6)
	for i := range s.Base {
		s.Base[i] = make([]int, 258)
	}
	s.Perm = make([][]int, 6)
	for i := range s.Perm {
		s.Perm[i] = make([]int, 258)
	}
	s.MinLens = make([]int, 6)
	return &s
}

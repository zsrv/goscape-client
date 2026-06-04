package bzip2

import (
	"fmt"
	"sync"

	"github.com/zsrv/goscape-client/pkg/jagex2/io/bzip2state"
)

var (
	State = bzip2state.NewBZip2State()
	// mu serializes access to State (and bzip2state.TT), mirroring the
	// `synchronized` on Java BZip2.read. Without it, RunMidi decoding the
	// title music races Client.Load decoding the title fonts and the bit
	// reader's NextIn walks off the end of the buffer.
	// Java: BZip2.read (deob/BZip2.java)
	mu sync.Mutex
)

// Decompress
func Read(decompressed []byte, length int, stream []byte, availIn int, nextIn int) int {
	mu.Lock()
	defer mu.Unlock()
	State.Stream = stream
	State.NextIn = nextIn
	State.Decompressed = decompressed
	State.NextOut = 0
	State.AvailIn = availIn
	State.AvailOut = length
	State.BsLive = 0
	State.BsBuff = 0
	State.TotalInLo32 = 0
	State.TotalInHi32 = 0
	State.TotalOutLo32 = 0
	State.TotalOutHi32 = 0
	State.CurrBlockNo = 0
	Decompress(State)
	return length - State.AvailOut
}

// unRLE_obuf_to_output_FAST
func Finish(s *bzip2state.BZip2State) {
	cStateOutCh := s.StateOutCh
	cStateOutLen := s.StateOutLen
	cNBlockUsed := s.CNBlockUsed
	cK0 := s.K0
	cTT := bzip2state.TT
	cTPos := s.TPos
	csDecompressed := s.Decompressed
	csNextOut := s.NextOut
	csAvailOut := s.AvailOut
	availOutInit := csAvailOut
	sSaveNBlockPP := s.SaveNBlock + 1

label67:
	for {
		if cStateOutLen > 0 {
			for {
				if csAvailOut == 0 {
					break label67
				}

				if cStateOutLen == 1 {
					if csAvailOut == 0 {
						cStateOutLen = 1
						break label67
					}

					csDecompressed[csNextOut] = cStateOutCh
					csNextOut++
					csAvailOut--
					break
				}

				csDecompressed[csNextOut] = cStateOutCh
				cStateOutLen--
				csNextOut++
				csAvailOut--
			}
		}

		next := true
		k1 := byte(0)
		for next {
			next = false
			if cNBlockUsed == sSaveNBlockPP {
				cStateOutLen = 0
				break label67
			}

			// macro: BZ_GET_FAST_C
			cStateOutCh = byte(cK0)
			cTPos = cTT[cTPos]
			k1 = byte(cTPos & 0xFF)
			cTPos >>= 0x8
			cNBlockUsed++

			if int(k1) != cK0 {
				cK0 = int(k1)
				if csAvailOut == 0 {
					cStateOutLen = 1
					break label67
				}

				csDecompressed[csNextOut] = cStateOutCh
				csNextOut++
				csAvailOut--
				next = true
			} else if cNBlockUsed == sSaveNBlockPP {
				if csAvailOut == 0 {
					cStateOutLen = 1
					break label67
				}

				csDecompressed[csNextOut] = cStateOutCh
				csNextOut++
				csAvailOut--
				next = true
			}
		}

		// macro: BZ_GET_FAST_C
		cStateOutLen = 2
		cTPos = cTT[cTPos]
		k1 = byte(cTPos & 0xFF)
		cTPos >>= 0x8
		cNBlockUsed++

		if cNBlockUsed != sSaveNBlockPP {
			if int(k1) == cK0 {
				// macro: BZ_GET_FAST_C
				cStateOutLen = 3
				cTPos = cTT[cTPos]
				k1 = byte(cTPos & 0xFF)
				cTPos >>= 0x8
				cNBlockUsed++

				if cNBlockUsed != sSaveNBlockPP {
					if int(k1) == cK0 {
						// macro: BZ_GET_FAST_C
						cTPos = cTT[cTPos]
						k1 = byte(cTPos & 0xFF)
						cTPos >>= 0x8
						cNBlockUsed++

						// macro: BZ_GET_FAST_C
						cStateOutLen = int(k1&0xFF) + 4
						cTPos = cTT[cTPos]
						cK0 = cTPos & 0xFF
						cTPos >>= 0x8
						cNBlockUsed++
					} else {
						cK0 = int(k1)
					}
				}
			} else {
				cK0 = int(k1)
			}
		}
	}

	var13 := s.TotalOutLo32
	s.TotalOutLo32 += int32(availOutInit - csAvailOut)
	if s.TotalOutLo32 < var13 {
		s.TotalOutHi32++
	}

	// save
	s.StateOutCh = cStateOutCh
	s.StateOutLen = cStateOutLen
	s.CNBlockUsed = cNBlockUsed
	s.K0 = cK0
	bzip2state.TT = cTT
	s.TPos = cTPos
	s.Decompressed = csDecompressed
	s.NextOut = csNextOut
	s.AvailOut = csAvailOut
	// end save
}

func Decompress(s *bzip2state.BZip2State) {
	// libbzip2 uses these variables in a save area
	/*boolean save_i = false;
	boolean save_j = false;
	boolean save_t = false;
	boolean save_alphaSize = false;
	boolean save_nGroups = false;
	boolean save_nSelectors = false;
	boolean save_EOB = false;
	boolean save_groupNo = false;
	boolean save_groupPos = false;
	boolean save_nextSym = false;
	boolean save_nblockMAX = false;
	boolean save_nblock = false;
	boolean save_es = false;
	boolean save_N = false;
	boolean save_curr = false;
	boolean save_zt = false;
	boolean save_zn = false;
	boolean save_zvec = false;
	boolean save_zj = false;*/

	gMinLen := 0
	var gLimit []int
	var gBase []int
	var gPerm []int

	s.BlockSize100k = 1
	if bzip2state.TT == nil {
		bzip2state.TT = make([]int, s.BlockSize100k*100_000)
	}

	reading := true
	// Java: BZip2.decompress's outer `while (true)` (BZip2.java:180-468) wraps the
	// inner `while (var27)` loop. Both inner break-paths fall through to the
	// outer `return`, so the outer for-loop body only executes once. Preserved
	// verbatim from the obfuscated source; gopls correctly notes the outer loop
	// is unconditionally terminated.
	for {
		for reading {
			uc := GetUnsignedChar(s)
			if uc == 0x17 {
				return
			}

			// Java: BZip2.java:186-195 — these repeated getUnsignedChar reads advance
			// the bit-stream for their side effect; the assignment to uc is discarded.
			uc = GetUnsignedChar(s) //nolint:ineffassign,staticcheck
			uc = GetUnsignedChar(s) //nolint:ineffassign,staticcheck
			uc = GetUnsignedChar(s) //nolint:ineffassign,staticcheck
			uc = GetUnsignedChar(s) //nolint:ineffassign,staticcheck
			uc = GetUnsignedChar(s) //nolint:ineffassign,staticcheck

			s.CurrBlockNo++

			uc = GetUnsignedChar(s) //nolint:ineffassign,staticcheck // Java: BZip2.java skip-bytes (stream advance side effect)
			uc = GetUnsignedChar(s) //nolint:ineffassign,staticcheck
			uc = GetUnsignedChar(s) //nolint:ineffassign,staticcheck
			uc = GetUnsignedChar(s) //nolint:ineffassign,staticcheck

			uc = GetBit(s)
			if uc == 0 {
				s.BlockRandomized = false
			} else {
				s.BlockRandomized = true
			}

			if s.BlockRandomized {
				fmt.Println("PANIC! RANDOMISED BLOCK!")
			}

			s.OrigPtr = 0
			uc = GetUnsignedChar(s)
			s.OrigPtr = s.OrigPtr<<8 | int(uc&0xFF)
			uc = GetUnsignedChar(s)
			s.OrigPtr = s.OrigPtr<<8 | int(uc&0xFF)
			uc = GetUnsignedChar(s)
			s.OrigPtr = s.OrigPtr<<8 | int(uc&0xFF)

			// Receive the mapping table
			for i := range 16 {
				uc = GetBit(s)
				if uc == 1 {
					s.InUse16[i] = true
				} else {
					s.InUse16[i] = false
				}
			}

			for i := range 256 {
				s.InUse[i] = false
			}

			for i := range 16 {
				if s.InUse16[i] {
					for j := range 16 {
						uc = GetBit(s)
						if uc == 1 {
							s.InUse[i*16+j] = true
						}
					}
				}
			}
			MakeMaps(s)
			alphaSize := s.NInUse + 2

			nGroups := GetBits(3, s)
			nSelectors := GetBits(15, s)
			for i := range nSelectors {
				j := 0
				for {
					uc = GetBit(s)
					if uc == 0 {
						s.SelectorMTF[i] = byte(j)
						break
					}

					j++
				}
			}

			// Undo the MTF values for the selectors
			pos := make([]byte, bzip2state.BZ_N_GROUPS)
			v := 0
			for v < nGroups {
				pos[v] = byte(v)
				v++
			}

			for i := range nSelectors {
				v = int(s.SelectorMTF[i])
				tmp := pos[v]
				for v > 0 {
					pos[v] = pos[v-1]
					v--
				}
				pos[0] = tmp
				s.Selector[i] = tmp
			}

			// Now the coding tables
			for t := range nGroups {
				curr := GetBits(5, s)

				for i := range alphaSize {
					for {
						uc = GetBit(s)
						if uc == 0 {
							s.Len[t][i] = byte(curr)
							break
						}

						uc = GetBit(s)
						if uc == 0 {
							curr++
						} else {
							curr--
						}
					}
				}
			}

			// Create the Huffman decoding tables
			for t := range nGroups {
				minLen := byte(32)
				maxLen := byte(0)
				for i := range alphaSize {
					maxLen = max(s.Len[t][i], maxLen)

					minLen = min(s.Len[t][i], minLen)
				}

				CreateDecodeTables(s.Limit[t], s.Base[t], s.Perm[t], s.Len[t], int(minLen), int(maxLen), alphaSize)
				s.MinLens[t] = int(minLen)
			}

			// Now the MTF values
			eob := s.NInUse + 1
			groupNo := -1
			groupPos := byte(0)

			for i := 0; i <= 0xFF; i++ {
				s.UnZFTab[i] = 0
			}

			// MTF init
			kk := bzip2state.MTFA_SIZE - 1
			for ii := 256/bzip2state.MTFL_SIZE - 1; ii >= 0; ii-- {
				for jj := bzip2state.MTFL_SIZE - 1; jj >= 0; jj-- {
					s.MTFA[kk] = byte(ii*16 + jj)
					kk--
				}

				s.MTFBase[ii] = kk + 1
			}
			// end MTF init

			nBlock := 0

			// macro: GET_MTF_VAL
			if groupPos == 0 {
				groupNo++
				groupPos = 50
				gSel := s.Selector[groupNo]
				gMinLen = s.MinLens[gSel]
				gLimit = s.Limit[gSel]
				gPerm = s.Perm[gSel]
				gBase = s.Base[gSel]
			}

			gPos := groupPos - 1
			zn := gMinLen
			zvec := 0
			zj := byte(0)
			for zvec = GetBits(gMinLen, s); zvec > gLimit[zn]; zvec = zvec<<1 | int(zj) {
				zn++
				zj = GetBit(s)
			}

			nextSym := gPerm[zvec-gBase[zn]]
			for {
				for nextSym != eob {
					if nextSym == 0 || nextSym == 1 {
						es := -1
						n := 1

						for ok := true; ok; ok = nextSym == 0 || nextSym == 1 {
							switch nextSym {
							case 0:
								es += n
							case 1:
								es += n * 2
							}

							n *= 2

							// macro: GET_MTF_VAL
							if gPos == 0 {
								groupNo++
								gPos = 50
								gSel := s.Selector[groupNo]
								gMinLen = s.MinLens[gSel]
								gLimit = s.Limit[gSel]
								gPerm = s.Perm[gSel]
								gBase = s.Base[gSel]
							}

							gPos--
							zn = gMinLen
							for zvec = GetBits(gMinLen, s); zvec > gLimit[zn]; zvec = zvec<<1 | int(zj) {
								zn++
								zj = GetBit(s)
							}

							nextSym = gPerm[zvec-gBase[zn]]
						}

						es++
						var84 := s.SeqToUnseq[s.MTFA[s.MTFBase[0]]&0xFF]
						s.UnZFTab[var84&0xFF] += es

						for es > 0 {
							bzip2state.TT[nBlock] = int(var84 & 0xFF)
							nBlock++
							es--
						}
					} else {
						// uc = MTF ( nextSym-1 )

						nn := nextSym - 1

						if nn < bzip2state.MTFL_SIZE {
							pp := s.MTFBase[0]
							uc = s.MTFA[pp+nn]

							for nn > 3 {
								z := pp + nn
								s.MTFA[z] = s.MTFA[z-1]
								s.MTFA[z-1] = s.MTFA[z-2]
								s.MTFA[z-2] = s.MTFA[z-3]
								s.MTFA[z-3] = s.MTFA[z-4]
								nn -= 4
							}

							for nn > 0 {
								s.MTFA[pp+nn] = s.MTFA[pp+nn-1]
								nn--
							}

							s.MTFA[pp] = uc
						} else {
							// general case
							lno := nn / bzip2state.MTFL_SIZE
							off := nn % bzip2state.MTFL_SIZE

							pp := s.MTFBase[lno] + off
							uc = s.MTFA[pp]

							for pp > s.MTFBase[lno] {
								s.MTFA[pp] = s.MTFA[pp-1]
								pp--
							}

							s.MTFBase[lno]++

							for lno > 0 {
								s.MTFBase[lno]--
								s.MTFA[s.MTFBase[lno]] = s.MTFA[s.MTFBase[lno-1]+16-1]
								lno--
							}

							s.MTFBase[0]--
							s.MTFA[s.MTFBase[0]] = uc

							if s.MTFBase[0] == 0 {
								kk = bzip2state.MTFA_SIZE - 1

								for ii := 256/bzip2state.MTFL_SIZE - 1; ii >= 0; ii-- {
									for jj := bzip2state.MTFL_SIZE - 1; jj >= 0; jj-- {
										s.MTFA[kk] = s.MTFA[s.MTFBase[ii]+jj]
										kk--
									}

									s.MTFBase[ii] = kk + 1
								}
							}
						}
						// end uc = MTF ( nextSym-1 )

						s.UnZFTab[s.SeqToUnseq[uc&0xFF]&0xFF]++
						bzip2state.TT[nBlock] = int(s.SeqToUnseq[uc&0xFF] & 0xFF)
						nBlock++

						// macro: GET_MTF_VAL
						if gPos == 0 {
							groupNo++
							gPos = 50
							gSel := s.Selector[groupNo]
							gMinLen = s.MinLens[gSel]
							gLimit = s.Limit[gSel]
							gPerm = s.Perm[gSel]
							gBase = s.Base[gSel]
						}

						gPos--
						zn = gMinLen
						for zvec = GetBits(gMinLen, s); zvec > gLimit[zn]; zvec = zvec<<1 | int(zj) {
							zn++
							zj = GetBit(s)
						}
						nextSym = gPerm[zvec-gBase[zn]]
					}
				}

				// Set up cftab to facilitate generation of T^(-1)

				// Actually generate cftab
				s.StateOutLen = 0
				s.StateOutCh = 0
				s.CFTab[0] = 0

				for i := 1; i <= 256; i++ {
					s.CFTab[i] = s.UnZFTab[i-1]
				}

				for i := 1; i <= 256; i++ {
					s.CFTab[i] += s.CFTab[i-1]
				}

				for i := range nBlock {
					uc = byte(bzip2state.TT[i] & 0xFF)
					bzip2state.TT[s.CFTab[uc&0xFF]] |= i << 8
					s.CFTab[uc&0xFF]++
				}

				s.TPos = bzip2state.TT[s.OrigPtr] >> 8
				s.CNBlockUsed = 0

				// macro: BZ_GET_FAST
				s.TPos = bzip2state.TT[s.TPos]
				s.K0 = s.TPos & 0xFF
				s.TPos >>= 0x8
				s.CNBlockUsed++

				s.SaveNBlock = nBlock
				Finish(s)

				if s.CNBlockUsed == s.SaveNBlock+1 && s.StateOutLen == 0 {
					reading = true
					break
				}
				reading = false
				break
			}
		}
		return //nolint:staticcheck // SA4004: outer `for` intentionally runs once (faithful port; see comment above)
	}
}

func GetUnsignedChar(s *bzip2state.BZip2State) byte {
	return byte(GetBits(8, s))
}

func GetBit(s *bzip2state.BZip2State) byte {
	return byte(GetBits(1, s))
}

func GetBits(n int, s *bzip2state.BZip2State) int {
	for s.BsLive < n {
		s.BsBuff = s.BsBuff<<8 | int(s.Stream[s.NextIn])
		s.BsLive += 8

		s.NextIn++
		s.AvailIn--

		s.TotalInLo32++
		if s.TotalInLo32 == 0 {
			s.TotalInHi32++
		}
	}

	// Theme C: BsBuff is Go int (64-bit) but Java's is 32-bit int. The int32()
	// cast is LOAD-BEARING — it makes the right shift drop the high bits exactly
	// as Java's 32-bit `bsBuff >> (bsLive - n)` does. Do not remove it.
	value := int32(s.BsBuff) >> (int32(s.BsLive) - int32(n)) & ((0x1 << int32(n)) - 1)
	s.BsLive -= n
	return int(value)
}

func MakeMaps(s *bzip2state.BZip2State) {
	s.NInUse = 0
	for i := range 256 {
		if s.InUse[i] {
			s.SeqToUnseq[s.NInUse] = byte(i)
			s.NInUse++
		}
	}
}

func CreateDecodeTables(limit []int, base []int, perm []int, length []byte, minLen, maxLen, alphaSize int) {
	pp := 0

	for i := minLen; i <= maxLen; i++ {
		for j := range alphaSize {
			if int(length[j]) == i {
				perm[pp] = j
				pp++
			}
		}
	}

	for i := range bzip2state.BZ_MAX_CODE_LEN {
		base[i] = 0
	}

	for i := range alphaSize {
		base[length[i]+1]++
	}

	for i := 1; i < bzip2state.BZ_MAX_CODE_LEN; i++ {
		base[i] += base[i-1]
	}

	for i := range bzip2state.BZ_MAX_CODE_LEN {
		limit[i] = 0
	}

	vec := 0
	for i := minLen; i <= maxLen; i++ {
		vec += base[i+1] - base[i]
		limit[i] = vec - 1
		vec <<= 1
	}

	for i := minLen + 1; i <= maxLen; i++ {
		base[i] = ((limit[i-1] + 1) << 1) - base[i]
	}
}

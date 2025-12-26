package bzip2

import (
	"fmt"

	"goscape-client/pkg/jagex2/io/bzip2state"
)

var (
	State = bzip2state.NewBZip2State()
)

func Read(arg0 []byte, arg1 int, arg2 []byte, arg3 int, arg4 int) int {
	// TODO: synchronized
	State.Stream = arg2
	State.NextIn = arg4
	State.Decompressed = arg0
	State.NextOut = 0
	State.AvailIn = arg3
	State.AvailOut = arg1
	State.BsLive = 0
	State.BsBuff = 0
	State.TotalInLo32 = 0
	State.TotalInHi32 = 0
	State.TotalOutLo32 = 0
	State.TotalOutHi32 = 0
	State.CurrBlockNo = 0
	Decompress(State)
	return arg1 - State.AvailOut
}

func Finish(arg0 *bzip2state.BZip2State) {
	var2 := arg0.StateOutCh
	var3 := arg0.StateOutLen
	var4 := arg0.CNBlockUsed
	var5 := arg0.K0
	var6 := bzip2state.TT
	var7 := arg0.TPos
	var8 := arg0.Decompressed
	var9 := arg0.NextOut
	var10 := arg0.AvailOut
	var11 := var10
	var12 := arg0.SaveNBlock + 1
label67:
	for {
		if var3 > 0 {
			for {
				if var10 == 0 {
					break label67
				}
				if var3 == 1 {
					if var10 == 0 {
						var3 = 1
						break label67
					}
					var8[var9] = var2
					var9++
					var10--
					break
				}
				var8[var9] = var2
				var3--
				var9++
				var10--
			}
		}
		var14 := true
		var1 := byte(0)
		for var14 {
			var14 = false
			if var4 == var12 {
				var3 = 0
				break label67
			}
			var2 = byte(var5)
			var7 = var6[var7]
			var1 = byte(var7 & 0xFF)
			var7 >>= 0x8
			var4++
			if int(var1) != var5 {
				var5 = int(var1)
				if var10 == 0 {
					var3 = 1
					break label67
				}
				var8[var9] = var2
				var9++
				var10--
				var14 = true
			} else if var4 == var12 {
				if var10 == 0 {
					var3 = 1
					break label67
				}
				var8[var9] = var2
				var9++
				var10--
				var14 = true
			}
		}
		var3 = 2
		var7 = var6[var7]
		var1 = byte(var7 & 0xFF)
		var7 >>= 0x8
		var4++
		if var4 != var12 {
			if int(var1) == var5 {
				var3 = 3
				var7 = var6[var7]
				var1 = byte(var7 & 0xFF)
				var7 >>= 0x8
				var4++
				if var4 != var12 {
					if int(var1) == var5 {
						var7 = var6[var7]
						var1 = byte(var7 & 0xFF)
						var7 >>= 0x8
						var4++
						var3 = int((var1 & 0xFF) + 4)
						var7 = var6[var7]
						var5 = var7 & 0xFF // TODO: java converts to byte?
						var7 >>= 0x8
						var4++
					} else {
						var5 = int(var1)
					}
				}
			} else {
				var5 = int(var1)
			}
		}
	}
	var13 := arg0.TotalOutLo32
	arg0.TotalOutLo32 += var11 - var10
	if arg0.TotalOutLo32 < var13 {
		arg0.TotalOutHi32++
	}
	arg0.StateOutCh = var2
	arg0.StateOutLen = var3
	arg0.CNBlockUsed = var4
	arg0.K0 = var5
	bzip2state.TT = var6
	arg0.TPos = var7
	arg0.Decompressed = var8
	arg0.NextOut = var9
	arg0.AvailOut = var10
}

func Decompress(arg0 *bzip2state.BZip2State) {
	var23 := 0
	var var24 [258]int
	var var25 [258]int
	var var26 [258]int
	arg0.BlockSize100k = 1
	if bzip2state.TT == nil {
		bzip2state.TT = make([]int, arg0.BlockSize100k*100_000)
	}
	var27 := true
	for {
		for var27 {
			var1 := GetUnsignedChar(arg0)
			if var1 == 23 {
				return
			}
			var1 = GetUnsignedChar(arg0)
			var1 = GetUnsignedChar(arg0)
			var1 = GetUnsignedChar(arg0)
			var1 = GetUnsignedChar(arg0)
			var1 = GetUnsignedChar(arg0)
			arg0.CurrBlockNo++
			var1 = GetUnsignedChar(arg0)
			var1 = GetUnsignedChar(arg0)
			var1 = GetUnsignedChar(arg0)
			var1 = GetUnsignedChar(arg0)
			var1 = GetBit(arg0)
			if var1 == 0 {
				arg0.BlockRandomized = false
			} else {
				arg0.BlockRandomized = true
			}
			if arg0.BlockRandomized {
				fmt.Println("PANIC! RANDOMISED BLOCK!")
			}
			arg0.OrigPtr = 0
			var1 = GetUnsignedChar(arg0)
			arg0.OrigPtr = arg0.OrigPtr<<8 | int(var1&0xFF)
			var1 = GetUnsignedChar(arg0)
			arg0.OrigPtr = arg0.OrigPtr<<8 | int(var1&0xFF)
			var1 = GetUnsignedChar(arg0)
			arg0.OrigPtr = arg0.OrigPtr<<8 | int(var1&0xFF)
			for i := range 16 {
				var1 = GetBit(arg0)
				if var1 == 1 {
					arg0.InUse16[i] = true
				} else {
					arg0.InUse16[i] = false
				}
			}
			for i := range 256 {
				arg0.InUse[i] = false
			}
			for i := range 16 {
				if arg0.InUse16[i] {
					for j := range 16 {
						var1 = GetBit(arg0)
						if var1 == 1 {
							arg0.InUse[i*16+j] = true
						}
					}
				}
			}
			MakeMaps(arg0)
			var45 := arg0.NInUse + 2
			var46 := GetBits(3, arg0)
			var47 := GetBits(15, arg0)
			for i := range var47 {
				var43 := 0
				for {
					var1 = GetBit(arg0)
					if var1 == 0 {
						arg0.SelectorMTF[i] = byte(var43)
						break
					}
					var43++
				}
			}
			var28 := make([]byte, 6)
			var30 := 0
			for var30 < var46 {
				var28[var30] = byte(var30)
				var30++
			}
			for i := range var47 {
				var30 = int(arg0.SelectorMTF[i])
				var29 := var28[var30]
				for var30 > 0 {
					var28[var30] = var28[var30-1]
					var30--
				}
				var28[0] = var29
				arg0.Selector[i] = var29
			}
			for i := range var46 {
				var57 := GetBits(5, arg0)
				for j := range var45 {
					for {
						var1 = GetBit(arg0)
						if var1 == 0 {
							arg0.Len[i][j] = byte(var57)
							break
						}
						var1 = GetBit(arg0)
						if var1 == 0 {
							var57++
						} else {
							var57--
						}
					}
				}
			}
			for i := range var46 {
				var2 := byte(32)
				var3 := byte(0)
				for j := range var45 {
					if arg0.Len[i][j] > var3 {
						var3 = arg0.Len[i][j]
					}
					if arg0.Len[i][j] < var2 {
						var2 = arg0.Len[i][j]
					}
				}
				CreateDecodeTables(arg0.Limit[i], arg0.Base[i], arg0.Perm[i], arg0.Len[i], int(var2), int(var3), var45)
				arg0.MinLens[i] = int(var2)
			}
			var48 := arg0.NInUse + 1
			var49 := -1
			var50 := byte(0)
			for i := 0; i <= 255; i++ {
				arg0.UnZFTab[i] = 0
			}
			var33 := 4095
			for i := 15; i >= 0; i-- {
				for j := 15; j >= 0; j-- {
					arg0.MTFA[var33] = byte(i*16 + j)
					var33--
				}
				arg0.MTFBase[i] = var33 + 1
			}
			var54 := 0
			var61 := byte(0)
			if var50 == 0 {
				var49++
				var50 = 50
				var61 = arg0.Selector[var49]
				var23 = arg0.MinLens[var61]
				var24 = arg0.Limit[var61]
				var26 = arg0.Perm[var61]
				var25 = arg0.Base[var61]
			}
			var51 := var50 - 1
			var58 := var23
			var59 := 0
			var60 := byte(0)
			for var59 = GetBits(var23, arg0); var59 > var24[var58]; var59 = var59<<1 | int(var60) { // TODO: my conversion to int
				var58++
				var60 = GetBit(arg0)
			}
			var52 := var26[var59-var25[var58]]
			for {
				for var52 != var48 {
					if var52 == 0 || var52 == 1 {
						var55 := -1
						var56 := 1
						for ok := true; ok; ok = var52 == 0 || var52 == 1 {
							if var52 == 0 {
								var55 += var56
							} else if var52 == 1 {
								var55 += var56 * 2
							}
							var56 *= 2
							if var51 == 0 {
								var49++
								var51 = 50
								var61 = arg0.Selector[var49]
								var23 = arg0.MinLens[var61]
								var24 = arg0.Limit[var61]
								var26 = arg0.Perm[var61]
								var25 = arg0.Base[var61]
							}
							var51--
							var58 = var23
							for var59 = GetBits(var23, arg0); var59 > var24[var58]; var59 = var59<<1 | int(var60) { // TODO: my conversion to int
								var58++
								var60 = GetBit(arg0)
							}
							var52 = var26[var59-var25[var58]]
						}
						var55++
						var1 = arg0.SeqToUnseq[arg0.MTFA[arg0.MTFBase[0]]&0xFF]
						arg0.UnZFTab[var1&0xFF] += var55
						for var55 > 0 {
							bzip2state.TT[var54] = int(var1 & 0xFF)
							var54++
							var55--
						}
					} else {
						var40 := var52 - 1
						var37 := 0
						if var40 < 16 {
							var37 = arg0.MTFBase[0]
							var1 = arg0.MTFA[var37+var40]
							for var40 > 3 {
								var41 := var37 + var40
								arg0.MTFA[var41] = arg0.MTFA[var41-1]
								arg0.MTFA[var41-1] = arg0.MTFA[var41-2]
								arg0.MTFA[var41-2] = arg0.MTFA[var41-3]
								arg0.MTFA[var41-3] = arg0.MTFA[var41-4]
								var40 -= 4
							}
							for var40 > 0 {
								arg0.MTFA[var37+var40] = arg0.MTFA[var37+var40-1]
								var40--
							}
							arg0.MTFA[var37] = var1
						} else {
							var38 := var40 / 16
							var39 := var40 % 16
							var37 = arg0.MTFBase[var38] + var39
							var1 = arg0.MTFA[var37]
							for var37 > arg0.MTFBase[var38] {
								arg0.MTFA[var37] = arg0.MTFA[var37-1]
								var37--
							}
							arg0.MTFBase[var38]++
							for var38 > 0 {
								arg0.MTFBase[var38]--
								arg0.MTFA[arg0.MTFBase[var38]] = arg0.MTFA[arg0.MTFBase[var38-1]+16-1]
								var38--
							}
							arg0.MTFBase[0]--
							arg0.MTFA[arg0.MTFBase[0]] = var1
							if arg0.MTFBase[0] == 0 {
								var36 := 4095
								for i := 15; i >= 0; i-- {
									for j := 15; j >= 0; j-- {
										arg0.MTFA[var36] = arg0.MTFA[arg0.MTFBase[i]+j]
										var36--
									}
									arg0.MTFBase[i] = var36 + 1
								}
							}
						}
						arg0.UnZFTab[arg0.SeqToUnseq[var1&0xFF]&0xFF]++
						bzip2state.TT[var54] = int(arg0.SeqToUnseq[var1&0xFF] & 0xFF)
						var54++
						if var51 == 0 {
							var49++
							var51 = 50
							var61 = arg0.Selector[var49]
							var23 = arg0.MinLens[var61]
							var24 = arg0.Limit[var61]
							var26 = arg0.Perm[var61]
							var25 = arg0.Base[var61]
						}
						var51--
						var58 = var23
						for var59 = GetBits(var23, arg0); var59 > var24[var58]; var59 = var59<<1 | int(var60) { // TODO: my conversion to int
							var58++
							var60 = GetBit(arg0)
						}
						var52 = var26[var59-var25[var58]]
					}
				}
				arg0.StateOutLen = 0
				arg0.StateOutCh = 0
				arg0.CFTab[0] = 0
				for i := 1; i <= 256; i++ {
					arg0.CFTab[i] = arg0.UnZFTab[i-1]
				}
				for i := 1; i <= 256; i++ {
					arg0.CFTab[i] += arg0.CFTab[i-1]
				}
				for i := range var54 {
					var1 = byte(bzip2state.TT[i] & 0xFF)
					bzip2state.TT[arg0.CFTab[var1&0xFF]] |= i << 8
					arg0.CFTab[var1&0xFF]++
				}
				arg0.TPos = bzip2state.TT[arg0.OrigPtr] >> 8
				arg0.CNBlockUsed = 0
				arg0.TPos = bzip2state.TT[arg0.TPos]
				arg0.K0 = int(byte(arg0.TPos & 0xFF)) // TODO: my double conversion
				arg0.TPos >>= 0x8
				arg0.CNBlockUsed++
				arg0.SaveNBlock = var54
				Finish(arg0)
				if arg0.CNBlockUsed == arg0.SaveNBlock+1 && arg0.StateOutLen == 0 {
					var27 = true
					break
				}
				var27 = false
				break
			}
		}
		return
	}
}

func GetUnsignedChar(arg0 *bzip2state.BZip2State) byte {
	return byte(GetBits(8, arg0))
}

func GetBit(arg0 *bzip2state.BZip2State) byte {
	return byte(GetBits(1, arg0))
}

func GetBits(arg0 int, arg1 *bzip2state.BZip2State) int {
	for arg1.BsLive < arg0 {
		arg1.BsBuff = arg1.BsBuff<<8 | int(arg1.Stream[arg1.NextIn]&0xFF) // TODO: my conversion to int
		arg1.BsLive += 8
		arg1.NextIn++
		arg1.AvailIn--
		arg1.TotalInLo32++
		if arg1.TotalInLo32 == 0 {
			arg1.TotalInHi32++
		}
	}
	var3 := int32(arg1.BsBuff) >> (int32(arg1.BsLive) - int32(arg0)) & ((0x1 << int32(arg0)) - 1)
	arg1.BsLive -= arg0
	return int(var3)
}

func MakeMaps(arg0 *bzip2state.BZip2State) {
	arg0.NInUse = 0
	for i := range 256 {
		if arg0.InUse[i] {
			arg0.SeqToUnseq[arg0.NInUse] = byte(i)
			arg0.NInUse++
		}
	}
}

func CreateDecodeTables(arg0 [258]int, arg1 [258]int, arg2 [258]int, arg3 [258]byte, arg4, arg5, arg6 int) {
	var7 := 0
	for i := arg4; i <= arg5; i++ {
		for j := range arg6 {
			if int(arg3[j]) == i {
				arg2[var7] = j
				var7++
			}
		}
	}
	for i := range 23 {
		arg1[i] = 0
	}
	for i := range arg6 {
		arg1[arg3[i]+1]++
	}
	for i := 1; i < 23; i++ {
		arg1[i] += arg1[i-1]
	}
	for i := range 23 {
		arg0[i] = 0
	}
	var10 := 0
	for i := arg4; i <= arg5; i++ {
		var10 += arg1[i+1] - arg1[i]
		arg0[i] = var10 - 1
		var10 <<= 0x1
	}
	for i := arg4 + 1; i <= arg5; i++ {
		arg1[i] = ((arg0[i-1] + 1) << 1) - arg1[i]
	}
}

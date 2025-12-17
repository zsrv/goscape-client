package wordfilter

import (
	"strings"

	"goscape-client/pkg/jagex2/io"
)

var (
	Fragments       []int
	BadWords        [][]rune
	BadCombinations [][][]byte
	Domains         [][]rune
	TLDs            [][]rune
	TLDType         []int
	ALLOWLIST       []string = []string{"cook", "cook's", "cooks", "seeks", "sheet"}
)

func Unpack(arg0 *io.Jagfile) {
	var1 := io.NewPacket(arg0.Read("fragmentsenc.txt", nil))
	var2 := io.NewPacket(arg0.Read("badenc.txt", nil))
	var3 := io.NewPacket(arg0.Read("domainenc.txt", nil))
	var4 := io.NewPacket(arg0.Read("tldlist.txt", nil))
	Read(var1, var2, var3, var4)
}

func Read(arg0, arg1, arg2, arg3 *io.Packet) {
	ReadBadWords(arg1)
	ReadDomains(arg2)
	ReadFragments(arg0)
	ReadTLD(arg3)
}

func ReadTLD(arg1 *io.Packet) {
	var2 := arg1.G4()
	TLDs = make([][]rune, var2)
	TLDType = make([]int, var2)
	for i := range var2 {
		TLDType[i] = arg1.G1()
		var4 := make([]rune, arg1.G1())
		for j := range len(var4) {
			var4[j] = rune(arg1.G1())
		}
		TLDs[i] = var4
	}
}

func ReadBadWords(arg1 *io.Packet) {
	var2 := arg1.G4()
	BadWords = make([][]rune, var2)
	BadCombinations = make([][][]byte, var2)
	ReadBadCombinations(BadCombinations, BadWords, arg1)
}

func ReadDomains(arg0 *io.Packet) {
	var2 := arg0.G4()
	Domains = make([][]rune, var2)
	ReadDomain(arg0, Domains)
}

func ReadFragments(arg1 *io.Packet) {
	Fragments = make([]int, arg1.G4())
	for i := range len(Fragments) {
		Fragments[i] = arg1.G2()
	}
}

func ReadBadCombinations(arg0 [][][]byte, arg1 [][]rune, arg2 *io.Packet) {
	for i := range len(arg1) {
		var5 := make([]rune, arg2.G1())
		for j := range len(var5) {
			var5[j] = rune(arg2.G1())
		}
		arg1[i] = var5
		var7 := make([][]byte, arg2.G1())
		for j := range len(var7) {
			var7[j] = make([]byte, 2)
			var7[j][0] = byte(arg2.G1())
			var7[j][1] = byte(arg2.G1())
		}
		if len(var7) > 0 {
			arg0[i] = var7
		}
	}
}

func ReadDomain(arg1 *io.Packet, arg2 [][]rune) {
	for i := range len(arg2) {
		var4 := make([]rune, arg1.G1())
		for j := range len(var4) {
			var4[j] = rune(arg1.G1())
		}
		arg2[i] = var4
	}
}

func FilterCharacters(arg0 []rune) {
	var2 := 0
	for i := range len(arg0) {
		if AllowCharacter(arg0[i]) {
			arg0[var2] = arg0[i]
		} else {
			arg0[var2] = ' '
		}
		if var2 == 0 || arg0[var2] != ' ' || arg0[var2-1] != ' ' {
			var2++
		}
	}
	for i := var2; i < len(arg0); i++ {
		arg0[i] = ' '
	}
}

func AllowCharacter(arg1 rune) bool {
	return arg1 >= ' ' && arg1 <= 127 || arg1 == ' ' || arg1 == '\n' || arg1 == '\t' || arg1 == 163 || arg1 == 8364
}

func Filter(arg0 string) string {
	var4 := []rune(arg0)
	FilterCharacters(var4)
	var5 := strings.TrimSpace(string(var4))
	var11 := []rune(strings.ToLower(var5))
	var6 := strings.ToLower(var5)
	FilterTLD(var11)
	FilterBad(var11)
	FilterDomains(var11)
	FilterFragments(var11)
	var8 := 0
	for i := range len(ALLOWLIST) {
		var8 = -1
		for var8 = strings.Index(ALLOWLIST[i], var6[var8+1:]); var8 != -1; {
			var9 := []rune(ALLOWLIST[i])
			for j := range len(var9) {
				var11[j+var8] = var9[j]
			}
		}
	}
	ReplaceUpperCases(var11, []rune(var5))
	FormatUpperCases(var11)
	return strings.TrimSpace(string(var11))
}

func ReplaceUpperCases(arg0, arg2 []rune) {
	for i := range len(arg2) {
		if arg0[i] != '*' && IsUpperCase(arg2[i]) {
			arg0[i] = arg2[i]
		}
	}
}

func FormatUpperCases(arg1 []rune) {
	var2 := true
	for i := range len(arg1) {
		var4 := arg1[i]
		if !IsAlpha(var4) {
			var2 = true
		} else if var2 {
			if IsLowerCase(var4) {
				var2 = false
			}
		} else if IsUpperCase(var4) {
			arg1[i] = var4 + 'a' - 65
		}
	}
}

func FilterBad(arg1 []rune) {
	for range 2 {
		for j := len(BadWords) - 1; j >= 0; j-- {
			Filter2(BadCombinations[j], arg1, BadWords[j])
		}
	}
}

func FilterDomains(arg1 []rune) {
	var2 := make([]rune, len(arg1))
	copy(var2, arg1)
	var3 := []rune{'(', 'a', ')'}
	Filter2(nil, var2, var3)
	var4 := make([]rune, len(arg1))
	copy(var4, arg1)
	var5 := []rune{'d', 'o', 't'}
	Filter2(nil, var4, var5)
	for i := len(Domains) - 1; i >= 0; i-- {
		FilterDomain(var4, var2, Domains[i], arg1)
	}
}

func FilterDomain(arg0, arg2, arg3, arg4 []rune) {
	if len(arg3) > len(arg4) {
		return
	}
	var13 := 0
	for i := 0; i <= len(arg4)-len(arg3); i += var13 {
		var7 := i
		var8 := 0
		var13 = 1
		var9 := false
		for {
			if var7 >= len(arg4) {
				break
			}
			var9 = false
			var10 := arg4[var7]
			var11 := rune(0)
			if var7+1 < len(arg4) {
				var11 = arg4[var7+1]
			}
			if var8 < len(arg3) && GetEmulatedDomainCharSize(var11, arg3[var8], var10) > 0 {
				var7 += GetEmulatedDomainCharSize(var11, arg3[var8], var10)
				var8++
			} else {
				if var8 == 0 {
					break
				}
				if GetEmulatedDomainCharSize(var11, arg3[var8-1], var10) > 0 {
					var7 += GetEmulatedDomainCharSize(var11, arg3[var8-1], var10)
					if var8 == 1 {
						var13++
					}
				} else {
					if var8 >= len(arg3) || !IsSymbol(var10) {
						break
					}
					var7++
				}
			}
		}

		if var8 >= len(arg3) {
			var9 = false
			var16 := GetDomainAtFilterStatus(i, arg4, arg2)
			var17 := GetDomainDotFilterStatus(arg0, arg4, var7-1)
			if var16 > 2 || var17 > 2 {
				var9 = true
			}
			if var9 {
				for j := i; j < var7; j++ {
					arg4[j] = '*'
				}
			}
		}
	}
}

func GetDomainAtFilterStatus(arg0 int, arg1 []rune, arg3 []rune) int {
	if arg0 == 0 {
		return 2
	}
	for i := arg0 - 1; i >= 0 && IsSymbol(arg1[i]); i-- {
		if arg1[i] == '@' {
			return 3
		}
	}
	var5 := 0
	for i := arg0 - 1; i >= 0 && IsSymbol(arg3[i]); i-- {
		if arg3[i] == '*' {
			var5++
		}
	}
	if var5 >= 3 {
		return 4
	} else if IsSymbol(arg1[arg0-1]) {
		return 1
	} else {
		return 0
	}
}

func GetDomainDotFilterStatus(arg0 []rune, arg1 []rune, arg2 int) int {
	if arg2+1 == len(arg1) {
		return 2
	}
	var4 := arg2 + 1
	for {
		if var4 < len(arg1) && IsSymbol(arg1[var4]) {
			if arg1[var4] != '.' && arg1[var4] != ',' {
				var4++
				continue
			}
			return 3
		}
		var5 := 0
		for i := arg2 + 1; i < len(arg1) && IsSymbol(arg0[i]); i++ {
			if arg0[i] == '*' {
				var5++
			}
		}
		if var5 >= 3 {
			return 4
		}
		if IsSymbol(arg1[arg2+1]) {
			return 1
		}
		return 0
	}
}

func FilterTLD(arg0 []rune) {
	var2 := make([]rune, len(arg0))
	copy(var2, arg0)
	var3 := []rune{'d', 'o', 't'}
	Filter2(nil, var2, var3)
	var4 := make([]rune, len(arg0))
	copy(var4, arg0)
	var5 := []rune{'s', 'l', 'a', 's', 'h'}
	Filter2(nil, var4, var5)
	for i := range len(TLDs) {
		FilterTLD2(var4, TLDType[i], arg0, TLDs[i], var2)
	}
}

func FilterTLD2(arg0 []rune, arg1 int, arg3, arg4, arg5 []rune) {
	var6 := 0
	if len(arg4) > len(arg3) {
		return
	}
	for i := 0; i <= len(arg3)-len(arg4); i += var6 {
		var8 := i
		var9 := 0
		var6 = 1
		var10 := false
		for {
			if var8 >= len(arg3) {
				break
			}
			var10 = false
			var11 := arg3[var8]
			var12 := rune(0)
			if var8+1 < len(arg3) {
				var12 = arg3[var8+1]
			}
			if var9 < len(arg4) && GetEmulatedDomainCharSize(var12, arg4[var9], var11) > 0 {
				var8 += GetEmulatedDomainCharSize(var12, arg4[var9], var11)
				var9++
			} else {
				if var9 == 0 {
					break
				}
				if GetEmulatedDomainCharSize(var12, arg4[var9-1], var11) > 0 {
					var8 += GetEmulatedDomainCharSize(var12, arg4[var9-1], var11)
					if var9 == 1 {
						var6++
					}
				} else {
					if var9 >= len(arg4) || !IsSymbol(var11) {
						break
					}
					var8++
				}
			}
		}
		if var9 >= len(arg4) {
			var10 = false
			var20 := GetTLDDotFilterStatus(arg3, arg5, i)
			var21 := GetTLDSlashFilterStatus(arg0, -678, var8-1, arg3)
			if arg1 == 1 && var20 > 0 && var21 > 0 {
				var10 = true
			}
			if arg1 == 2 && (var20 > 2 && var21 > 0 || var20 > 0 && var21 > 2) {
				var10 = true
			}
			if arg1 == 3 && var20 > 0 && var21 > 2 {
				var10 = true
			}
			if var10 {
				var13 := i
				var14 := var8 - 1
				var15 := false
				if var20 > 2 {
					if var20 == 4 {
						var15 = false
						for j := i - 1; j >= 0; j-- {
							if var15 {
								if arg5[j] != '*' {
									break
								}
								var13 = j
							} else if arg5[j] == '*' {
								var13 = j
								var15 = true
							}
						}
					}
					var15 = false
					for j := var13 - 1; j >= 0; j-- {
						if var15 {
							if IsSymbol(arg3[j]) {
								break
							}
							var13 = j
						} else if !IsSymbol(arg3[j]) {
							var15 = true
							var13 = j
						}
					}
				}
				if var21 > 2 {
					if var21 == 4 {
						var15 = false
						for j := var14 + 1; j < len(arg3); j++ {
							if var15 {
								if arg0[j] != '*' {
									break
								}
								var14 = j
							} else if arg0[j] == '*' {
								var14 = j
								var15 = true
							}
						}
					}
					var15 = false
					for j := var14 + 1; j < len(arg3); j++ {
						if var15 {
							if IsSymbol(arg3[j]) {
								break
							}
							var14 = j
						} else if !IsSymbol(arg3[j]) {
							var15 = true
							var14 = j
						}
					}
				}
				for j := var13; j <= var14; j++ {
					arg3[j] = '*'
				}
			}
		}
	}
}

func GetTLDDotFilterStatus(arg0 []rune, arg2 []rune, arg3 int) int {
	if arg3 == 0 {
		return 2
	}
	var4 := arg3 - 1
	for {
		if var4 >= 0 && IsSymbol(arg0[var4]) {
			if arg0[var4] != ',' && arg0[var4] != '.' {
				var4--
				continue
			}
			return 3
		}
		var5 := 0
		for i := arg3 - 1; i >= 0 && IsSymbol(arg2[i]); i-- {
			if arg2[i] == '*' {
				var5++
			}
		}
		if var5 >= 3 {
			return 4
		}
		if IsSymbol(arg0[arg3-1]) {
			return 1
		}
		return 0
	}
}

func GetTLDSlashFilterStatus(arg0 []rune, arg1 int, arg2 int, arg3 []rune) int {
	if arg2+1 == len(arg3) {
		return 2
	}
	var4 := arg2 + 1
	for {
		if var4 < len(arg3) && IsSymbol(arg3[var4]) {
			if arg3[var4] != '\\' && arg3[var4] != '/' {
				var4++
				continue
			}
			return 3
		}
		var5 := 0
		for i := arg2 + 1; i < len(arg3) && IsSymbol(arg0[i]); i++ {
			if arg0[i] == '*' {
				var5++
			}
		}
		if arg1 >= 0 {
			return 3
		}
		if var5 >= 5 {
			return 4
		}
		if IsSymbol(arg3[arg2+1]) {
			return 1
		}
		return 0
	}
}

func Filter2(arg1 [][]byte, arg2 []rune, arg3 []rune) {
	if len(arg3) > len(arg2) {
		return
	}
	var20 := 0
	for i := 0; i <= len(arg2)-len(arg3); i += var20 {
		var6 := i
		var7 := 0
		var8 := 0
		var20 = 1
		var9 := false
		var10 := false
		var11 := false
		var12 := false
		var13 := rune(0)
		var14 := rune(0)
		for {
			if var6 >= len(arg2) || var10 && var11 {
				break
			}
			var12 = false
			var13 = arg2[var6]
			var14 = 0
			if var6+1 < len(arg2) {
				var14 = arg2[var6+1]
			}
			if var7 < len(arg3) && GetEmulatedSize(var14, arg3[var7], var13) > 0 {
				var21 := GetEmulatedSize(var14, arg3[var7], var13)
				if var21 == 1 && IsNumber(var13) {
					var10 = true
				}
				if var21 == 2 && (IsNumber(var13) || IsNumber(var14)) {
					var10 = true
				}
				var6 += var21
				var7++
			} else {
				if var7 == 0 {
					break
				}
				if GetEmulatedSize(var14, arg3[var7-1], var13) > 0 {
					var6 += GetEmulatedSize(var14, arg3[var7-1], var13)
					if var7 == 1 {
						var20++
					}
				} else {
					if var7 >= len(arg3) || !IsLowerCaseAlpha(var13) {
						break
					}
					if IsSymbol(var13) && var13 != '\'' {
						var9 = true
					}
					if IsNumber(var13) {
						var11 = true
					}
					var6++
					var8++
					if var8*100/(var6-i) > 90 {
						break
					}
				}
			}
		}
		if var7 >= len(arg3) && (!var10 || !var11) {
			var12 = true
			var28 := 0
			if var9 {
				var23 := false
				var27 := false
				if i-1 < 0 || IsSymbol(arg2[i-1]) && arg2[i-1] != '\'' {
					var23 = true
				}
				if var6 >= len(arg2) || IsSymbol(arg2[var6]) && arg2[var6] != '\'' {
					var27 = true
				}
				if !var23 || !var27 {
					var24 := false
					var28 = i - 2
					if var23 {
						var28 = i
					}
					for !var24 && var28 < var6 {
						if var28 >= 0 && (!IsSymbol(arg2[var28]) || arg2[var28] == '\'') {
							var17 := make([]rune, 3)
							var18 := 0
							for var18 = 0; var18 < 3 && var28+var18 < len(arg2) && (!IsSymbol(arg2[var28+var18]) || arg2[var28+var18] == '\''); var18++ {
								var17[var18] = arg2[var28+var18]
							}
							var19 := true
							if var18 == 0 {
								var19 = false
							}
							if var18 < 3 && var28-1 >= 0 && (!IsSymbol(arg2[var28-1]) || arg2[var28-1] == '\'') {
								var19 = false
							}
							if var19 && !IsBadFragment(var17) {
								var24 = true
							}
						}
						var28++
					}
					if !var24 {
						var12 = false
					}
				}
			} else {
				var13 = ' '
				if i-1 >= 0 {
					var13 = arg2[i-1]
				}
				var14 = ' '
				if var6 < len(arg2) {
					var14 = arg2[var6]
				}
				var15 := GetIndex(var13)
				var16 := GetIndex(var14)
				if arg1 != nil && ComboMatches(var15, arg1, var16) {
					var12 = false
				}
			}
			if var12 {
				var25 := 0
				var29 := 0
				for j := i; j < var6; j++ {
					if IsNumber(arg2[j]) {
						var25++
					} else if IsAlpha(arg2[j]) {
						var29++
					}
				}
				if var25 <= var29 {
					for j := i; j < var6; j++ {
						arg2[j] = '*'
					}
				}
			}
		}
	}
}

func ComboMatches(arg1 byte, arg2 [][]byte, arg3 byte) bool {
	var4 := 0
	if arg2[var4][0] == arg1 && arg2[var4][1] == arg3 {
		return true
	}
	var5 := len(arg2) - 1
	if arg2[var5][0] == arg1 && arg2[var5][1] == arg3 {
		return true
	}
	for ok := true; ok; ok = var4 != var5 && var4+1 != var5 {
		var6 := (var4 + var5) / 2
		if arg2[var6][0] == arg1 && arg2[var6][1] == arg3 {
			return true
		}
		if arg1 < arg2[var6][0] || arg1 == arg2[var6][0] && arg3 < arg2[var6][1] {
			var5 = var6
		} else {
			var4 = var6
		}
	}
	return false
}

func GetEmulatedDomainCharSize(arg1, arg2, arg3 rune) int {
	if arg2 == arg3 {
		return 1
	}
	if arg2 == 'o' && arg3 == '0' {
		return 1
	}
	if arg2 == 'o' && arg3 == '(' && arg1 == ')' {
		return 2
	}
	if arg2 == 'c' && (arg3 == '(' || arg3 == '<' || arg3 == '[') {
		return 1
	}
	if arg2 == 'e' && arg3 == 8364 {
		return 1
	}
	if arg2 == 's' && arg3 == '$' {
		return 1
	}
	if arg2 == 'l' && arg3 == 'i' {
		return 1
	}
	return 0
}

func GetEmulatedSize(arg0, arg1, arg2 rune) int {
	if arg1 == arg2 {
		return 1
	}
	if arg1 >= 'a' && arg1 <= 'm' {
		if arg1 == 'a' {
			if arg2 != '4' && arg2 != '@' && arg2 != '^' {
				if arg2 == '/' && arg0 == '\\' {
					return 2
				}
				return 0
			}
			return 1
		}
		if arg1 == 'b' {
			if arg2 != '6' && arg2 != '8' {
				if arg2 == '1' && arg0 == '3' {
					return 2
				}
				return 0
			}
			return 1
		}
		if arg1 == 'c' {
			if arg2 != '(' && arg2 != '<' && arg2 != '{' && arg2 != '[' {
				return 0
			}
			return 1
		}
		if arg1 == 'd' {
			if arg2 == '[' && arg0 == ')' {
				return 2
			}
			return 0
		}
		if arg1 == 'e' {
			if arg2 != '3' && arg2 != 8364 {
				return 0
			}
			return 1
		}
		if arg1 == 'f' {
			if arg2 == 'p' && arg0 == 'h' {
				return 2
			}
			if arg2 == 163 {
				return 1
			}
			return 0
		}
		if arg1 == 'g' {
			if arg2 != '9' && arg2 != '6' {
				return 0
			}
			return 1
		}
		if arg1 == 'h' {
			if arg2 == '#' {
				return 1
			}
			return 0
		}
		if arg1 == 'i' {
			if arg2 != 'y' && arg2 != 'l' && arg2 != 'j' && arg2 != '1' && arg2 != '!' && arg2 != ':' && arg2 != ';' && arg2 != '|' {
				return 0
			}
			return 1
		}
		if arg1 == 'j' {
			return 0
		}
		if arg1 == 'k' {
			return 0
		}
		if arg1 == 'l' {
			if arg2 != '1' && arg2 != '|' && arg2 != 'i' {
				return 0
			}
			return 1
		}
		if arg1 == 'm' {
			return 0
		}
	}
	if arg1 >= 'n' && arg1 <= 'z' {
		if arg1 == 'n' {
			return 0
		}
		if arg1 == 'o' {
			if arg2 != '0' && arg2 != '*' {
				if (arg2 != '(' || arg0 != ')') && (arg2 != '[' || arg0 != ']') && (arg2 != '{' || arg0 != '}') && (arg2 != '<' || arg0 != '>') {
					return 0
				}
				return 2
			}
			return 1
		}
		if arg1 == 'p' {
			return 0
		}
		if arg1 == 'q' {
			return 0
		}
		if arg1 == 'r' {
			return 0
		}
		if arg1 == 's' {
			if arg2 != '5' && arg2 != 'z' && arg2 != '$' && arg2 != '2' {
				return 0
			}
			return 1
		}
		if arg1 == 't' {
			if arg2 != '7' && arg2 != '+' {
				return 0
			}
			return 1
		}
		if arg1 == 'u' {
			if arg2 == 'v' {
				return 1
			}
			if arg2 == '\\' && arg0 == '/' || arg2 == '\\' && arg0 == '|' || arg2 == '|' && arg0 == '/' {
				return 2
			}
			return 0
		}
		if arg1 == 'v' {
			if (arg2 != '\\' || arg0 != '/') && (arg2 != '\\' || arg0 != '|') && (arg2 != '|' || arg0 != '/') {
				return 0
			}
			return 2
		}
		if arg1 == 'w' {
			if arg2 == 'v' && arg0 == 'v' {
				return 2
			}
			return 0
		}
		if arg1 == 'x' {
			if (arg2 != ')' || arg0 != '(') && (arg2 != '}' || arg0 != '{') && (arg2 != ']' || arg0 != '[') && (arg2 != '>' || arg0 != '<') {
				return 0
			}
			return 2
		}
		if arg1 == 'y' {
			return 0
		}
		if arg1 == 'z' {
			return 0
		}
	}
	if arg1 >= '0' && arg1 <= '9' {
		if arg1 == '0' {
			if arg2 == 'o' || arg2 == 'O' {
				return 1
			}
			if (arg2 != '(' || arg0 != ')') && (arg2 != '{' || arg0 != '}') && (arg2 != '[' || arg0 != ']') {
				return 0
			}
			return 2
		}
		if arg1 == '1' {
			if arg2 == 'l' {
				return 1
			}
			return 0
		}
		return 0
	}
	if arg1 == ',' {
		if arg2 == '.' {
			return 1
		}
		return 0
	}
	if arg1 == '.' {
		if arg2 == ',' {
			return 1
		}
		return 0
	}
	if arg1 == '!' {
		if arg2 == 'i' {
			return 1
		}
		return 0
	}
	return 0
}

func GetIndex(arg1 rune) byte {
	if arg1 >= 'a' && arg1 <= 'z' {
		return byte(arg1 - 'a' + 1)
	}
	if arg1 == '\'' {
		return 28
	}
	if arg1 >= '0' && arg1 <= '9' {
		return byte(arg1 - '0' + 29)
	}
	return 27
}

func FilterFragments(arg1 []rune) {
	var3 := 0
	var4 := 0
	var5 := 0
	for {
		for ok := true; ok; ok = var4 != 4 {
			var11 := IndexOfNumber(arg1, var3)
			if var11 == -1 {
				return
			}
			var6 := false
			for i := var3; i >= 0 && i < var11 && !var6; i++ {
				if !IsSymbol(arg1[i]) && !IsLowerCaseAlpha(arg1[i]) {
					var6 = true
				}
			}
			if var6 {
				var4 = 0
			}
			if var4 == 0 {
				var5 = var11
			}
			var3 = IndexOfNonNumber(var11, arg1)
			var8 := 0
			for i := var11; i < var3; i++ {
				var8 = var8*10 + int(arg1[i]) - 48
			}
			if var8 <= 255 && var3-var11 <= 8 {
				var4++
			} else {
				var4 = 0
			}
		}
		for i := var5; i < var3; i++ {
			arg1[i] = '*'
		}
		var4 = 0
	}
}

func IndexOfNumber(arg1 []rune, arg2 int) int {
	for i := arg2; i < len(arg1) && i >= 0; i++ {
		if arg1[i] >= '0' && arg1[i] <= '9' {
			return i
		}
	}
	return -1
}

func IndexOfNonNumber(arg1 int, arg2 []rune) int {
	var3 := arg1
	for {
		if var3 < len(arg2) && var3 >= 0 {
			if arg2[var3] >= '0' && arg2[var3] <= '9' {
				var3++
				continue
			}
			return var3
		}
		return len(arg2)
	}
}

func IsSymbol(arg0 rune) bool {
	return !IsAlpha(arg0) && !IsNumber(arg0)
}

func IsLowerCaseAlpha(arg0 rune) bool {
	if arg0 >= 'a' && arg0 <= 'z' {
		return arg0 == 'v' || arg0 == 'x' || arg0 == 'j' || arg0 == 'q' || arg0 == 'z'
	}
	return true
}

func IsAlpha(arg1 rune) bool {
	return arg1 >= 'a' && arg1 <= 'z' || arg1 >= 'A' && arg1 <= 'Z'
}

func IsNumber(arg0 rune) bool {
	return arg0 >= '0' && arg0 <= '9'
}

func IsLowerCase(arg1 rune) bool {
	return arg1 >= 'a' && arg1 <= 'z'
}

func IsUpperCase(arg1 rune) bool {
	return arg1 >= 'A' && arg1 <= 'Z'
}

func IsBadFragment(arg0 []rune) bool {
	var2 := true
	for i := range len(arg0) {
		if !IsNumber(arg0[i]) && arg0[i] != 0 {
			var2 = false
		}
	}
	if var2 {
		return true
	}
	var4 := FirstFragmentID(arg0)
	var5 := 0
	var6 := len(Fragments) - 1
	if var4 == Fragments[var5] || var4 == Fragments[var6] {
		return true
	}
	for ok := true; ok; ok = var5 != var6 && var5+1 != var6 {
		var7 := (var5 + var6) / 2
		if var4 == Fragments[var7] {
			return true
		}
		if var4 < Fragments[var7] {
			var6 = var7
		} else {
			var5 = var7
		}
	}
	return false
}

func FirstFragmentID(arg1 []rune) int {
	if len(arg1) > 6 {
		return 0
	}
	var2 := 0
	for i := range len(arg1) {
		var4 := arg1[len(arg1)-i-1]
		if var4 >= 'a' && var4 <= 'z' {
			var2 = var2*38 + int(var4) - 'a' + 1
		} else if var4 == '\'' {
			var2 = var2*38 + 27
		} else if var4 >= '0' && var4 <= '9' {
			var2 = var2*38 + int(var4) - '0' + 28
		} else if var4 != 0 {
			return 0
		}
	}
	return var2
}

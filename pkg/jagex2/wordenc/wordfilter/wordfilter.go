package wordfilter

import (
	"strings"

	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

var (
	Fragments       []int
	BadWords        [][]rune
	BadCombinations [][][]int8
	Domains         [][]rune
	TLDs            [][]rune
	TLDType         []int
	// Java: WordFilter.java:29 — 244 added "woop"/"woops" (225 had 5 entries).
	ALLOWLIST []string = []string{"cook", "cook's", "cooks", "seeks", "sheet", "woop", "woops"}
)

func Unpack(jag *io.JagFile) {
	fragments := io.NewPacket(jag.Read("fragmentsenc.txt", nil))
	bad := io.NewPacket(jag.Read("badenc.txt", nil))
	domain := io.NewPacket(jag.Read("domainenc.txt", nil))
	tld := io.NewPacket(jag.Read("tldlist.txt", nil))
	Read(fragments, bad, domain, tld)
}

// DecodeAll
func Read(fragments, bad, domain, tld *io.Packet) {
	ReadBadWords(bad)
	ReadDomains(domain)
	ReadFragments(fragments)
	ReadTLD(tld)
}

// DecodeTldsTxt
func ReadTLD(buf *io.Packet) {
	count := buf.G4()
	TLDs = make([][]rune, count)
	TLDType = make([]int, count)

	for i := range count {
		TLDType[i] = buf.G1()

		tld := make([]rune, buf.G1())
		for j := range len(tld) {
			tld[j] = rune(buf.G1())
		}

		TLDs[i] = tld
	}
}

// DecodeBadWordsTxt
func ReadBadWords(buf *io.Packet) {
	count := buf.G4()
	BadWords = make([][]rune, count)
	BadCombinations = make([][][]int8, count)

	ReadBadCombinations(BadCombinations, BadWords, buf)
}

// DecodeDomainsTxt
func ReadDomains(buf *io.Packet) {
	count := buf.G4()
	Domains = make([][]rune, count)

	ReadDomain(buf, Domains)
}

// DecodeFragmentsTxt
func ReadFragments(buf *io.Packet) {
	Fragments = make([]int, buf.G4())
	for i := range len(Fragments) {
		Fragments[i] = buf.G2()
	}
}

// DecodeBadCombinations
func ReadBadCombinations(badCombinations [][][]int8, badWords [][]rune, buf *io.Packet) {
	for i := range len(badWords) {
		badWord := make([]rune, buf.G1())
		for j := range len(badWord) {
			badWord[j] = rune(buf.G1())
		}

		badWords[i] = badWord

		combination := make([][]int8, buf.G1())
		for j := range len(combination) {
			combination[j] = make([]int8, 2)
			// Java: combination[j][0] = (byte) buf.g1() (WordFilter.java:96-97) —
			// signed byte storage so values >127 read back negative, matching
			// the signed comparison in ComboMatches.
			combination[j][0] = int8(buf.G1())
			combination[j][1] = int8(buf.G1())
		}

		if len(combination) > 0 {
			badCombinations[i] = combination
		}
	}
}

// DecodeDomains
func ReadDomain(buf *io.Packet, domains [][]rune) {
	for i := range len(domains) {
		domain := make([]rune, buf.G1())
		for j := range len(domain) {
			domain[j] = rune(buf.G1())
		}

		domains[i] = domain
	}
}

func FilterCharacters(in []rune) {
	pos := 0
	for i := range len(in) {
		if AllowCharacter(in[i]) {
			in[pos] = in[i]
		} else {
			in[pos] = ' '
		}

		if pos == 0 || in[pos] != ' ' || in[pos-1] != ' ' {
			pos++
		}
	}

	for i := pos; i < len(in); i++ {
		in[i] = ' '
	}
}

func AllowCharacter(c rune) bool {
	return c >= ' ' && c <= 127 || c == ' ' || c == '\n' || c == '\t' || c == 163 || c == 8364
}

// indexOfRunesFrom mirrors Java String.indexOf(String, int) for rune slices:
// it searches haystack for the first occurrence of needle starting at index
// from, and returns the haystack index where needle starts (or -1).
func indexOfRunesFrom(haystack, needle []rune, from int) int {
	if from < 0 {
		from = 0
	}
	if len(needle) == 0 {
		if from > len(haystack) {
			return len(haystack)
		}
		return from
	}
	last := len(haystack) - len(needle)
	for i := from; i <= last; i++ {
		match := true
		for k := range len(needle) {
			if haystack[i+k] != needle[k] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func Filter(input string) string {
	outputPre := []rune(input)
	FilterCharacters(outputPre)

	trimmed := strings.TrimSpace(string(outputPre))
	output := []rune(strings.ToLower(trimmed))
	// lowercase mirrors output in rune-index space so positions returned by
	// indexOfRunesFrom can be used directly to write into output.
	lowercase := append([]rune(nil), output...)

	FilterTLD(output)
	FilterBad(output)
	FilterDomains(output)
	FilterFragments(output)

	// Java: var6.indexOf(ALLOWLIST[i], var8 + 1) — find needle ALLOWLIST[i]
	// inside haystack var6 (lowercase) starting at var8+1
	// (WordFilter.java:152-160).
	for i := range len(ALLOWLIST) {
		needle := []rune(ALLOWLIST[i])
		j := -1
		for {
			j = indexOfRunesFrom(lowercase, needle, j+1)
			if j == -1 {
				break
			}
			for k := range len(needle) {
				output[j+k] = needle[k]
			}
		}
	}

	ReplaceUpperCases(output, []rune(trimmed))
	FormatUpperCases(output)

	return strings.TrimSpace(string(output))
}

// ReplaceUppercase
func ReplaceUpperCases(in, unfiltered []rune) {
	for i := range len(unfiltered) {
		if in[i] != '*' && IsUpperCase(unfiltered[i]) {
			in[i] = unfiltered[i]
		}
	}
}

// FormatUppercase
func FormatUpperCases(in []rune) {
	upper := true

	for i := range len(in) {
		c := in[i]

		if !IsAlpha(c) {
			upper = true
		} else if upper {
			if IsLowerCase(c) {
				upper = false
			}
		} else if IsUpperCase(c) {
			in[i] = c + 'a' - 65
		}
	}
}

func FilterBad(in []rune) {
	for range 2 { // passes
		for i := len(BadWords) - 1; i >= 0; i-- {
			Filter2(BadCombinations[i], in, BadWords[i])
		}
	}
}

func FilterDomains(in []rune) {
	filteredAt := make([]rune, len(in))
	copy(filteredAt, in)
	at := []rune{'(', 'a', ')'}
	Filter2(nil, filteredAt, at)

	filteredDot := make([]rune, len(in))
	copy(filteredDot, in)
	dot := []rune{'d', 'o', 't'}
	Filter2(nil, filteredDot, dot)

	for i := len(Domains) - 1; i >= 0; i-- {
		FilterDomain(filteredDot, filteredAt, Domains[i], in)
	}
}

func FilterDomain(arg0, arg2, domain, in []rune) {
	if len(domain) > len(in) {
		return
	}

	stride := 0
	for start := 0; start <= len(in)-len(domain); start += stride {
		end := start
		offset := 0
		stride = 1

		match := false
		for {
			if end >= len(in) { //nolint:staticcheck // QF1006: explicit break mirrors Java while(true){ if(...) break }
				break
			}

			match = false
			b := in[end]
			c := rune(0)
			if end+1 < len(in) {
				c = in[end+1]
			}

			if offset < len(domain) && GetEmulatedDomainCharSize(c, domain[offset], b) > 0 {
				end += GetEmulatedDomainCharSize(c, domain[offset], b)
				offset++
			} else {
				if offset == 0 {
					break
				}

				charSize2 := GetEmulatedDomainCharSize(c, domain[offset-1], b)
				if charSize2 > 0 {
					end += charSize2

					if offset == 1 {
						stride++
					}
				} else {
					if offset >= len(domain) || !IsSymbol(b) {
						break
					}

					end++
				}
			}
		}

		if offset >= len(domain) {
			match = false
			atFilter := GetDomainAtFilterStatus(start, in, arg2)
			dotFilter := GetDomainDotFilterStatus(arg0, in, end-1)

			if atFilter > 2 || dotFilter > 2 {
				match = true
			}

			if match {
				for i := start; i < end; i++ {
					in[i] = '*'
				}
			}
		}
	}
}

func GetDomainAtFilterStatus(end int, a []rune, b []rune) int {
	if end == 0 {
		return 2
	}

	for i := end - 1; i >= 0 && IsSymbol(a[i]); i-- {
		if a[i] == '@' {
			return 3
		}
	}

	asteriskCount := 0
	for i := end - 1; i >= 0 && IsSymbol(b[i]); i-- {
		if b[i] == '*' {
			asteriskCount++
		}
	}

	if asteriskCount >= 3 {
		return 4
	} else if IsSymbol(a[end-1]) {
		return 1
	} else {
		return 0
	}
}

func GetDomainDotFilterStatus(b []rune, a []rune, start int) int {
	if start+1 == len(a) {
		return 2
	}

	i := start + 1
	for {
		if i < len(a) && IsSymbol(a[i]) {
			if a[i] != '.' && a[i] != ',' {
				i++
				continue
			}

			return 3
		}

		asteriskCount := 0
		for j := start + 1; j < len(a) && IsSymbol(b[j]); j++ {
			if b[j] == '*' {
				asteriskCount++
			}
		}

		if asteriskCount >= 3 {
			return 4
		}
		if IsSymbol(a[start+1]) {
			return 1
		}
		return 0
	}
}

func FilterTLD(in []rune) {
	filteredDot := make([]rune, len(in))
	copy(filteredDot, in)

	dot := []rune{'d', 'o', 't'}
	Filter2(nil, filteredDot, dot)

	filteredSlash := make([]rune, len(in))
	copy(filteredSlash, in)

	slash := []rune{'s', 'l', 'a', 's', 'h'}
	Filter2(nil, filteredSlash, slash)

	for i := range len(TLDs) {
		FilterTLD2(filteredSlash, TLDType[i], in, TLDs[i], filteredDot)
	}
}

func FilterTLD2(filteredSlash []rune, typ int, chars, tld, filteredDot []rune) {
	if len(tld) > len(chars) {
		return
	}

	stride := 0
	for start := 0; start <= len(chars)-len(tld); start += stride {
		end := start
		offset := 0
		stride = 1
		match := false

		for {
			if end >= len(chars) { //nolint:staticcheck // QF1006: explicit break mirrors Java while(true){ if(...) break }
				break
			}

			match = false
			b := chars[end]
			c := rune(0)

			if end+1 < len(chars) {
				c = chars[end+1]
			}

			if offset < len(tld) && GetEmulatedDomainCharSize(c, tld[offset], b) > 0 {
				end += GetEmulatedDomainCharSize(c, tld[offset], b)
				offset++
			} else {
				if offset == 0 {
					break
				}

				charLen2 := GetEmulatedDomainCharSize(c, tld[offset-1], b)
				if charLen2 > 0 {
					end += charLen2

					if offset == 1 {
						stride++
					}
				} else {
					if offset >= len(tld) || !IsSymbol(b) {
						break
					}

					end++
				}
			}
		}

		if offset >= len(tld) {
			match = false

			status0 := GetTLDDotFilterStatus(chars, filteredDot, start)
			status1 := GetTLDSlashFilterStatus(filteredSlash, end-1, chars)

			if typ == 1 && status0 > 0 && status1 > 0 {
				match = true
			}

			if typ == 2 && (status0 > 2 && status1 > 0 || status0 > 0 && status1 > 2) {
				match = true
			}

			if typ == 3 && status0 > 0 && status1 > 2 {
				match = true
			}

			if match {
				first := start
				last := end - 1

				if status0 > 2 {
					if status0 == 4 {
						findStart := false
						for i := start - 1; i >= 0; i-- {
							if findStart {
								if filteredDot[i] != '*' {
									break
								}

								first = i
							} else if filteredDot[i] == '*' {
								first = i
								findStart = true
							}
						}
					}

					findStart := false
					for i := first - 1; i >= 0; i-- {
						if findStart {
							if IsSymbol(chars[i]) {
								break
							}

							first = i
						} else if !IsSymbol(chars[i]) {
							findStart = true
							first = i
						}
					}
				}

				if status1 > 2 {
					if status1 == 4 {
						findStart := false
						for i := last + 1; i < len(chars); i++ {
							if findStart {
								if filteredSlash[i] != '*' {
									break
								}

								last = i
							} else if filteredSlash[i] == '*' {
								last = i
								findStart = true
							}
						}
					}

					findStart := false
					for i := last + 1; i < len(chars); i++ {
						if findStart {
							if IsSymbol(chars[i]) {
								break
							}

							last = i
						} else if !IsSymbol(chars[i]) {
							findStart = true
							last = i
						}
					}
				}

				for j := first; j <= last; j++ {
					chars[j] = '*'
				}
			}
		}
	}
}

func GetTLDDotFilterStatus(a []rune, b []rune, start int) int {
	if start == 0 {
		return 2
	}

	i := start - 1
	for {
		if i >= 0 && IsSymbol(a[i]) {
			if a[i] != ',' && a[i] != '.' {
				i--
				continue
			}

			return 3
		}

		asteriskCount := 0
		for j := start - 1; j >= 0 && IsSymbol(b[j]); j-- {
			if b[j] == '*' {
				asteriskCount++
			}
		}

		if asteriskCount >= 3 {
			return 4
		}
		if IsSymbol(a[start-1]) {
			return 1
		}
		return 0
	}
}

func GetTLDSlashFilterStatus(b []rune, start int, a []rune) int {
	if start+1 == len(a) {
		return 2
	}

	i := start + 1
	for {
		if i < len(a) && IsSymbol(a[i]) {
			if a[i] != '\\' && a[i] != '/' {
				i++
				continue
			}
			return 3
		}

		asteriskCount := 0
		for j := start + 1; j < len(a) && IsSymbol(b[j]); j++ {
			if b[j] == '*' {
				asteriskCount++
			}
		}

		if asteriskCount >= 5 {
			return 4
		}
		if IsSymbol(a[start+1]) {
			return 1
		}
		return 0
	}
}

func Filter2(badCombinations [][]int8, chars []rune, fragment []rune) {
	if len(fragment) > len(chars) {
		return
	}

	stride := 0
	for start := 0; start <= len(chars)-len(fragment); start += stride {
		end := start
		fragOff := 0
		iterations := 0
		stride = 1

		isSymbol := false
		isEmulated := false
		isNumeral := false

		bad := false
		b := rune(0)
		c := rune(0)
		for {
			if end >= len(chars) || isEmulated && isNumeral { //nolint:staticcheck // QF1006: explicit break mirrors Java while(true){ if(...) break }
				break
			}

			bad = false
			b = chars[end]
			c = 0

			if end+1 < len(chars) {
				c = chars[end+1]
			}

			if fragOff < len(fragment) && GetEmulatedSize(c, fragment[fragOff], b) > 0 {
				charLen := GetEmulatedSize(c, fragment[fragOff], b)

				if charLen == 1 && IsNumber(b) {
					isEmulated = true
				}

				if charLen == 2 && (IsNumber(b) || IsNumber(c)) {
					isEmulated = true
				}

				end += charLen
				fragOff++
			} else {
				if fragOff == 0 {
					break
				}

				if GetEmulatedSize(c, fragment[fragOff-1], b) > 0 {
					end += GetEmulatedSize(c, fragment[fragOff-1], b)

					if fragOff == 1 {
						stride++
					}
				} else {
					if fragOff >= len(fragment) || !IsLowerCaseAlpha(b) {
						break
					}

					if IsSymbol(b) && b != '\'' {
						isSymbol = true
					}

					if IsNumber(b) {
						isNumeral = true
					}

					end++
					iterations++

					if iterations*100/(end-start) > 90 {
						break
					}
				}
			}
		}

		if fragOff >= len(fragment) && (!isEmulated || !isNumeral) {
			bad = true

			if isSymbol {
				badCurrent := false
				badNext := false

				if start-1 < 0 || IsSymbol(chars[start-1]) && chars[start-1] != '\'' {
					badCurrent = true
				}

				if end >= len(chars) || IsSymbol(chars[end]) && chars[end] != '\'' {
					badNext = true
				}

				if !badCurrent || !badNext {
					good := false
					cur := start - 2
					if badCurrent {
						cur = start
					}

					for !good && cur < end {
						if cur >= 0 && (!IsSymbol(chars[cur]) || chars[cur] == '\'') {
							frag := make([]rune, 3)

							off := 0
							for off = 0; off < 3 && cur+off < len(chars) && (!IsSymbol(chars[cur+off]) || chars[cur+off] == '\''); off++ {
								frag[off] = chars[cur+off]
							}

							valid := true
							if off == 0 {
								valid = false
							}
							if off < 3 && cur-1 >= 0 && (!IsSymbol(chars[cur-1]) || chars[cur-1] == '\'') {
								valid = false
							}
							if valid && !IsBadFragment(frag) {
								good = true
							}
						}

						cur++
					}

					if !good {
						bad = false
					}
				}
			} else {
				b = ' '
				if start-1 >= 0 {
					b = chars[start-1]
				}

				c = ' '
				if end < len(chars) {
					c = chars[end]
				}

				bIndex := GetIndex(b)
				cIndex := GetIndex(c)

				if badCombinations != nil && ComboMatches(bIndex, badCombinations, cIndex) {
					bad = false
				}
			}

			if bad {
				numeralCount := 0
				alphaCount := 0
				// Java: alphaIndex tracks the LAST alpha position; before the
				// masking gate, numeralCount is reduced by the distance from it
				// to the span end (WordFilter.java:756-771, missing in 225).
				alphaIndex := -1
				for i := start; i < end; i++ {
					if IsNumber(chars[i]) {
						numeralCount++
					} else if IsAlpha(chars[i]) {
						alphaCount++
						alphaIndex = i
					}
				}

				if alphaIndex > -1 {
					numeralCount -= end - alphaIndex + 1
				}

				if numeralCount <= alphaCount {
					for i := start; i < end; i++ {
						chars[i] = '*'
					}
				}
			}
		}
	}
}

// ComboMatches mirrors Java comboMatches(byte, byte[][], byte) (WordFilter.java:679).
// All operands are signed int8 so the binary-search ordering test below matches
// Java's signed-byte comparison; a stored combination byte >127 sorts negative.
func ComboMatches(a int8, combos [][]int8, b int8) bool {
	first := 0
	if combos[first][0] == a && combos[first][1] == b {
		return true
	}

	last := len(combos) - 1
	if combos[last][0] == a && combos[last][1] == b {
		return true
	}

	for ok := true; ok; ok = first != last && first+1 != last {
		middle := (first + last) / 2
		if combos[middle][0] == a && combos[middle][1] == b {
			return true
		}

		if a < combos[middle][0] || a == combos[middle][0] && b < combos[middle][1] {
			last = middle
		} else {
			first = middle
		}
	}

	return false
}

func GetEmulatedDomainCharSize(c, b, a rune) int {
	if b == a {
		return 1
	}
	if b == 'o' && a == '0' {
		return 1
	}
	if b == 'o' && a == '(' && c == ')' {
		return 2
	}
	if b == 'c' && (a == '(' || a == '<' || a == '[') {
		return 1
	}
	if b == 'e' && a == 8364 {
		return 1
	}
	if b == 's' && a == '$' {
		return 1
	}
	if b == 'l' && a == 'i' {
		return 1
	}
	return 0
}

func GetEmulatedSize(c, a, b rune) int {
	if a == b {
		return 1
	}

	if a >= 'a' && a <= 'm' {
		if a == 'a' {
			if b != '4' && b != '@' && b != '^' {
				if b == '/' && c == '\\' {
					return 2
				}
				return 0
			}
			return 1
		}

		if a == 'b' {
			if b != '6' && b != '8' {
				// Java 244: `if ((b != '1' || c != '3') && (b != 'i' || c != '3'))
				// return 0; return 2;` — 225 lacked the 'i3' alternative.
				if (b == '1' && c == '3') || (b == 'i' && c == '3') {
					return 2
				}
				return 0
			}
			return 1
		}

		if a == 'c' {
			if b != '(' && b != '<' && b != '{' && b != '[' {
				return 0
			}
			return 1
		}

		if a == 'd' {
			// Java 244: `if ((b != '[' || c != ')') && (b != 'i' || c != ')'))
			// return 0; return 2;` — 225 lacked the 'i)' alternative.
			if (b == '[' && c == ')') || (b == 'i' && c == ')') {
				return 2
			}
			return 0
		}

		if a == 'e' {
			if b != '3' && b != 8364 {
				return 0
			}
			return 1
		}

		if a == 'f' {
			if b == 'p' && c == 'h' {
				return 2
			}
			if b == 163 {
				return 1
			}
			return 0
		}

		if a == 'g' {
			// Java 244 also accepts 'q' as an emulated 'g' (WordFilter.java:899-905).
			if b != '9' && b != '6' && b != 'q' {
				return 0
			}
			return 1
		}

		if a == 'h' {
			if b == '#' {
				return 1
			}
			return 0
		}

		if a == 'i' {
			if b != 'y' && b != 'l' && b != 'j' && b != '1' && b != '!' && b != ':' && b != ';' && b != '|' {
				return 0
			}
			return 1
		}

		if a == 'j' {
			return 0
		}

		if a == 'k' {
			return 0
		}

		if a == 'l' {
			if b != '1' && b != '|' && b != 'i' {
				return 0
			}
			return 1
		}

		if a == 'm' {
			return 0
		}
	}

	if a >= 'n' && a <= 'z' {
		if a == 'n' {
			return 0
		}

		if a == 'o' {
			if b != '0' && b != '*' {
				if (b != '(' || c != ')') && (b != '[' || c != ']') && (b != '{' || c != '}') && (b != '<' || c != '>') {
					return 0
				}
				return 2
			}
			return 1
		}

		if a == 'p' {
			return 0
		}

		if a == 'q' {
			return 0
		}

		if a == 'r' {
			return 0
		}

		if a == 's' {
			if b != '5' && b != 'z' && b != '$' && b != '2' {
				return 0
			}
			return 1
		}

		if a == 't' {
			if b != '7' && b != '+' {
				return 0
			}
			return 1
		}

		if a == 'u' {
			if b == 'v' {
				return 1
			}
			if b == '\\' && c == '/' || b == '\\' && c == '|' || b == '|' && c == '/' {
				return 2
			}
			return 0
		}

		if a == 'v' {
			if (b != '\\' || c != '/') && (b != '\\' || c != '|') && (b != '|' || c != '/') {
				return 0
			}
			return 2
		}

		if a == 'w' {
			if b == 'v' && c == 'v' {
				return 2
			}
			return 0
		}

		if a == 'x' {
			if (b != ')' || c != '(') && (b != '}' || c != '{') && (b != ']' || c != '[') && (b != '>' || c != '<') {
				return 0
			}
			return 2
		}

		if a == 'y' {
			return 0
		}

		if a == 'z' {
			return 0
		}
	}

	if a >= '0' && a <= '9' {
		if a == '0' {
			if b == 'o' || b == 'O' {
				return 1
			}
			if (b != '(' || c != ')') && (b != '{' || c != '}') && (b != '[' || c != ']') {
				return 0
			}
			return 2
		}

		if a == '1' {
			if b == 'l' {
				return 1
			}
			return 0
		}

		return 0
	}

	if a == ',' {
		if b == '.' {
			return 1
		}
		return 0
	}

	if a == '.' {
		if b == ',' {
			return 1
		}
		return 0
	}

	if a == '!' {
		if b == 'i' {
			return 1
		}
		return 0
	}

	return 0
}

// GetIndex mirrors Java getIndex (WordFilter.java) which returns a byte; the Go
// return is int8 so its result feeds ComboMatches's signed comparison directly.
// All returned values are positive (1..38), so the signedness is inert here, but
// the type matches Java's byte return.
func GetIndex(c rune) int8 {
	if c >= 'a' && c <= 'z' {
		return int8(c - 'a' + 1)
	}
	if c == '\'' {
		return 28
	}
	if c >= '0' && c <= '9' {
		return int8(c - '0' + 29)
	}
	return 27
}

func FilterFragments(chars []rune) {
	end := 0
	count := 0
	start := 0

	for {
		for ok := true; ok; ok = count != 4 {
			index := IndexOfNumber(chars, end)
			if index == -1 {
				return
			}

			foundLowercase := false
			for i := end; i >= 0 && i < index && !foundLowercase; i++ {
				if !IsSymbol(chars[i]) && !IsLowerCaseAlpha(chars[i]) {
					foundLowercase = true
				}
			}

			if foundLowercase {
				count = 0
			}

			if count == 0 {
				start = index
			}

			end = IndexOfNonNumber(index, chars)

			value := 0
			for i := index; i < end; i++ {
				value = value*10 + int(chars[i]) - 48
			}

			if value <= 0xFF && end-index <= 8 {
				count++
			} else {
				count = 0
			}
		}

		for i := start; i < end; i++ {
			chars[i] = '*'
		}

		count = 0
	}
}

func IndexOfNumber(in []rune, off int) int {
	for i := off; i < len(in) && i >= 0; i++ {
		if in[i] >= '0' && in[i] <= '9' {
			return i
		}
	}
	return -1
}

func IndexOfNonNumber(off int, in []rune) int {
	i := off
	for {
		if i < len(in) && i >= 0 {
			if in[i] >= '0' && in[i] <= '9' {
				i++
				continue
			}
			return i
		}
		return len(in)
	}
}

func IsSymbol(c rune) bool {
	return !IsAlpha(c) && !IsNumber(c)
}

func IsLowerCaseAlpha(c rune) bool {
	if c >= 'a' && c <= 'z' {
		return c == 'v' || c == 'x' || c == 'j' || c == 'q' || c == 'z'
	}
	return true
}

func IsAlpha(c rune) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func IsNumber(c rune) bool {
	return c >= '0' && c <= '9'
}

func IsLowerCase(c rune) bool {
	return c >= 'a' && c <= 'z'
}

func IsUpperCase(c rune) bool {
	return c >= 'A' && c <= 'Z'
}

func IsBadFragment(in []rune) bool {
	skip := true
	for i := range len(in) {
		if !IsNumber(in[i]) && in[i] != 0 {
			skip = false
		}
	}

	if skip {
		return true
	}

	i := FirstFragmentID(in)
	start := 0
	end := len(Fragments) - 1

	if i == Fragments[start] || i == Fragments[end] {
		return true
	}

	for ok := true; ok; ok = start != end && start+1 != end {
		middle := (start + end) / 2
		if i == Fragments[middle] {
			return true
		}

		if i < Fragments[middle] {
			end = middle
		} else {
			start = middle
		}
	}

	return false
}

func FirstFragmentID(chars []rune) int {
	if len(chars) > 6 {
		return 0
	}

	value := 0
	for i := range len(chars) {
		c := chars[len(chars)-i-1]
		if c >= 'a' && c <= 'z' {
			value = value*38 + int(c) - 'a' + 1
		} else if c == '\'' {
			value = value*38 + 27
		} else if c >= '0' && c <= '9' {
			value = value*38 + int(c) - '0' + 28
		} else if c != 0 {
			return 0
		}
	}

	return value
}

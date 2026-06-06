package jstring

import (
	"fmt"
	"strings"
)

var (
	BASE37_LOOKUP []rune = []rune{'_', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
)

func ToBase37(s string) int64 {
	// Java: JString.toBase37 walks arg1.charAt(i) for i < arg1.length() && i < 12,
	// where length() is UTF-16 code units. RuneScape usernames are validated
	// ASCII-only, so this matters for fidelity only. []rune approximates
	// charAt sufficiently for the BMP-only chars the alphabet allows; the
	// non-matching branch silently multiplies hash by 37 in Java but is
	// reached only on invalid input.
	runes := []rune(s)
	var hash int64

	for i := 0; i < len(runes) && i < 12; i++ {
		c := int64(runes[i])
		hash *= 37

		if c >= 'A' && c <= 'Z' {
			hash += c + 1 - 65
		} else if c >= 'a' && c <= 'z' {
			hash += c + 1 - 97
		} else if c >= '0' && c <= '9' {
			hash += c + 27 - 48
		}
	}

	for hash%37 == 0 && hash != 0 {
		hash /= 37
	}

	return hash
}

func FromBase37(username int64) string {
	// Java: if (arg0 <= 0L || arg0 >= 6582952005840035281L) (JString.java:33
	// @2e62978) — the first clause rejects 0 too. Match it literally with
	// <= 0 (0 is also caught by the %37==0 branch, so behavior was already
	// identical).
	if username <= 0 || username >= 6582952005840035281 {
		return "invalid_name"
	} else if username%37 == 0 {
		return "invalid_name"
	} else {
		length := 0
		// Java: char[] var4 = new char[12] (JString.java:39 @2e62978) — 254
		// replaces ≤245.2's static 12-char builder with a per-call buffer;
		// the package-level Builder global is deleted with it (also fixes a
		// latent data race between goroutines formatting names concurrently).
		builder := make([]rune, 12)

		for username != 0 {
			last := username
			username /= 37
			builder[11-length] = BASE37_LOOKUP[last-username*37]
			length++
		}

		return string(builder[12-length : 12-length+length])
	}
}

func HashCode(s string) int64 {
	// Java: JString.hashCode walks charAt(i) for i < length() (UTF-16 code
	// units). Sprite/file names supplied by callers are ASCII-only, but
	// iterating runes matches Java's semantics for any BMP input.
	upper := []rune(strings.ToUpper(s))

	hash := int64(0)
	for i := range len(upper) {
		hash = hash*61 + int64(upper[i]) - 32
		hash = (hash + (hash >> 56)) & 0xFFFFFFFFFFFFFF
	}

	return hash
}

func FormatIPv4(ip int32) string {
	return fmt.Sprintf("%d.%d.%d.%d", (ip>>24)&0xFF, (ip>>16)&0xFF, (ip>>8)&0xFF, ip&0xFF)
}

// FormatDisplayName
func FormatName(username string) string {
	if len(username) <= 0 {
		return username
	}

	chars := []rune(username)
	for i := range len(chars) {
		if chars[i] == '_' {
			chars[i] = ' '
			if i+1 < len(chars) && chars[i+1] >= 'a' && chars[i+1] <= 'z' {
				chars[i+1] = chars[i+1] + 'A' - 97
			}
		}
	}

	if chars[0] >= 'a' && chars[0] <= 'z' {
		chars[0] = chars[0] + 'A' - 97
	}

	return string(chars)
}

func ToSentenceCase(s string) string {
	lower := strings.ToLower(s)
	chars := []rune(lower)
	length := len(chars)

	capital := true
	for i := range length {
		c := chars[i]

		if capital && c >= 'a' && c <= 'z' {
			chars[i] = chars[i] + -32
			capital = false
		}

		if c == '.' || c == '!' {
			capital = true
		}
	}

	return string(chars)
}

// ToAsterisks returns a string of `*` characters with the same character
// count as s. Java: JString.toAsterisks (JString.java:108-114) iterates
// `arg1.length()`, which is the UTF-16 code-unit count — one star per char,
// not per UTF-8 byte. Go's `len(s)` is byte-based: for any non-ASCII char
// (e.g. '£' = 2 bytes) it would produce too many stars. Censored chat can
// include '£', so this matters in practice.
//
// Latent (audit jstring-01, accepted): `for range s` counts runes, while
// Java counts UTF-16 units — these differ only for astral (non-BMP) chars,
// which encode as surrogate PAIRS in Java (2 stars) but single runes here
// (1 star). Chat input is CHARSET-gated to BMP characters, so the divergence
// is unreachable; the exact port would be len(utf16.Encode([]rune(s))).
func ToAsterisks(s string) string {
	var sb strings.Builder
	for range s {
		sb.WriteString("*")
	}
	return sb.String()
}

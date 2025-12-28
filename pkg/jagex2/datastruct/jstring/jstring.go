package jstring

import (
	"fmt"
	"strings"
)

var (
	Builder       []rune = make([]rune, 12)
	BASE37_LOOKUP []rune = []rune{'_', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
)

func ToBase37(s string) int64 {
	var hash int64

	for i := 0; i < len(s) && i < 12; i++ {
		c := int64(s[i])
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
	if username < 0 || username >= 6582952005840035281 {
		return "invalid_name"
	} else if username%37 == 0 {
		return "invalid_name"
	} else {
		length := 0

		for username != 0 {
			last := username
			username /= 37
			Builder[11-length] = BASE37_LOOKUP[last-username*37]
			length++
		}

		return string(Builder[12-length : 12-length+length]) // TODO: test this
	}
}

func HashCode(s string) int64 {
	upper := strings.ToUpper(s)

	hash := int64(0)
	for i := 0; i < len(upper); i++ {
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
	for i := 0; i < len(chars); i++ {
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
	for i := 0; i < length; i++ {
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

// Censor
func ToAsterisks(s string) string {
	var sb strings.Builder

	for i := 0; i < len(s); i++ {
		sb.WriteString("*")
	}

	return sb.String()
}

package wordpack

import (
	"strings"

	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

var (
	CharBuffer []rune = make([]rune, 100)
	TABLE      []rune = []rune{' ', 'e', 't', 'a', 'o', 'i', 'h', 'n', 's', 'r', 'd', 'l', 'u', 'm', 'w', 'c', 'y', 'f', 'g', 'p', 'b', 'v', 'k', 'x', 'j', 'q', 'z', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', ' ', '!', '?', '.', ',', ':', ';', '(', ')', '-', '&', '*', '\\', '\'', '@', '#', '+', '=', '£', '$', '%', '"', '[', ']'}
)

func Unpack(buf *io.Packet, length int) string {
	pos := 0
	carry := -1

	for range length {
		value := buf.G1()

		nibble := (value >> 4) & 0xF
		if carry != -1 {
			CharBuffer[pos] = TABLE[(carry<<4)+nibble-195]
			pos++
			carry = -1
		} else if nibble < 13 {
			CharBuffer[pos] = TABLE[nibble]
			pos++
		} else {
			carry = nibble
		}

		nibble = value & 0xF
		if carry != -1 {
			CharBuffer[pos] = TABLE[(carry<<4)+nibble-195]
			pos++
			carry = -1
		} else if nibble < 13 {
			CharBuffer[pos] = TABLE[nibble]
			pos++
		} else {
			carry = nibble
		}
	}

	uppercase := true
	for i := range pos {
		c := CharBuffer[i]
		if uppercase && c >= 'a' && c <= 'z' {
			CharBuffer[i] = CharBuffer[i] + -32
			uppercase = false
		}

		if c == '.' || c == '!' {
			uppercase = true
		}
	}

	// Java: WordPack.java:51 — `new String(charBuffer, 0, var3)`. The (offset,count)
	// String constructor takes a length, matching the Go slice expression here.
	return string(CharBuffer[0:pos])
}

func Pack(buf *io.Packet, terminate bool, str string) {
	// Java: arg2.length() / arg2.charAt(i) walk UTF-16 code units, not bytes.
	// TABLE contains BMP-only chars (incl. '£' U+00A3), so []rune matches charAt
	// exactly for this input. See WordPack.java:56-62.
	runes := []rune(str)
	if len(runes) > 80 {
		runes = runes[:80]
	}
	runes = []rune(strings.ToLower(string(runes)))

	carry := -1
	for _, c := range runes {
		index := 0
		for j := range len(TABLE) {
			if c == TABLE[j] {
				index = j
				break
			}
		}

		if index > 12 {
			index += 195
		}

		if carry == -1 {
			if index < 13 {
				carry = index
			} else {
				buf.P1(index)
			}
		} else if index < 13 {
			buf.P1((carry << 4) + index)
			carry = -1
		} else {
			buf.P1((carry << 4) + (index >> 4))
			carry = index & 0xF
		}
	}

	if terminate && carry != -1 {
		buf.P1(carry << 4)
	}
}

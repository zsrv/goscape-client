package wordpack

import (
	"strings"

	"goscape-client/pkg/jagex2/io"
)

var (
	CharBuffer []rune = make([]rune, 100)
	TABLE      []rune = []rune{' ', 'e', 't', 'a', 'o', 'i', 'h', 'n', 's', 'r', 'd', 'l', 'u', 'm', 'w', 'c', 'y', 'f', 'g', 'p', 'b', 'v', 'k', 'x', 'j', 'q', 'z', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', ' ', '!', '?', '.', ',', ':', ';', '(', ')', '-', '&', '*', '\\', '\'', '@', '#', '+', '=', '£', '$', '%', '"', '[', ']'}
)

func Unpack(arg0 *io.Packet, arg2 int) string {
	var3 := 0
	var4 := -1
	for range arg2 {
		var6 := arg0.G1()
		var7 := (var6 >> 4) & 0xF
		if var4 != -1 {
			CharBuffer[var3] = TABLE[(var4<<4)+var7-195]
			var3++
			var4 = -1
		} else if var7 < 13 {
			CharBuffer[var3] = TABLE[var7]
			var3++
		} else {
			var4 = var7
		}
		var7 = var6 & 0xF
		if var4 != -1 {
			CharBuffer[var3] = TABLE[(var4<<4)+var7-195]
			var3++
			var4 = -1
		} else if var7 < 13 {
			CharBuffer[var3] = TABLE[var7]
			var3++
		} else {
			var4 = var7
		}
	}
	var10 := true
	for i := range var3 {
		var8 := CharBuffer[i]
		if var10 && var8 >= 'a' && var8 <= 'z' {
			CharBuffer[i] = CharBuffer[i] + -32
			var10 = false
		}
		if var8 == '.' || var8 == '!' {
			var10 = true
		}
	}
	return string(CharBuffer[0:var3]) // TODO: var3 or var3-1?
}

func Pack(arg0 *io.Packet, arg1 bool, arg2 string) {
	if len(arg2) > 80 {
		arg2 = arg2[0:80] // TODO: verify 80
	}
	arg2 = strings.ToLower(arg2)
	var3 := -1
	for i := range len(arg2) {
		var5 := rune(arg2[i])
		var6 := 0
		for j := range len(TABLE) {
			if var5 == TABLE[j] {
				var6 = j
				break
			}
		}
		if var6 > 12 {
			var6 += 195
		}
		if var3 == -1 {
			if var6 < 13 {
				var3 = var6
			} else {
				arg0.P1(var6)
			}
		} else if var6 < 13 {
			arg0.P1((var3 << 4) + var6)
			var3 = -1
		} else {
			arg0.P1((var3 << 4) + (var6 >> 4))
			var3 = var6 & 0xF
		}
	}
	if arg1 && var3 != -1 {
		arg0.P1(var3 << 4)
	}
}

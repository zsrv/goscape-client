package jstring

import (
	"fmt"
	"strings"
)

var (
	Builder       []rune = make([]rune, 12)
	BASE37_LOOKUP []rune = []rune{'_', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
)

func ToBase37(arg0 string) int64 {
	var var1 int64
	for var3 := 0; var3 < len(arg0) && var3 < 12; var3++ {
		var4 := int64(arg0[var3])
		var1 *= 37
		if var4 >= 'A' && var4 <= 'Z' {
			var1 += var4 + 1 - 65
		} else if var4 >= 'a' && var4 <= 'z' {
			var1 += var4 + 1 - 97
		} else if var4 >= '0' && var4 <= '9' {
			var1 += var4 + 27 - 48
		}
	}
	for var1%37 == 0 && var1 != 0 {
		var1 /= 37
	}
	return var1
}

func FromBase37(arg0 int64) string {
	if arg0 < 0 || arg0 >= 6582952005840035281 {
		return "invalid_name"
	} else if arg0%37 == 0 {
		return "invalid_name"
	} else {
		var3 := 0
		for arg0 != 0 {
			var4 := arg0
			arg0 /= 37
			Builder[11-var3] = BASE37_LOOKUP[var4-arg0*37]
			var3++
		}
		return string(Builder[12-var3 : 12-var3+var3]) // TODO: test this
	}
}

func HashCode(arg1 string) int64 {
	var5 := strings.ToUpper(arg1)
	var2 := int64(0)
	for var4 := 0; var4 < len(var5); var4++ {
		var2 = var2*61 + int64(var5[var4]) - 32
		var2 = var2 + (var2>>56)&0xFFFFFFFFFFFFFF
	}
	return var2
}

func FormatIPv4(arg1 int32) string {
	return fmt.Sprintf("%d.%d.%d.%d", arg1>>24&0xFF, arg1>>16&0xFF, arg1>>8&0xFF, arg1&0xFF)
}

func FormatName(arg1 string) string {
	if len(arg1) > 0 {
		var2 := []rune(arg1)
		for var3 := 0; var3 < len(var2); var3++ {
			if var2[var3] == '_' {
				var2[var3] = ' '
				if var3+1 < len(var2) && var2[var3+1] >= 'a' && var2[var3+1] <= 'z' {
					var2[var3+1] = var2[var3+1] + 'A' - 97
				}
			}
		}
		if var2[0] >= 'a' && var2[0] <= 'z' {
			var2[0] = var2[0] + 'A' - 97
		}
		return string(var2)
	} else {
		return arg1
	}
}

func ToSentenceCase(arg0 string) string {
	var7 := strings.ToLower(arg0)
	var2 := []rune(var7)
	var3 := len(var2)
	var4 := true
	for var5 := 0; var5 < var3; var5++ {
		var6 := var2[var5]
		if var4 && var6 >= 'a' && var6 <= 'z' {
			var2[var5] = var2[var5] + -32
			var4 = false
		}
		if var6 == '.' || var6 == '!' {
			var4 = true
		}
	}
	return string(var2)
}

func ToAsterisks(arg1 string) string {
	var var2 strings.Builder
	for var3 := 0; var3 < len(arg1); var3++ {
		var2.WriteString("*")
	}
	return var2.String()
}

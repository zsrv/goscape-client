// Package bootfont renders text during the boot phase before c.JagTitle
// (and thus the RuneScape pixel fonts in pixfont) has been loaded. It
// wraps golang.org/x/image/font/basicfont.Face7x13, a monospace 7x13
// font shipped in x/image. Used exclusively by DrawProgressGameShell.
package bootfont

import (
	"unicode/utf8"

	"golang.org/x/image/font/basicfont"
)

// Height returns the font's inter-line height in pixels.
func Height() int {
	return basicfont.Face7x13.Height
}

// StringWidth returns the rendered pixel width of s, assuming
// basicfont.Face7x13's fixed 7-pixel advance per glyph.
func StringWidth(s string) int {
	return utf8.RuneCountInString(s) * basicfont.Face7x13.Advance
}

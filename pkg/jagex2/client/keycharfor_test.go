package client

import (
	"testing"

	"gioui.org/io/key"
)

// TestKeyCharFor_AppliesShiftToDigitsAndPunctuation pins the post-fix
// US-keyboard Shift mapping. User-reported: special characters like
// `:` and `$` could not be typed in chat / PM input. Root cause:
// Gio's key.Event reports only the physical key name plus the
// modifier bitmask (unlike AWT's KeyEvent.getKeyChar() which gives
// the typed character directly). The pre-fix keyCharFor applied
// Shift only to A-Z letters; digits and punctuation returned their
// physical-key value regardless of Shift, so chat inputs that
// required Shift+number or Shift+punctuation silently dropped the
// modifier.
func TestKeyCharFor_AppliesShiftToDigitsAndPunctuation(t *testing.T) {
	cases := []struct {
		name  key.Name
		shift bool
		want  int
	}{
		// Letters: shift selects case
		{"A", false, 'a'},
		{"A", true, 'A'},
		{"Z", false, 'z'},
		{"Z", true, 'Z'},

		// Digits: shift selects punctuation
		{"1", false, '1'},
		{"1", true, '!'},
		{"2", true, '@'},
		{"4", true, '$'},
		{"0", true, ')'},

		// Punctuation: shift maps to upper variant
		{";", false, ';'},
		{";", true, ':'},
		{"'", false, '\''},
		{"'", true, '"'},
		{"-", true, '_'},
		{"=", true, '+'},
		{",", true, '<'},
		{".", true, '>'},
		{"/", true, '?'},

		// Multi-char names (special keys) → 0
		{"LeftArrow", false, 0},
		{"F1", false, 0},
	}
	for _, tc := range cases {
		var mods key.Modifiers
		if tc.shift {
			mods = key.ModShift
		}
		got := keyCharFor(key.Event{Name: tc.name, Modifiers: mods, State: key.Press})
		if got != tc.want {
			t.Errorf("keyCharFor(name=%q, shift=%v) = %d (%q); want %d (%q)",
				tc.name, tc.shift, got, rune(got), tc.want, rune(tc.want))
		}
	}
}

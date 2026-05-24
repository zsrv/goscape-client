package wordfilter

import "testing"

// TestIndexOfRunesFrom verifies the Java String.indexOf(String, int) semantics
// implemented over rune slices.
func TestIndexOfRunesFrom(t *testing.T) {
	hay := []rune("seeks cook and cook's cooks")
	cases := []struct {
		needle string
		from   int
		want   int
	}{
		{"cook", 0, 6},
		{"cook", 7, 15}, // skips first match
		{"cook", 16, 22},
		{"cook", 23, -1},
		{"seeks", 0, 0},
		{"missing", 0, -1},
		{"", 3, 3},
		{"", 100, len(hay)},
		{"cook", -1, 6}, // negative from is clamped
	}
	for _, c := range cases {
		got := indexOfRunesFrom(hay, []rune(c.needle), c.from)
		if got != c.want {
			t.Errorf("indexOfRunesFrom(%q, %q, %d) = %d, want %d", string(hay), c.needle, c.from, got, c.want)
		}
	}
}

// TestFilterAllowlistRestoresMaskedWords ensures that words in ALLOWLIST
// (e.g. "cook") are restored after FilterBad / FilterFragments would
// otherwise mask them. Prior to fixing the needle/haystack swap and the
// broken loop, the restoration never happened — ALLOWLIST words stayed
// as asterisks.
func TestFilterAllowlistRestoresMaskedWords(t *testing.T) {
	// Pre-seed BadWords with "cook" so FilterBad will mask it. Restore the
	// global after the test so other tests are unaffected.
	prevBad := BadWords
	prevCombos := BadCombinations
	t.Cleanup(func() {
		BadWords = prevBad
		BadCombinations = prevCombos
	})
	BadWords = [][]rune{[]rune("cook")}
	BadCombinations = [][][]int8{nil}

	got := Filter("cook")
	if got != "cook" {
		t.Errorf("Filter(\"cook\") = %q, want %q (ALLOWLIST should restore it)", got, "cook")
	}

	// Also confirm multiple occurrences are restored — exercises the loop.
	got = Filter("cook and cook")
	if got != "cook and cook" {
		t.Errorf("Filter(\"cook and cook\") = %q, want %q", got, "cook and cook")
	}
}

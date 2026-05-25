package client

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
)

func TestAwtFor(t *testing.T) {
	cases := map[platform.Key]int{
		platform.KeyLeft: 37, platform.KeyRight: 39, platform.KeyUp: 38,
		platform.KeyDown: 40, platform.KeyCtrl: 17, platform.KeyBackspace: 8,
		platform.KeyDelete: 127, platform.KeyTab: 9, platform.KeyReturn: 10,
		platform.KeyEnter: 10, platform.KeyHome: 36, platform.KeyEnd: 35,
		platform.KeyPageUp: 33, platform.KeyPageDown: 34,
		platform.KeyF1: 112, platform.KeyF12: 123, platform.KeyRune: 0,
	}
	for k, want := range cases {
		if got := awtFor(k); got != want {
			t.Errorf("awtFor(%d) = %d, want %d", k, got, want)
		}
	}
}

func TestCharFor(t *testing.T) {
	if got := charFor(platform.KeyPress{Key: platform.KeyRune, Rune: 'A'}); got != int('a') {
		t.Errorf("'A' no shift = %d, want %d", got, int('a'))
	}
	if got := charFor(platform.KeyPress{Key: platform.KeyRune, Rune: 'A', Mods: platform.ModShift}); got != int('A') {
		t.Errorf("'A' shift = %d, want %d", got, int('A'))
	}
	if got := charFor(platform.KeyPress{Key: platform.KeyRune, Rune: '1', Mods: platform.ModShift}); got != int('!') {
		t.Errorf("'1' shift = %d, want %d", got, int('!'))
	}
	if got := charFor(platform.KeyPress{Key: platform.KeyLeft}); got != 0 {
		t.Errorf("named key char = %d, want 0", got)
	}
}

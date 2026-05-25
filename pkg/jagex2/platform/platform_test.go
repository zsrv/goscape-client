package platform

import "testing"

func TestKeyEnumDistinct(t *testing.T) {
	keys := []Key{
		KeyNone, KeyLeft, KeyRight, KeyUp, KeyDown, KeyReturn, KeyEnter,
		KeyEscape, KeyHome, KeyEnd, KeyBackspace, KeyDelete, KeyPageUp,
		KeyPageDown, KeyTab, KeySpace, KeyCtrl, KeyShift, KeyAlt, KeySuper,
		KeyCommand, KeyBack, KeyRune,
		KeyF1, KeyF2, KeyF3, KeyF4, KeyF5, KeyF6,
		KeyF7, KeyF8, KeyF9, KeyF10, KeyF11, KeyF12,
	}
	seen := map[Key]bool{}
	for _, k := range keys {
		if seen[k] {
			t.Fatalf("duplicate Key value %d", k)
		}
		seen[k] = true
	}
}

func TestEventsImplementInterface(t *testing.T) {
	var evs = []Event{
		MouseMove{X: 1, Y: 2},
		MouseButton{X: 1, Y: 2, Button: 1, Pressed: true},
		MouseCross{Entered: true},
		KeyPress{Key: KeyLeft, Down: true},
		CharInput{Rune: 'a'},
		FocusChange{Gained: true},
	}
	if len(evs) != 6 {
		t.Fatalf("want 6 events, got %d", len(evs))
	}
}

func TestModContains(t *testing.T) {
	m := ModShift | ModCtrl
	if !m.Has(ModShift) || !m.Has(ModCtrl) || m.Has(ModAlt) {
		t.Fatalf("Mod.Has wrong: %v", m)
	}
}

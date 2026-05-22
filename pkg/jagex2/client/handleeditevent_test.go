package client

import (
	"testing"

	"gioui.org/io/key"
)

// TestHandleEditEvent_PushesTextRunesToKeyQueue pins the OS-level
// typed-text path. User-reported: special characters like `:` and
// `$` couldn't be typed in chat / PM input even after a US-QWERTY
// shift table was added to keyCharFor. Root cause: Gio's key.Event
// is a *physical*-key dispatch with a modifier bitmask, but the
// OS keyboard-layout resolution happens upstream of key.Event
// (X11/Wayland/macOS/Windows native text-input pipelines). For
// proper layout/dead-key/IME-aware text input, the application
// must filter for key.EditEvent and consume its .Text field —
// which already contains the OS-resolved character(s).
//
// handleEditEvent pushes each rune of e.Text onto KeyQueue (where
// the game's PollKey reader consumes them) and into the
// InputTracking byte stream. It owns the typed-text path; handleKey
// continues to own the modal-sentinel path (arrows, F-keys, Enter,
// Backspace, etc.).
func TestHandleEditEvent_PushesTextRunesToKeyQueue(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []int
	}{
		{"colon (Shift+; on US)", ":", []int{':'}},
		{"dollar (Shift+4 on US)", "$", []int{'$'}},
		{"question (Shift+/ on US)", "?", []int{'?'}},
		{"ampersand (Shift+7 on US)", "&", []int{'&'}},
		{"underscore (Shift+- on US)", "_", []int{'_'}},
		{"plus (Shift+= on US)", "+", []int{'+'}},
		{"angle brackets", "<>", []int{'<', '>'}},
		{"quote (Shift+' on US)", "\"", []int{'"'}},
		{"plain letter", "a", []int{'a'}},
		{"uppercase via Shift", "A", []int{'A'}},
		{"digit", "1", []int{'1'}},
		{"multi-rune (IME-style)", "hi", []int{'h', 'i'}},
		{"NUL is dropped (var3 <= 4)", "\x00", []int{}},
		{"space passes (var3=32 > 4)", " ", []int{' '}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient()
			c.handleEditEvent(key.EditEvent{Text: tc.text})

			got := []int{}
			for c.KeyQueueReadPos != c.KeyQueueWritePos {
				got = append(got, c.KeyQueue[c.KeyQueueReadPos])
				c.KeyQueueReadPos = (c.KeyQueueReadPos + 1) & 0x7F
			}
			if len(got) != len(tc.want) {
				t.Fatalf("KeyQueue contents len=%d, want %d (got=%v, want=%v)", len(got), len(tc.want), got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("KeyQueue[%d] = %d (%q); want %d (%q)", i, got[i], rune(got[i]), tc.want[i], rune(tc.want[i]))
				}
			}
		})
	}
}

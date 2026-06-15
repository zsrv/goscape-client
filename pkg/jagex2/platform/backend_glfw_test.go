//go:build !js

package platform

import (
	"testing"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// TestKeyDownFromActionRepeatsCountAsDown guards backspace auto-repeat: holding
// a non-printable sentinel key (Backspace, Tab, Enter, the F-keys,
// Home/End/PgUp/PgDn) must keep firing KeyPress{Down:true} on every OS
// auto-repeat. The GLFW key callback is the ONLY source of these sentinels —
// they produce no CharCallback rune — so a dropped glfw.Repeat means e.g.
// holding Backspace clears just one character instead of continuing to clear.
//
// This matches the reference clients: AWT keyPressed fires on OS auto-repeat,
// and the TS client's onkeydown (GameShell.ts) does not filter auto-repeat
// either; both push the sentinel to the keyQueue on every repeat.
func TestKeyDownFromActionRepeatsCountAsDown(t *testing.T) {
	cases := []struct {
		name     string
		action   glfw.Action
		wantDown bool
		wantEmit bool
	}{
		{"press", glfw.Press, true, true},
		{"repeat", glfw.Repeat, true, true},
		{"release", glfw.Release, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			down, emit := keyDownFromAction(tc.action)
			if emit != tc.wantEmit {
				t.Fatalf("emit = %v, want %v", emit, tc.wantEmit)
			}
			if down != tc.wantDown {
				t.Fatalf("down = %v, want %v", down, tc.wantDown)
			}
		})
	}
}

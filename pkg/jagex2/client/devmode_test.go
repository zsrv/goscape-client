package client

import "testing"

func TestExamineIDSuffix(t *testing.T) {
	// Save and restore the package flag so the test is independent of the
	// DEVELOPER_MODE env var the binary was launched with.
	orig := developerMode
	defer func() { developerMode = orig }()

	t.Run("off returns empty", func(t *testing.T) {
		developerMode = false
		if got := examineIDSuffix(1276); got != "" {
			t.Errorf("examineIDSuffix(1276) off = %q, want %q", got, "")
		}
	})

	t.Run("on returns green id markup", func(t *testing.T) {
		developerMode = true
		want := " @whi@(@gre@1276@whi@)"
		if got := examineIDSuffix(1276); got != want {
			t.Errorf("examineIDSuffix(1276) on = %q, want %q", got, want)
		}
	})
}

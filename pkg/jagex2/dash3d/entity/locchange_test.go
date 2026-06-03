package entity

import "testing"

// TestNewLocChangeDefaults verifies the Java field initializer `endTime = -1`
// is reproduced by the constructor (Go zero-values it to 0 otherwise), and that
// every other field starts at its zero value.
func TestNewLocChangeDefaults(t *testing.T) {
	loc := NewLocChange()

	if loc.EndTime != -1 {
		t.Errorf("EndTime = %d, want -1", loc.EndTime)
	}

	checks := []struct {
		name string
		got  int
	}{
		{"Level", loc.Level},
		{"Layer", loc.Layer},
		{"X", loc.X},
		{"Z", loc.Z},
		{"OldType", loc.OldType},
		{"OldAngle", loc.OldAngle},
		{"OldShape", loc.OldShape},
		{"NewType", loc.NewType},
		{"NewAngle", loc.NewAngle},
		{"NewShape", loc.NewShape},
		{"StartTime", loc.StartTime},
	}
	for _, c := range checks {
		if c.got != 0 {
			t.Errorf("%s = %d, want 0", c.name, c.got)
		}
	}
}

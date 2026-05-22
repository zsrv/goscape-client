package profiling

import (
	"testing"
	"time"
)

func TestSessionTimestamp_Format(t *testing.T) {
	got := sessionTimestamp(time.Date(2026, 5, 22, 14, 30, 15, 123_000_000, time.UTC))
	want := "20260522T143015Z"
	if got != want {
		t.Errorf("sessionTimestamp = %q; want %q", got, want)
	}
}

func TestSessionTimestamp_SortableByTime(t *testing.T) {
	early := sessionTimestamp(time.Date(2026, 5, 22, 14, 30, 15, 0, time.UTC))
	later := sessionTimestamp(time.Date(2026, 5, 22, 14, 30, 16, 0, time.UTC))
	if !(early < later) {
		t.Errorf("expected %q < %q lexicographically", early, later)
	}
}

func TestSessionTimestamp_AlwaysUTC(t *testing.T) {
	// Caller may pass a non-UTC time; the formatted string must still
	// reflect UTC so two captures from machines in different timezones
	// sort sensibly together.
	losAngeles, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skipf("tz data unavailable: %v", err)
	}
	got := sessionTimestamp(time.Date(2026, 5, 22, 7, 30, 15, 0, losAngeles))
	want := "20260522T143015Z"
	if got != want {
		t.Errorf("sessionTimestamp on LA-local 07:30 = %q; want %q (UTC)", got, want)
	}
}

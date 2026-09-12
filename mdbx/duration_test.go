package mdbx

import (
	"math"
	"testing"
	"time"
)

func assertEqualDuration(t *testing.T, actual time.Duration, expected time.Duration) {
	t.Helper()
	diff := actual.Nanoseconds() - expected.Nanoseconds()
	threshold := int64(time.Millisecond)
	if (diff > threshold) || (diff < -threshold) {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func assertEqual16dot16(t *testing.T, actual Duration16dot16, expected Duration16dot16) {
	t.Helper()
	diff := int64(actual) - int64(expected)
	threshold := int64(66)
	if (diff > threshold) || (diff < -threshold) {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func TestDurationWithDuration(t *testing.T) {
	for i := range 1001 {
		expected := time.Duration(i) * time.Second
		assertEqualDuration(t, NewDuration16dot16(expected).ToDuration(), expected)
	}
}

func TestDurationWith16dot16(t *testing.T) {
	for i := range 1001 {
		expected := Duration16dot16(i * 65536)
		assertEqual16dot16(t, NewDuration16dot16(expected.ToDuration()), expected)
	}
}

// TestNewDuration16dot16_Saturates covers the conversions that used to wrap.
// libmdbx narrows every 16.16 duration to 32 bits and reads 0 as "no bound",
// so a value that survives neither must not silently become a different bound.
func TestNewDuration16dot16_Saturates(t *testing.T) {
	const unit = time.Second / 65536 // 15.258us

	for _, tc := range []struct {
		name string
		in   time.Duration
		want Duration16dot16
	}{
		{"zero stays no-bound", 0, 0},
		{"one unit", unit, 1},
		{"below the unit rounds up, never to no-bound", unit - 1, 1},
		{"negative becomes the smallest bound", -time.Second, 1},
		{"negative below the unit too", -1, 1},
		{"at the ceiling", maxDuration16dot16.ToDuration(), maxDuration16dot16},
		{"past the ceiling saturates", 24 * time.Hour, maxDuration16dot16},
		{"far past the ceiling saturates", math.MaxInt64, maxDuration16dot16},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := NewDuration16dot16(tc.in); got != tc.want {
				t.Errorf("NewDuration16dot16(%v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

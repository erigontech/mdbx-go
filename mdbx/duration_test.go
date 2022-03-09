package mdbx

import (
	"testing"
	"time"
)

func assertEqualDuration(t *testing.T, actual time.Duration, expected time.Duration) {
	diff := actual.Nanoseconds() - expected.Nanoseconds()
	threshold := int64(time.Millisecond)
	if (diff > threshold) || (diff < -threshold) {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func assertEqual16dot16(t *testing.T, actual Duration16dot16, expected Duration16dot16) {
	diff := int64(actual) - int64(expected)
	threshold := int64(66)
	if (diff > threshold) || (diff < -threshold) {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func TestDurationWithDuration(t *testing.T) {
	for i := 0; i <= 1000; i++ {
		expected := time.Duration(i) * time.Second
		assertEqualDuration(t, NewDuration16dot16(expected).ToDuration(), expected)
	}
}

func TestDurationWith16dot16(t *testing.T) {
	for i := 0; i <= 1000; i++ {
		expected := Duration16dot16(i * 65536)
		assertEqual16dot16(t, NewDuration16dot16(expected.ToDuration()), expected)
	}
}

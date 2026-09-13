package mdbx

import (
	"math"
	"time"
)

/*
#include <stdint.h>
*/
import "C"

type Duration16dot16 uint64

// maxDuration16dot16 (~18h12m) is the largest value libmdbx acts on: every
// consumer narrows it to 32 bits.
const maxDuration16dot16 = Duration16dot16(math.MaxUint32)

func (d Duration16dot16) ToDuration() time.Duration {
	return time.Duration(d) * (time.Second / 65536)
}

// NewDuration16dot16 converts duration to libmdbx's 1/65536-second units,
// saturating rather than wrapping. libmdbx reads 0 as "no bound", so only a
// zero duration may produce it: negative and sub-unit durations become the
// smallest bound instead.
func NewDuration16dot16(duration time.Duration) Duration16dot16 {
	ticks := duration / (time.Second / 65536)
	switch {
	case duration == 0:
		return 0
	case duration < 0 || ticks == 0:
		return 1
	case Duration16dot16(ticks) > maxDuration16dot16:
		return maxDuration16dot16
	}
	return Duration16dot16(ticks)
}

func toDuration(seconds16dot16 C.uint32_t) time.Duration {
	return Duration16dot16(seconds16dot16).ToDuration()
}

func toDurationU64(seconds16dot16 C.uint64_t) time.Duration {
	return Duration16dot16(seconds16dot16).ToDuration()
}

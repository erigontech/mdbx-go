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

// maxDuration16dot16 is the largest 16.16 value libmdbx can act on. Every
// consumer narrows it to 32 bits -- mdbx_env_set_syncperiod and
// mdbx_env_warmup take unsigned, defrag_init casts to uint32_t -- so a larger
// value wraps into a shorter, arbitrary bound. It is about 18h12m13s.
const maxDuration16dot16 = Duration16dot16(math.MaxUint32)

func (d Duration16dot16) ToDuration() time.Duration {
	return time.Duration(d) * (time.Second / 65536)
}

// NewDuration16dot16 converts duration into libmdbx's 1/65536-second units,
// saturating rather than wrapping.
//
// Every libmdbx entry point taking one of these reads 0 as "no bound", so only
// a zero duration may produce 0. A negative duration -- what
// deadline.Sub(time.Now()) returns once the deadline has passed -- would
// otherwise wrap through the unsigned type into a bound of roughly 18 hours,
// and a positive duration under the 15.26us unit would round down to "no
// bound". Both yield the smallest real bound instead. Durations past the
// 32-bit ceiling saturate there rather than wrapping to a shorter one.
func NewDuration16dot16(duration time.Duration) Duration16dot16 {
	if duration == 0 {
		return 0
	}
	ticks := duration / (time.Second / 65536)
	switch {
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

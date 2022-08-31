package mdbx

import "time"

/*
#include <stdint.h>
*/
import "C"

type Duration16dot16 uint64

func (d Duration16dot16) ToDuration() time.Duration {
	return time.Duration(d) * (time.Second / 65536)
}

func NewDuration16dot16(duration time.Duration) Duration16dot16 {
	return Duration16dot16(duration / (time.Second / 65536))
}

func toDuration(seconds16dot16 C.uint32_t) time.Duration {
	return Duration16dot16(seconds16dot16).ToDuration()
}

func toDurationU64(seconds16dot16 C.uint64_t) time.Duration {
	return Duration16dot16(seconds16dot16).ToDuration()
}

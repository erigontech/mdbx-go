//go:build !windows

package mdbx

/*
#include "mdbxgo.h"
*/
import "C"
import (
	"syscall"
)

func operrno(op string, ret C.int) error {
	if ret == C.MDBX_SUCCESS || ret == C.MDBX_RESULT_TRUE {
		return nil
	}
	if ret == C.MDBX_NOTFOUND || int(ret) == int(syscall.ENODATA) {
		return ErrNotFound
	}
	if minErrno <= ret && ret <= maxErrno {
		return &OpError{Op: op, Errno: Errno(ret)}
	}
	return &OpError{Op: op, Errno: syscall.Errno(ret)}
}

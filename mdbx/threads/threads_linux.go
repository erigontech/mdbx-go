//go:build linux

package threads

import "syscall"

// CurrentThreadID returns the Linux thread ID.
// Note: gettid() is not directly available in Go so we use a raw syscall.
func CurrentThreadID() uint64 {
	tid, _, _ := syscall.RawSyscall(syscall.SYS_GETTID, 0, 0, 0)
	return uint64(tid)
}

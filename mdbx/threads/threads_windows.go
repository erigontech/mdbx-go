//go:build windows

package threads

import "syscall"

// CurrentThreadID returns the Windows thread ID.
func CurrentThreadID() uint64 {
	return uint64(syscall.GetCurrentThreadId())
}

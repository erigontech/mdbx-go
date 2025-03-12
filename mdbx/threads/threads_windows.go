//go:build windows

package threads

import (
	"golang.org/x/sys/windows"
)

// CurrentThreadID returns the Windows thread ID.
func CurrentThreadID() uint64 {
	return uint64(windows.GetCurrentThreadId())
}

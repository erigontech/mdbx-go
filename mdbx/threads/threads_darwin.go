//go:build darwin

package threads

/*
#include <pthread.h>
#include <stdint.h>

// getThreadID uses the pthread API to get the thread ID.
uint64_t getThreadID() {
    uint64_t tid;
    pthread_threadid_np(NULL, &tid);
    return tid;
}
*/
import "C"

//TODO: maybe there's go func for that)

// CurrentThreadID returns the macOS thread ID.
func CurrentThreadID() uint64 {
	return uint64(C.getThreadID())
}

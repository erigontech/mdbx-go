package mdbx

import (
	"runtime"
	"testing"
	"time"

	"github.com/erigontech/mdbx-go/mdbx/threads"
)

// Under strict thread mode Txn.abort panics by design when it runs on a
// different OS thread than the one that began the txn.  That panic must not
// leave Env.closeLock read-locked: Env.Close takes the same lock for writing,
// so a leaked read lock deadlocks every later Close.
func TestTxn_AbortForeignThreadPanicDoesNotLeakCloseLock(t *testing.T) {
	env, err := NewEnv(Default)
	if err != nil {
		t.Fatalf("env: %v", err)
	}
	// Deliberately not using setup(), which registers Close as a Cleanup: this
	// test has to call Close itself and bound how long it may block.
	const pageSize = 4096
	if err := env.SetGeometry(-1, -1, 64*1024*pageSize, -1, -1, pageSize); err != nil {
		t.Fatalf("geometry: %v", err)
	}
	if err := env.Open(t.TempDir(), 0, 0664); err != nil {
		t.Fatalf("open: %v", err)
	}
	env.SetStrictThreadMode(true)

	// Goroutine A begins the txn and stays pinned to its OS thread for the
	// rest of the test, so that thread cannot be recycled under goroutine B.
	var (
		txnc    = make(chan *Txn, 1)
		tidA    = make(chan uint64, 1)
		release = make(chan struct{})
	)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		txn, err := env.BeginTxn(nil, Readonly)
		if err != nil {
			t.Errorf("begin: %v", err)
			txnc <- nil
			return
		}
		tidA <- threads.CurrentThreadID()
		txnc <- txn
		<-release
	}()
	txn := <-txnc
	if txn == nil {
		t.FailNow()
	}
	threadA := <-tidA

	// Goroutine B aborts it from a different thread and recovers the expected
	// panic -- what a Txn pool's Close does when it drains a txn begun
	// elsewhere.
	var (
		panicked = make(chan any, 1)
		tidB     = make(chan uint64, 1)
	)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		tidB <- threads.CurrentThreadID()
		defer func() { panicked <- recover() }()
		txn.Abort()
	}()
	threadB := <-tidB
	recovered := <-panicked
	close(release)

	if threadA == threadB {
		t.Skipf("goroutines landed on the same OS thread (%d); nothing to check", threadA)
	}
	if recovered == nil {
		t.Fatalf("cross-thread Abort did not panic (thread %d begun, %d aborting)", threadA, threadB)
	}
	t.Logf("recovered expected panic: %v", recovered)

	// The regression: Close must not block on a read lock the panic leaked.
	env.SetStrictThreadMode(false)
	closed := make(chan error, 1)
	go func() { closed <- env.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatalf("close: %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("Env.Close blocked: the panicking Abort leaked env.closeLock")
	}
}

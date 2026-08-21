package mdbxpool

import (
	"bytes"
	"errors"
	"runtime"
	"sync"
	"testing"

	"github.com/erigontech/mdbx-go/mdbx"
)

func TestPoolReusesTheSameTransaction(t *testing.T) {
	env, dbi := setup(t)
	p := New(env)
	t.Cleanup(func() { p.Close() })

	first := mustGet(t, p)
	read(t, first, dbi)
	p.Put(first)

	second := mustGet(t, p)
	if second != first {
		t.Errorf("Get returned a new transaction %p, want the pooled %p", second, first)
	}
	read(t, second, dbi)
	p.Put(second)

	st := p.Stats()
	if st.Hits != 1 || st.Misses != 1 {
		t.Errorf("stats = hits %d, misses %d; want 1 and 1", st.Hits, st.Misses)
	}
}

// A reused transaction must be renewed onto the current snapshot, not resume
// the one it saw last time.
func TestPoolGetSeesCommitsMadeWhileIdle(t *testing.T) {
	env, dbi := setup(t)
	p := New(env)
	t.Cleanup(func() { p.Close() })

	txn := mustGet(t, p)
	before := bytes.Clone(read(t, txn, dbi))
	p.Put(txn)

	want := []byte("written-while-idle")
	if err := env.Update(func(txn *mdbx.Txn) error {
		return txn.Put(dbi, firstKey(), want, 0)
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	reused := mustGet(t, p)
	defer p.Put(reused)
	if reused != txn {
		t.Fatalf("expected the pooled transaction back, got a fresh one")
	}
	got := read(t, reused, dbi)
	if !bytes.Equal(got, want) {
		t.Errorf("reused txn read %q, want %q (it read %q before the commit)", got, want, before)
	}
}

// The invariant the pool rests on: an idle pooled reader holds no snapshot, so
// writes can keep recycling pages while it sits in the free list.
func TestPoolIdleTransactionsRetainNoPages(t *testing.T) {
	env, dbi := setup(t)
	p := New(env)
	t.Cleanup(func() { p.Close() })

	txn := mustGet(t, p)
	read(t, txn, dbi)
	p.Put(txn)

	churn(t, env, dbi, 6)

	idle := readerStats(t, env)
	if idle.SumBytesRetained != 0 {
		t.Errorf("idle pooled reader retains %d bytes, want 0", idle.SumBytesRetained)
	}
	if idle.OldestTxID != 0 {
		t.Errorf("idle pooled reader pins snapshot %d, want none", idle.OldestTxID)
	}

	// Contrast: a live reader over the same churn does retain pages, which is
	// what makes the assertions above meaningful rather than vacuous.
	live := mustGet(t, p)
	read(t, live, dbi)
	churn(t, env, dbi, 6)
	st := readerStats(t, env)
	p.Put(live)
	if st.SumBytesRetained == 0 {
		t.Error("a live reader retained nothing over the same churn; the test cannot distinguish idle from live")
	}
	t.Logf("retained: idle %d bytes, live %d bytes", idle.SumBytesRetained, st.SumBytesRetained)
}

// Close must reclaim every reader slot. A sync.Pool could not: objects in
// per-P caches are unreachable to the draining goroutine, and mdbx installs no
// Txn finalizers, so whatever Close misses leaks for the life of the Env.
func TestPoolCloseReleasesEveryReaderSlot(t *testing.T) {
	env, dbi := setup(t)
	const n = 8
	p := New(env, WithMaxIdle(n))

	live := make([]*mdbx.Txn, n)
	for i := range live {
		live[i] = mustGet(t, p)
		read(t, live[i], dbi)
	}
	for _, txn := range live {
		p.Put(txn)
	}

	if got := p.Stats().Idle; got != n {
		t.Fatalf("idle = %d, want %d", got, n)
	}
	if got := readerStats(t, env).Count; got != n {
		t.Fatalf("reader slots before Close = %d, want %d", got, n)
	}

	if err := p.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if got := readerStats(t, env).Count; got != 0 {
		t.Errorf("reader slots after Close = %d, want 0 (leaked)", got)
	}
	if got := p.Stats().Idle; got != 0 {
		t.Errorf("idle after Close = %d, want 0", got)
	}
}

func TestPoolMaxIdleBoundsReaderSlots(t *testing.T) {
	env, dbi := setup(t)
	const (
		maxIdle = 2
		live    = 8
	)
	p := New(env, WithMaxIdle(maxIdle))
	t.Cleanup(func() { p.Close() })

	txns := make([]*mdbx.Txn, live)
	for i := range txns {
		txns[i] = mustGet(t, p)
		read(t, txns[i], dbi)
	}
	if got := readerStats(t, env).Count; got != live {
		t.Fatalf("reader slots with %d checked out = %d", live, got)
	}
	for _, txn := range txns {
		p.Put(txn)
	}

	if got := p.Stats().Idle; got != maxIdle {
		t.Errorf("idle = %d, want %d", got, maxIdle)
	}
	if got := readerStats(t, env).Count; got != maxIdle {
		t.Errorf("reader slots after returning %d transactions = %d, want %d", live, got, maxIdle)
	}
	if got := p.Stats().Discards; got != live-maxIdle {
		t.Errorf("discards = %d, want %d", got, live-maxIdle)
	}
}

// MaxIdle 0 is a legitimate configuration: the pool degenerates to
// BeginTxn/Abort and keeps nothing.
func TestPoolMaxIdleZeroKeepsNothing(t *testing.T) {
	env, dbi := setup(t)
	p := New(env, WithMaxIdle(0))
	t.Cleanup(func() { p.Close() })

	for range 4 {
		txn := mustGet(t, p)
		read(t, txn, dbi)
		p.Put(txn)
	}
	if got := p.Stats().Idle; got != 0 {
		t.Errorf("idle = %d, want 0", got)
	}
	if st := p.Stats(); st.Hits != 0 || st.Misses != 4 {
		t.Errorf("stats = hits %d, misses %d; want 0 and 4", st.Hits, st.Misses)
	}
	if got := readerStats(t, env).Count; got != 0 {
		t.Errorf("reader slots = %d, want 0", got)
	}
}

// A negative MaxIdle is clamped rather than panicking or wrapping into a huge
// free list.
func TestPoolMaxIdleNegativeClampsToZero(t *testing.T) {
	env, _ := setup(t)
	p := New(env, WithMaxIdle(-5))
	t.Cleanup(func() { p.Close() })

	if got := p.Stats().MaxIdle; got != 0 {
		t.Errorf("MaxIdle = %d, want 0", got)
	}
}

func TestPoolDefaultMaxIdleIsGOMAXPROCS(t *testing.T) {
	env, _ := setup(t)
	p := New(env)
	t.Cleanup(func() { p.Close() })

	if got, want := p.Stats().MaxIdle, runtime.GOMAXPROCS(0); got != want {
		t.Errorf("MaxIdle = %d, want %d", got, want)
	}
}

func TestPoolAfterClose(t *testing.T) {
	env, dbi := setup(t)
	p := New(env)

	txn := mustGet(t, p)
	read(t, txn, dbi)

	if err := p.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}

	if _, err := p.Get(); !errors.Is(err, ErrClosed) {
		t.Errorf("Get after Close = %v, want ErrClosed", err)
	}
	if err := p.View(func(*mdbx.Txn) error { return nil }); !errors.Is(err, ErrClosed) {
		t.Errorf("View after Close = %v, want ErrClosed", err)
	}

	// A transaction checked out before Close is still the caller's to return,
	// and Put must abort rather than resurrect the free list.
	p.Put(txn)
	if got := p.Stats().Idle; got != 0 {
		t.Errorf("idle after Put on a closed pool = %d, want 0", got)
	}
	if got := readerStats(t, env).Count; got != 0 {
		t.Errorf("reader slots = %d, want 0", got)
	}
}

func TestPoolPutNil(t *testing.T) {
	env, _ := setup(t)
	p := New(env)
	t.Cleanup(func() { p.Close() })

	p.Put(nil) // must not panic
	if got := p.Stats().Idle; got != 0 {
		t.Errorf("idle = %d, want 0", got)
	}
}

func TestPoolView(t *testing.T) {
	env, dbi := setup(t)
	p := New(env)
	t.Cleanup(func() { p.Close() })

	var got []byte
	if err := p.View(func(txn *mdbx.Txn) error {
		v, err := txn.Get(dbi, firstKey())
		got = bytes.Clone(v)
		return err
	}); err != nil {
		t.Fatalf("view: %v", err)
	}
	if len(got) != seedValue {
		t.Errorf("read %d bytes, want %d", len(got), seedValue)
	}
	if idle := p.Stats().Idle; idle != 1 {
		t.Errorf("idle after View = %d, want 1", idle)
	}
}

func TestPoolViewReturnsOperationError(t *testing.T) {
	env, _ := setup(t)
	p := New(env)
	t.Cleanup(func() { p.Close() })

	want := errors.New("boom")
	if err := p.View(func(*mdbx.Txn) error { return want }); !errors.Is(err, want) {
		t.Errorf("View = %v, want %v", err, want)
	}
	if idle := p.Stats().Idle; idle != 1 {
		t.Errorf("idle after a failed View = %d, want 1 (the txn must still be returned)", idle)
	}
}

func TestPoolViewReturnsTransactionOnPanic(t *testing.T) {
	env, dbi := setup(t)
	p := New(env)
	t.Cleanup(func() { p.Close() })

	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected the panic to propagate")
			}
		}()
		_ = p.View(func(txn *mdbx.Txn) error {
			read(t, txn, dbi)
			panic("boom")
		})
	}()

	if idle := p.Stats().Idle; idle != 1 {
		t.Errorf("idle after a panicking View = %d, want 1 (the txn must not leak)", idle)
	}
	// And the recovered transaction is still usable.
	if err := p.View(func(txn *mdbx.Txn) error {
		read(t, txn, dbi)
		return nil
	}); err != nil {
		t.Errorf("View after a panic: %v", err)
	}
}

// Inside View the transaction is managed, so terminating it is a programming
// error and panics -- the same contract as Env.View.
func TestPoolViewTransactionIsManaged(t *testing.T) {
	env, _ := setup(t)
	p := New(env)
	t.Cleanup(func() { p.Close() })

	for _, tc := range []struct {
		name string
		op   func(*mdbx.Txn)
	}{
		{"Abort", func(txn *mdbx.Txn) { txn.Abort() }},
		{"Reset", func(txn *mdbx.Txn) { _ = txn.Reset() }},
		{"Renew", func(txn *mdbx.Txn) { _ = txn.Renew() }},
		{"Commit", func(txn *mdbx.Txn) { _, _ = txn.Commit() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("%s inside View did not panic", tc.name)
				}
			}()
			_ = p.View(func(txn *mdbx.Txn) error {
				tc.op(txn)
				return nil
			})
		})
	}
}

func TestPoolConcurrentViews(t *testing.T) {
	env, dbi := setup(t)
	p := New(env, WithMaxIdle(4))
	t.Cleanup(func() { p.Close() })

	const (
		goroutines = 16
		iterations = 64
	)
	stop := make(chan struct{})
	var writer sync.WaitGroup
	writer.Go(func() {
		value := make([]byte, seedValue)
		for {
			select {
			case <-stop:
				return
			default:
			}
			if err := env.Update(func(txn *mdbx.Txn) error {
				return txn.Put(dbi, firstKey(), value, 0)
			}); err != nil {
				t.Errorf("writer: %v", err)
				return
			}
		}
	})

	var readers sync.WaitGroup
	for range goroutines {
		readers.Go(func() {
			for range iterations {
				if err := p.View(func(txn *mdbx.Txn) error {
					_, err := txn.Get(dbi, firstKey())
					return err
				}); err != nil {
					t.Errorf("view: %v", err)
					return
				}
			}
		})
	}
	readers.Wait()
	close(stop)
	writer.Wait()

	st := p.Stats()
	if total := st.Hits + st.Misses; total != goroutines*iterations {
		t.Errorf("hits+misses = %d, want %d", total, goroutines*iterations)
	}
	if st.Idle > st.MaxIdle {
		t.Errorf("idle %d exceeds MaxIdle %d", st.Idle, st.MaxIdle)
	}
	if st.RenewFails != 0 {
		t.Errorf("renew failures = %d, want 0", st.RenewFails)
	}
	if got, want := readerStats(t, env).Count, uint64(st.Idle); got != want {
		t.Errorf("reader slots = %d, want %d (only idle transactions remain)", got, want)
	}
	t.Logf("stats: %+v", st)
}

// Concurrent Get/Put/Stats/Close must not race; run with -race for value.
func TestPoolCloseDuringConcurrentUse(t *testing.T) {
	env, dbi := setup(t)
	p := New(env, WithMaxIdle(4))

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 32 {
				txn, err := p.Get()
				if errors.Is(err, ErrClosed) {
					return
				}
				if err != nil {
					t.Errorf("get: %v", err)
					return
				}
				if _, err := txn.Get(dbi, firstKey()); err != nil {
					t.Errorf("read: %v", err)
					p.Put(txn)
					return
				}
				_ = p.Stats()
				p.Put(txn)
			}
		})
	}
	if err := p.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	wg.Wait()

	if got := p.Stats().Idle; got != 0 {
		t.Errorf("idle after Close = %d, want 0", got)
	}
	if got := readerStats(t, env).Count; got != 0 {
		t.Errorf("reader slots after Close = %d, want 0", got)
	}
}

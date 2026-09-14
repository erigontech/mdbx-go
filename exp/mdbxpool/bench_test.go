package mdbxpool

import (
	"sync"
	"testing"

	"github.com/erigontech/mdbx-go/mdbx"
)

// BenchmarkPool measures what pooling buys over beginning and aborting a
// transaction per read.
func BenchmarkPool(b *testing.B) {
	b.Run("Serial", func(b *testing.B) {
		b.Run("NoPool", func(b *testing.B) {
			env, dbi := setup(b)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := env.View(func(txn *mdbx.Txn) error {
					_, err := txn.Get(dbi, firstKey())
					return err
				}); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run("Pool", func(b *testing.B) {
			env, dbi := setup(b)
			p := New(env)
			defer p.Close()
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := p.View(func(txn *mdbx.Txn) error {
					_, err := txn.Get(dbi, firstKey())
					return err
				}); err != nil {
					b.Fatal(err)
				}
			}
		})
	})

	b.Run("Parallel", func(b *testing.B) {
		b.Run("NoPool", func(b *testing.B) {
			env, dbi := setup(b)
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if err := env.View(func(txn *mdbx.Txn) error {
						_, err := txn.Get(dbi, firstKey())
						return err
					}); err != nil {
						b.Error(err)
						return
					}
				}
			})
		})
		b.Run("Pool", func(b *testing.B) {
			env, dbi := setup(b)
			p := New(env)
			defer p.Close()
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if err := p.View(func(txn *mdbx.Txn) error {
						_, err := txn.Get(dbi, firstKey())
						return err
					}); err != nil {
						b.Error(err)
						return
					}
				}
			})
		})
	})
}

// strategy is one way of recycling a read-only transaction between uses.
//
// acquire is handed the previously released transaction, or nil when there is
// none, and returns one ready to read from. release makes it idle again.
type strategy struct {
	name string
	// staleView reports that acquire may return a transaction still viewing
	// an older snapshot rather than the latest committed one.
	staleView bool
	acquire   func(env *mdbx.Env, txn *mdbx.Txn) (*mdbx.Txn, error)
	release   func(txn *mdbx.Txn) error
	// reusable reports whether release leaves a handle worth passing back to
	// acquire, as opposed to destroying it.
	reusable bool
}

func begin(env *mdbx.Env) (*mdbx.Txn, error) { return env.BeginTxn(nil, mdbx.Readonly) }

var strategies = []strategy{{
	name:    "BeginAbort",
	acquire: func(env *mdbx.Env, _ *mdbx.Txn) (*mdbx.Txn, error) { return begin(env) },
	release: func(txn *mdbx.Txn) error { txn.Abort(); return nil },
}, {
	// What Pool does.
	name:     "ResetRenew",
	reusable: true,
	acquire: func(env *mdbx.Env, txn *mdbx.Txn) (*mdbx.Txn, error) {
		if txn == nil {
			return begin(env)
		}
		return txn, txn.Renew()
	},
	release: func(txn *mdbx.Txn) error { return txn.Reset() },
}, {
	name:     "ResetRefresh",
	reusable: true,
	acquire: func(env *mdbx.Env, txn *mdbx.Txn) (*mdbx.Txn, error) {
		if txn == nil {
			return begin(env)
		}
		_, err := txn.Refresh()
		return txn, err
	},
	release: func(txn *mdbx.Txn) error { return txn.Reset() },
}, {
	// Faster serially, but only because an un-ousted transaction resumes the
	// snapshot it was parked on instead of advancing to the tip.
	name:      "ParkUnpark",
	reusable:  true,
	staleView: true,
	acquire: func(env *mdbx.Env, txn *mdbx.Txn) (*mdbx.Txn, error) {
		if txn == nil {
			return begin(env)
		}
		_, err := txn.Unpark(true)
		return txn, err
	},
	release: func(txn *mdbx.Txn) error { return txn.Park(false) },
}, {
	name:     "ParkUnparkRefresh",
	reusable: true,
	acquire: func(env *mdbx.Env, txn *mdbx.Txn) (*mdbx.Txn, error) {
		if txn == nil {
			return begin(env)
		}
		restarted, err := txn.Unpark(true)
		if err != nil || restarted {
			// A restarted transaction already views the latest snapshot.
			return txn, err
		}
		_, err = txn.Refresh()
		return txn, err
	},
	release: func(txn *mdbx.Txn) error { return txn.Park(false) },
}}

// BenchmarkStrategies compares the ways libmdbx offers to recycle a read-only
// transaction, and is the evidence for Pool using Reset/Renew. See the package
// documentation for the numbers it produced and what they mean.
//
// Only ParkUnpark returns a possibly stale view, so its results are not
// directly comparable with the rest; ParkUnparkRefresh is its like-for-like
// counterpart.
func BenchmarkStrategies(b *testing.B) {
	for _, s := range strategies {
		b.Run(s.name, func(b *testing.B) {
			if s.staleView {
				b.Logf("%s may hand back an older snapshot; ParkUnparkRefresh is the comparable variant", s.name)
			}
			b.Run("Serial", func(b *testing.B) { benchSerial(b, s) })
			b.Run("ParallelWithWriter", func(b *testing.B) { benchParallelWithWriter(b, s) })
		})
	}
}

func benchSerial(b *testing.B, s strategy) {
	b.Helper()
	env, dbi := setup(b)
	var cur *mdbx.Txn
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		txn, err := s.acquire(env, cur)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := txn.Get(dbi, firstKey()); err != nil {
			b.Fatal(err)
		}
		if err := s.release(txn); err != nil {
			b.Fatal(err)
		}
		if s.reusable {
			cur = txn
		}
	}
	b.StopTimer()
	if cur != nil {
		cur.Abort()
	}
}

// benchParallelWithWriter is the workload a pool exists for: several readers
// recycling transactions while a writer keeps superseding pages, so parked
// readers actually get ousted.
func benchParallelWithWriter(b *testing.B, s strategy) {
	b.Helper()
	env, dbi := setup(b)

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
				b.Error(err)
				return
			}
		}
	})

	// A plain mutex-guarded stack, so the benchmark measures the recycling
	// strategy rather than a particular free-list design.
	var (
		mu   sync.Mutex
		free []*mdbx.Txn
	)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var cur *mdbx.Txn
			if s.reusable {
				mu.Lock()
				if n := len(free); n > 0 {
					cur, free = free[n-1], free[:n-1]
				}
				mu.Unlock()
			}

			txn, err := s.acquire(env, cur)
			if err != nil {
				b.Error(err)
				return
			}
			if _, err := txn.Get(dbi, firstKey()); err != nil {
				b.Error(err)
				return
			}
			if err := s.release(txn); err != nil {
				b.Error(err)
				return
			}
			if s.reusable {
				mu.Lock()
				free = append(free, txn)
				mu.Unlock()
			}
		}
	})
	b.StopTimer()

	close(stop)
	writer.Wait()
	for _, txn := range free {
		txn.Abort()
	}
}

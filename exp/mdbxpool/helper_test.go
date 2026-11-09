package mdbxpool

import (
	"fmt"
	"testing"

	"github.com/erigontech/mdbx-go/mdbx"
)

const (
	seedKeys  = 2048
	seedValue = 512
	pageSize  = 4096
)

func firstKey() []byte { return []byte("k-00000") }

// setup opens an Env seeded with seedKeys entries and returns it with the root
// DBI. The Env is closed by t.Cleanup, after any pool Cleanup registered later
// -- cleanups run last-registered-first -- so pooled readers release their
// slots before the Env goes away.
func setup(tb testing.TB) (*mdbx.Env, mdbx.DBI) {
	tb.Helper()

	env, err := mdbx.NewEnv(mdbx.Default)
	if err != nil {
		tb.Fatalf("new env: %v", err)
	}
	if err := env.SetOption(mdbx.OptMaxDB, 8); err != nil {
		tb.Fatalf("max db: %v", err)
	}
	if err := env.SetGeometry(-1, -1, 64*1024*pageSize, -1, -1, pageSize); err != nil {
		tb.Fatalf("geometry: %v", err)
	}
	// libmdbx's default logger writes page-management chatter to stdout, which
	// interleaves with test and benchmark output. mdbx_setup_debug is global,
	// so this quiets the whole test binary.
	if err := env.SetDebug(mdbx.LogLvlWarn, mdbx.DbgDoNotChange, mdbx.LoggerDoNotChange); err != nil {
		tb.Fatalf("set debug: %v", err)
	}
	if err := env.Open(tb.TempDir(), 0, 0664); err != nil {
		tb.Fatalf("open: %v", err)
	}
	tb.Cleanup(func() { env.Close() })

	var dbi mdbx.DBI
	err = env.Update(func(txn *mdbx.Txn) error {
		var err error
		if dbi, err = txn.OpenRoot(0); err != nil {
			return err
		}
		value := make([]byte, seedValue)
		for i := range seedKeys {
			if err := txn.Put(dbi, key(i), value, 0); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		tb.Fatalf("seed: %v", err)
	}
	return env, dbi
}

func key(i int) []byte { return fmt.Appendf(nil, "k-%05d", i) }

// churn rewrites the whole table `rounds` times, superseding every page. Pages
// the previous versions occupied can only be recycled once no reader still
// views them, which is what makes retention observable.
func churn(tb testing.TB, env *mdbx.Env, dbi mdbx.DBI, rounds int) {
	tb.Helper()
	for r := range rounds {
		value := make([]byte, seedValue+256)
		value[0] = byte(r)
		err := env.Update(func(txn *mdbx.Txn) error {
			for i := range seedKeys {
				if err := txn.Put(dbi, key(i), value, 0); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			tb.Fatalf("churn round %d: %v", r, err)
		}
	}
}

func readerStats(tb testing.TB, env *mdbx.Env) mdbx.ReaderStats {
	tb.Helper()
	st, err := env.ReaderStats()
	if err != nil {
		tb.Fatalf("reader stats: %v", err)
	}
	return st
}

func mustGet(tb testing.TB, p *Pool) *mdbx.Txn {
	tb.Helper()
	txn, err := p.Get()
	if err != nil {
		tb.Fatalf("get: %v", err)
	}
	return txn
}

func read(tb testing.TB, txn *mdbx.Txn, dbi mdbx.DBI) []byte {
	tb.Helper()
	v, err := txn.Get(dbi, firstKey())
	if err != nil {
		tb.Fatalf("read: %v", err)
	}
	return v
}

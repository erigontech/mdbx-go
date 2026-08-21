package mdbxpool_test

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/erigontech/mdbx-go/exp/mdbxpool"
	"github.com/erigontech/mdbx-go/mdbx"
)

// openEnv opens a throwaway environment holding a single key.
func openEnv() (*mdbx.Env, mdbx.DBI, func()) {
	dir, err := os.MkdirTemp("", "mdbxpool-example-")
	if err != nil {
		log.Fatal(err)
	}
	env, err := mdbx.NewEnv(mdbx.Default)
	if err != nil {
		log.Fatal(err)
	}
	if err := env.SetGeometry(-1, -1, 1<<26, -1, -1, 4096); err != nil {
		log.Fatal(err)
	}
	// Quiet libmdbx's page-management chatter, which otherwise interleaves
	// with test output. mdbx_setup_debug is global.
	if err := env.SetDebug(mdbx.LogLvlWarn, mdbx.DbgDoNotChange, mdbx.LoggerDoNotChange); err != nil {
		log.Fatal(err)
	}
	if err := env.Open(dir, 0, 0664); err != nil {
		log.Fatal(err)
	}

	var dbi mdbx.DBI
	err = env.Update(func(txn *mdbx.Txn) error {
		var err error
		if dbi, err = txn.OpenRoot(0); err != nil {
			return err
		}
		return txn.Put(dbi, []byte("greeting"), []byte("hello"), 0)
	})
	if err != nil {
		log.Fatal(err)
	}
	return env, dbi, func() {
		env.Close()
		os.RemoveAll(dir)
	}
}

func ExamplePool_View() {
	env, dbi, cleanup := openEnv()
	defer cleanup()

	// Close the pool before the Env, so pooled readers release their slots
	// first.
	p := mdbxpool.New(env)
	defer p.Close()

	for range 3 {
		err := p.View(func(txn *mdbx.Txn) error {
			v, err := txn.Get(dbi, []byte("greeting"))
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", v)
			return nil
		})
		if err != nil {
			// panic, not log.Fatal: os.Exit would skip the deferred Close.
			panic(err)
		}
	}

	// Only the first read had to begin a transaction; the other two reused it.
	st := p.Stats()
	fmt.Printf("hits=%d misses=%d idle=%d\n", st.Hits, st.Misses, st.Idle)

	// Output:
	// hello
	// hello
	// hello
	// hits=2 misses=1 idle=1
}

// Get and Put are for callers that need the transaction to outlive a single
// function call. Every Get must be matched by a Put, or the transaction leaks
// its reader slot.
func ExamplePool_Get() {
	env, dbi, cleanup := openEnv()
	defer cleanup()

	p := mdbxpool.New(env)
	defer p.Close()

	txn, err := p.Get()
	if err != nil {
		panic(err) // not log.Fatal: os.Exit would skip the deferred Close.
	}
	defer p.Put(txn)

	v, err := txn.Get(dbi, []byte("greeting"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", v)

	// Output:
	// hello
}

// WithMaxIdle bounds how many reader-table slots the pool holds while idle.
// Transactions returned beyond that are aborted rather than kept.
func ExampleWithMaxIdle() {
	env, dbi, cleanup := openEnv()
	defer cleanup()

	p := mdbxpool.New(env, mdbxpool.WithMaxIdle(2))
	defer p.Close()

	// Read concurrently, so more transactions are live at once than the pool
	// will keep.
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			err := p.View(func(txn *mdbx.Txn) error {
				_, err := txn.Get(dbi, []byte("greeting"))
				return err
			})
			if err != nil {
				log.Print(err)
			}
		})
	}
	wg.Wait()

	fmt.Println(p.Stats().Idle <= 2)

	// Output:
	// true
}

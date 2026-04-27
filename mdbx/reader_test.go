package mdbx

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestEnv_ReaderListEmpty(t *testing.T) {
	env, _ := setup(t)

	called := false
	if err := env.ReaderList(func(ReaderInfo) error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("reader list: %v", err)
	}
	if called {
		t.Fatalf("unexpected reader record in empty reader table")
	}
}

func TestEnv_ReadersIncludesOpenReadTxn(t *testing.T) {
	env, dbi := setupReaderInfoEnv(t)

	txn, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatalf("begin reader: %v", err)
	}
	defer txn.Abort()
	if _, err = txn.Get(dbi, []byte("key-0000")); err != nil {
		t.Fatalf("get: %v", err)
	}

	readers, err := env.Readers()
	if err != nil {
		t.Fatalf("readers: %v", err)
	}
	if !hasReader(readers, txn.ID()) {
		t.Fatalf("reader txid %d not found in %#v", txn.ID(), readers)
	}
}

func TestEnv_ReaderListCallbackError(t *testing.T) {
	env, _ := setupReaderInfoEnv(t)

	txn, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatalf("begin reader: %v", err)
	}
	defer txn.Abort()

	want := errors.New("stop reader list")
	err = env.ReaderList(func(ReaderInfo) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnv_ReaderStatsLagAndRetained(t *testing.T) {
	env, dbi := setupReaderInfoEnv(t)

	old, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatalf("begin old reader: %v", err)
	}
	defer old.Abort()
	if _, err = old.Get(dbi, []byte("key-0000")); err != nil {
		t.Fatalf("old get: %v", err)
	}

	if err = churnReaderInfoPages(env, dbi); err != nil {
		t.Fatalf("churn pages: %v", err)
	}

	stats, err := env.ReaderStats()
	if err != nil {
		t.Fatalf("reader stats: %v", err)
	}
	if stats.Count == 0 {
		t.Fatalf("expected at least one reader")
	}
	if old.ID() != 0 && stats.OldestTxID != old.ID() {
		t.Fatalf("oldest txid = %d, want %d", stats.OldestTxID, old.ID())
	}
	if stats.MaxLag == 0 {
		t.Fatalf("expected reader lag after write commit, got %#v", stats)
	}
	if stats.MaxBytesRetained == 0 || stats.SumBytesRetained == 0 {
		t.Fatalf("expected retained bytes after deletes with old reader, got %#v", stats)
	}
}

func TestEnv_ReaderListParked(t *testing.T) {
	env, dbi := setupReaderInfoEnv(t)

	txn, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatalf("begin reader: %v", err)
	}
	defer func() {
		_ = txn.Unpark(true)
		txn.Abort()
	}()
	if _, err = txn.Get(dbi, []byte("key-0000")); err != nil {
		t.Fatalf("get: %v", err)
	}
	if err = txn.Park(true); err != nil {
		t.Fatalf("park: %v", err)
	}

	readers, err := env.Readers()
	if err != nil {
		t.Fatalf("readers: %v", err)
	}
	for _, reader := range readers {
		if reader.Parked {
			return
		}
	}
	t.Fatalf("parked reader not found in %#v", readers)
}

func TestTxn_GCInfo(t *testing.T) {
	env, dbi := setupReaderInfoEnv(t)
	if err := churnReaderInfoPages(env, dbi); err != nil {
		t.Fatalf("churn pages: %v", err)
	}

	if err := env.View(func(txn *Txn) error {
		info, err := txn.GCInfo()
		if err != nil {
			return err
		}
		if info.PagesAllocated == 0 {
			t.Fatalf("expected allocated pages, got %#v", info)
		}
		if info.PagesBacked > info.PagesTotal {
			t.Fatalf("backed pages exceed total pages: %#v", info)
		}
		if info.PagesAllocated > info.PagesTotal {
			t.Fatalf("allocated pages exceed total pages: %#v", info)
		}
		if info.PagesReclaimable > info.PagesGC {
			t.Fatalf("reclaimable pages exceed GC pages: %#v", info)
		}
		if info.PagesRetained != info.PagesGC-info.PagesReclaimable {
			t.Fatalf("retained pages mismatch: %#v", info)
		}
		return nil
	}); err != nil {
		t.Fatalf("gc info: %v", err)
	}
}

func setupReaderInfoEnv(t *testing.T) (*Env, DBI) {
	t.Helper()
	env, _ := setup(t)
	var dbi DBI
	err := env.Update(func(txn *Txn) error {
		var err error
		dbi, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		value := make([]byte, 512)
		for i := range value {
			value[i] = byte(i)
		}
		for i := 0; i < 2048; i++ {
			key := []byte(fmt.Sprintf("key-%04d", i))
			if err = txn.Put(dbi, key, value, 0); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed env: %v", err)
	}
	return env, dbi
}

func churnReaderInfoPages(env *Env, dbi DBI) error {
	return env.Update(func(txn *Txn) error {
		value := make([]byte, 768)
		for i := range value {
			value[i] = byte(255 - i)
		}
		for i := 0; i < 1024; i++ {
			key := []byte(fmt.Sprintf("key-%04d", i))
			if err := txn.Put(dbi, key, value, 0); err != nil {
				return err
			}
		}
		for i := 1024; i < 2048; i++ {
			key := []byte(fmt.Sprintf("key-%04d", i))
			if err := txn.Del(dbi, key, nil); err != nil {
				return err
			}
		}
		return nil
	})
}

func hasReader(readers []ReaderInfo, txid uint64) bool {
	for _, reader := range readers {
		if reader.PID == os.Getpid() && reader.TxID == txid {
			return true
		}
	}
	return false
}

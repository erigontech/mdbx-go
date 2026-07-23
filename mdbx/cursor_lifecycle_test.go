package mdbx

import (
	"runtime"
	"testing"
)

// In libmdbx (unlike LMDB) cursors are never freed automatically when their
// transaction ends; Close must release the C cursor even after the write
// transaction has been committed, and must be a safe no-op the second time.
func TestCursor_CloseAfterWriteTxnCommit(t *testing.T) {
	env, _ := setup(t)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	txn, err := env.BeginTxn(nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	db, err := txn.OpenRoot(0)
	if err != nil {
		txn.Abort()
		t.Fatal(err)
	}
	cur, err := txn.OpenCursor(db)
	if err != nil {
		txn.Abort()
		t.Fatal(err)
	}
	if err := cur.Put([]byte("k"), []byte("v"), 0); err != nil {
		txn.Abort()
		t.Fatal(err)
	}
	if _, err := txn.Commit(); err != nil {
		t.Fatal(err)
	}

	cur.Close()
	cur.Close()
}

func TestCursor_CloseAfterWriteTxnAbort(t *testing.T) {
	env, _ := setup(t)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	txn, err := env.BeginTxn(nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	db, err := txn.OpenRoot(0)
	if err != nil {
		txn.Abort()
		t.Fatal(err)
	}
	cur, err := txn.OpenCursor(db)
	if err != nil {
		txn.Abort()
		t.Fatal(err)
	}
	txn.Abort()

	cur.Close()
	cur.Close()
}

// Close on a cursor that was never bound to a transaction
// (CreateCursor/CursorFromPool) must not panic.
func TestCursor_CloseNeverBound(t *testing.T) {
	c := CreateCursor()
	c.Close()
	c.Close()
}

// Get and PutReserve on an unbound cursor must return an error, not panic.
func TestCursor_UseNeverBound(t *testing.T) {
	c := CreateCursor()
	defer c.Close()

	if _, _, err := c.Get(nil, nil, First); err == nil {
		t.Error("Get on unbound cursor: expected error, got nil")
	}
	if _, err := c.PutReserve([]byte("k"), 1, 0); err == nil {
		t.Error("PutReserve on unbound cursor: expected error, got nil")
	}
}

// Unbind must disassociate the cursor from its transaction so a later Close
// (after the transaction ended) still releases the C cursor.
func TestCursor_UnbindThenClose(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	if err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		return txn.Put(db, []byte("k"), []byte("v"), 0)
	}); err != nil {
		t.Fatal(err)
	}

	txn, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatal(err)
	}
	cur, err := txn.OpenCursor(db)
	if err != nil {
		txn.Abort()
		t.Fatal(err)
	}
	if _, _, err := cur.Get(nil, nil, First); err != nil {
		txn.Abort()
		t.Fatal(err)
	}
	if err := cur.Unbind(); err != nil {
		txn.Abort()
		t.Fatal(err)
	}
	if cur.Txn() != nil {
		t.Error("Txn() should report nil after Unbind")
	}
	if _, _, err := cur.Get(nil, nil, First); err == nil {
		t.Error("Get on unbound cursor: expected error, got nil")
	}
	txn.Abort()
	cur.Close()
}

// A cursor returned to the pool must be unbound: pulling it back out and
// using it without Bind/Renew must error, not read through the old binding.
func TestCursorPool_UnboundOnReturn(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	if err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		return txn.Put(db, []byte("k"), []byte("v"), 0)
	}); err != nil {
		t.Fatal(err)
	}

	err := env.View(func(txn *Txn) error {
		c := CursorFromPool()
		if err := c.Bind(txn, db); err != nil {
			return err
		}
		if _, _, err := c.Get(nil, nil, First); err != nil {
			return err
		}
		CursorToPool(c) // must unbind

		reused := CursorFromPool()
		defer reused.Close()
		if reused.Txn() != nil {
			t.Error("pooled cursor still reports a Txn")
		}
		if _, _, err := reused.Get(nil, nil, First); err == nil {
			t.Error("Get on a pooled (unbound) cursor: expected error, got nil")
		}
		// Rebinding makes it usable again.
		if err := reused.Bind(txn, db); err != nil {
			return err
		}
		if _, _, err := reused.Get(nil, nil, First); err != nil {
			t.Errorf("Get after rebind: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

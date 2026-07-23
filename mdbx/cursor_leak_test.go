package mdbx

import (
	"runtime"
	"testing"
)

// Regression: in libmdbx (unlike LMDB) cursors are not freed when their
// transaction ends, so Close must release the C cursor even after the write
// transaction has committed/aborted, and must be a safe no-op the second time.
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

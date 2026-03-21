package mdbx

import (
	"testing"
)

func assertNoAllocs(t *testing.T, name string, fn func()) {
	t.Helper()
	// testing.AllocsPerRun runs fn once as a warm-up before measuring.
	allocs := testing.AllocsPerRun(100, fn)
	if allocs != 0 {
		t.Errorf("%s allocates %.0f allocs/op, want 0", name, allocs)
	}
}

func TestCursor_Count_NoAllocs(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple("testingdup_noalloc", Create|DupSort)
		if err != nil {
			return err
		}
		if err = txn.Put(db, []byte("k"), []byte("v0"), 0); err != nil {
			return err
		}
		return txn.Put(db, []byte("k"), []byte("v1"), 0)
	})
	if err != nil {
		t.Fatal(err)
	}

	err = env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		if _, _, err = cur.Get(nil, nil, First); err != nil {
			return err
		}

		assertNoAllocs(t, "Cursor.Count()", func() { _, _ = cur.Count() })
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestTxn_Sequence_NoAllocs(t *testing.T) {
	env, _ := setup(t)

	err := env.Update(func(txn *Txn) error {
		db, err := txn.OpenDBISimple("testingseq_noalloc", Create)
		if err != nil {
			return err
		}
		assertNoAllocs(t, "Txn.Sequence()", func() { _, _ = txn.Sequence(db, 0) })
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestEnv_GetOption_NoAllocs(t *testing.T) {
	env, _ := setup(t)
	assertNoAllocs(t, "Env.GetOption()", func() { _, _ = env.GetOption(OptMaxDB) })
}

func TestEnv_GetSyncPeriod_NoAllocs(t *testing.T) {
	env, _ := setup(t)
	assertNoAllocs(t, "Env.GetSyncPeriod()", func() { _, _ = env.GetSyncPeriod() })
}

func TestEnv_GetSyncBytes_NoAllocs(t *testing.T) {
	env, _ := setup(t)
	assertNoAllocs(t, "Env.GetSyncBytes()", func() { _, _ = env.GetSyncBytes() })
}

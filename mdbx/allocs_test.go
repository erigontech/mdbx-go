package mdbx

import (
	"testing"
)

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

		// warm up: ensure no lazy init on first call
		if _, err = cur.Count(); err != nil {
			return err
		}

		allocs := testing.AllocsPerRun(100, func() {
			_, _ = cur.Count()
		})
		if allocs != 0 {
			t.Errorf("Cursor.Count() allocates %.0f allocs/op, want 0", allocs)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

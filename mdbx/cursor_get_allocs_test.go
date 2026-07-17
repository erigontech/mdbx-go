package mdbx

import "testing"

func TestCursor_Get_NoAllocs(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple("getnoalloc", Create)
		if err != nil {
			return err
		}
		if err = txn.Put(db, []byte("k1"), []byte("v1"), 0); err != nil {
			return err
		}
		return txn.Put(db, []byte("k2"), []byte("v2"), 0)
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

		setkey := []byte("k1")
		assertNoAllocs(t, "Cursor.Get(nil, nil, First)", func() { _, _, _ = cur.Get(nil, nil, First) })
		assertNoAllocs(t, "Cursor.Get(nil, nil, Next)", func() { _, _, _ = cur.Get(nil, nil, First); _, _, _ = cur.Get(nil, nil, Next) })
		assertNoAllocs(t, "Cursor.Get(k, nil, SetRange)", func() { _, _, _ = cur.Get(setkey, nil, SetRange) })
		assertNoAllocs(t, "Cursor.Get(k, nil, Set)", func() { _, _, _ = cur.Get(setkey, nil, Set) })
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

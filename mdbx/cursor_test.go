//nolint:goconst
package mdbx

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"testing"
)

func TestCursor_Txn(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}

		_txn := cur.Txn()
		if _txn == nil {
			t.Errorf("nil cursor txn")
		}

		cur.Close()

		_txn = cur.Txn()
		if _txn != nil {
			t.Errorf("non-nil cursor txn")
		}

		return err
	})
	if err != nil {
		t.Error(err)
		return
	}
}

func TestCursor_DBI(t *testing.T) {
	env, _ := setup(t)

	err := env.Update(func(txn *Txn) (err error) {
		db, err := txn.OpenDBISimple("db", Create)
		if err != nil {
			return err
		}
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		dbcur := cur.DBI()
		if dbcur != db {
			cur.Close()
			return fmt.Errorf("unequal db: %v != %v", dbcur, db)
		}
		cur.Close()
		dbcur = cur.DBI()
		if dbcur == db {
			return fmt.Errorf("db: %v", dbcur)
		}
		if dbcur != ^DBI(0) {
			return fmt.Errorf("db: %v", dbcur)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCursor_Close(t *testing.T) {
	env, _ := setup(t)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	txn, err := env.BeginTxn(nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer txn.Abort()

	db, err := txn.OpenDBISimple("testing", Create)
	if err != nil {
		t.Fatal(err)
	}

	cur, err := txn.OpenCursor(db)
	if err != nil {
		t.Fatal(err)
	}
	cur.Close()
	cur.Close()
	err = cur.Put([]byte("closedput"), []byte("shouldfail"), 0)
	if err == nil {
		t.Fatalf("expected error: put on closed cursor")
	}
}

func TestCursor_bytesBuffer(t *testing.T) {
	env, _ := setup(t)

	db, err := openRoot(env, 0)
	if err != nil {
		t.Error(err)
		return
	}

	err = env.Update(func(txn *Txn) (err error) {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()
		k := new(bytes.Buffer)
		k.WriteString("hello")
		v := new(bytes.Buffer)
		v.WriteString("world")
		return cur.Put(k.Bytes(), v.Bytes(), 0)
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = env.View(func(txn *Txn) (err error) {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()
		k := new(bytes.Buffer)
		k.WriteString("hello")
		_k, v, err := cur.Get(k.Bytes(), nil, SetKey)
		if err != nil {
			return err
		}
		if !bytes.Equal(_k, k.Bytes()) {
			return fmt.Errorf("unexpected key: %q", _k)
		}
		if !bytes.Equal(v, []byte("world")) {
			return fmt.Errorf("unexpected value: %q", v)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}
}

func TestCursor_PutReserve(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	key := "reservekey"
	val := "reserveval"
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.CreateDBI("testing")
		if err != nil {
			return err
		}

		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		p, err := cur.PutReserve([]byte(key), len(val), 0)
		if err != nil {
			return err
		}
		copy(p, val)

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = env.View(func(txn *Txn) (err error) {
		dbval, err := txn.Get(db, []byte(key))
		if err != nil {
			return err
		}
		if !bytes.Equal(dbval, []byte(val)) {
			return fmt.Errorf("unexpected val %q != %q", dbval, val)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCursor_Get_KV(t *testing.T) {
	env, _ := setup(t)

	var dbi DBI
	err := env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenDBISimple("testdb", Create|DupSort)
		return err
	})
	if err != nil {
		t.Errorf("%s", err)
		return
	}

	err = env.Update(func(txn *Txn) (err error) {
		put := func(k, v []byte) {
			if err == nil {
				err = txn.Put(dbi, k, v, 0)
			}
		}
		put([]byte("k1"), []byte("v1"))
		put([]byte("k1"), []byte("v2"))
		put([]byte("k1"), []byte("v3"))
		return err
	})
	if err != nil {
		t.Errorf("%s", err)
	}

	err = env.View(func(txn *Txn) (err error) {
		cur, err := txn.OpenCursor(dbi)
		if err != nil {
			return err
		}
		defer cur.Close()

		k, v, err := cur.Get([]byte("k1"), []byte("v0"), GetBothRange)
		if err != nil {
			return err
		}
		if string(k) != "k1" {
			t.Errorf("unexpected key: %q (not %q)", k, "k1")
		}
		if string(v) != "v1" {
			t.Errorf("unexpected value: %q (not %q)", k, "1")
		}

		_, _, err = cur.Get([]byte("k0"), []byte("v0"), GetBothRange)
		if !IsNotFound(err) {
			t.Errorf("unexpected error: %s", err)
		}

		_, _, err = cur.Get([]byte("k1"), []byte("v1"), GetBoth)
		return err
	})
	if err != nil {
		t.Errorf("%s", err)
	}
}

func FromHex(in string) []byte {
	out, err := hex.DecodeString(in)
	if err != nil {
		panic(err)
	}
	return out
}

func TestLastDup(t *testing.T) {
	env, _ := setup(t)

	var dbi DBI
	err := env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenDBISimple("testdb", Create|DupSort)
		if err != nil {
			return err
		}

		err = txn.Put(dbi, []byte("key1"), []byte("value1.1"), 0)
		if err != nil {
			return err
		}
		err = txn.Put(dbi, []byte("key3"), []byte("value3.1"), 0)
		if err != nil {
			return err
		}
		err = txn.Put(dbi, []byte("key1"), []byte("value1.3"), 0)
		if err != nil {
			return err
		}
		err = txn.Put(dbi, []byte("key3"), []byte("value3.3"), 0)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = env.View(func(txn *Txn) error {
		c, err := txn.OpenCursor(dbi)
		if err != nil {
			return err
		}
		defer c.Close()

		i := 0
		for k, _, err := c.Get(nil, nil, First); k != nil; k, _, err = c.Get(nil, nil, NextNoDup) {
			if err != nil {
				return err
			}
			i++

			_, v, err := c.Get(nil, nil, LastDup)
			if err != nil {
				return err
			}
			if i == 1 && string(v) != "value1.3" {
				panic(1)
			}
			if i == 2 && string(v) != "value3.3" {
				panic(1)
			}

			_, v, err = c.Get(nil, nil, FirstDup)
			if err != nil {
				return err
			}
			if i == 1 && string(v) != "value1.1" {
				panic(1)
			}
			if i == 2 && string(v) != "value3.1" {
				panic(1)
			}

		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

}

func TestCursor_Get_op_Set_bytesBuffer(t *testing.T) {
	env, _ := setup(t)

	var dbi DBI
	err := env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenDBISimple("testdb", Create|DupSort)
		return err
	})
	if err != nil {
		t.Errorf("%s", err)
		return
	}

	err = env.Update(func(txn *Txn) (err error) {
		put := func(k, v []byte) {
			if err == nil {
				err = txn.Put(dbi, k, v, 0)
			}
		}
		put([]byte("k1"), []byte("v11"))
		put([]byte("k1"), []byte("v12"))
		put([]byte("k1"), []byte("v13"))
		put([]byte("k2"), []byte("v21"))
		put([]byte("k2"), []byte("v22"))
		put([]byte("k2"), []byte("v23"))
		return err
	})
	if err != nil {
		t.Errorf("%s", err)
	}

	err = env.View(func(txn *Txn) (err error) {
		cur, err := txn.OpenCursor(dbi)
		if err != nil {
			return err
		}
		defer cur.Close()

		// Create bytes.Buffer values containing a amount of bytes.  Byte
		// slices returned from buf.Bytes() have a history of tricking the cgo
		// argument checker.
		var kbuf bytes.Buffer
		kbuf.WriteString("k2")

		k, _, err := cur.Get(kbuf.Bytes(), nil, Set)
		if err != nil {
			return err
		}
		if string(k) != kbuf.String() {
			t.Errorf("unexpected key: %q (not %q)", k, kbuf.String())
		}

		// No guarantee is made about the return value of mdb_cursor_get when
		// MDB_SET is the op, so its value is not checked as part of this test.
		// That said, it is important that Cursor.Get not panic if given a
		// short buffer as an input value for a Set op (despite that not really
		// having any significance)
		var vbuf bytes.Buffer
		vbuf.WriteString("v22")

		k, _, err = cur.Get(kbuf.Bytes(), vbuf.Bytes(), Set)
		if err != nil {
			return err
		}
		if string(k) != kbuf.String() {
			t.Errorf("unexpected key: %q (not %q)", k, kbuf.String())
		}

		return nil
	})
	if err != nil {
		t.Errorf("%s", err)
	}
}

func TestCursor_Get_DupFixed(t *testing.T) {
	env, _ := setup(t)

	const datasize = 16
	pagesize := os.Getpagesize()
	numitems := (2 * pagesize / datasize) + 1

	var dbi DBI
	key := []byte("key")
	err := env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenDBISimple("test", DupSort|DupFixed|Create)
		if err != nil {
			return err
		}

		for i := int64(0); i < int64(numitems); i++ {
			err = txn.Put(dbi, key, []byte(fmt.Sprintf("%016x", i)), 0)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}

	var items [][]byte
	err = env.View(func(txn *Txn) (err error) {
		cur, err := txn.OpenCursor(dbi)
		if err != nil {
			return err
		}
		defer cur.Close()

		for {
			k, first, err := cur.Get(nil, nil, NextNoDup)
			if IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}

			if string(k) != string(key) {
				return fmt.Errorf("key: %s", k)
			}

			stride := len(first)

			for {
				_, v, err := cur.Get(nil, nil, NextMultiple)
				if IsNotFound(err) {
					break
				}
				if err != nil {
					return err
				}

				multi := WrapMulti(v, stride)
				for i := 0; i < multi.Len(); i++ {
					items = append(items, multi.Val(i))
				}
			}
		}
	})
	if err != nil {
		t.Error(err)
	}

	if len(items) != numitems {
		t.Errorf("unexpected number of items: %d (!= %d)", len(items), numitems)
	}

	for i, b := range items {
		expect := fmt.Sprintf("%016x", i)
		if string(b) != expect {
			t.Errorf("unexpected value: %q (!= %q)", b, expect)
		}
	}
}

func TestCursor_Get_reverse(t *testing.T) {
	env, _ := setup(t)

	var dbi DBI
	err := env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		err = txn.Put(dbi, []byte("k0"), []byte("v0"), 0)
		if err != nil {
			return err
		}
		err = txn.Put(dbi, []byte("k1"), []byte("v1"), 0)
		if err != nil {
			return err
		}
		return err
	})
	if err != nil {
		t.Error(err)
	}

	type Item struct{ k, v []byte }
	var items []Item

	err = env.View(func(txn *Txn) (err error) {
		cur, err := txn.OpenCursor(dbi)
		if err != nil {
			return err
		}
		defer cur.Close()

		for {
			k, v, err := cur.Get(nil, nil, Prev)
			if IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			items = append(items, Item{k, v})
		}
	})
	if err != nil {
		t.Error(err)
	}

	expect := []Item{
		{[]byte("k1"), []byte("v1")},
		{[]byte("k0"), []byte("v0")},
	}
	if !reflect.DeepEqual(items, expect) {
		t.Errorf("unexpected items %q (!= %q)", items, expect)
	}
}

// func TestCursor_PutMulti(t *testing.T) {
//	env := setup(t)
//
//
//	key := []byte("k")
//	items := [][]byte{
//		[]byte("v0"),
//		[]byte("v2"),
//		[]byte("v1"),
//	}
//	page := bytes.Join(items, nil)
//	stride := 2
//
//	var dbi DBI
//	err := env.Update(func(txn *Txn) (err error) {
//		dbi, err = txn.OpenDBISimple("test2", Create|DupSort|DupFixed)
//		if err != nil {
//			return err
//		}
//
//		cur, err := txn.OpenCursor(dbi)
//		if err != nil {
//			return err
//		}
//		defer cur.Close()
//
//		return cur.PutMulti(key, page, stride, 0)
//	})
//	if err != nil {
//		t.Error(err)
//	}
//
//	expect := [][]byte{
//		[]byte("v0"),
//		[]byte("v1"),
//		[]byte("v2"),
//	}
//	var dbitems [][]byte
//	err = env.View(func(txn *Txn) (err error) {
//		cur, err := txn.OpenCursor(dbi)
//		if err != nil {
//			return err
//		}
//		defer cur.Close()
//
//		for {
//			k, v, err := cur.Get(nil, nil, Next)
//			if IsNotFound(err) {
//				return nil
//			}
//			if err != nil {
//				return err
//			}
//			if string(k) != "k" {
//				return fmt.Errorf("key: %q", k)
//			}
//			dbitems = append(dbitems, v)
//		}
//	})
//	if err != nil {
//		t.Error(err)
//	}
//	if !reflect.DeepEqual(dbitems, expect) {
//		t.Errorf("unexpected items: %q (!= %q)", dbitems, items)
//	}
//}

func TestCursor_Del(t *testing.T) {

	env, _ := setup(t)

	var db DBI
	type Item struct{ k, v string }
	items := []Item{
		{"k0", "k0"},
		{"k1", "k1"},
		{"k2", "k2"},
	}
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.CreateDBI("testing")
		if err != nil {
			return err
		}

		for _, item := range items {
			err := txn.Put(db, []byte(item.k), []byte(item.v), 0)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}

	err = env.Update(func(txn *Txn) (err error) {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}

		item := items[1]
		k, v, err := cur.Get([]byte(item.k), nil, SetKey)
		if err != nil {
			return err
		}
		if !bytes.Equal(k, []byte(item.k)) {
			return fmt.Errorf("found key %q (!= %q)", k, item.k)
		}
		if !bytes.Equal(v, []byte(item.v)) {
			return fmt.Errorf("found value %q (!= %q)", k, item.v)
		}

		err = cur.Del(0)
		if err != nil {
			return err
		}

		k, v, err = cur.Get(nil, nil, Next)
		if err != nil {
			return fmt.Errorf("post-delete: %w", err)
		}
		item = items[2]
		if !bytes.Equal(k, []byte(item.k)) {
			return fmt.Errorf("found key %q (!= %q)", k, item.k)
		}
		if !bytes.Equal(v, []byte(item.v)) {
			return fmt.Errorf("found value %q (!= %q)", k, item.v)
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}

	var newitems []Item
	err = env.View(func(txn *Txn) (err error) {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		next := func(cur *Cursor) (k, v []byte, err error) { return cur.Get(nil, nil, Next) }
		for k, v, err := next(cur); !IsNotFound(err); k, v, err = next(cur) {
			if err != nil {
				return err
			}
			newitems = append(newitems, Item{string(k), string(v)})
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}

	expectitems := []Item{
		items[0],
		items[2],
	}
	if !reflect.DeepEqual(newitems, expectitems) {
		t.Errorf("unexpected items %q (!= %q)", newitems, expectitems)
	}
}

func TestDupCursor_EmptyKeyValues1(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple("testingdup", Create|DupSort)
		if err != nil {
			return err
		}
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		// empty value - must function as valid dupsort value
		if err = txn.Put(db, []byte{1}, []byte{}, 0); err != nil {
			panic(err)
		}
		if err = txn.Put(db, []byte{1}, []byte{8}, 0); err != nil {
			panic(err)
		}

		_, v, err := cur.Get([]byte{1}, []byte{}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{}) {
			panic(v)
		}
		_, v, err = cur.Get([]byte{1}, []byte{0}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{8}) {
			panic(v)
		}
		_, v, err = cur.Get([]byte{}, []byte{0}, GetBoth)
		if err == nil {
			panic("expecting 'not found' error")
		}
		if v != nil {
			panic(v)
		}

		// can use empty key as valid key in non-dupsort operations
		k, v, err := cur.Get([]byte{}, nil, SetRange)
		if err != nil {
			panic(err)
		}
		if k == nil {
			panic("nil")
		}
		if !bytes.Equal(k, []byte{1}) {
			panic(hex.EncodeToString(k))
		}
		if !bytes.Equal(v, []byte{}) {
			panic(hex.EncodeToString(k))
		}
		k, _, err = cur.Get([]byte{}, nil, Set)
		if err == nil {
			panic("expected 'not found' error")
		}
		if k != nil {
			panic("nil")
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestDupCursor_EmptyKeyValues2(t *testing.T) {
	t.Skip()
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple("testingdup", Create|DupSort)
		if err != nil {
			return err
		}
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		// empty value - must function as valid dupsort value
		if err = txn.Put(db, []byte{1}, []byte{}, 0); err != nil {
			panic(err)
		}
		if err = txn.Put(db, []byte{1}, []byte{8}, 0); err != nil {
			panic(err)
		}

		_, v, err := cur.Get([]byte{1}, []byte{}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{}) {
			panic(v)
		}
		_, v, err = cur.Get([]byte{1}, []byte{0}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{8}) {
			panic(v)
		}
		_, v, err = cur.Get([]byte{}, []byte{0}, GetBoth)
		if err == nil {
			panic("expecting 'not found' error")
		}
		if v != nil {
			panic(v)
		}

		// can use empty key as valid key in non-dupsort operations
		k, v, err := cur.Get([]byte{}, nil, SetRange)
		if err != nil {
			panic(err)
		}
		if k == nil {
			panic("nil")
		}
		if !bytes.Equal(k, []byte{1}) {
			panic(hex.EncodeToString(k))
		}
		if !bytes.Equal(v, []byte{}) {
			panic(hex.EncodeToString(k))
		}
		k, _, err = cur.Get([]byte{}, nil, Set)
		if err == nil {
			panic("expected 'not found' error")
		}
		if k != nil {
			panic("nil")
		}

		// empty key - must function as valid dupsort key
		if err = txn.Put(db, []byte{}, []byte{}, 0); err != nil {
			panic(err)
		}
		if err = txn.Put(db, []byte{}, []byte{2}, 0); err != nil {
			panic(err)
		}
		_, v, err = cur.Get([]byte{}, []byte{}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{}) {
			panic(v)
		}
		_, v, err = cur.Get([]byte{}, []byte{0}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{2}) {
			panic(v)
		}
		_, v, err = cur.Get([]byte{}, []byte{0}, GetBoth)
		if err == nil {
			panic("expecting 'not found' error ")
		}
		if v != nil {
			panic(v)
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestDupCursor_EmptyKeyValues3(t *testing.T) {
	t.Skip()
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple("testingdup", Create|DupSort)
		if err != nil {
			return err
		}
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		// empty value - must function as valid dupsort value
		if err = txn.Put(db, []byte{1}, []byte{}, 0); err != nil {
			panic(err)
		}
		if err = txn.Put(db, []byte{1}, []byte{8}, 0); err != nil {
			panic(err)
		}

		_, v, err := cur.Get([]byte{1}, []byte{}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{}) {
			panic(v)
		}
		_, v, err = cur.Get([]byte{1}, []byte{0}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{8}) {
			panic(v)
		}
		_, v, err = cur.Get([]byte{}, []byte{0}, GetBoth)
		if err == nil {
			panic("expecting 'not found' error")
		}
		if v != nil {
			panic(v)
		}

		// can use empty key as valid key in non-dupsort operations
		k, v, err := cur.Get([]byte{}, nil, SetRange)
		if err != nil {
			panic(err)
		}
		if k == nil {
			panic("nil")
		}
		if !bytes.Equal(k, []byte{1}) {
			panic(hex.EncodeToString(k))
		}
		if !bytes.Equal(v, []byte{}) {
			panic(hex.EncodeToString(k))
		}
		k, _, err = cur.Get([]byte{}, nil, Set)
		if err == nil {
			panic("expected 'not found' error")
		}
		if k != nil {
			panic("nil")
		}

		// empty key - must function as valid dupsort key
		if err = txn.Put(db, []byte{}, []byte{}, 0); err != nil {
			panic(err)
		}
		if err = txn.Put(db, []byte{}, []byte{2}, 0); err != nil {
			panic(err)
		}
		_, v, err = cur.Get([]byte{}, []byte{}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{}) {
			panic(v)
		}
		_, v, err = cur.Get([]byte{}, []byte{0}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{2}) {
			panic(v)
		}
		_, v, err = cur.Get([]byte{}, []byte{0}, GetBoth)
		if err == nil {
			panic("expecting 'not found' error ")
		}
		if v != nil {
			panic(v)
		}

		// non-existing key
		_, v, err = cur.Get([]byte{7}, []byte{}, GetBoth)
		if err == nil {
			panic("expecting 'not found' error")
		}
		if v != nil {
			panic(v)
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestDupCursor_EmptyKeyValues(t *testing.T) {
	t.Skip()
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple("testingdup", Create|DupSort)
		if err != nil {
			return err
		}
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		// empty value - must function as valid dupsort value
		if err = txn.Put(db, []byte{1}, []byte{}, 0); err != nil {
			panic(err)
		}
		if err = txn.Put(db, []byte{1}, []byte{8}, 0); err != nil {
			panic(err)
		}

		_, v, err := cur.Get([]byte{1}, []byte{}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{}) {
			panic(v)
		}
		_, v, err = cur.Get([]byte{1}, []byte{0}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{8}) {
			panic(v)
		}
		_, v, err = cur.Get([]byte{}, []byte{0}, GetBoth)
		if err == nil {
			panic("expecting 'not found' error")
		}
		if v != nil {
			panic(v)
		}

		// can use empty key as valid key in non-dupsort operations
		k, v, err := cur.Get([]byte{}, nil, SetRange)
		if err != nil {
			panic(err)
		}
		if k == nil {
			panic("nil")
		}
		if !bytes.Equal(k, []byte{1}) {
			panic(hex.EncodeToString(v))
		}
		if !bytes.Equal(v, []byte{}) {
			panic(hex.EncodeToString(v))
		}
		k, _, err = cur.Get([]byte{}, nil, Set)
		if err == nil {
			panic("expected 'not found' error")
		}
		if k != nil {
			panic("nil")
		}

		// empty key - must function as valid dupsort key
		if err = txn.Put(db, []byte{}, []byte{}, 0); err != nil {
			panic(err)
		}
		if err = txn.Put(db, []byte{}, []byte{2}, 0); err != nil {
			panic(err)
		}
		_, v, err = cur.Get([]byte{}, []byte{}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{}) {
			panic(v)
		}
		_, v, err = cur.Get([]byte{}, []byte{0}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{2}) {
			panic(v)
		}
		_, v, err = cur.Get([]byte{}, []byte{0}, GetBoth)
		if err == nil {
			panic("expecting 'not found' error ")
		}
		if v != nil {
			panic(v)
		}

		// non-existing key
		_, v, err = cur.Get([]byte{7}, []byte{}, GetBoth)
		if err == nil {
			panic("expecting 'not found' error")
		}
		if v != nil {
			panic(v)
		}

		// sub-db doesn't have empty value, but we must be able to search from it
		if err = txn.Put(db, []byte{2}, []byte{1}, 0); err != nil {
			return err
		}
		if err = txn.Put(db, []byte{2}, []byte{3}, 0); err != nil {
			return err
		}
		_, v, err = cur.Get([]byte{2}, []byte{}, GetBothRange)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{1}) {
			panic(v)
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

// This test verifies the behavior of Cursor.Count when DupSort is provided.
func TestCursor_Count_DupSort(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple("testingdup", Create|DupSort)
		if err != nil {
			return err
		}

		put := func(k, v string) {
			if err != nil {
				return
			}
			err = txn.Put(db, []byte(k), []byte(v), 0)
		}
		put("k", "v0")
		put("k", "v1")

		return err
	})
	if err != nil {
		t.Error(err)
	}

	err = env.View(func(txn *Txn) (err error) {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		_, _, err = cur.Get(nil, nil, First)
		if err != nil {
			return err
		}
		numdup, err := cur.Count()
		if err != nil {
			return err
		}

		if numdup != 2 {
			t.Errorf("unexpected count: %d != %d", numdup, 2)
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCursor_Del_DupSort(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple("testingdup", Create|DupSort)
		if err != nil {
			return err
		}

		put := func(k, v string) {
			if err != nil {
				return
			}
			err = txn.Put(db, []byte(k), []byte(v), 0)
		}
		put("k", "v0")
		put("k", "v1")

		return err
	})
	if err != nil {
		t.Error(err)
	}

	err = env.Update(func(txn *Txn) (err error) {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		_, _, err = cur.Get(nil, nil, First)
		if err != nil {
			return err
		}
		numdup, err := cur.Count()
		if err != nil {
			panic(err)
		}

		if numdup != 2 {
			t.Errorf("unexpected count: %d != %d", numdup, 2)
		}
		err = cur.Del(0)
		if err != nil {
			panic(err)
		}

		kk, vv, err := cur.Get(nil, nil, NextDup)
		if err != nil {
			panic(err)
		}
		_, _ = kk, vv //TODO: add assert
		// fmt.Printf("kk: %s\n", kk)
		// fmt.Printf("vv: %s\n", vv)

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

// This test verifies the behavior of Cursor.Count when DupSort is not enabled
// on the database.
func TestCursor_Count_noDupSort(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple("testingnodup", Create)
		if err != nil {
			return err
		}

		return txn.Put(db, []byte("k"), []byte("v1"), 0)
	})
	if err != nil {
		t.Error(err)
	}

	// it is an error to call Count if the underlying database does not allow
	// duplicate keys.
	err = env.View(func(txn *Txn) (err error) {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		_, _, err = cur.Get(nil, nil, First)
		if err != nil {
			return err
		}
		_, err = cur.Count()
		if err != nil {
			t.Errorf("expected no error: %v", err)
			return nil
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCursor_Renew(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		return err
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = env.Update(func(txn *Txn) (err error) {
		put := func(k, v string) {
			if err == nil {
				err = txn.Put(db, []byte(k), []byte(v), 0)
			}
		}
		put("k1", "v1")
		put("k2", "v2")
		put("k3", "v3")
		return err
	})
	if err != nil {
		t.Error("err")
	}

	var cur *Cursor
	err = env.View(func(txn *Txn) (err error) {
		cur, err = txn.OpenCursor(db)
		if err != nil {
			return err
		}

		k, v, err := cur.Get(nil, nil, Next)
		if err != nil {
			return err
		}
		if string(k) != "k1" {
			return fmt.Errorf("key: %q", k)
		}
		if string(v) != "v1" {
			return fmt.Errorf("val: %q", v)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}

	err = env.View(func(txn *Txn) (err error) {
		err = cur.Renew(txn)
		if err != nil {
			return err
		}

		k, v, err := cur.Get(nil, nil, Next)
		if err != nil {
			return err
		}
		if string(k) != "k1" {
			return fmt.Errorf("key: %q", k)
		}
		if string(v) != "v1" {
			return fmt.Errorf("val: %q", v)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCursor_Get_OneKey_Parallel(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	k := make([]byte, 8)
	binary.BigEndian.PutUint64(k, uint64(1))

	if err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		c, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		err = txn.Put(db, k, k, 0)
		if err != nil {
			return err
		}

		ch := make(chan error)
		go func() {
			_, _, err = c.Get(k, nil, Set)
			if err != nil {
				ch <- err
			}
			ch <- nil
		}()
		err = <-ch
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Errorf("dbi: %v", err)
		return
	}
}

func TestCursor_Get_OneKey_SeparateProcess(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	k := make([]byte, 8)
	binary.BigEndian.PutUint64(k, uint64(1))

	if err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		err = txn.Put(db, k, k, 0)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		t.Errorf("dbi: %v", err)
		return
	}

	if err := env.View(func(txn *Txn) (err error) {
		c, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		ch := make(chan error)
		go func() {
			_, _, err = c.Get(k, nil, Set)
			if err != nil {
				ch <- err
			}
			ch <- nil
		}()
		err = <-ch
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Errorf("dbi: %v", err)
		return
	}
}

func TestCursor_Bind(t *testing.T) {
	env, _ := setup(t)

	var db1, db2 DBI
	err := env.Update(func(txn *Txn) (err error) {
		db1, err = txn.CreateDBI("testing1")
		return err
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = env.Update(func(txn *Txn) (err error) {
		db2, err = txn.CreateDBI("testing2")
		return err
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = env.Update(func(txn *Txn) (err error) {
		put := func(k, v string) {
			if err == nil {
				err = txn.Put(db1, []byte(k), []byte(v), 0)
			}
		}
		put("k1", "v1")
		put("k2", "v2")
		put("k3", "v3")
		return err
	})
	if err != nil {
		t.Error("err")
	}

	var cur *Cursor
	err = env.View(func(txn *Txn) (err error) {
		cur, err = txn.OpenCursor(db2)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}

	err = env.View(func(txn *Txn) (err error) {
		{
			err = cur.Bind(txn, db1)
			if err != nil {
				return err
			}

			k, v, err := cur.Get(nil, nil, Next)
			if err != nil {
				return err
			}
			if string(k) != "k1" {
				return fmt.Errorf("key: %q", k)
			}
			if string(v) != "v1" {
				return fmt.Errorf("val: %q", v)
			}
		}

		{
			c2 := CreateCursor()
			err = c2.Bind(txn, db1)
			if err != nil {
				return err
			}

			k, v, err := c2.Get(nil, nil, Next)
			if err != nil {
				return err
			}
			if string(k) != "k1" {
				return fmt.Errorf("key: %q", k)
			}
			if string(v) != "v1" {
				return fmt.Errorf("val: %q", v)
			}
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCursor_BindOnEmptyDbi(t *testing.T) {
	env, _ := setup(t)

	var db1, db2 DBI
	err := env.Update(func(txn *Txn) (err error) {
		db1, err = txn.CreateDBI("testing1")
		return err
	})
	if err != nil {
		t.Error(err)
		return
	}

	var cur *Cursor
	err = env.View(func(txn *Txn) (err error) {
		cur, err = txn.OpenCursor(db2)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}

	err = env.View(func(txn *Txn) (err error) {
		err = cur.Bind(txn, db1)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func BenchmarkCursor(b *testing.B) {
	env, _ := setup(b)

	var db DBI
	err := env.View(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		b.Error(err)
		return
	}

	err = env.View(func(txn *Txn) (err error) {
		b.ResetTimer()
		defer b.StopTimer()

		for i := 0; i < b.N; i++ {
			cur, err := txn.OpenCursor(db)
			if err != nil {
				return err
			}
			cur.Close()
		}
		return
	})
	if err != nil {
		b.Error(err)
		return
	}
}

func BenchmarkCursor_Renew(b *testing.B) {
	env, _ := setup(b)

	var db DBI
	var cur *Cursor
	err := env.View(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		cur, err = txn.OpenCursor(db)
		return err
	})
	if err != nil {
		b.Error(err)
		return
	}

	_ = env.View(func(txn *Txn) (err error) {
		b.Run("1", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if err := cur.Renew(txn); err != nil {
					panic(err)
				}
			}
		})
		b.Run("2", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if err := cur.Bind(txn, db); err != nil {
					panic(err)
				}
			}
		})
		b.Run("3", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				c, err := txn.OpenCursor(db)
				if err != nil {
					panic(err)
				}
				c.Close()
			}
		})
		b.Run("4", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				cur := CursorFromPool()
				if err := cur.Bind(txn, db); err != nil {
					panic(err)
				}
				CursorToPool(cur)
			}
		})
		return nil
	})
}

func BenchmarkCursor_Set_OneKey(b *testing.B) {
	env, _ := setup(b)

	var db DBI
	k := make([]byte, 8)
	binary.BigEndian.PutUint64(k, uint64(1))

	if err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		err = txn.Put(db, k, k, 0)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		b.Errorf("dbi: %v", err)
		return
	}

	if err := env.View(func(txn *Txn) (err error) {
		c, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := c.Get(k, nil, Set)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		b.Errorf("put: %v", err)
	}
}

func BenchmarkCursor_Set_Sequence(b *testing.B) {
	env, _ := setup(b)

	var db DBI
	keys := make([][]byte, b.N)
	for i := range keys {
		keys[i] = make([]byte, 8)
		binary.BigEndian.PutUint64(keys[i], uint64(i))
	}

	if err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		for _, k := range keys {
			err = txn.Put(db, k, k, 0)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		b.Errorf("dbi: %v", err)
		return
	}

	if err := env.View(func(txn *Txn) (err error) {
		c, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err = c.Get(keys[i], nil, Set)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		b.Errorf("put: %v", err)
	}
}

func BenchmarkCursor_Set_Random(b *testing.B) {
	env, _ := setup(b)

	var db DBI
	keys := make([][]byte, b.N)
	for i := range keys {
		keys[i] = make([]byte, 8)
		binary.BigEndian.PutUint64(keys[i], uint64(rand.Intn(100*b.N)))
	}

	if err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		for _, k := range keys {
			err = txn.Put(db, k, k, 0)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		b.Errorf("dbi: %v", err)
		return
	}

	if err := env.View(func(txn *Txn) (err error) {
		b.ResetTimer()
		c, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err = c.Get(keys[i], nil, Set)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		b.Errorf("put: %v", err)
	}
}

type kv struct{ k, v string }

// ---------- helpers ----------

func mustOpenUniqueDB(t *testing.T, env *Env, name string) DBI {
	t.Helper()
	var db DBI
	if err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple(name, Create)
		return err
	}); err != nil {
		t.Fatalf("OpenDBISimple(unique): %v", err)
	}
	return db
}

func mustOpenDupSortDB(t *testing.T, env *Env, name string) DBI {
	t.Helper()
	var db DBI
	if err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple(name, Create|DupSort)
		return err
	}); err != nil {
		t.Fatalf("OpenDBISimple(dupsort): %v", err)
	}
	return db
}

func mustPutUniqueSeq(t *testing.T, env *Env, db DBI, n int) {
	t.Helper()
	if err := env.Update(func(txn *Txn) error {
		for i := 0; i < n; i++ {
			k := fmt.Sprintf("k%d", i)
			v := fmt.Sprintf("v%d", i)
			if err := txn.Put(db, []byte(k), []byte(v), 0); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("put unique seq: %v", err)
	}
}

func mustPutDupVals(t *testing.T, env *Env, db DBI, key string, vals []string) {
	t.Helper()
	if err := env.Update(func(txn *Txn) error {
		for _, v := range vals {
			if err := txn.Put(db, []byte(key), []byte(v), 0); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("put dup vals: %v", err)
	}
}

func dumpAll(t *testing.T, env *Env, db DBI) []kv {
	t.Helper()
	var out []kv
	if err := env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		k, v, err := cur.Get(nil, nil, First)
		if IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for {
			out = append(out, kv{string(k), string(v)})
			k, v, err = cur.Get(nil, nil, Next)
			if IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}); err != nil {
		t.Fatalf("dumpAll: %v", err)
	}
	return out
}

func dumpKeyDups(t *testing.T, env *Env, db DBI, key string) []string {
	t.Helper()
	var out []string
	if err := env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		k, v, err := cur.Get([]byte(key), nil, SetKey)
		if err != nil {
			return err
		}
		if string(k) != key {
			return fmt.Errorf("positioned on unexpected key: %q", k)
		}
		out = append(out, string(v))

		for {
			_, v, err = cur.Get(nil, nil, NextDup)
			if IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			out = append(out, string(v))
		}
	}); err != nil {
		t.Fatalf("dumpKeyDups(%s): %v", key, err)
	}
	return out
}

const RangeDel Label = "RangeDel"

func requireEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected:\n got:  %#v\n want: %#v", got, want)
	}
}

// ---------- tests: unique DB ----------

func TestCursor_RangeDel_Unique_DeleteCurrentValue(t *testing.T) {
	env, _ := setupWithLabel(t, RangeDel)
	db := mustOpenUniqueDB(t, env, "rangedel_unique_cur")
	mustPutUniqueSeq(t, env, db, 5) // k0..k4

	var affected uint64
	if err := env.Update(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		_, _, err = cur.Get([]byte("k2"), nil, SetKey)
		if err != nil {
			return err
		}
		affected, err = cur.RangeDel(DeleteCurrentValue)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if affected != 1 {
		t.Fatalf("affected=%d, want 1", affected)
	}
	requireEqual(t, dumpAll(t, env, db), []kv{
		{"k0", "v0"}, {"k1", "v1"}, {"k3", "v3"}, {"k4", "v4"},
	})
}

func TestCursor_RangeDel_Unique_DeleteBeforeIncluding(t *testing.T) {
	env, _ := setupWithLabel(t, RangeDel)
	db := mustOpenUniqueDB(t, env, "rangedel_unique_before_incl")
	mustPutUniqueSeq(t, env, db, 5)

	var affected uint64
	if err := env.Update(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		_, _, err = cur.Get([]byte("k2"), nil, SetKey)
		if err != nil {
			return err
		}
		affected, err = cur.RangeDel(DeleteBeforeIncluding) // k0,k1,k2
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if affected != 3 {
		t.Fatalf("affected=%d, want 3", affected)
	}
	requireEqual(t, dumpAll(t, env, db), []kv{{"k3", "v3"}, {"k4", "v4"}})
}

func TestCursor_RangeDel_Unique_DeleteBeforeExcluding(t *testing.T) {
	env, _ := setupWithLabel(t, RangeDel)
	db := mustOpenUniqueDB(t, env, "rangedel_unique_before_excl")
	mustPutUniqueSeq(t, env, db, 5)

	var affected uint64
	if err := env.Update(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		_, _, err = cur.Get([]byte("k2"), nil, SetKey)
		if err != nil {
			return err
		}
		affected, err = cur.RangeDel(DeleteBeforeExcluding) // k0,k1
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if affected != 2 {
		t.Fatalf("affected=%d, want 2", affected)
	}
	requireEqual(t, dumpAll(t, env, db), []kv{
		{"k2", "v2"}, {"k3", "v3"}, {"k4", "v4"},
	})
}

func TestCursor_RangeDel_Unique_DeleteAfterIncluding(t *testing.T) {
	env, _ := setupWithLabel(t, RangeDel)
	db := mustOpenUniqueDB(t, env, "rangedel_unique_after_incl")
	mustPutUniqueSeq(t, env, db, 5)

	var affected uint64
	if err := env.Update(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		_, _, err = cur.Get([]byte("k2"), nil, SetKey)
		if err != nil {
			return err
		}
		affected, err = cur.RangeDel(DeleteAfterIncluding) // k2,k3,k4
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if affected != 3 {
		t.Fatalf("affected=%d, want 3", affected)
	}
	requireEqual(t, dumpAll(t, env, db), []kv{{"k0", "v0"}, {"k1", "v1"}})
}

func TestCursor_RangeDel_Unique_DeleteAfterExcluding(t *testing.T) {
	env, _ := setupWithLabel(t, RangeDel)
	db := mustOpenUniqueDB(t, env, "rangedel_unique_after_excl")
	mustPutUniqueSeq(t, env, db, 5)

	var affected uint64
	if err := env.Update(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		_, _, err = cur.Get([]byte("k2"), nil, SetKey)
		if err != nil {
			return err
		}
		affected, err = cur.RangeDel(DeleteAfterExcluding) // k3,k4
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if affected != 2 {
		t.Fatalf("affected=%d, want 2", affected)
	}
	requireEqual(t, dumpAll(t, env, db), []kv{
		{"k0", "v0"}, {"k1", "v1"}, {"k2", "v2"},
	})
}

func TestCursor_RangeDel_Unique_DeleteWhole(t *testing.T) {
	env, _ := setupWithLabel(t, RangeDel)
	db := mustOpenUniqueDB(t, env, "rangedel_unique_whole")
	mustPutUniqueSeq(t, env, db, 5)

	var affected uint64
	if err := env.Update(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		// Many “whole” APIs accept any cursor state; to be safe, position to First.
		_, _, err = cur.Get(nil, nil, First)
		if err != nil {
			return err
		}
		affected, err = cur.RangeDel(DeleteWhole)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if affected != 5 {
		t.Fatalf("affected=%d, want 5", affected)
	}
	requireEqual(t, dumpAll(t, env, db), []kv(nil))
}

// ---------- tests: dupsort DB (single key, multiple dup vals) ----------

func TestCursor_RangeDel_DupSort_DeleteCurrentValue(t *testing.T) {
	env, _ := setupWithLabel(t, RangeDel)
	db := mustOpenDupSortDB(t, env, "rangedel_dups_cur")
	mustPutDupVals(t, env, db, "k", []string{"v0", "v1", "v2", "v3"})

	var affected uint64
	if err := env.Update(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		// Position on specific dup-val via GetBoth.
		_, _, err = cur.Get([]byte("k"), []byte("v2"), GetBoth)
		if err != nil {
			return err
		}
		affected, err = cur.RangeDel(DeleteCurrentValue) // delete only v2
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if affected != 1 {
		t.Fatalf("affected=%d, want 1", affected)
	}
	requireEqual(t, dumpKeyDups(t, env, db, "k"), []string{"v0", "v1", "v3"})
}

func TestCursor_RangeDel_DupSort_DeleteCurrentValueMultiValAll(t *testing.T) {
	env, _ := setupWithLabel(t, RangeDel)
	db := mustOpenDupSortDB(t, env, "rangedel_dups_all")
	mustPutDupVals(t, env, db, "k", []string{"v0", "v1", "v2"})

	var affected uint64
	if err := env.Update(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		_, _, err = cur.Get([]byte("k"), []byte("v1"), GetBoth)
		if err != nil {
			return err
		}
		affected, err = cur.RangeDel(DeleteCurrentValueMultiValAll) // remove all dupvals of key "k"
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if affected != 3 {
		t.Fatalf("affected=%d, want 3", affected)
	}
	requireEqual(t, dumpAll(t, env, db), []kv(nil))
}

func TestCursor_RangeDel_DupSort_DeleteCurrentMultiValBeforeIncluding(t *testing.T) {
	env, _ := setupWithLabel(t, RangeDel)
	db := mustOpenDupSortDB(t, env, "rangedel_dups_before_incl")
	mustPutDupVals(t, env, db, "k", []string{"v0", "v1", "v2", "v3"})

	var affected uint64
	if err := env.Update(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		_, _, err = cur.Get([]byte("k"), []byte("v2"), GetBoth)
		if err != nil {
			return err
		}
		affected, err = cur.RangeDel(DeleteCurrentMultiValBeforeIncluding) // v0,v1,v2
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if affected != 3 {
		t.Fatalf("affected=%d, want 3", affected)
	}
	requireEqual(t, dumpKeyDups(t, env, db, "k"), []string{"v3"})
}

func TestCursor_RangeDel_DupSort_DeleteCurrentMultiValBeforeExcluding(t *testing.T) {
	env, _ := setupWithLabel(t, RangeDel)
	db := mustOpenDupSortDB(t, env, "rangedel_dups_before_excl")
	mustPutDupVals(t, env, db, "k", []string{"v0", "v1", "v2", "v3"})

	var affected uint64
	if err := env.Update(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		_, _, err = cur.Get([]byte("k"), []byte("v2"), GetBoth)
		if err != nil {
			return err
		}
		affected, err = cur.RangeDel(DeleteCurrentMultiValBeforeExcluding) // v0,v1
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if affected != 2 {
		t.Fatalf("affected=%d, want 2", affected)
	}
	requireEqual(t, dumpKeyDups(t, env, db, "k"), []string{"v2", "v3"})
}

func TestCursor_RangeDel_DupSort_DeleteCurrentMultiValAfterIncluding(t *testing.T) {
	env, _ := setupWithLabel(t, RangeDel)
	db := mustOpenDupSortDB(t, env, "rangedel_dups_after_incl")
	mustPutDupVals(t, env, db, "k", []string{"v0", "v1", "v2", "v3"})

	var affected uint64
	if err := env.Update(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		_, _, err = cur.Get([]byte("k"), []byte("v1"), GetBoth)
		if err != nil {
			return err
		}
		affected, err = cur.RangeDel(DeleteCurrentMultiValAfterIncluding) // v1,v2,v3
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if affected != 3 {
		t.Fatalf("affected=%d, want 3", affected)
	}
	requireEqual(t, dumpKeyDups(t, env, db, "k"), []string{"v0"})
}

func TestCursor_RangeDel_DupSort_DeleteCurrentMultiValAfterExcluding(t *testing.T) {
	env, _ := setupWithLabel(t, RangeDel)
	db := mustOpenDupSortDB(t, env, "rangedel_dups_after_excl")
	mustPutDupVals(t, env, db, "k", []string{"v0", "v1", "v2", "v3"})

	var affected uint64
	if err := env.Update(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		_, _, err = cur.Get([]byte("k"), []byte("v1"), GetBoth)
		if err != nil {
			return err
		}
		affected, err = cur.RangeDel(DeleteCurrentMultiValAfterExcluding) // v2,v3
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if affected != 2 {
		t.Fatalf("affected=%d, want 2", affected)
	}
	requireEqual(t, dumpKeyDups(t, env, db, "k"), []string{"v0", "v1"})
}

func TestCursor_RangeDel_DupSort_DeleteWhole(t *testing.T) {
	env, _ := setupWithLabel(t, RangeDel)
	db := mustOpenDupSortDB(t, env, "rangedel_dups_whole")

	// 2 keys to ensure whole is whole, not only current key.
	mustPutDupVals(t, env, db, "k1", []string{"a0", "a1"})
	mustPutDupVals(t, env, db, "k2", []string{"b0", "b1", "b2"})

	var affected uint64
	if err := env.Update(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		_, _, err = cur.Get(nil, nil, First)
		if err != nil {
			return err
		}
		affected, err = cur.RangeDel(DeleteWhole)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if affected != 5 { // 2 + 3
		t.Fatalf("affected=%d, want 5", affected)
	}
	requireEqual(t, dumpAll(t, env, db), []kv(nil))
}

// ---------- optional: empty table contract ----------

func TestCursor_RangeDel_EmptyTable_DeleteWhole(t *testing.T) {
	env, _ := setupWithLabel(t, RangeDel)
	db := mustOpenUniqueDB(t, env, "rangedel_empty_whole")

	// No puts.

	var affected uint64
	if err := env.Update(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		// For empty DB, positioning may return NotFound; we still try DeleteWhole.
		affected, err = cur.RangeDel(DeleteWhole)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if affected != 0 {
		t.Fatalf("affected=%d, want 0", affected)
	}
}

package mdbx

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestEmptyKeysAndValues(t *testing.T) {
	env, err1 := NewEnv(Default)
	if err1 != nil {
		t.Fatalf("Cannot create environment: %s", err1)
	}
	err1 = env.SetGeometry(-1, -1, 1024*1024, -1, -1, 4096)
	if err1 != nil {
		t.Fatalf("Cannot set mapsize: %s", err1)
	}
	path := t.TempDir()
	err1 = env.Open(path, 0, 0664)
	defer env.Close()
	if err1 != nil {
		t.Fatalf("Cannot open environment: %s", err1)
	}

	var db DBI
	numEntries := 4
	if err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		if err != nil {
			panic(err)
		}

		err = txn.Put(db, nil, []byte{}, NoOverwrite)
		if err != nil {
			panic(err)
		}
		err = txn.Put(db, []byte{}, []byte{}, NoOverwrite)
		if err == nil { // expect err: MDBX_KEYEXIST
			panic(err)
		}
		err = txn.Put(db, []byte{1}, []byte{}, NoOverwrite)
		if err != nil {
			panic(err)
		}
		err = txn.Put(db, []byte{2}, nil, NoOverwrite)
		if err != nil {
			panic(err)
		}
		err = txn.Put(db, []byte{3}, []byte{1}, NoOverwrite)
		if err != nil {
			panic(err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	stat, err1 := env.Stat()
	if err1 != nil {
		t.Fatalf("Cannot get stat %s", err1)
	} else if stat.Entries != uint64(numEntries) {
		t.Errorf("Less entry in the database than expected: %d <> %d", stat.Entries, numEntries)
	}
	if err := env.View(func(txn *Txn) error {
		v, err := txn.Get(db, nil)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{}) {
			panic(hex.EncodeToString(v))
		}
		v, err = txn.Get(db, []byte{})
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{}) {
			panic(hex.EncodeToString(v))
		}
		v, err = txn.Get(db, []byte{1})
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{}) {
			panic(hex.EncodeToString(v))
		}
		v, err = txn.Get(db, []byte{2})
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{}) {
			panic(hex.EncodeToString(v))
		}
		v, err = txn.Get(db, []byte{3})
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, []byte{1}) {
			panic(hex.EncodeToString(v))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestTest1(t *testing.T) {
	env, err1 := NewEnv(Default)
	if err1 != nil {
		t.Fatalf("Cannot create environment: %s", err1)
	}
	err1 = env.SetGeometry(-1, -1, 1024*1024, -1, -1, 4096)
	if err1 != nil {
		t.Fatalf("Cannot set mapsize: %s", err1)
	}
	path := t.TempDir()
	err1 = env.Open(path, 0, 0664)
	defer env.Close()
	if err1 != nil {
		t.Fatalf("Cannot open environment: %s", err1)
	}

	var db DBI
	numEntries := 10
	var data = map[string]string{}
	var key string
	var val string
	for i := 0; i < numEntries; i++ {
		key = fmt.Sprintf("Key-%d", i)
		val = fmt.Sprintf("Val-%d", i)
		data[key] = val
	}
	if err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}

		for k, v := range data {
			err = txn.Put(db, []byte(k), []byte(v), NoOverwrite)
			if err != nil {
				return fmt.Errorf("put: %w", err)
			}
		}

		return nil
	}); err != nil {
		t.Fatal(err)
	}

	stat, err1 := env.Stat()
	if err1 != nil {
		t.Fatalf("Cannot get stat %s", err1)
	} else if stat.Entries != uint64(numEntries) {
		t.Errorf("Less entry in the database than expected: %d <> %d", stat.Entries, numEntries)
	}
	t.Logf("%#v", stat)

	if err := env.View(func(txn *Txn) error {
		cursor, err := txn.OpenCursor(db)
		if err != nil {
			cursor.Close()
			return fmt.Errorf("cursor: %w", err)
		}
		var bkey, bval []byte
		var bNumVal int
		for {
			bkey, bval, err = cursor.Get(nil, nil, Next)
			if IsNotFound(err) {
				break
			}
			if err != nil {
				return fmt.Errorf("cursor get: %w", err)
			}
			bNumVal++
			skey := string(bkey)
			sval := string(bval)
			t.Logf("Val: %s", sval)
			t.Logf("Key: %s", skey)
			var d string
			var ok bool
			if d, ok = data[skey]; !ok {
				return fmt.Errorf("cursor get: key does not exist %q", skey)
			}
			if d != sval {
				return fmt.Errorf("cursor get: value %q does not match %q", sval, d)
			}
		}
		if bNumVal != numEntries {
			t.Errorf("cursor iterated over %d entries when %d expected", bNumVal, numEntries)
		}
		cursor.Close()
		bval, err = txn.Get(db, []byte("Key-0"))
		if err != nil {
			return fmt.Errorf("get: %w", err)
		}
		if string(bval) != "Val-0" {
			return fmt.Errorf("get: value %q does not match %q", bval, "Val-0")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// func TestVersion(t *testing.T) {
//	maj, min, patch, str := Version()
//	if maj < 0 || min < 0 || patch < 0 {
//		t.Error("invalid version number: ", maj, min, patch)
//	}
//	if maj == 0 && min == 0 && patch == 0 {
//		t.Error("invalid version number: ", maj, min, patch)
//	}
//	if str == "" {
//		t.Error("empty version string")
//	}
//
//	str = VersionString()
//	if str == "" {
//		t.Error("empty version string")
//	}
// }

func TestGetSysRamInfo(t *testing.T) {
	env, err1 := NewEnv(Default)
	if err1 != nil {
		t.Fatalf("Cannot create environment: %s", err1)
	}
	err1 = env.SetGeometry(-1, -1, 1024*1024, -1, -1, 4096)
	if err1 != nil {
		t.Fatalf("Cannot set mapsize: %s", err1)
	}
	path := t.TempDir()
	err1 = env.Open(path, 0, 0664)
	defer env.Close()
	if err1 != nil {
		t.Fatalf("Cannot open environment: %s", err1)
	}

	var db DBI
	if err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		if err != nil {
			panic(err)
		}

		err = txn.Put(db, nil, []byte{}, NoOverwrite)
		if err != nil {
			panic(err)
		}
		err = txn.Put(db, []byte{}, []byte{}, NoOverwrite)
		if err == nil { // expect err: MDBX_KEYEXIST
			panic(err)
		}
		err = txn.Put(db, []byte{1}, []byte{}, NoOverwrite)
		if err != nil {
			panic(err)
		}
		err = txn.Put(db, []byte{2}, nil, NoOverwrite)
		if err != nil {
			panic(err)
		}
		err = txn.Put(db, []byte{3}, []byte{1}, NoOverwrite)
		if err != nil {
			panic(err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	pageSize, totalPages, availablePages, err := GetSysRamInfo()
	if err != nil {
		t.Fatal(err)
	}

	println(pageSize, totalPages, availablePages) // no asserts because it's (at least avP) pretty random TODO: think about how to avoid it
}

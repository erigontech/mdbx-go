//nolint:goconst
package mdbx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"syscall"
	"testing"
)

func TestTxn_ID(t *testing.T) {
	env, _ := setup(t)

	var id0, id1, id2, id3 uint64
	var txnInvalid *Txn
	err := env.View(func(txn *Txn) (err error) {
		id0 = txn.ID()
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	if id0 != 3 {
		t.Errorf("unexpected readonly id (before update): %v (!= %v)", id0, 3)
	}

	txnCached, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Error(err)
		return
	}
	defer txnCached.Abort()
	if txnCached.ID() != 3 {
		t.Errorf("unexpected readonly id (before update): %v (!= %v)", txnCached.ID(), 3)
	}
	if txnCached.getID() != txnCached.ID() {
		t.Errorf("unexpected readonly id (before update): %v (!= %v)", txnCached.ID(), txnCached.getID())
	}
	if err := txnCached.Reset(); err != nil {
		t.Fatal(err)
	}
	if txnCached.getID() != txnCached.ID() {
		t.Errorf("unexpected reset id: %v (!= %v)", txnCached.ID(), txnCached.getID())
	}

	err = env.Update(func(txn *Txn) (err error) {
		dbi, err := txn.OpenDBISimple("test", Create)
		if err != nil {
			return err
		}
		id1 = txn.ID()
		return txn.Put(dbi, []byte("key"), []byte("val"), 0)
	})
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	err = env.View(func(txn *Txn) (err error) {
		id2 = txn.ID()
		txnInvalid = txn
		return nil
	})
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	id3 = txnInvalid.ID()

	// The ID of txnCached will actually change during the call to
	// txnCached.Renew().  It's imperative that any ID cached in the Txn object
	// does not diverge.
	if txnCached.ID() != txnCached.getID() {
		t.Errorf("unexpected invalid id: %v (!= %v)", txnCached.ID(), txnCached.getID())
	}
	err = txnCached.Renew()
	if err != nil {
		t.Error(err)
		return
	}
	if txnCached.ID() != txnCached.getID() {
		t.Errorf("unexpected invalid id: %v (!= %v)", txnCached.ID(), txnCached.getID())
	}

	t.Logf("ro txn id:: %v", id1)
	t.Logf("txn id: %v", id1)
	t.Logf("ro txn id: %v", id2)
	t.Logf("bad txn id: %v", id3)
	if id1 != id2 {
		t.Errorf("unexpected readonly id: %v (!= %v)", id2, id1)
	}
	if id3 != 0 {
		t.Errorf("unexpected invalid id: %v (!= %v)", id3, 0)
	}
	if id1 != txnCached.ID() {
		t.Errorf("unexpected invalid id: %v (!= %v)", txnCached.ID(), 0)
	}
}

func TestTxn_errLogf(t *testing.T) {
	env, _ := setup(t)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	txn, err := env.BeginTxn(nil, 0)
	if err != nil {
		t.Error(err)
	} else {
		defer txn.Abort()
		txn.errf("this is just a test")
	}
}

func TestTxn_Drop(t *testing.T) {
	env, _ := setup(t)

	db, err := openDBI(env, "db", Create)
	if err != nil {
		t.Error(err)
		return
	}

	err = env.Update(func(txn *Txn) (err error) {
		return txn.Put(db, []byte("k"), []byte("v"), 0)
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = env.Update(func(txn *Txn) (err error) {
		return txn.Drop(db, false)
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = env.View(func(txn *Txn) (err error) {
		_, err = txn.Get(db, []byte("k"))
		return err
	})
	if !IsNotFound(err) {
		t.Error(err)
		return
	}

	err = env.Update(func(txn *Txn) (err error) {
		return txn.Drop(db, true)
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = env.View(func(txn *Txn) (err error) {
		_, err = txn.Get(db, []byte("k"))
		return err
	})
	if !IsErrno(err, BadDBI) {
		t.Errorf("mdb_get: %v", err)
	}
}

func TestTxn_Del(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple("text", Create)
		return err
	})
	if err != nil {
		panic(err)
	}

	err = env.Update(func(txn *Txn) (err error) {
		return txn.Put(db, []byte("k"), []byte("v"), 0)
	})
	if err != nil {
		t.Error(err)
	}

	err = env.Update(func(txn *Txn) (err error) {
		return txn.Del(db, []byte("k"), nil)
	})
	if err != nil {
		t.Error(err)
	}

	err = env.View(func(txn *Txn) (err error) {
		_, err = txn.Get(db, []byte("k"))
		return err
	})
	if !IsNotFound(err) {
		t.Errorf("mdb_txn_get: %v", err)
	}
}

func TestTxn_Del_dup(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple("text", Create|DupSort)
		return err
	})
	if err != nil {
		panic(err)
	}

	err = env.Update(func(txn *Txn) (err error) {
		return txn.Put(db, []byte("k"), []byte("v"), 0)
	})
	if err != nil {
		t.Error(err)
	}

	err = env.Update(func(txn *Txn) (err error) {
		return txn.Del(db, []byte("k"), []byte("valignored"))
	})
	if !IsNotFound(err) {
		t.Errorf("mdb_del: %v", err)
	}

	err = env.View(func(txn *Txn) (err error) {
		v, err := txn.Get(db, []byte("k"))
		if err != nil {
			return err
		}
		if string(v) != "v" {
			return fmt.Errorf("unexpected value: %q", v)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestTexn_Put_emptyValue(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple("text", Create)
		if err != nil {
			return err
		}
		err = txn.Put(db, []byte("k"), nil, 0)
		if err != nil {
			return err
		}
		v, err := txn.Get(db, []byte("k"))
		if err != nil {
			return err
		}
		if len(v) != 0 {
			t.Errorf("value: %q (!= \"\")", v)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}
}

func TestTxn_PutReserve(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		val := "v"
		err = txn.Put(db, []byte("k"), []byte(val), 0)
		if err != nil {
			return err
		}
		p, err := txn.PutReserve(db, []byte("k"), len(val), 0)
		if err != nil {
			return err
		}
		copy(p, val)
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = env.View(func(txn *Txn) (err error) {
		v, err := txn.Get(db, []byte("k"))
		if err != nil {
			return err
		}
		if string(v) != "v" {
			return fmt.Errorf("value: %q", v)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}
}

func TestTxn_bytesBuffer(t *testing.T) {
	env, _ := setup(t)

	db, err := openRoot(env, 0)
	if err != nil {
		t.Error(err)
		return
	}

	err = env.Update(func(txn *Txn) (err error) {
		k := new(bytes.Buffer)
		k.WriteString("hello")
		v := new(bytes.Buffer)
		v.WriteString("world")
		return txn.Put(db, k.Bytes(), v.Bytes(), 0)
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = env.View(func(txn *Txn) (err error) {
		k := new(bytes.Buffer)
		k.WriteString("hello")
		v, err := txn.Get(db, k.Bytes())
		if err != nil {
			return err
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

func TestTxn_Put_overwrite(t *testing.T) {
	env, _ := setup(t)

	db, err := openRoot(env, 0)
	if err != nil {
		t.Error(err)
		return
	}

	err = env.Update(func(txn *Txn) (err error) {
		k := []byte("hello")
		v := []byte("world")
		err = txn.Put(db, k, v, 0)
		if err != nil {
			return err
		}
		copy(k, "bye!!")
		copy(v, "toodles")
		err = txn.Put(db, k, v, 0)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = env.View(func(txn *Txn) (err error) {
		v, err := txn.Get(db, []byte("hello"))
		if err != nil {
			return err
		}
		if !bytes.Equal(v, []byte("world")) {
			return errors.New("unexpected value")
		}
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}
}

func TestTxn_OpenDBI_emptyName(t *testing.T) {
	env, _ := setup(t)

	err := env.View(func(txn *Txn) (err error) {
		_, err = txn.OpenDBISimple("", 0)
		return err
	})
	if !IsNotFound(err) {
		t.Errorf("mdb_dbi_open: %v", err)
	}

	err = env.View(func(txn *Txn) (err error) {
		_, err = txn.OpenDBISimple("", Create)
		return err
	})
	if runtime.GOOS == "windows" {
		if !IsErrnoSys(err, syscall.EIO) {
			t.Errorf("mdb_dbi_open: %v", err)
		}
	} else {
		if !IsErrnoSys(err, syscall.EACCES) {
			t.Errorf("mdb_dbi_open: %v", err)
		}
	}
}

func TestTxn_OpenDBI_zero(t *testing.T) {
	env, _ := setup(t)

	txn, err := env.BeginTxn(nil, 0)
	if err != nil {
		panic(err)
	}
	defer txn.Abort()

	dbi, err := txn.OpenRoot(0)
	if err != nil {
		panic(err)
	}
	_, err = txn.Get(dbi, []byte("k"))
	if !errors.Is(err, ErrNotFound) {
		panic(err)
	}
}

func TestTxn_Commit_managed(t *testing.T) {
	env, _ := setup(t)

	err := env.View(func(txn *Txn) (err error) {
		defer func() {
			if e := recover(); e == nil {
				t.Errorf("expected panic: %v", err)
			}
		}()
		_, err2 := txn.Commit()
		return err2
	})
	if err != nil {
		t.Error(err)
	}

	err = env.View(func(txn *Txn) (err error) {
		defer func() {
			if e := recover(); e == nil {
				t.Errorf("expected panic: %v", err)
			}
		}()
		txn.Abort()
		return errors.New("abort")
	})
	if err != nil {
		t.Error(err)
	}

	err = env.View(func(txn *Txn) (err error) {
		defer func() {
			if e := recover(); e == nil {
				t.Errorf("expected panic: %v", err)
			}
		}()
		return txn.Renew()
	})
	if err != nil {
		t.Error(err)
	}

	err = env.View(func(txn *Txn) (err error) {
		defer func() {
			if e := recover(); e == nil {
				t.Errorf("expected panic: %v", err)
			}
		}()
		_ = txn.Reset()
		return errors.New("reset")
	})
	if err != nil {
		t.Error(err)
	}
}

func TestTxn_Commit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fix me")
	}
	env, _ := setup(t)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	txn, err := env.BeginTxn(nil, 0)
	if err != nil {
		t.Error(err)
		return
	}
	txn.Abort()
	_, err = txn.Commit()
	if !IsErrnoSys(err, syscall.EINVAL) {
		t.Errorf("mdb_txn_commit: %s", err.Error())
	}
}

func TestTxn_Update(t *testing.T) {
	env, _ := setup(t)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(Create)
		if err != nil {
			return err
		}
		err = txn.Put(db, []byte("mykey"), []byte("myvalue"), 0)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Errorf("update: %v", err)
		return
	}

	err = env.View(func(txn *Txn) (err error) {
		v, err := txn.Get(db, []byte("mykey"))
		if err != nil {
			return err
		}
		if string(v) != "myvalue" {
			return fmt.Errorf("value: %q", v)
		}
		return nil
	})
	if err != nil {
		t.Errorf("view: %v", err)
		return
	}
}

func TestTxn_Flags(t *testing.T) {
	env, path := setup(t)

	dbflags := uint(ReverseKey | ReverseDup | DupSort | DupFixed)
	err := env.Update(func(txn *Txn) (err error) {
		db, err := txn.OpenDBISimple("testdb", dbflags|Create)
		if err != nil {
			return err
		}
		err = txn.Put(db, []byte("bcd"), []byte("exval1"), 0)
		if err != nil {
			return err
		}
		err = txn.Put(db, []byte("abcda"), []byte("exval3"), 0)
		if err != nil {
			return err
		}
		err = txn.Put(db, []byte("abcda"), []byte("exval2"), 0)
		if err != nil {
			return err
		}
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()
		k, v, err := cur.Get(nil, nil, Next)
		if err != nil {
			return err
		}
		if string(k) != "abcda" { // ReverseKey does not do what one might expect
			return fmt.Errorf("unexpected first key: %q", k)
		}
		if string(v) != "exval2" {
			return fmt.Errorf("unexpected first value: %q", v)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}
	env.Close()

	// opening the database after it is created inherits the original flags.
	env, err = NewEnv(Default)
	if err != nil {
		t.Error(err)
		return
	}
	err = env.SetOption(OptMaxDB, uint64(1))
	if err != nil {
		t.Error(err)
		return
	}
	defer env.Close()
	err = env.Open(path, 0, 0644)
	if err != nil {
		t.Error(err)
		return
	}
	err = env.View(func(txn *Txn) (err error) {
		db, err := txn.OpenDBISimple("testdb", dbflags)
		if err != nil {
			return err
		}
		flags, err := txn.Flags(db)
		if err != nil {
			return err
		}
		if flags != dbflags {
			return errors.New("unexpected flags")
		}
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		k, v, err := cur.Get(nil, nil, Next)
		if err != nil {
			return err
		}
		if string(k) != "abcda" {
			return fmt.Errorf("unexpected first key: %q", k)
		}
		if string(v) != "exval2" {
			return fmt.Errorf("unexpected first value: %q", v)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}
}

func TestTxn_Renew(t *testing.T) {
	env, _ := setup(t)

	// It is not necessary to call runtime.LockOSThread in this test because
	// the only unmanaged Txn is Readonly.

	var dbroot DBI
	err := env.Update(func(txn *Txn) (err error) {
		dbroot, err = txn.OpenRoot(0)
		return err
	})
	if err != nil {
		t.Error(err)
		return
	}

	txn, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Error(err)
		return
	}
	defer txn.Abort()
	_, err = txn.Get(dbroot, []byte("k"))
	if !IsNotFound(err) {
		t.Errorf("get: %v", err)
	}

	err = env.Update(func(txn *Txn) (err error) {
		dbroot, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		return txn.Put(dbroot, []byte("k"), []byte("v"), 0)
	})
	if err != nil {
		t.Error(err)
	}

	_, err = txn.Get(dbroot, []byte("k"))
	if !IsNotFound(err) {
		t.Errorf("get: %v", err)
	}
	if err := txn.Reset(); err != nil {
		t.Fatal(err)
	}

	err = txn.Renew()
	if err != nil {
		t.Error(err)
	}
	val, err := txn.Get(dbroot, []byte("k"))
	if err != nil {
		t.Error(err)
	}
	if string(val) != "v" {
		t.Errorf("unexpected value: %q", val)
	}
}

func TestTxn_Reset_doubleReset(t *testing.T) {
	env, _ := setup(t)

	txn, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Error(err)
		return
	}
	defer txn.Abort()

	_ = txn.Reset()
	_ = txn.Reset()
}

func TestTxn_ParkUnpark(t *testing.T) {
	env, _ := setup(t)

	txn, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Error(err)
		return
	}
	defer txn.Abort()
	err = txn.Park(true)
	if err != nil {
		t.Error(err)
		return
	}
	err = txn.Unpark(true)
	if err != nil {
		t.Error(err)
		return
	}
}

// This test demonstrates that Reset/Renew have no effect on writable
// transactions. The transaction may be committed after Reset/Renew are called.
func TestTxn_Reset_writeTxn(t *testing.T) {
	env, _ := setup(t)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	txn, err := env.BeginTxn(nil, 0)
	if err != nil {
		t.Error(err)
		return
	}
	defer txn.Abort()

	db, err := txn.OpenRoot(0)
	if err != nil {
		t.Error(err)
	}
	err = txn.Put(db, []byte("k"), []byte("v"), 0)
	if err != nil {
		t.Error(err)
	}

	// Reset is a noop and Renew will always error out.
	_ = txn.Reset()
	err = txn.Renew()
	if runtime.GOOS == "windows" {
		// todo
	} else if !IsErrnoSys(err, syscall.EINVAL) {
		t.Errorf("renew: %v", err)
	}

	_, err = txn.Commit()
	if err != nil {
		t.Errorf("commit: %v", err)
	}

	err = env.View(func(txn *Txn) (err error) {
		val, err := txn.Get(db, []byte("k"))
		if err != nil {
			return err
		}
		if string(val) != "v" {
			return fmt.Errorf("unexpected value: %q", val)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestTxn_UpdateLocked(t *testing.T) {
	env, _ := setup(t)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var dbi DBI
	err := env.UpdateLocked(func(txn *Txn) (err error) {
		dbi, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		return txn.Put(dbi, []byte("k0"), []byte("v0"), 0)
	})
	if err != nil {
		t.Error(err)
	}

	err = env.View(func(txn *Txn) (err error) {
		v, err := txn.Get(dbi, []byte("k0"))
		if err != nil {
			return err
		}
		if string(v) != "v0" {
			return fmt.Errorf("unexpected value: %q (!= %q)", v, "v0")
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestTxn_RunTxn(t *testing.T) {
	env, _ := setup(t)

	var dbi DBI
	err := env.RunTxn(0, func(txn *Txn) (err error) {
		dbi, err = txn.OpenRoot(0)
		if err != nil {
			return err
		}
		return txn.Put(dbi, []byte("k0"), []byte("v0"), 0)
	})
	if err != nil {
		t.Error(err)
	}

	err = env.RunTxn(Readonly, func(txn *Txn) (err error) {
		v, err := txn.Get(dbi, []byte("k0"))
		if err != nil {
			return err
		}
		if string(v) != "v0" {
			return fmt.Errorf("unexpected value: %q (!= %q)", v, "v0")
		}
		err = txn.Put(dbi, []byte("k1"), []byte("v1"), 0)
		if err == nil {
			return errors.New("allowed to Put in a readonly Txn")
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestTxn_Stat(t *testing.T) {
	env, _ := setup(t)

	var dbi DBI
	err := env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenDBISimple("testdb", Create)
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
		put([]byte("a"), []byte("1"))
		put([]byte("b"), []byte("2"))
		put([]byte("c"), []byte("3"))
		return err
	})
	if err != nil {
		t.Errorf("%s", err)
	}

	var stat *Stat
	err = env.View(func(txn *Txn) (err error) {
		stat, err = txn.StatDBI(dbi)
		return err
	})
	if err != nil {
		t.Errorf("%s", err)
		return
	}
	if stat.Entries != 3 {
		t.Errorf("unexpected entries: %d (expected %d)", stat.Entries, 3)
	}
}

func TestTxn_StatOnEmpty(t *testing.T) {
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
	env.CloseDBI(dbi)

	err = env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenDBISimple("testdb", DupSort)
		return err
	})
	if err != nil {
		t.Errorf("%s", err)
		return
	}

	err = env.Update(func(txn *Txn) (err error) {
		err = txn.Drop(dbi, true)
		return err
	})
	if err != nil {
		t.Errorf("%s", err)
		return
	}
}

func TestSequence(t *testing.T) {
	env, _ := setup(t)

	var dbi1 DBI
	var dbi2 DBI
	err := env.Update(func(txn *Txn) (err error) {
		dbi1, err = txn.OpenDBISimple("testdb", Create)
		if err != nil {
			return err
		}
		dbi2, err = txn.OpenDBISimple("testdb2", Create)
		return err
	})
	if err != nil {
		t.Errorf("%s", err)
		return
	}

	err = env.Update(func(txn *Txn) (err error) {
		v, err := txn.Sequence(dbi1, 0) // 0 accepted, validate on app level
		if err != nil {
			return err
		}
		if v != 0 {
			t.Errorf("unexpected value: %d (expected %d)", v, 0)
		}
		v, err = txn.Sequence(dbi2, 2)
		if err != nil {
			return err
		}
		if v != 0 {
			t.Errorf("unexpected value: %d (expected %d)", v, 1)
		}

		v, err = txn.Sequence(dbi1, 3)
		if err != nil {
			return err
		}
		if v != 0 {
			t.Errorf("unexpected value: %d (expected %d)", 0, 0)
		}

		return nil
	})
	if err != nil {
		t.Errorf("%s", err)
	}

	err = env.View(func(txn *Txn) (err error) {
		v, err := txn.Sequence(dbi1, 0)
		if err != nil {
			return err
		}
		if v != 3 {
			t.Errorf("unexpected value: %d (expected %d)", v, 3)
		}

		_, err = txn.Sequence(dbi1, 3) // error if > 0 in read tx
		if err == nil {
			t.Errorf("error expected")
		}

		return nil
	})
	if err != nil {
		t.Errorf("%s", err)
		return
	}
}

func TestListDBI(t *testing.T) {
	env, _ := setup(t)

	if err := env.View(func(txn *Txn) (err error) {
		v, err := txn.ListDBI()
		if err != nil {
			return err
		}
		if len(v) != 0 {
			t.Errorf("unexpected value: %d (expected %d)", len(v), 0)
		}

		return nil
	}); err != nil {
		t.Errorf("%s", err)
	}

	if err := env.Update(func(txn *Txn) (err error) {
		_, err = txn.OpenDBISimple("testdb", Create)
		if err != nil {
			return err
		}
		_, err = txn.OpenDBISimple("testdb2", Create)
		return err
	}); err != nil {
		t.Errorf("%s", err)
		return
	}

	if err := env.View(func(txn *Txn) (err error) {
		v, err := txn.ListDBI()
		if err != nil {
			return err
		}
		if v == nil {
			t.Errorf("unexpected nil")
		}
		if len(v) != 2 {
			t.Errorf("unexpected value: %d (expected %d)", len(v), 2)
		}

		return nil
	}); err != nil {
		t.Errorf("%s", err)
	}
}

func BenchmarkTxn_abort(b *testing.B) {
	env, _ := setup(b)

	var e = errors.New("abort")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = env.Update(func(txn *Txn) error { return e })
	}
}

func BenchmarkTxn_commit(b *testing.B) {
	env, _ := setup(b)

	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple("testdb", Create)
		if err != nil {
			return err
		}
		return err
	})
	if err != nil {
		b.Errorf("%s", err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var k [8]byte
		binary.BigEndian.PutUint64(k[:], uint64(i))
		err = env.Update(func(txn *Txn) error {
			err = txn.Put(db, k[:], k[:], 0)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			b.Error(err)
			return
		}
	}
}

func BenchmarkTxn_ro(b *testing.B) {
	env, _ := setup(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := env.View(func(txn *Txn) error { return nil })
		if err != nil {
			b.Error(err)
			return
		}
	}
}

func BenchmarkTxn_unmanaged_abort(b *testing.B) {
	env, _ := setup(b)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txn, err := env.BeginTxn(nil, 0)
		if err != nil {
			b.Error(err)
			return
		}
		txn.Abort()
	}
}

func BenchmarkTxn_unmanaged_commit(b *testing.B) {
	env, _ := setup(b)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txn, err := env.BeginTxn(nil, 0)
		if err != nil {
			b.Error(err)
			return
		}
		txn.Abort()
	}
}

func BenchmarkTxn_unmanaged_ro(b *testing.B) {
	env, _ := setup(b)

	// It is not necessary to call runtime.LockOSThread here because the txn is
	// Readonly

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txn, err := env.BeginTxn(nil, Readonly)
		if err != nil {
			b.Error(err)
			return
		}
		txn.Abort()
	}
}

func BenchmarkTxn_renew(b *testing.B) {
	env, _ := setup(b)

	// It is not necessary to call runtime.LockOSThread here because the txn is
	// Readonly

	txn, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		b.Error(err)
		return
	}
	defer txn.Abort()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = txn.Reset()
		err = txn.Renew()
		if err != nil {
			b.Error(err)
			return
		}
	}
}

func BenchmarkTxn_Put_append(b *testing.B) {
	env, _ := setup(b)
	err := env.SetGeometry(-1, -1, 256*1024*1024, -1, -1, 4096)
	if err != nil {
		b.Error(err)
		return
	}

	var db DBI

	err = env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		return err
	})
	if err != nil {
		b.Errorf("dbi: %v", err)
		return
	}

	b.ResetTimer()
	err = env.Update(func(txn *Txn) (err error) {
		for i := 0; i < b.N; i++ {
			var k [8]byte
			binary.BigEndian.PutUint64(k[:], uint64(i))
			err = txn.Put(db, k[:], k[:], Append)
			if err != nil {
				return err
			}
		}

		b.StopTimer()
		defer b.StartTimer()

		return txn.Drop(db, false)
	})
	if err != nil {
		b.Errorf("put: %v", err)
	}
}

func BenchmarkTxn_Put_append_noflag(b *testing.B) {
	env, _ := setup(b)
	err := env.SetGeometry(-1, -1, 256*1024*1024, -1, -1, 4096)
	if err != nil {
		b.Fatalf("Cannot set mapsize: %s", err)
	}

	var db DBI

	err = env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(0)
		return err
	})
	if err != nil {
		b.Errorf("dbi: %v", err)
		return
	}

	err = env.Update(func(txn *Txn) (err error) {
		var k [8]byte
		for i := 0; i < b.N; i++ {
			binary.BigEndian.PutUint64(k[:], uint64(i))
			err = txn.Put(db, k[:], k[:], 0)
			if err != nil {
				return err
			}
		}

		b.StopTimer()
		defer b.StartTimer()
		return txn.Drop(db, false)
	})
	if err != nil {
		b.Errorf("put: %v", err)
	}
}

func openRoot(env *Env, flags uint) (DBI, error) {
	var db DBI
	dotxn := env.View
	if flags != 0 {
		dotxn = env.Update
	}
	err := dotxn(func(txn *Txn) (err error) {
		db, err = txn.OpenRoot(flags)
		return err
	})
	if err != nil {
		return 0, err
	}
	return db, nil
}

func openDBI(env *Env, key string, flags uint) (DBI, error) {
	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple(key, flags)
		return err
	})
	if err != nil {
		return 0, err
	}
	return db, nil
}

func BenchmarkTxn_Get_OneKey(b *testing.B) {
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
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := txn.Get(db, k)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		b.Errorf("put: %v", err)
	}
}

func BenchmarkTxn_Get_Sequence(b *testing.B) {
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
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := txn.Get(db, keys[i])
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		b.Errorf("put: %v", err)
	}
}

func BenchmarkTxn_Get_Random(b *testing.B) {
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
		for i := 0; i < b.N; i++ {
			_, err := txn.Get(db, keys[i])
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		b.Errorf("put: %v", err)
	}
}

func TestTxnEnvWarmup(t *testing.T) {
	env, _ := setup(t)

	txn, err := env.BeginTxn(nil, EnvDefaults)
	if err != nil {
		t.Errorf("%s", err)
		return
	}
	err = txn.EnvWarmup(WarmupDefault, 2)
	if err != nil {
		t.Errorf("%s", err)
		return
	}
	defer txn.Abort()
}

// TestTxn_Refresh exercises mdbx_txn_refresh against a read-only transaction.
// After a write has occurred, refreshing the older reader must expose the
// new value without going through Reset/Renew.
func TestTxn_Refresh(t *testing.T) {
	env, _ := setup(t)

	var dbi DBI
	if err := env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenDBISimple("refresh", Create)
		return err
	}); err != nil {
		t.Fatalf("create dbi: %v", err)
	}

	roTxn, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatalf("begin ro: %v", err)
	}
	defer roTxn.Abort()

	// Already at tip — nothing committed since this snapshot started.
	atTip, err := roTxn.Refresh()
	if err != nil {
		t.Fatalf("refresh (tip): %v", err)
	}
	if !atTip {
		t.Errorf("expected atTip=true on freshly-started reader")
	}

	if _, err := roTxn.Get(dbi, []byte("k")); !IsNotFound(err) {
		t.Errorf("expected NotFound before write, got %v", err)
	}

	if err := env.Update(func(txn *Txn) error {
		return txn.Put(dbi, []byte("k"), []byte("v"), 0)
	}); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Reader still sees the old snapshot before refresh.
	if _, err := roTxn.Get(dbi, []byte("k")); !IsNotFound(err) {
		t.Errorf("expected NotFound on stale snapshot, got %v", err)
	}

	atTip, err = roTxn.Refresh()
	if err != nil {
		t.Fatalf("refresh (advance): %v", err)
	}
	if atTip {
		t.Errorf("expected atTip=false after a fresh commit landed")
	}

	v, err := roTxn.Get(dbi, []byte("k"))
	if err != nil {
		t.Fatalf("get after refresh: %v", err)
	}
	if string(v) != "v" {
		t.Errorf("unexpected value after refresh: %q", v)
	}
}

// TestTxn_Checkpoint commits a chunk of work in the middle of a longer write
// pipeline and verifies the data is visible to a new reader without
// releasing the write lock.
func TestTxn_Checkpoint(t *testing.T) {
	env, _ := setup(t)

	var dbi DBI
	if err := env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenDBISimple("ckpt", Create)
		return err
	}); err != nil {
		t.Fatalf("create dbi: %v", err)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	txn, err := env.BeginTxn(nil, 0)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() {
		if txn._txn != nil {
			txn.Abort()
		}
	}()

	if err := txn.Put(dbi, []byte("a"), []byte("1"), 0); err != nil {
		t.Fatalf("put a: %v", err)
	}

	lat, noChanges, err := txn.Checkpoint(0)
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	if noChanges {
		t.Errorf("expected noChanges=false after a put")
	}
	_ = lat // latency is informational

	// Pipeline continues in the same write txn.
	if err := txn.Put(dbi, []byte("b"), []byte("2"), 0); err != nil {
		t.Fatalf("put b: %v", err)
	}

	// First key must be visible to a fresh reader (mid-pipeline).
	if err := env.View(func(view *Txn) error {
		got, gerr := view.Get(dbi, []byte("a"))
		if gerr != nil {
			return fmt.Errorf("get a: %w", gerr)
		}
		if string(got) != "1" {
			return fmt.Errorf("a=%q, want %q", got, "1")
		}
		// b not yet committed
		if _, gerr := view.Get(dbi, []byte("b")); !IsNotFound(gerr) {
			return fmt.Errorf("b visible before second commit: %w", gerr)
		}
		return nil
	}); err != nil {
		t.Errorf("mid-pipeline view: %v", err)
	}

	if _, err := txn.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	if err := env.View(func(view *Txn) error {
		got, gerr := view.Get(dbi, []byte("b"))
		if gerr != nil {
			return fmt.Errorf("get b: %w", gerr)
		}
		if string(got) != "2" {
			return fmt.Errorf("b=%q, want %q", got, "2")
		}
		return nil
	}); err != nil {
		t.Errorf("final view: %v", err)
	}
}

// TestTxn_Checkpoint_NoChanges asserts the noChanges flag is returned when a
// write transaction is checkpointed without any modifications.
func TestTxn_Checkpoint_NoChanges(t *testing.T) {
	env, _ := setup(t)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	txn, err := env.BeginTxn(nil, 0)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() {
		if txn._txn != nil {
			txn.Abort()
		}
	}()

	_, noChanges, err := txn.Checkpoint(0)
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	if !noChanges {
		t.Errorf("expected noChanges=true for an empty write txn")
	}
}

// TestTxn_CommitEmbarkRead commits a write txn and immediately observes the
// resulting snapshot through the same handle (now in read-only mode).
func TestTxn_CommitEmbarkRead(t *testing.T) {
	env, _ := setup(t)

	var dbi DBI
	if err := env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenDBISimple("embark", Create)
		return err
	}); err != nil {
		t.Fatalf("create dbi: %v", err)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	txn, err := env.BeginTxn(nil, 0)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() {
		if txn._txn != nil {
			txn.Abort()
		}
	}()

	if err := txn.Put(dbi, []byte("k"), []byte("v"), 0); err != nil {
		t.Fatalf("put: %v", err)
	}

	_, noChanges, err := txn.CommitEmbarkRead()
	if err != nil {
		t.Fatalf("commit embark read: %v", err)
	}
	if noChanges {
		t.Errorf("expected noChanges=false after a put")
	}
	if !txn.readonly {
		t.Errorf("expected txn to be read-only after CommitEmbarkRead")
	}

	got, err := txn.Get(dbi, []byte("k"))
	if err != nil {
		t.Fatalf("get on embarked read: %v", err)
	}
	if string(got) != "v" {
		t.Errorf("unexpected value on embarked read: %q", got)
	}
}

// TestTxn_Rollback verifies that writes performed before Rollback are
// discarded but the same Txn handle can continue to be used.
func TestTxn_Rollback(t *testing.T) {
	env, _ := setup(t)

	var dbi DBI
	if err := env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenDBISimple("rollback", Create)
		return err
	}); err != nil {
		t.Fatalf("create dbi: %v", err)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	txn, err := env.BeginTxn(nil, 0)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() {
		if txn._txn != nil {
			txn.Abort()
		}
	}()

	if err := txn.Put(dbi, []byte("dropped"), []byte("x"), 0); err != nil {
		t.Fatalf("put: %v", err)
	}

	if err := txn.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if txn._txn == nil {
		t.Fatal("Rollback unexpectedly tore down the txn handle")
	}

	// The handle must still be usable for further writes.
	if _, err := txn.Get(dbi, []byte("dropped")); !IsNotFound(err) {
		t.Errorf("expected NotFound after Rollback, got %v", err)
	}
	if err := txn.Put(dbi, []byte("kept"), []byte("y"), 0); err != nil {
		t.Fatalf("put after rollback: %v", err)
	}
	if _, err := txn.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	if err := env.View(func(view *Txn) error {
		if _, gerr := view.Get(dbi, []byte("dropped")); !IsNotFound(gerr) {
			return fmt.Errorf("rolled-back key visible: %w", gerr)
		}
		got, gerr := view.Get(dbi, []byte("kept"))
		if gerr != nil {
			return fmt.Errorf("get kept: %w", gerr)
		}
		if string(got) != "y" {
			return fmt.Errorf("kept=%q, want %q", got, "y")
		}
		return nil
	}); err != nil {
		t.Errorf("post-rollback view: %v", err)
	}
}

// TestTxn_Amend converts a fresh read snapshot into a write transaction and
// commits a modification to that snapshot.
func TestTxn_Amend(t *testing.T) {
	env, _ := setup(t)

	var dbi DBI
	if err := env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenDBISimple("amend", Create)
		return err
	}); err != nil {
		t.Fatalf("create dbi: %v", err)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ro, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatalf("begin ro: %v", err)
	}
	defer func() {
		if ro._txn != nil {
			ro.Abort()
		}
	}()

	writeTxn, snapshotTooOld, err := ro.Amend(0)
	if err != nil {
		t.Fatalf("amend: %v", err)
	}
	if snapshotTooOld {
		t.Fatal("unexpected snapshotTooOld for an untouched snapshot")
	}
	if writeTxn == nil {
		t.Fatal("expected non-nil write txn")
	}
	if ro._txn != nil {
		t.Errorf("expected original read txn handle to be consumed without TxPrepareRO")
	}

	if err := writeTxn.Put(dbi, []byte("amended"), []byte("ok"), 0); err != nil {
		writeTxn.Abort()
		t.Fatalf("put: %v", err)
	}
	if _, err := writeTxn.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	if err := env.View(func(view *Txn) error {
		got, gerr := view.Get(dbi, []byte("amended"))
		if gerr != nil {
			return fmt.Errorf("get amended: %w", gerr)
		}
		if string(got) != "ok" {
			return fmt.Errorf("amended=%q, want %q", got, "ok")
		}
		return nil
	}); err != nil {
		t.Errorf("post-amend view: %v", err)
	}
}

// TestTxn_Amend_SnapshotTooOld asserts the snapshotTooOld signal is set when
// the read transaction's snapshot has been superseded by a committed write.
func TestTxn_Amend_SnapshotTooOld(t *testing.T) {
	env, _ := setup(t)

	var dbi DBI
	if err := env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenDBISimple("amend_stale", Create)
		return err
	}); err != nil {
		t.Fatalf("create dbi: %v", err)
	}

	ro, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatalf("begin ro: %v", err)
	}
	defer ro.Abort()

	// Race a write past the read snapshot.
	if err := env.Update(func(txn *Txn) error {
		return txn.Put(dbi, []byte("racer"), []byte("1"), 0)
	}); err != nil {
		t.Fatalf("race write: %v", err)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	writeTxn, snapshotTooOld, err := ro.Amend(0)
	if err != nil {
		t.Fatalf("amend: %v", err)
	}
	if !snapshotTooOld {
		if writeTxn != nil {
			writeTxn.Abort()
		}
		t.Fatalf("expected snapshotTooOld=true after concurrent commit")
	}
	if writeTxn != nil {
		t.Errorf("expected nil writeTxn on snapshotTooOld")
	}
}

// TestTxn_Clone creates a clone of a read transaction and verifies it sees
// the same MVCC snapshot independently of the origin.
func TestTxn_Clone(t *testing.T) {
	env, _ := setup(t)

	var dbi DBI
	if err := env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenDBISimple("clone", Create)
		if err != nil {
			return err
		}
		return txn.Put(dbi, []byte("k"), []byte("v0"), 0)
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	origin, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer origin.Abort()

	clone, err := origin.Clone()
	if err != nil {
		t.Fatalf("clone: %v", err)
	}
	defer clone.Abort()

	if clone.ID() != origin.ID() {
		t.Errorf("clone id %d != origin id %d", clone.ID(), origin.ID())
	}

	// A subsequent commit must not be visible to either the origin or the
	// clone — they both share the same pre-commit MVCC snapshot.
	if err := env.Update(func(txn *Txn) error {
		return txn.Put(dbi, []byte("k"), []byte("v1"), 0)
	}); err != nil {
		t.Fatalf("post-clone write: %v", err)
	}

	for name, view := range map[string]*Txn{"origin": origin, "clone": clone} {
		got, gerr := view.Get(dbi, []byte("k"))
		if gerr != nil {
			t.Errorf("%s get: %v", name, gerr)
			continue
		}
		if string(got) != "v0" {
			t.Errorf("%s saw %q, want %q", name, got, "v0")
		}
	}
}

// TestTxn_Checkpoint_OnReadOnly verifies the !readonly guard on Checkpoint:
// calling it on a read txn must return BadTxn without destroying the handle.
func TestTxn_Checkpoint_OnReadOnly(t *testing.T) {
	env, _ := setup(t)

	ro, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatalf("begin ro: %v", err)
	}
	defer ro.Abort()

	_, _, err = ro.Checkpoint(0)
	if !IsErrno(err, BadTxn) {
		t.Fatalf("expected BadTxn from Checkpoint on read txn, got %v", err)
	}
	if ro._txn == nil {
		t.Fatal("read txn handle was destroyed by failed Checkpoint guard")
	}
	// Reader must still be usable after the failed guard.
	if _, gerr := ro.OpenRoot(0); gerr != nil {
		t.Fatalf("read txn unusable after guard: %v", gerr)
	}
}

// TestTxn_Rollback_OnReadOnly verifies the !readonly guard on Rollback.
func TestTxn_Rollback_OnReadOnly(t *testing.T) {
	env, _ := setup(t)

	ro, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatalf("begin ro: %v", err)
	}
	defer ro.Abort()

	if err := ro.Rollback(); !IsErrno(err, BadTxn) {
		t.Fatalf("expected BadTxn from Rollback on read txn, got %v", err)
	}
	if ro._txn == nil {
		t.Fatal("read txn handle was destroyed by failed Rollback guard")
	}
	if _, gerr := ro.OpenRoot(0); gerr != nil {
		t.Fatalf("read txn unusable after guard: %v", gerr)
	}
}

// TestTxn_CommitEmbarkRead_OnReadOnly verifies the !readonly guard.
func TestTxn_CommitEmbarkRead_OnReadOnly(t *testing.T) {
	env, _ := setup(t)

	ro, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatalf("begin ro: %v", err)
	}
	defer ro.Abort()

	_, _, err = ro.CommitEmbarkRead()
	if !IsErrno(err, BadTxn) {
		t.Fatalf("expected BadTxn from CommitEmbarkRead on read txn, got %v", err)
	}
	if ro._txn == nil {
		t.Fatal("read txn handle was destroyed by failed CommitEmbarkRead guard")
	}
}

// TestTxn_CloneInto_RejectsWriteTarget verifies that CloneInto refuses a
// write-txn target before invoking libmdbx (which would otherwise reset the
// write handle via its read-only bailout, corrupting state).
func TestTxn_CloneInto_RejectsWriteTarget(t *testing.T) {
	env, _ := setup(t)

	// A live read source.
	src, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatalf("begin src: %v", err)
	}
	defer src.Abort()

	// A live write target — must be rejected at the wrapper boundary.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	writeTarget, err := env.BeginTxn(nil, 0)
	if err != nil {
		t.Fatalf("begin write target: %v", err)
	}
	defer func() {
		if writeTarget._txn != nil {
			writeTarget.Abort()
		}
	}()

	if err := src.CloneInto(writeTarget); !IsErrno(err, BadTxn) {
		t.Fatalf("expected BadTxn from CloneInto with write target, got %v", err)
	}
	if writeTarget._txn == nil {
		t.Fatal("write target handle was corrupted by CloneInto")
	}
	// Sanity: the write target is still usable as a write txn.
	root, gerr := writeTarget.OpenRoot(0)
	if gerr != nil {
		t.Fatalf("write target unusable after rejected CloneInto: %v", gerr)
	}
	if perr := writeTarget.Put(root, []byte("k"), []byte("v"), 0); perr != nil {
		t.Fatalf("put on write target failed: %v", perr)
	}
}

// TestTxn_CloneInto_Reuse exercises the happy path of reusing an existing
// (reset) read-only handle as the clone destination.
func TestTxn_CloneInto_Reuse(t *testing.T) {
	env, _ := setup(t)

	var dbi DBI
	if err := env.Update(func(txn *Txn) (err error) {
		dbi, err = txn.OpenDBISimple("clone_into", Create)
		if err != nil {
			return err
		}
		return txn.Put(dbi, []byte("k"), []byte("v"), 0)
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	src, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatalf("begin src: %v", err)
	}
	defer src.Abort()

	target, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatalf("begin target: %v", err)
	}
	defer target.Abort()

	if err := src.CloneInto(target); err != nil {
		t.Fatalf("clone into: %v", err)
	}
	if target.ID() != src.ID() {
		t.Errorf("target id %d != src id %d after CloneInto", target.ID(), src.ID())
	}
	got, gerr := target.Get(dbi, []byte("k"))
	if gerr != nil {
		t.Fatalf("get on cloned target: %v", gerr)
	}
	if string(got) != "v" {
		t.Errorf("target saw %q, want %q", got, "v")
	}
}

func TestTxn_Reset_ReturnsError(t *testing.T) {
	env, _ := setup(t)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// mdbx_txn_reset is only valid for read-only transactions.
	wtxn, err := env.BeginTxn(nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer wtxn.Abort()
	// The exact errno is platform-dependent (EINVAL on POSIX,
	// ERROR_INVALID_PARAMETER on Windows); pin only that Reset fails.
	if err := wtxn.Reset(); err == nil {
		t.Error("Reset on a write txn: expected error, got nil")
	}

	rtxn, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		t.Fatal(err)
	}
	defer rtxn.Abort()
	if err := rtxn.Reset(); err != nil {
		t.Errorf("Reset on a read txn: %v", err)
	}
	if err := rtxn.Renew(); err != nil {
		t.Errorf("Renew after Reset: %v", err)
	}
}

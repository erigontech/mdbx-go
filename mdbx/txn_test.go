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
	txnCached.Reset()
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
		txn.Reset()
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
		db, err := txn.OpenDBISimple("testdb", 0)
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
	txn.Reset()

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

	txn.Reset()
	txn.Reset()
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
	txn.Reset()
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
		dbi, err = txn.OpenDBISimple("testdb", 0)
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
		txn.Reset()
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

package mdbx

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestEnv_Path_notOpen(t *testing.T) {
	env, err := NewEnv(Default)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer env.Close()

	// before Open the Path method returns "" and a non-nil error.
	path, err := env.Path()
	if err == nil {
		t.Errorf("no error returned before Open")
	}
	if path != "" {
		t.Errorf("non-zero path returned before Open")
	}
}

func TestEnv_Path(t *testing.T) {
	env, err := NewEnv(Default)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// open an environment
	dir := t.TempDir()
	err = env.Open(dir, 0, 0644)
	defer env.Close()
	if err != nil {
		t.Errorf("open: %v", err)
	}
	path, err := env.Path()
	if err != nil {
		t.Errorf("path: %v", err)
	}
	if path != dir {
		t.Errorf("path: %q (!= %q)", path, dir)
	}
}

func TestEnv_Open_notExist(t *testing.T) {
	env, err := NewEnv(Default)
	if err != nil {
		t.Fatalf("create: %s", err)
	}
	defer env.Close()

	// ensure that opening a non-existent path fails.
	err = env.Open("/path/does/not/exist/aoeu", 0, 0664)
	if !IsNotExist(err) {
		t.Errorf("open: %v", err)
	}
}

func TestEnv_Open(t *testing.T) {
	env, err1 := NewEnv(Default)
	if err1 != nil {
		t.Error(err1)
		return
	}
	defer env.Close()

	// open an environment at a temporary path.
	path := t.TempDir()
	err := env.Open(path, 0, 0664)
	if err != nil {
		t.Errorf("open: %s", err)
	}
}

func TestEnv_PreOpen(t *testing.T) {
	env, err1 := NewEnv(Default)
	if err1 != nil {
		t.Error(err1)
		return
	}
	defer env.Close()

	// open an environment at a temporary path.
	path := t.TempDir()
	err := env.Open(path, 0, 0664)
	if err != nil {
		t.Errorf("open: %s", err)
	}
	env.Close()

	_info, err := PreOpenSnapInfo(path)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n", _info)
}

func TestEnv_FD(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FD funcs not supported on windows")
	}
	env, err1 := NewEnv(Default)
	if err1 != nil {
		t.Error(err1)
		return
	}
	defer env.Close()

	fd, err := env.FD()
	if err != nil && !strings.Contains(err.Error(), "operation not permitted") {
		t.Errorf("fd: %x (%v)", fd, err)
	}

	// open an environment at a temporary path.
	path := t.TempDir()
	err = env.Open(path, 0, 0664)
	if err != nil {
		t.Errorf("open: %s", err)
	}

	fd, err = env.FD()
	if err != nil {
		t.Errorf("fd error: %v", err)
	}
	if fd == 0 {
		t.Errorf("fd: %x", fd)
	}
}

func TestEnv_Flags(t *testing.T) {
	env, _ := setup(t)

	flags, err := env.Flags()
	if err != nil {
		t.Error(err)
		return
	}

	if flags&NoStickyThreads == 0 {
		t.Errorf("NoStickyThreads is not set")
	}

	err = env.SetFlags(SafeNoSync)
	if err != nil {
		t.Error(err)
	}

	flags, err = env.Flags()
	if err != nil {
		t.Error(err)
	}
	if flags&SafeNoSync == 0 {
		t.Error("UtterlyNoSync is not set")
	}

	err = env.UnsetFlags(SafeNoSync)
	if err != nil {
		t.Error(err)
	}

	flags, err = env.Flags()
	if err != nil {
		t.Error(err)
	}
	if flags&SafeNoSync != 0 {
		t.Error("UtterlyNoSync is set")
	}
}

func TestEnv_SetMaxReader(t *testing.T) {
	dir := t.TempDir()

	env, err := NewEnv(Default)
	if err != nil {
		t.Error(err)
	}

	maxreaders := uint64(246)
	err = env.SetOption(OptMaxReaders, maxreaders)
	if err != nil {
		t.Fatal(err)
	}
	_maxreaders, err := env.GetOption(OptMaxReaders)
	if err != nil {
		t.Fatal(err)
	}
	if _maxreaders < maxreaders {
		t.Errorf("unexpected MaxReaders: %v (< %v)", _maxreaders, maxreaders)
	}

	err = env.Open(dir, 0, 0644)
	defer env.Close()
	if err != nil {
		env.Close()
		t.Error(err)
	}
}

func TestEnv_PresyncThreshold(t *testing.T) {
	env, _ := setup(t)

	const threshold = uint64(8 * 1024 * 1024)
	if err := env.SetOption(OptPresyncThreshold, threshold); err != nil {
		t.Fatal(err)
	}
	got, err := env.GetOption(OptPresyncThreshold)
	if err != nil {
		t.Fatal(err)
	}
	if got != threshold {
		t.Errorf("presync threshold: %v (!= %v)", got, threshold)
	}
}

func TestEnv_SetDebug(t *testing.T) {
	env, err := NewEnv(Default)
	if err != nil {
		t.Error(err)
	}

	err = env.SetDebug(LogLvlDoNotChange, DbgLegacyTxOverlap, LoggerDoNotChange)
	if err != nil {
		t.Error(err)
	}
}

// func TestEnv_SetMapSize(t *testing.T) {
//	env := setup(t)
//
//
//	const minsize = 100 << 20 // 100MB
//	err := env.SetMapSize(minsize)
//	if err != nil {
//		t.Error(err)
//	}
//
//	err = env.Update(func(txn *Txn) (err error) {
//		return nil
//	})
//	if err != nil {
//		t.Error(err)
//	}
//
//	info, err := env.Info()
//	if err != nil {
//		t.Error(err)
//	} else if info.MapSize < minsize {
//		t.Errorf("unexpected mapsize: %v (< %v)", info.MapSize, minsize)
//	}
//}

// func TestEnv_ReaderList(t *testing.T) {
//	env := setup(t)
//
//
//	var numreaders = 2
//
//	var fin sync.WaitGroup
//	defer fin.Wait()
//	ready := make(chan struct{})
//	done := make(chan struct{})
//	defer close(done)
//
//	t.Logf("starting")
//
//	for i := 0; i < numreaders; i++ {
//		fin.Add(1)
//		go func(i int) {
//			defer fin.Done()
//			err := env.View(func(txn *Txn) (err error) {
//				t.Logf("reader %v: ready", i)
//				ready <- struct{}{}
//
//				<-done
//				t.Logf("reader %v: done", i)
//				return nil
//			})
//			if err != nil {
//				t.Errorf("reader %d: %q", i, err)
//			}
//		}(i)
//
//		// wait for each reader to become ready
//		<-ready
//	}
//
//	var readers []string
//	_ = env.ReaderList(func(msg string) error {
//		t.Logf("reader: %q", msg)
//		readers = append(readers, msg)
//		return nil
//	})
//	if len(readers) != numreaders+1 {
//		t.Errorf("unexpected reader list size: %d (!= %d)", len(readers), numreaders)
//	}
// }

// func TestEnv_ReaderList_error(t *testing.T) {
//	env := setup(t)
//
//
//	var numreaders = 2
//
//	var fin sync.WaitGroup
//	defer fin.Wait()
//	ready := make(chan struct{})
//	done := make(chan struct{})
//	defer close(done)
//
//	t.Logf("starting")
//
//	for i := 0; i < numreaders; i++ {
//		fin.Add(1)
//		go func(i int) {
//			defer fin.Done()
//			err := env.View(func(txn *Txn) (err error) {
//				t.Logf("reader %v: ready", i)
//				ready <- struct{}{}
//
//				<-done
//				t.Logf("reader %v: done", i)
//				return nil
//			})
//			if err != nil {
//				t.Errorf("reader %d: %q", i, err)
//			}
//		}(i)
//
//		// wait for each reader to become ready
//		<-ready
//	}
//
//	e := fmt.Errorf("testerror")
//	var readers []string
//	err := env.ReaderList(func(msg string) error {
//		readers = append(readers, msg)
//		return e
//	})
//	if err == nil {
//		t.Errorf("expected error")
//	}
//	if !errors.Is(err, e) {
//		t.Errorf("unexpected error: %q (!= %q)", err, e)
//	}
//	if len(readers) != 1 {
//		t.Errorf("unexpected reader list size: %d (!= %d)", len(readers), 1)
//	}
// }
//
// func TestEnv_ReaderList_envInvalid(t *testing.T) {
//	err := (&Env{}).ReaderList(func(msg string) error {
//		t.Logf("%s", msg)
//		return nil
//	})
//	if err == nil {
//		t.Errorf("expected error")
//	}
// }
//
// func TestEnv_ReaderList_nilFunc(t *testing.T) {
//	env, err := NewEnv()
//	if err != nil {
//		t.Fatal(err)
//	}
//	err = env.ReaderList(nil)
//	if err == nil {
//		t.Errorf("expected error")
//	}
// }

func TestEnv_ReaderCheck(t *testing.T) {
	env, _ := setup(t)

	numDead, err := env.ReaderCheck()
	if err != nil {
		t.Error(err)
	}
	if numDead != 0 {
		t.Errorf("unexpected dead readers: %v (!= %v)", numDead, 0)
	}
}

// Copy / CopyFD are as-is; the *Flag forms take the flags verbatim.
func TestEnv_Copy(t *testing.T)               { testEnvCopy(t, 0, false, false) }
func TestEnv_CopyFD(t *testing.T)             { testEnvCopy(t, 0, false, true) }
func TestEnv_CopyFlag_Compact(t *testing.T)   { testEnvCopy(t, CopyCompact, true, false) }
func TestEnv_CopyFDFlag_Compact(t *testing.T) { testEnvCopy(t, CopyCompact, true, true) }

var copyItem = struct{ k, v []byte }{[]byte("k0"), []byte("v0")}

func seedCopyItem(t *testing.T, env *Env) {
	t.Helper()
	if err := env.Update(func(txn *Txn) error {
		db, err := txn.OpenRoot(0)
		if err != nil {
			return err
		}
		return txn.Put(db, copyItem.k, copyItem.v, 0)
	}); err != nil {
		t.Fatal(err)
	}
}

// verifyCopyItem reopens the copy read-only and checks copyItem survived.
func verifyCopyItem(t *testing.T, path string) {
	t.Helper()
	envcp, err := NewEnv(Default)
	if err != nil {
		t.Fatal(err)
	}
	defer envcp.Close()
	if err := envcp.Open(path, Readonly, 0644); err != nil {
		t.Fatalf("open copy: %v", err)
	}
	if err := envcp.View(func(txn *Txn) error {
		db, err := txn.OpenRoot(0)
		if err != nil {
			return err
		}
		v, err := txn.Get(db, copyItem.k)
		if err != nil {
			return err
		}
		if !bytes.Equal(v, copyItem.v) {
			return fmt.Errorf("unexpected value: %q (!= %q)", v, copyItem.v)
		}
		return nil
	}); err != nil {
		t.Errorf("verify: %v", err)
	}
}

// seedDefrag leaves reclaimable pages for the defragmenter. The delete must
// be a second transaction; done in one, the pages never reach the GC and
// defrag finds nothing to do.
func seedDefrag(t *testing.T, env *Env, records, valueSize int) {
	t.Helper()
	value := bytes.Repeat([]byte("x"), valueSize)
	if err := env.Update(func(txn *Txn) error {
		db, err := txn.OpenRoot(0)
		if err != nil {
			return err
		}
		for i := range records {
			if err := txn.Put(db, fmt.Appendf(nil, "k%08d", i), value, 0); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := env.Update(func(txn *Txn) error {
		db, err := txn.OpenRoot(0)
		if err != nil {
			return err
		}
		for i := range records / 2 {
			if err := txn.Del(db, fmt.Appendf(nil, "k%08d", i*2), nil); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestEnv_CopyFlag_Overwrite(t *testing.T) {
	env, _ := setup(t)
	seedCopyItem(t, env)

	dst := filepath.Join(t.TempDir(), "copy.mdbx")

	if err := env.CopyFlag(dst, CopyCompact); err != nil {
		t.Fatalf("first copy: %v", err)
	}

	// O_EXCL without CopyOverwrite: assert the errno so an unrelated
	// breakage cannot pass as this behaviour.
	err := env.CopyFlag(dst, CopyCompact)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("copy onto an existing target: err = %v, want os.ErrExist", err)
	}

	if err := env.CopyFlag(dst, CopyCompact|CopyOverwrite); err != nil {
		t.Fatalf("copy with overwrite: %v", err)
	}
	verifyCopyItem(t, dst)
}

func testEnvCopy(t *testing.T, flags uint, useflags bool, usefd bool) {
	t.Helper()
	tmp := t.TempDir()

	var (
		fd  uintptr
		dst string
		f   *os.File
	)
	if usefd {
		dst = filepath.Join(tmp, "data.mdb")
		var err error
		f, err = os.Create(dst)
		if err != nil {
			t.Fatal(err)
		}
		fd = f.Fd()
		defer f.Close() // safety net for error paths; happy path closes early below
	} else {
		dst = filepath.Join(tmp, "dst")
	}

	env, _ := setup(t)
	seedCopyItem(t, env)

	var err error
	switch {
	case usefd && useflags:
		err = env.CopyFDFlag(fd, flags)
	case usefd:
		err = env.CopyFD(fd)
	case useflags:
		err = env.CopyFlag(dst, flags)
	default:
		err = env.Copy(dst)
	}
	if usefd {
		// Release our handle on the copy target before re-opening it as an
		// env below, so the two handles don't overlap.
		f.Close()
	}
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	verifyCopyItem(t, dst)
}

func TestEnv_Defrag(t *testing.T) {
	// Exclusive: the shrink needs the whole-file lock, as mdbx_defrag does.
	env, _ := setupFlags(t, Exclusive, Default)
	seedDefrag(t, env, 256, 1024)

	res, err := env.Defrag(DefragOptions{AcceptableBacklash: -1})
	// LaggardReader means stopped early, and needs no reader. See Env.Defrag.
	if err != nil && !IsErrno(err, LaggardReader) {
		t.Fatalf("defrag: %v", err)
	}
	// Zero here means the binding is not wired to any real work.
	if res.Cycles == 0 {
		t.Errorf("defrag: Cycles = 0, expected defrag to run at least one cycle")
	}
	if res.PagesMoved == 0 {
		t.Errorf("defrag: PagesMoved = 0, expected defrag to relocate pages")
	}
	t.Logf("defrag: cycles=%d shrunk=%d moved=%d whole=%d stopping_reasons=0x%x spent=%s",
		res.Cycles, res.PagesShrunk, res.PagesMoved, res.PagesWhole, res.StoppingReasons, res.SpentTime)
}

// TestEnv_Defrag_TimeLimit covers DefragOptions.TimeLimit, which reaches
// libmdbx as a count of 1/65536-second units.
func TestEnv_Defrag_TimeLimit(t *testing.T) {
	env, _ := setupFlags(t, Exclusive, Default)
	// Unbounded defrag of this measures ~60ms, well clear of the limit below.
	seedDefrag(t, env, 20000, 2048)

	// Rejected up front, which proves both durations survive the conversion.
	if _, err := env.Defrag(DefragOptions{TimeAtLeast: time.Second, TimeLimit: time.Millisecond}); err == nil {
		t.Error("TimeLimit < TimeAtLeast: expected an error, got nil")
	}

	res, err := env.Defrag(DefragOptions{TimeLimit: time.Millisecond, AcceptableBacklash: -1})
	if err != nil && !IsErrno(err, LaggardReader) {
		t.Fatalf("defrag: %v", err)
	}
	if res.StoppingReasons&DefragTimeLimit == 0 {
		t.Errorf("StoppingReasons = 0x%x, want DefragTimeLimit (0x%x) set; spent=%s cycles=%d moved=%d",
			res.StoppingReasons, DefragTimeLimit, res.SpentTime, res.Cycles, res.PagesMoved)
	}
	// SpentTime may overshoot: the in-flight batch still completes.
	t.Logf("defrag: cycles=%d moved=%d whole=%d reasons=0x%x spent=%s",
		res.Cycles, res.PagesMoved, res.PagesWhole, res.StoppingReasons, res.SpentTime)
}

func TestEnv_Sync(t *testing.T) {
	env, _ := setupFlags(t, SafeNoSync, "NoSync")

	item := struct{ k, v []byte }{[]byte("k0"), []byte("v0")}

	err := env.Update(func(txn *Txn) (err error) {
		db, err := txn.OpenRoot(0)
		if err != nil {
			return err
		}
		return txn.Put(db, item.k, item.v, 0)
	})
	if err != nil {
		t.Error(err)
	}

	err = env.Sync(true, false)
	if err != nil {
		t.Error(err)
	}
}

func setup(tb testing.TB) (*Env, string) {
	tb.Helper()
	return setupFlags(tb, 0, Default)
}

func setupWithLabel(tb testing.TB, label Label) (*Env, string) {
	tb.Helper()
	return setupFlags(tb, 0, label)
}

func setupFlags(tb testing.TB, flags uint, label Label) (env *Env, path string) {
	tb.Helper()
	env, err := NewEnv(label)
	if err != nil {
		tb.Fatalf("env: %s", err)
	}
	path = tb.TempDir()
	err = env.SetOption(OptMaxDB, 1024)
	if err != nil {
		tb.Fatalf("setmaxdbs: %v", err)
	}
	const pageSize = 4096
	err = env.SetGeometry(-1, -1, 256*64*1024*pageSize, -1, -1, pageSize)
	if err != nil {
		tb.Fatalf("setmaxdbs: %v", err)
	}
	err = env.Open(path, flags, 0664)
	if err != nil {
		tb.Fatalf("open: %s", err)
	}
	tb.Cleanup(func() {
		// MDBX_BUSY here means a leaked write txn, which otherwise shows up
		// only as a Windows TempDir cleanup failure.
		if err := env.Close(); err != nil {
			tb.Errorf("env close: %v", err)
		}
	})
	return env, path
}
func TestEnv_MaxKeySize(t *testing.T) {
	env, _ := setup(t)

	n := env.MaxKeySize()
	if n <= 0 {
		t.Errorf("invaild maxkeysize: %d", n)
	}
}

func TestEnv_MaxKeySize_nil(t *testing.T) {
	var env *Env
	n := env.MaxKeySize()
	if n < -1 {
		t.Errorf("invaild maxkeysize: %d", n)
	}
	t.Logf("mdb_env_get_maxkeysize: %d", n)
}

func TestEnv_CloseDBI(t *testing.T) {
	env, _ := setup(t)

	const numdb = 1000
	for i := range numdb {
		dbname := fmt.Sprintf("db%d", i)

		var dbi DBI
		err := env.Update(func(txn *Txn) (err error) {
			dbi, err = txn.CreateDBI(dbname)
			return err
		})
		if err != nil {
			t.Errorf("%s", err)
		}

		env.CloseDBI(dbi)
	}

	stat, err := env.Stat()
	if err != nil {
		t.Errorf("%s", err)
		return
	}

	//nolint:err113
	if stat.Entries != numdb {
		t.Errorf("unexpected entries: %d (not %d)", stat.Entries, numdb)
	}
}

func TestEnv_Info_NilTxn(t *testing.T) {
	env, _ := setup(t)

	info, err := env.Info(nil)
	if err != nil {
		t.Fatal(err)
	}
	if info.PageSize == 0 {
		t.Error("PageSize = 0, want nonzero")
	}
	if info.MapSize == 0 {
		t.Error("MapSize = 0, want nonzero")
	}
	if info.MaxReaders == 0 {
		t.Error("MaxReaders = 0, want nonzero")
	}
}

func TestEnv_Info_NilTxn_Unopened(t *testing.T) {
	env, err := NewEnv(Default)
	if err != nil {
		t.Fatal(err)
	}
	defer env.Close()

	info, err := env.Info(nil)
	if err != nil {
		t.Fatal(err)
	}
	if info == nil {
		t.Fatal("Info(nil) on unopened env returned nil info")
	}
}

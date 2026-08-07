package mdbx

import (
	"bytes"
	"fmt"
	"testing"
)

func fillBatchDB(tb testing.TB, env *Env, name string, numItems int) DBI {
	tb.Helper()
	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple(name, Create)
		if err != nil {
			return err
		}
		for i := range numItems {
			k := fmt.Appendf(nil, "key-%08d", i)
			v := fmt.Appendf(nil, "val-%08d", i)
			if err := txn.Put(db, k, v, Append); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		tb.Fatal(err)
	}
	return db
}

func TestCursor_GetBatch(t *testing.T) {
	env, _ := setup(t)
	const numItems = 1000
	db := fillBatchDB(t, env, "testbatch", numItems)

	buf := NewGetBatchBuffer(64)
	defer buf.Close()

	err := env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		seen := 0
		for opFirst := uint(First); ; opFirst = Next {
			n, eof, err := cur.GetBatch(buf, opFirst, Next)
			if err != nil {
				return err
			}
			for i := range n {
				wantK := fmt.Sprintf("key-%08d", seen)
				wantV := fmt.Sprintf("val-%08d", seen)
				if !bytes.Equal(buf.Key(i), []byte(wantK)) {
					t.Fatalf("pair %d: key = %q, want %q", seen, buf.Key(i), wantK)
				}
				if !bytes.Equal(buf.Val(i), []byte(wantV)) {
					t.Fatalf("pair %d: val = %q, want %q", seen, buf.Val(i), wantV)
				}
				seen++
			}
			if eof {
				break
			}
		}
		if seen != numItems {
			t.Errorf("scanned %d items, want %d", seen, numItems)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCursor_GetBatch_EmptyDB(t *testing.T) {
	env, _ := setup(t)
	db := fillBatchDB(t, env, "testbatchempty", 0)

	buf := NewGetBatchBuffer(8)
	defer buf.Close()

	err := env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		n, eof, err := cur.GetBatch(buf, First, Next)
		if err != nil {
			return err
		}
		if n != 0 || !eof {
			t.Errorf("GetBatch on empty table: n=%d eof=%v, want 0/true", n, eof)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCursor_GetBatch_NoAllocs(t *testing.T) {
	env, _ := setup(t)
	db := fillBatchDB(t, env, "testbatchnoalloc", 100)

	buf := NewGetBatchBuffer(32)
	defer buf.Close()

	err := env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		assertNoAllocs(t, "Cursor.GetBatch", func() { _, _, _ = cur.GetBatch(buf, First, Next) })
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

// BenchmarkCursorScan compares a full forward scan performed with one cgo
// call per record (Get/Next) against batched retrieval (GetBatch).
func BenchmarkCursorScan(b *testing.B) {
	env, _ := setup(b)
	const numItems = 100_000
	db := fillBatchDB(b, env, "benchscan", numItems)

	txn, err := env.BeginTxn(nil, Readonly)
	if err != nil {
		b.Fatal(err)
	}
	defer txn.Abort()
	cur, err := txn.OpenCursor(db)
	if err != nil {
		b.Fatal(err)
	}
	defer cur.Close()

	b.Run("Get_Next", func(b *testing.B) {
		b.ResetTimer()
		var total int
		for range b.N {
			count := 0
			for _, _, err := cur.Get(nil, nil, First); err == nil; _, _, err = cur.Get(nil, nil, Next) {
				count++
			}
			total = count
		}
		if total != numItems {
			b.Fatalf("scanned %d, want %d", total, numItems)
		}
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/numItems, "ns/record")
	})

	for _, batch := range []int{64, 256, 1024} {
		b.Run(fmt.Sprintf("GetBatch_%d", batch), func(b *testing.B) {
			buf := NewGetBatchBuffer(batch)
			defer buf.Close()
			b.ResetTimer()
			var total int
			for range b.N {
				count := 0
				for opFirst := uint(First); ; opFirst = Next {
					n, eof, err := cur.GetBatch(buf, opFirst, Next)
					if err != nil {
						b.Fatal(err)
					}
					count += n
					if eof {
						break
					}
				}
				total = count
			}
			if total != numItems {
				b.Fatalf("scanned %d, want %d", total, numItems)
			}
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/numItems, "ns/record")
		})
	}
}

// Exact-fill boundary: the batch that fills completely on the table's last
// record reports eof=false; the follow-up call must return (0, true, nil).
func TestCursor_GetBatch_ExactFill(t *testing.T) {
	env, _ := setup(t)
	db := fillBatchDB(t, env, "testbatchexact", 128)

	buf := NewGetBatchBuffer(64)
	defer buf.Close()

	err := env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		for i, want := range []struct {
			n   int
			eof bool
		}{{64, false}, {64, false}, {0, true}} {
			op := uint(Next)
			if i == 0 {
				op = First
			}
			n, eof, err := cur.GetBatch(buf, op, Next)
			if err != nil {
				return err
			}
			if n != want.n || eof != want.eof {
				t.Errorf("batch %d: n=%d eof=%v, want n=%d eof=%v", i, n, eof, want.n, want.eof)
			}
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

// A failing step mid-batch must leave the pairs fetched before it valid.
func TestCursor_GetBatch_PartialOnError(t *testing.T) {
	env, _ := setup(t)
	db := fillBatchDB(t, env, "testbatchpartial", 10)

	buf := NewGetBatchBuffer(8)
	defer buf.Close()

	err := env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		// First succeeds; NextMultiple on a non-DupFixed table fails with
		// MDBX_INCOMPATIBLE on the second step.
		n, eof, err := cur.GetBatch(buf, First, NextMultiple)
		if err == nil {
			t.Fatal("GetBatch(First, NextMultiple): expected error, got nil")
		}
		if eof {
			t.Error("eof must be false on error")
		}
		if n != 1 {
			t.Fatalf("n = %d, want 1 pair fetched before the failing step", n)
		}
		if got := string(buf.Key(0)); got != "key-00000000" {
			t.Errorf("Key(0) = %q, want key-00000000", got)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

// Ops that need an input key/value must be rejected instead of run against an
// empty one: Set would match nothing and report a clean EOF, SetRange would
// silently rewind the scan to the first key.
func TestCursor_GetBatch_RejectsInputOps(t *testing.T) {
	env, _ := setup(t)
	db := fillBatchDB(t, env, "testbatchops", 10)

	buf := NewGetBatchBuffer(8)
	defer buf.Close()

	err := env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		for _, tc := range []struct {
			name        string
			first, next uint
		}{
			{"Set as opFirst", Set, Next},
			{"SetKey as opFirst", SetKey, Next},
			{"SetRange as opFirst", SetRange, Next},
			{"GetBoth as opFirst", GetBoth, Next},
			{"GetBothRange as opFirst", GetBothRange, Next},
			{"SetLowerBound as opFirst", SetLowerBound, Next},
			{"SetUpperBound as opFirst", SetUpperBound, Next},
			{"KeyGreaterThan as opFirst", KeyGreaterThan, Next},
			{"PairLesserThan as opFirst", PairLesserThan, Next},
			{"Set as opNext", First, Set},
			{"SetRange as opNext", First, SetRange},
			{"GetCurrent as opNext", First, GetCurrent},
			{"GetMultiple as opNext", First, GetMultiple},
			{"out-of-range op", ^uint(0), Next},
		} {
			n, eof, err := cur.GetBatch(buf, tc.first, tc.next)
			assertRejected(t, tc.name, n, eof, err)
		}

		// A rejected call must not leave the previous fill readable.
		if _, _, err := cur.GetBatch(buf, First, Next); err != nil {
			return err
		}
		if _, _, err := cur.GetBatch(buf, Set, Next); err == nil {
			t.Fatal("GetBatch(Set, Next): expected EINVAL, got nil")
		}
		assertPanics(t, "Key(0) after a rejected GetBatch", func() { _ = buf.Key(0) })
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

// Reverse scans use the same machinery via (Last, Prev).
func TestCursor_GetBatch_Reverse(t *testing.T) {
	env, _ := setup(t)
	const numItems = 100
	db := fillBatchDB(t, env, "testbatchreverse", numItems)

	buf := NewGetBatchBuffer(16)
	defer buf.Close()

	err := env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		seen := 0
		for opFirst := uint(Last); ; opFirst = Prev {
			n, eof, err := cur.GetBatch(buf, opFirst, Prev)
			if err != nil {
				return err
			}
			for i := range n {
				want := fmt.Sprintf("key-%08d", numItems-1-seen)
				if got := string(buf.Key(i)); got != want {
					t.Fatalf("pair %d: key = %q, want %q", seen, got, want)
				}
				seen++
			}
			if eof {
				break
			}
		}
		if seen != numItems {
			t.Errorf("scanned %d items, want %d", seen, numItems)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

// DupSort: (FirstDup, NextDup) batches the values of the positioned key.
func TestCursor_GetBatch_DupSort(t *testing.T) {
	env, _ := setup(t)
	const numDups = 20
	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple("testbatchdup", Create|DupSort)
		if err != nil {
			return err
		}
		for i := range 3 {
			for j := range numDups {
				k := fmt.Sprintf("key-%d", i)
				v := fmt.Sprintf("val-%d-%02d", i, j)
				if err := txn.Put(db, []byte(k), []byte(v), 0); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	buf := NewGetBatchBuffer(8)
	defer buf.Close()

	err = env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		// Position on the second key, then batch only its values. Key(i) is
		// unspecified for dup-only ops, so only the values are checked.
		if _, _, err := cur.Get([]byte("key-1"), nil, Set); err != nil {
			return err
		}
		seen := 0
		for opFirst := uint(FirstDup); ; opFirst = NextDup {
			n, eof, err := cur.GetBatch(buf, opFirst, NextDup)
			if err != nil {
				return err
			}
			for i := range n {
				want := fmt.Sprintf("val-1-%02d", seen)
				if got := string(buf.Val(i)); got != want {
					t.Fatalf("dup %d: val = %q, want %q", seen, got, want)
				}
				seen++
			}
			if eof {
				break
			}
		}
		if seen != numDups {
			t.Errorf("scanned %d dups, want %d", seen, numDups)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetBatchBuffer_Closed(t *testing.T) {
	env, _ := setup(t)
	db := fillBatchDB(t, env, "testbatchclosed", 10)

	buf := NewGetBatchBuffer(8)

	err := env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		n, _, err := cur.GetBatch(buf, First, Next)
		if err != nil {
			return err
		}
		if n == 0 {
			t.Fatal("GetBatch filled nothing")
		}

		buf.Close()
		buf.Close() // idempotent

		if got := buf.Cap(); got != 0 {
			t.Errorf("Cap() after Close = %d, want 0", got)
		}
		// Close must drop the last-fill count as well: Key/Val would otherwise
		// hand out views of freed memory.
		assertPanics(t, "Key(0) after Close", func() { _ = buf.Key(0) })
		assertPanics(t, "Val(0) after Close", func() { _ = buf.Val(0) })

		n, eof, err := cur.GetBatch(buf, First, Next)
		assertRejected(t, "GetBatch on a closed buffer", n, eof, err)

		n, eof, err = cur.GetBatch(nil, First, Next)
		assertRejected(t, "GetBatch(nil)", n, eof, err)
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

// Reading past the last fill must panic rather than expose stale entries.
func TestGetBatchBuffer_IndexOutOfFill(t *testing.T) {
	env, _ := setup(t)
	db := fillBatchDB(t, env, "testbatchindex", 3)

	buf := NewGetBatchBuffer(8)
	defer buf.Close()

	err := env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		n, _, err := cur.GetBatch(buf, First, Next)
		if err != nil {
			return err
		}
		if n != 3 {
			t.Fatalf("n = %d, want 3", n)
		}
		assertPanics(t, "Key(n)", func() { _ = buf.Key(n) })
		assertPanics(t, "Val(n)", func() { _ = buf.Val(n) })
		assertPanics(t, "Key(-1)", func() { _ = buf.Key(-1) })
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

// assertRejected pins that GetBatch refused the call without moving the
// cursor. The exact errno is platform-dependent (EINVAL on POSIX,
// ERROR_INVALID_PARAMETER on Windows), so the code itself is not asserted.
func assertRejected(t *testing.T, name string, n int, eof bool, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected an error, got nil", name)
	}
	if n != 0 || eof {
		t.Errorf("%s: n=%d eof=%v, want 0/false", name, n, eof)
	}
}

func assertPanics(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if e := recover(); e == nil {
			t.Errorf("%s: expected panic, got none", name)
		}
	}()
	fn()
}

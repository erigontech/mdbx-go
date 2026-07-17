package mdbx

import (
	"bytes"
	"fmt"
	"testing"
)

func fillBatchDB(t testing.TB, env *Env, name string, numItems int) DBI {
	t.Helper()
	var db DBI
	err := env.Update(func(txn *Txn) (err error) {
		db, err = txn.OpenDBISimple(name, Create)
		if err != nil {
			return err
		}
		for i := 0; i < numItems; i++ {
			k := []byte(fmt.Sprintf("key-%08d", i))
			v := []byte(fmt.Sprintf("val-%08d", i))
			if err := txn.Put(db, k, v, Append); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
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
			for i := 0; i < n; i++ {
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
		for i := 0; i < b.N; i++ {
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
			for i := 0; i < b.N; i++ {
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

		// First succeeds; Set without a search key fails on the second step.
		n, eof, err := cur.GetBatch(buf, First, Set)
		if err == nil {
			t.Fatal("GetBatch(First, Set): expected error, got nil")
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

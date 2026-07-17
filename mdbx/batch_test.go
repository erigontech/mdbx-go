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

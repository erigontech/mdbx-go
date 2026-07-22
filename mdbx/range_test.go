package mdbx

import (
	"encoding/binary"
	"slices"
	"testing"
)

// mustPutSeqBE inserts n keys 0..n-1 as 4-byte big-endian (so they sort
// numerically) with an identical value, into a freshly created unique DB.
func mustPutSeqBE(t *testing.T, env *Env, name string, n int) DBI {
	t.Helper()
	db := mustOpenUniqueDB(t, env, name)
	if err := env.Update(func(txn *Txn) error {
		var kb [4]byte
		for i := range n {
			binary.BigEndian.PutUint32(kb[:], uint32(i))
			if err := txn.Put(db, kb[:], kb[:], 0); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("put seq be: %v", err)
	}
	return db
}

func beKey(i uint32) []byte {
	var kb [4]byte
	binary.BigEndian.PutUint32(kb[:], i)
	return kb[:]
}

func TestTxn_EstimateRange(t *testing.T) {
	env, _ := setup(t)
	const n = 5000
	db := mustPutSeqBE(t, env, "estimate_range", n)

	if err := env.View(func(txn *Txn) error {
		// Whole table: nil begin + nil end.
		total, err := txn.EstimateRange(db, nil, nil, nil, nil)
		if err != nil {
			return err
		}
		// Rough estimate: must be in the right ballpark.
		if total < n/2 || total > n*2 {
			t.Fatalf("full-range estimate=%d, want ~%d", total, n)
		}

		// A half-open sub-range must be positive and not exceed the total.
		half, err := txn.EstimateRange(db, beKey(1000), nil, beKey(3000), nil)
		if err != nil {
			return err
		}
		if half <= 0 || half > total {
			t.Fatalf("sub-range estimate=%d, want in (0,%d]", half, total)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestCursor_EstimateDistanceAndDistance(t *testing.T) {
	env, _ := setup(t)
	const n = 1000
	db := mustPutSeqBE(t, env, "estimate_distance", n)

	if err := env.View(func(txn *Txn) error {
		first, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer first.Close()
		last, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer last.Close()

		if _, _, err := first.Get(beKey(100), nil, SetKey); err != nil {
			return err
		}
		if _, _, err := last.Get(beKey(900), nil, SetKey); err != nil {
			return err
		}

		// Exact distance with a large deepness. first (key 100) precedes last
		// (key 900), so the forward distance is exactly +800; the sign is
		// deterministic and a sign-inversion regression must be caught.
		d, err := first.Distance(last, 42)
		if err != nil {
			return err
		}
		if d != 800 {
			t.Fatalf("Distance(100->900)=%d, want 800", d)
		}
		// Reversed cursors yield the negated distance.
		dRev, err := last.Distance(first, 42)
		if err != nil {
			return err
		}
		if dRev != -800 {
			t.Fatalf("Distance(900->100)=%d, want -800", dRev)
		}

		// Rough estimate should be in the ballpark.
		ed, err := first.EstimateDistance(last)
		if err != nil {
			return err
		}
		if ed < 400 || ed > 1200 {
			t.Fatalf("EstimateDistance=%d, want ~800", ed)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestCursor_EstimateMove(t *testing.T) {
	env, _ := setup(t)
	const n = 1000
	db := mustPutSeqBE(t, env, "estimate_move", n)

	if err := env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		if _, _, err := cur.Get(nil, nil, First); err != nil {
			return err
		}
		// Estimate the move from First to Last; should be close to n.
		d, err := cur.EstimateMove(nil, nil, Last)
		if err != nil {
			return err
		}
		if d < n/2 || d > n*2 {
			t.Fatalf("EstimateMove(Last)=%d, want ~%d", d, n)
		}
		// The cursor must not have moved.
		pos, ok := curKeyInt(cur)
		if !ok {
			t.Fatalf("cursor not positioned after EstimateMove")
		}
		if pos != 0 {
			t.Fatalf("cursor moved after EstimateMove: key=%d", pos)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestCursor_Scroll(t *testing.T) {
	env, _ := setup(t)
	const n = 100
	db := mustPutSeqBE(t, env, "scroll", n)

	if err := env.View(func(txn *Txn) error {
		cur, err := txn.OpenCursor(db)
		if err != nil {
			return err
		}
		defer cur.Close()

		if _, _, err := cur.Get(nil, nil, First); err != nil {
			return err
		}
		if err := cur.Scroll(5, 42); err != nil {
			return err
		}
		if got, ok := curKeyInt(cur); !ok || got != 5 {
			t.Fatalf("after Scroll(+5) key=%d ok=%v, want 5", got, ok)
		}

		if err := cur.Scroll(-2, 42); err != nil {
			return err
		}
		if got, ok := curKeyInt(cur); !ok || got != 3 {
			t.Fatalf("after Scroll(-2) key=%d ok=%v, want 3", got, ok)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// curKeyInt returns the current cursor key decoded as a uint32, and ok=false if
// the cursor is not positioned on any data (e.g. an unset cursor after a partial
// DistributeCursors).
func curKeyInt(c *Cursor) (int, bool) {
	k, _, err := c.Get(nil, nil, GetCurrent)
	if err != nil || len(k) != 4 {
		return 0, false
	}
	return int(binary.BigEndian.Uint32(k)), true
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// openCursors opens count cursors on db within txn (each closed via t.Cleanup).
func openCursors(t *testing.T, txn *Txn, db DBI, count int) []*Cursor {
	t.Helper()
	cursors := make([]*Cursor, count)
	for i := range cursors {
		c, err := txn.OpenCursor(db)
		if err != nil {
			t.Fatalf("OpenCursor: %v", err)
		}
		cursors[i] = c
	}
	return cursors
}

func TestDistributeCursors(t *testing.T) {
	// Even distribution over the whole table for several worker counts. Keys are
	// dense (0..n-1), so a key's integer value equals its rank, which lets us
	// assert that consecutive cursors delimit approximately equally-sized chunks.
	t.Run("even", func(t *testing.T) {
		env, _ := setup(t)
		const n = 1000
		db := mustPutSeqBE(t, env, "distribute_even", n)

		for _, workers := range []int{1, 2, 3, 4, 7, 8} {
			if err := env.View(func(txn *Txn) error {
				first := openCursors(t, txn, db, 1)[0]
				defer first.Close()
				if _, _, err := first.Get(nil, nil, First); err != nil {
					return err
				}
				cursors := openCursors(t, txn, db, workers)
				defer func() {
					for _, c := range cursors {
						c.Close()
					}
				}()

				allSet, err := DistributeCursors(first, nil, cursors, 42)
				if err != nil {
					return err
				}
				if !allSet {
					t.Fatalf("workers=%d: allSet=false over %d keys", workers, n)
				}

				chunk := n / workers
				tol := chunk/2 + 2 // generous: page boundaries may shift positions
				prev := -1
				prevPos := 0 // first cursor's range starts at rank 0
				for i, c := range cursors {
					pos, ok := curKeyInt(c)
					if !ok {
						t.Fatalf("workers=%d: cursor %d not positioned", workers, i)
					}
					if pos <= prev {
						t.Fatalf("workers=%d: cursor %d pos=%d not > prev=%d", workers, i, pos, prev)
					}
					if pos < 0 || pos >= n {
						t.Fatalf("workers=%d: cursor %d pos=%d out of [0,%d)", workers, i, pos, n)
					}
					// Cursor i marks the upper boundary of chunk i, expected near
					// (i+1)*n/workers (the final cursor lands on the last key).
					want := (i + 1) * n / workers
					if i == workers-1 {
						want = n - 1
					}
					if abs(pos-want) > tol {
						t.Fatalf("workers=%d: cursor %d pos=%d, want ~%d (tol %d)", workers, i, pos, want, tol)
					}
					// Chunk size between boundaries must be balanced.
					size := pos - prevPos
					if abs(size-chunk) > tol {
						t.Fatalf("workers=%d: chunk %d size=%d, want ~%d (tol %d)", workers, i, size, chunk, tol)
					}
					prev = pos
					prevPos = pos
				}
				// Last cursor must reach the final key so the whole range is covered.
				if prev != n-1 {
					t.Fatalf("workers=%d: last cursor at %d, want %d", workers, prev, n-1)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
	})

	// Distribution constrained to an explicit [first, last] sub-range.
	t.Run("explicit-bounds", func(t *testing.T) {
		env, _ := setup(t)
		const n = 1000
		const lo, hi = 200, 800
		db := mustPutSeqBE(t, env, "distribute_bounds", n)

		if err := env.View(func(txn *Txn) error {
			bounds := openCursors(t, txn, db, 2)
			first, last := bounds[0], bounds[1]
			defer first.Close()
			defer last.Close()
			if _, _, err := first.Get(beKey(lo), nil, SetKey); err != nil {
				return err
			}
			if _, _, err := last.Get(beKey(hi), nil, SetKey); err != nil {
				return err
			}

			cursors := openCursors(t, txn, db, 3)
			defer func() {
				for _, c := range cursors {
					c.Close()
				}
			}()
			allSet, err := DistributeCursors(first, last, cursors, 42)
			if err != nil {
				return err
			}
			if !allSet {
				t.Fatalf("allSet=false for sub-range [%d,%d]", lo, hi)
			}
			chunk := (hi - lo) / len(cursors)
			tol := chunk/2 + 2
			prev := lo
			for i, c := range cursors {
				pos, ok := curKeyInt(c)
				if !ok {
					t.Fatalf("cursor %d not positioned", i)
				}
				if pos <= prev || pos > hi {
					t.Fatalf("cursor %d pos=%d not in (%d,%d]", i, pos, prev, hi)
				}
				// Chunks within the explicit bounds must be balanced too, not just
				// monotonic (guards against cursors bunching at one end).
				if abs(pos-prev-chunk) > tol {
					t.Fatalf("cursor %d chunk size=%d, want ~%d (tol %d)", i, pos-prev, chunk, tol)
				}
				prev = pos
			}
			if prev != hi {
				t.Fatalf("last cursor at %d, want %d", prev, hi)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	})

	// More cursors than available positions: allSet must be false (with nil
	// error), only the first few cursors get positioned, and they stay valid.
	t.Run("not-enough-positions", func(t *testing.T) {
		env, _ := setup(t)
		const n = 3
		db := mustPutSeqBE(t, env, "distribute_small", n)

		if err := env.View(func(txn *Txn) error {
			first := openCursors(t, txn, db, 1)[0]
			defer first.Close()
			if _, _, err := first.Get(nil, nil, First); err != nil {
				return err
			}
			cursors := openCursors(t, txn, db, 8)
			defer func() {
				for _, c := range cursors {
					c.Close()
				}
			}()

			allSet, err := DistributeCursors(first, nil, cursors, 42)
			if err != nil {
				return err
			}
			if allSet {
				t.Fatalf("allSet=true, want false (only %d keys, 8 cursors)", n)
			}
			set, prev := 0, -1
			for _, c := range cursors {
				pos, ok := curKeyInt(c)
				if !ok {
					continue
				}
				set++
				if pos <= prev {
					t.Fatalf("positioned cursor pos=%d not > prev=%d", pos, prev)
				}
				prev = pos
			}
			// first occupies key 0, leaving keys 1..n-1 as the only boundary
			// positions, so exactly n-1 cursors can be set; the rest stay unset.
			if set != n-1 {
				t.Fatalf("positioned %d cursors, want %d", set, n-1)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	})

	// Using distributed boundaries to delete the whole table in balanced chunks,
	// the way parallel workers would. Validates the end-to-end workflow:
	// distribute -> per-chunk DeleteRange -> full coverage, no gaps/overlaps.
	t.Run("delete-in-chunks", func(t *testing.T) {
		env, _ := setup(t)
		const n = 1000
		const workers = 4
		db := mustPutSeqBE(t, env, "distribute_delete", n)

		// 1) Read-only: compute the worker boundary keys.
		var bounds [][]byte
		if err := env.View(func(txn *Txn) error {
			first := openCursors(t, txn, db, 1)[0]
			defer first.Close()
			if _, _, err := first.Get(nil, nil, First); err != nil {
				return err
			}
			cursors := openCursors(t, txn, db, workers)
			defer func() {
				for _, c := range cursors {
					c.Close()
				}
			}()
			allSet, err := DistributeCursors(first, nil, cursors, 42)
			if err != nil {
				return err
			}
			if !allSet {
				t.Fatalf("allSet=false")
			}
			for _, c := range cursors {
				pos, ok := curKeyInt(c)
				if !ok {
					t.Fatalf("cursor not positioned")
				}
				bounds = append(bounds, beKey(uint32(pos)))
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}

		// 2) Write: each worker deletes its chunk. Chunk 0 covers [start,bounds[0]]
		// and chunk i covers (bounds[i-1], bounds[i]] (boundary inclusive). Chunks
		// are processed back-to-front so that a boundary key is never looked up
		// after an earlier chunk already deleted it.
		deleteChunk := func(txn *Txn, i int) (uint64, error) {
			begin, err := txn.OpenCursor(db)
			if err != nil {
				return 0, err
			}
			defer begin.Close()
			end, err := txn.OpenCursor(db)
			if err != nil {
				return 0, err
			}
			defer end.Close()

			if i == 0 {
				if _, _, err := begin.Get(nil, nil, First); err != nil {
					return 0, err
				}
			} else {
				// Position at the previous boundary, then step to the first key
				// strictly after it (the start of this chunk).
				if _, _, err := begin.Get(bounds[i-1], nil, SetKey); err != nil {
					return 0, err
				}
				if _, _, err := begin.Get(nil, nil, Next); err != nil {
					return 0, err
				}
			}
			if _, _, err := end.Get(bounds[i], nil, SetKey); err != nil {
				return 0, err
			}
			return begin.DeleteRange(end, true) // inclusive of the boundary key
		}

		var total uint64
		if err := env.Update(func(txn *Txn) error {
			for i := range slices.Backward(bounds) {
				affected, err := deleteChunk(txn, i)
				if err != nil {
					return err
				}
				total += affected
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}

		if total != n {
			t.Fatalf("deleted %d items across chunks, want %d", total, n)
		}
		if got := dumpAll(t, env, db); len(got) != 0 {
			t.Fatalf("table not empty after chunked delete: %d keys remain", len(got))
		}
	})

	// Error and edge cases.
	t.Run("errors", func(t *testing.T) {
		env, _ := setup(t)
		db := mustPutSeqBE(t, env, "distribute_errors", 10)

		if err := env.View(func(txn *Txn) error {
			first := openCursors(t, txn, db, 1)[0]
			defer first.Close()
			if _, _, err := first.Get(nil, nil, First); err != nil {
				return err
			}

			// Empty cursors slice.
			if _, err := DistributeCursors(first, nil, nil, 42); err == nil {
				t.Fatalf("expected error for empty cursors slice")
			}

			// Nil cursor inside the slice.
			c0 := openCursors(t, txn, db, 1)[0]
			defer c0.Close()
			if _, err := DistributeCursors(first, nil, []*Cursor{c0, nil}, 42); err == nil {
				t.Fatalf("expected error for nil cursor in slice")
			}

			// Both bounds nil is invalid per mdbx.
			c1 := openCursors(t, txn, db, 1)[0]
			defer c1.Close()
			if _, err := DistributeCursors(nil, nil, []*Cursor{c1}, 42); err == nil {
				t.Fatalf("expected error when both first and last are nil")
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestCursor_DeleteRange(t *testing.T) {
	t.Run("whole", func(t *testing.T) {
		env, _ := setup(t)
		db := mustPutUniqueSeq2(t, env, "delrange_whole", 10)
		var affected uint64
		if err := env.Update(func(txn *Txn) error {
			begin, err := txn.OpenCursor(db)
			if err != nil {
				return err
			}
			defer begin.Close()
			if _, _, err := begin.Get(nil, nil, First); err != nil {
				return err
			}
			affected, err = begin.DeleteRange(nil, true) // begin..last inclusive
			return err
		}); err != nil {
			t.Fatal(err)
		}
		if affected != 10 {
			t.Fatalf("affected=%d, want 10", affected)
		}
		if got := dumpAll(t, env, db); len(got) != 0 {
			t.Fatalf("remaining=%v, want empty", got)
		}
	})

	t.Run("partial-excluding", func(t *testing.T) {
		env, _ := setup(t)
		db := mustPutUniqueSeq2(t, env, "delrange_partial", 10)
		var affected uint64
		if err := env.Update(func(txn *Txn) error {
			begin, err := txn.OpenCursor(db)
			if err != nil {
				return err
			}
			defer begin.Close()
			end, err := txn.OpenCursor(db)
			if err != nil {
				return err
			}
			defer end.Close()

			if _, _, err := begin.Get([]byte("k2"), nil, SetKey); err != nil {
				return err
			}
			if _, _, err := end.Get([]byte("k7"), nil, SetKey); err != nil {
				return err
			}
			affected, err = begin.DeleteRange(end, false) // [k2, k7)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		remaining := dumpAll(t, env, db)
		if uint64(10-len(remaining)) != affected {
			t.Fatalf("affected=%d but %d keys removed", affected, 10-len(remaining))
		}
		// [k2,k7) excluding => k2..k6 removed (5), keep k0,k1,k7,k8,k9.
		if affected != 5 {
			t.Fatalf("affected=%d, want 5; remaining=%v", affected, remaining)
		}
		requireEqual(t, remaining, []kv{
			{"k0", "v0"}, {"k1", "v1"}, {"k7", "v7"}, {"k8", "v8"}, {"k9", "v9"},
		})
	})
}

// mustPutUniqueSeq2 is mustPutUniqueSeq but returns the created DBI.
func mustPutUniqueSeq2(t *testing.T, env *Env, name string, n int) DBI {
	t.Helper()
	db := mustOpenUniqueDB(t, env, name)
	mustPutUniqueSeq(t, env, db, n)
	return db
}

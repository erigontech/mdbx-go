package mdbx

/*
#include "mdbxgo.h"
*/
import "C"

import "errors"

var errNilReaderListFunc = errors.New("mdbx: nil ReaderList callback")

var (
	readerTIDTxnParked = uint64(C.mdbxgo_tid_txn_parked())
	readerTIDTxnOusted = uint64(C.mdbxgo_tid_txn_ousted())
)

// ReaderTIDTxnParked is the raw thread marker used by libmdbx for parked
// read transactions. Prefer ReaderInfo.Parked for application logic.
var ReaderTIDTxnParked = readerTIDTxnParked

// ReaderTIDTxnOusted is the raw thread marker used by libmdbx for ousted
// read transactions. Prefer ReaderInfo.Ousted for application logic.
var ReaderTIDTxnOusted = readerTIDTxnOusted

// ReaderInfo describes one entry in the MDBX reader lock table.
type ReaderInfo struct {
	Num           int
	Slot          int
	PID           int
	TID           uint64
	TxID          uint64
	Lag           uint64
	BytesUsed     uint64
	BytesRetained uint64
	Parked        bool
	Ousted        bool
}

// ReaderStats summarizes MDBX reader lock table state for metrics and
// monitoring. Collecting it may require scanning the reader lock table.
type ReaderStats struct {
	Count            uint64
	OldestTxID       uint64
	MaxLag           uint64
	MaxBytesUsed     uint64
	MaxBytesRetained uint64
	SumBytesRetained uint64
	Parked           uint64
	Ousted           uint64
}

// ReaderList enumerates the MDBX reader lock table as structured records.
//
// BytesRetained is the approximate amount of data prevented from reuse by the
// reader's MVCC snapshot.
func (env *Env) ReaderList(fn func(ReaderInfo) error) error {
	if fn == nil {
		return errNilReaderListFunc
	}
	ctx, done := newReaderFunc(fn)
	defer done()

	ret := C.mdbxgo_reader_list(env._env, C.size_t(ctx))
	if ctxerr := ctx.get().err; ctxerr != nil {
		return ctxerr
	}
	return operrno("mdbx_reader_list", ret)
}

// Readers returns a snapshot of all entries in the MDBX reader lock table.
func (env *Env) Readers() ([]ReaderInfo, error) {
	var readers []ReaderInfo
	err := env.ReaderList(func(info ReaderInfo) error {
		readers = append(readers, info)
		return nil
	})
	return readers, err
}

// ReaderStats returns aggregate MDBX reader metrics.
func (env *Env) ReaderStats() (ReaderStats, error) {
	var stats ReaderStats
	err := env.ReaderList(func(info ReaderInfo) error {
		stats.Count++
		if info.TxID != 0 && (stats.OldestTxID == 0 || info.TxID < stats.OldestTxID) {
			stats.OldestTxID = info.TxID
		}
		if info.Lag > stats.MaxLag {
			stats.MaxLag = info.Lag
		}
		if info.BytesUsed > stats.MaxBytesUsed {
			stats.MaxBytesUsed = info.BytesUsed
		}
		if info.BytesRetained > stats.MaxBytesRetained {
			stats.MaxBytesRetained = info.BytesRetained
		}
		stats.SumBytesRetained += info.BytesRetained
		if info.Parked {
			stats.Parked++
		}
		if info.Ousted {
			stats.Ousted++
		}
		return nil
	})
	return stats, err
}

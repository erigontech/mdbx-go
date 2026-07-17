package mdbx

/*
#include <stdlib.h>
#include "mdbxgo.h"
*/
import "C"
import "unsafe"

// GetBatchBuffer is a reusable buffer of MDBX_val (key, value) pairs for
// Cursor.GetBatch. It lives in C memory so cgo does not scan it for Go
// pointers on every call. Reusable across cursors and transactions; not safe
// for concurrent use. Close releases the C allocation.
type GetBatchBuffer struct {
	ptr  *C.MDBX_val
	size int
}

// maxBatchPairs bounds GetBatchBuffer sizes; it keeps 2*numPairs and the
// calloc byte count far from integer overflow on any platform.
const maxBatchPairs = 1 << 28

// NewGetBatchBuffer allocates a buffer holding numPairs key/value pairs
// (64-512 is a reasonable range).
func NewGetBatchBuffer(numPairs int) *GetBatchBuffer {
	if numPairs < 1 || numPairs > maxBatchPairs {
		panic("mdbx: NewGetBatchBuffer: number of pairs out of range")
	}
	p := C.calloc(C.size_t(2*numPairs), C.size_t(unsafe.Sizeof(C.MDBX_val{})))
	if p == nil {
		panic("mdbx: NewGetBatchBuffer: OOM")
	}
	return &GetBatchBuffer{ptr: (*C.MDBX_val)(p), size: numPairs}
}

// Close releases the C allocation. No-op if already closed.
func (b *GetBatchBuffer) Close() {
	if b.ptr != nil {
		C.free(unsafe.Pointer(b.ptr))
		b.ptr = nil
		b.size = 0
	}
}

// Cap returns the buffer capacity in key/value pairs.
func (b *GetBatchBuffer) Cap() int { return b.size }

func (b *GetBatchBuffer) at(i int) *C.MDBX_val {
	if b.ptr == nil || i < 0 || i >= 2*b.size {
		panic("mdbx: GetBatchBuffer: index out of range or buffer closed")
	}
	return (*C.MDBX_val)(unsafe.Add(unsafe.Pointer(b.ptr), uintptr(i)*unsafe.Sizeof(C.MDBX_val{})))
}

// Key returns the i-th key of the most recent GetBatch (i < its pair count).
// Zero-copy view: read-only, invalid after the txn ends or the buffer is
// refilled.
func (b *GetBatchBuffer) Key(i int) []byte { return castToBytes(b.at(2 * i)) }

// Val returns the i-th value of the most recent GetBatch. See Key.
func (b *GetBatchBuffer) Val(i int) []byte { return castToBytes(b.at(2*i + 1)) }

// GetBatch fetches up to buf.Cap() key/value pairs in one cgo call: the first
// record with opFirst, the rest with opNext. It amortizes cgo overhead over
// large scans.
//
// There is no way to pass a search key or value, so ops that need one (Set,
// SetRange, GetBoth, ...) cannot be used. For a ranged scan, position the
// cursor with Get first and batch with (GetCurrent, Next).
//
// n is the number of pairs stored (read via buf.Key/Val). eof is true when
// iteration was exhausted before the buffer filled; otherwise continue with
// GetBatch(buf, opNext, opNext).
//
//	buf := mdbx.NewGetBatchBuffer(256)
//	defer buf.Close()
//	for opFirst := uint(mdbx.First); ; opFirst = mdbx.Next {
//		n, eof, err := cur.GetBatch(buf, opFirst, mdbx.Next)
//		if err != nil {
//			return err
//		}
//		for i := 0; i < n; i++ {
//			handle(buf.Key(i), buf.Val(i))
//		}
//		if eof {
//			break
//		}
//	}
func (c *Cursor) GetBatch(buf *GetBatchBuffer, opFirst, opNext uint) (n int, eof bool, err error) {
	if c._c == nil || buf == nil || buf.ptr == nil || buf.size <= 0 {
		return 0, false, operrno("mdbx_cursor_get", C.MDBX_EINVAL)
	}
	r := C.mdbxgo_cursor_get_batch(
		c._c, buf.ptr, C.size_t(buf.size),
		C.MDBX_cursor_op(opFirst), C.MDBX_cursor_op(opNext),
	)
	n = int(r.val)
	if r.err == C.MDBX_NOTFOUND {
		return n, true, nil
	}
	if r.err != success {
		return n, false, operrno("mdbx_cursor_get", r.err)
	}
	return n, false, nil
}

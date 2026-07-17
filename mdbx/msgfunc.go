package mdbx

/*
#include "mdbxgo.h"
*/
import "C"
import (
	"fmt"
	"sync"
	"sync/atomic"
)

// mdbxgoMDBReaderListBridge provides a static C function for handling
// MDBX_reader_list_func callbacks.  It performs type conversion and dynamic
// dispatch to a callback provided to Env.ReaderList.  Any error returned by the
// callback is cached and MDBX_RESULT_TRUE is returned to terminate the
// iteration.
//
//export mdbxgoMDBReaderListBridge
func mdbxgoMDBReaderListBridge(_ctx C.size_t, num C.int, slot C.int, pid C.mdbx_pid_t, thread C.uint64_t, txnid C.uint64_t, lag C.uint64_t, bytesUsed C.size_t, bytesRetained C.size_t) (rc C.int) {
	ctx := readerctx(_ctx).get()
	defer func() {
		if r := recover(); r != nil {
			ctx.err = fmt.Errorf("mdbx: panic in ReaderList callback: %v", r)
			rc = C.MDBX_RESULT_TRUE
		}
	}()

	info := ReaderInfo{
		Num:           int(num),
		Slot:          int(slot),
		PID:           int(pid),
		TID:           uint64(thread),
		TxID:          uint64(txnid),
		Lag:           uint64(lag),
		BytesUsed:     uint64(bytesUsed),
		BytesRetained: uint64(bytesRetained),
	}
	info.Parked = info.TID == readerTIDTxnParked
	info.Ousted = info.TID == readerTIDTxnOusted
	err := ctx.fn(info)
	if err != nil {
		ctx.err = err
		return C.MDBX_RESULT_TRUE
	}
	return C.MDBX_RESULT_FALSE
}

type readerfunc func(ReaderInfo) error

// readerctx is the type used for context pointers passed to mdbx_reader_list.
// It keeps Go pointers out of C memory by storing callback state in a Go map
// and passing only an integer handle through libmdbx.
//
// An external map is used because struct pointers passed to C functions must
// not contain pointers in their struct fields.  See the following language
// proposal which discusses the restrictions on passing pointers to C.
//
//	https://github.com/golang/proposal/blob/master/design/12416-cgo-pointers.md
type readerctx uintptr
type _readerctx struct {
	fn  readerfunc
	err error
}

var readerctxn uint32
var readerctxm = map[readerctx]*_readerctx{}
var readerctxmlock sync.RWMutex

func newReaderFunc(fn readerfunc) (ctx readerctx, done func()) {
	ctx = readerctx(atomic.AddUint32(&readerctxn, 1))
	ctx.register(fn)
	return ctx, ctx.deregister
}

func (ctx readerctx) register(fn readerfunc) {
	readerctxmlock.Lock()
	if _, ok := readerctxm[ctx]; ok {
		readerctxmlock.Unlock()
		panic("readerfunc conflict")
	}
	readerctxm[ctx] = &_readerctx{fn: fn}
	readerctxmlock.Unlock()
}

func (ctx readerctx) deregister() {
	readerctxmlock.Lock()
	delete(readerctxm, ctx)
	readerctxmlock.Unlock()
}

func (ctx readerctx) get() *_readerctx {
	readerctxmlock.RLock()
	_ctx := readerctxm[ctx]
	readerctxmlock.RUnlock()
	return _ctx
}

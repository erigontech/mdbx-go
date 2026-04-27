package mdbx

/*
#include "mdbxgo.h"
*/
import "C"
import (
	"sync"
	"sync/atomic"
)

// mdbxgoMDBMsgFuncBridge provides a static C function for handling MDB_msgfunc
// callbacks.  It performs string conversion and dynamic dispatch to a msgfunc
// provided to Env.ReaderList.  Any error returned by the msgfunc is cached and
// -1 is returned to terminate the iteration.

//export mdbxgoMDBMsgFuncBridge
func mdbxgoMDBMsgFuncBridge(cmsg C.mdbxgo_ConstCString, _ctx C.size_t) C.int {
	ctx := msgctx(_ctx).get()
	msg := C.GoString(cmsg.p)
	err := ctx.fn(msg)
	if err != nil {
		ctx.err = err
		return -1
	}
	return 0
}

// mdbxgoMDBReaderListBridge provides a static C function for handling
// MDBX_reader_list_func callbacks.  It performs type conversion and dynamic
// dispatch to a callback provided to Env.ReaderList.  Any error returned by the
// callback is cached and MDBX_RESULT_TRUE is returned to terminate the
// iteration.
//
//export mdbxgoMDBReaderListBridge
func mdbxgoMDBReaderListBridge(_ctx C.size_t, num C.int, slot C.int, pid C.mdbx_pid_t, thread C.uint64_t, txnid C.uint64_t, lag C.uint64_t, bytesUsed C.size_t, bytesRetained C.size_t) C.int {
	ctx := readerctx(_ctx).get()
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

type msgfunc func(string) error

// msgctx is the type used for context pointers passed to mdb_reader_list.  A
// msgctx stores its corresponding msgfunc, and any error encountered in an
// external map.  The corresponding function is called once for each
// mdb_reader_list entry using the msgctx.
//
// An External map is used because struct pointers passed to C functions must
// not contain pointers in their struct fields.  See the following language
// proposal which discusses the restrictions on passing pointers to C.
//
//	https://github.com/golang/proposal/blob/master/design/12416-cgo-pointers.md
type msgctx uintptr
type _msgctx struct {
	fn  msgfunc
	err error
}

var msgctxn uint32
var msgctxm = map[msgctx]*_msgctx{}
var msgctxmlock sync.RWMutex

func nextctx() msgctx {
	return msgctx(atomic.AddUint32(&msgctxn, 1))
}

//nolint:deadcode,unused
func newMsgFunc(fn msgfunc) (ctx msgctx, done func()) {
	ctx = nextctx()
	ctx.register(fn)
	return ctx, ctx.deregister
}

func (ctx msgctx) register(fn msgfunc) {
	msgctxmlock.Lock()
	if _, ok := msgctxm[ctx]; ok {
		msgctxmlock.Unlock()
		panic("msgfunc conflict")
	}
	msgctxm[ctx] = &_msgctx{fn: fn}
	msgctxmlock.Unlock()
}

func (ctx msgctx) deregister() {
	msgctxmlock.Lock()
	delete(msgctxm, ctx)
	msgctxmlock.Unlock()
}

func (ctx msgctx) get() *_msgctx {
	msgctxmlock.RLock()
	_ctx := msgctxm[ctx]
	msgctxmlock.RUnlock()
	return _ctx
}

type readerfunc func(ReaderInfo) error

// readerctx is the type used for context pointers passed to mdbx_reader_list.
// It keeps Go pointers out of C memory by storing callback state in a Go map
// and passing only an integer handle through libmdbx.
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

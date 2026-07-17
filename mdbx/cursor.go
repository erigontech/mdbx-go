package mdbx

/*
#include <stdlib.h>
#include <stdio.h>
#include "mdbxgo.h"
*/
import "C"
import (
	"errors"
	"sync"
	"unsafe"
)

const (
	// Flags for Cursor.Get
	//
	// See MDB_cursor_op.

	First         = C.MDBX_FIRST          // The first item.
	FirstDup      = C.MDBX_FIRST_DUP      // The first value of current key (DupSort).
	GetBoth       = C.MDBX_GET_BOTH       // Get the key as well as the value (DupSort).
	GetBothRange  = C.MDBX_GET_BOTH_RANGE // Get the key and the nearsest value (DupSort).
	GetCurrent    = C.MDBX_GET_CURRENT    // Get the key and value at the current position.
	GetMultiple   = C.MDBX_GET_MULTIPLE   // Get up to a page dup values for key at current position (DupFixed).
	Last          = C.MDBX_LAST           // Last item.
	LastDup       = C.MDBX_LAST_DUP       // Position at last value of current key (DupSort).
	Next          = C.MDBX_NEXT           // Next value.
	NextDup       = C.MDBX_NEXT_DUP       // Next value of the current key (DupSort).
	NextMultiple  = C.MDBX_NEXT_MULTIPLE  // Get key and up to a page of values from the next cursor position (DupFixed).
	NextNoDup     = C.MDBX_NEXT_NODUP     // The first value of the next key (DupSort).
	Prev          = C.MDBX_PREV           // The previous item.
	PrevDup       = C.MDBX_PREV_DUP       // The previous item of the current key (DupSort).
	PrevNoDup     = C.MDBX_PREV_NODUP     // The last data item of the previous key (DupSort).
	PrevMultiple  = C.MDBX_PREV_MULTIPLE  //
	Set           = C.MDBX_SET            // The specified key.
	SetKey        = C.MDBX_SET_KEY        // Get key and data at the specified key.
	SetRange      = C.MDBX_SET_RANGE      // The first key no less than the specified key.
	SetLowerBound = C.MDBX_SET_LOWERBOUND // The first key/value pair no less than the specified key/value pair.
	SetUpperBound = C.MDBX_SET_UPPERBOUND // The first key/value pair greater than the specified key/value pair.

	KeyLesserThan     = C.MDBX_TO_KEY_LESSER_THAN
	KeyLesserOrEqual  = C.MDBX_TO_KEY_LESSER_OR_EQUAL
	KeyEqual          = C.MDBX_TO_KEY_EQUAL
	KeyGreaterOrEqual = C.MDBX_TO_KEY_GREATER_OR_EQUAL
	KeyGreaterThan    = C.MDBX_TO_KEY_GREATER_THAN

	ExactKeyValueLesserThan     = C.MDBX_TO_EXACT_KEY_VALUE_LESSER_THAN
	ExactKeyValueLesserOrEqual  = C.MDBX_TO_EXACT_KEY_VALUE_LESSER_OR_EQUAL
	ExactKeyValueEqual          = C.MDBX_TO_EXACT_KEY_VALUE_EQUAL
	ExactKeyValueGreaterOrEqual = C.MDBX_TO_EXACT_KEY_VALUE_GREATER_OR_EQUAL
	ExactKeyValueGreaterThan    = C.MDBX_TO_EXACT_KEY_VALUE_GREATER_THAN

	PairLesserThan     = C.MDBX_TO_PAIR_LESSER_THAN
	PairLesserOrEqual  = C.MDBX_TO_PAIR_LESSER_OR_EQUAL
	PairEqual          = C.MDBX_TO_PAIR_EQUAL
	PairGreaterOrEqual = C.MDBX_TO_PAIR_GREATER_OR_EQUAL
	PairGreaterThan    = C.MDBX_TO_PAIR_GREATER_THAN

	LesserThan = KeyLesserThan // Deprecated: use KeyLesserThan.
)

// The MDB_MULTIPLE and MDB_RESERVE flags are special and do not fit the
// calling pattern of other calls to Put.  They are not exported because they
// require special methods, PutMultiple and PutReserve in which the flag is
// implied and does not need to be passed.
const (
	// Flags for Txn.Put and Cursor.Put.
	//
	// See mdb_put and mdb_cursor_put.
	Upsert      = C.MDBX_UPSERT      // Replace the item at the current key position (Cursor only)
	Current     = C.MDBX_CURRENT     // Replace the item at the current key position (Cursor only)
	NoDupData   = C.MDBX_NODUPDATA   // Store the key-value pair only if key is not present (DupSort).
	NoOverwrite = C.MDBX_NOOVERWRITE // Store a new key-value pair only if key is not present.
	Append      = C.MDBX_APPEND      // Append an item to the database.
	AppendDup   = C.MDBX_APPENDDUP   // Append an item to the database (DupSort).
	AllDups     = C.MDBX_ALLDUPS
)

const (
	// Flags for Cursor.RangeDel
	//
	// See mdbx_cursor_bunch_delete.
	DeleteCurrentValue                   = C.MDBX_DELETE_CURRENT_VALUE
	DeleteCurrentMultiValBeforeExcluding = C.MDBX_DELETE_CURRENT_MULTIVAL_BEFORE_EXCLUDING
	DeleteCurrentMultiValBeforeIncluding = C.MDBX_DELETE_CURRENT_MULTIVAL_BEFORE_INCLUDING
	DeleteCurrentMultiValAfterIncluding  = C.MDBX_DELETE_CURRENT_MULTIVAL_AFTER_INCLUDING
	DeleteCurrentMultiValAfterExcluding  = C.MDBX_DELETE_CURRENT_MULTIVAL_AFTER_EXCLUDING
	DeleteCurrentValueMultiValAll        = C.MDBX_DELETE_CURRENT_MULTIVAL_ALL
	DeleteBeforeExcluding                = C.MDBX_DELETE_BEFORE_EXCLUDING
	DeleteBeforeIncluding                = C.MDBX_DELETE_BEFORE_INCLUDING
	DeleteAfterIncluding                 = C.MDBX_DELETE_AFTER_INCLUDING
	DeleteAfterExcluding                 = C.MDBX_DELETE_AFTER_EXCLUDING
	DeleteWhole                          = C.MDBX_DELETE_WHOLE
)

// Cursor operates on data inside a transaction and holds a position in the
// database.
//
// See MDB_cursor.
type Cursor struct {
	txn *Txn
	_c  *C.MDBX_cursor
}

//nolint:gocritic // reason: false positive on dupSubExpr
func openCursor(txn *Txn, db DBI) (*Cursor, error) {
	c := &Cursor{txn: txn}
	ret := C.mdbx_cursor_open(txn._txn, C.MDBX_dbi(db), &c._c)
	if ret != success {
		return nil, operrno("mdbx_cursor_open", ret)
	}
	return c, nil
}

// Renew associates cursor with txn.
//
// See mdb_cursor_renew.
func (c *Cursor) Renew(txn *Txn) error {
	ret := C.mdbx_cursor_renew(txn._txn, c._c)
	if ret != success {
		return operrno("mdbx_cursor_renew", ret)
	}
	c.txn = txn
	return nil
}

// Bind Using of the `mdbx_cursor_bind()` is equivalent to calling mdbx_cursor_renew() but with specifying an arbitrary
// dbi handle.
func (c *Cursor) Bind(txn *Txn, db DBI) error {
	ret := C.mdbx_cursor_bind(txn._txn, c._c, C.MDBX_dbi(db))
	if ret != success {
		return operrno("mdbx_cursor_bind", ret)
	}
	c.txn = txn
	return nil
}

// Unbind Unbinded cursor is disassociated with any transactions but still holds
// the original DBI-handle internally. Thus, it could be renewed with any running
// transaction or closed.
func (c *Cursor) Unbind() error {
	ret := C.mdbx_cursor_unbind(c._c)
	if ret != success {
		return operrno("mdbx_cursor_unbind", ret)
	}
	return nil
}

// Close the cursor handle and clear the finalizer on c.  Cursors belonging to
// write transactions are closed automatically when the transaction is
// terminated.
//
// See mdb_cursor_close.
func (c *Cursor) Close() {
	if c._c != nil {
		if c.txn._txn == nil && !c.txn.readonly {
			// the cursor has already been released by MDBX.
		} else {
			C.mdbx_cursor_close(c._c)
		}
		c.txn = nil
		c._c = nil
	}
}

// Txn returns the cursor's transaction.
func (c *Cursor) Txn() *Txn {
	return c.txn
}

// DBI returns the cursor's database handle.  If c has been closed than an
// invalid DBI is returned.
func (c *Cursor) DBI() DBI {
	// dbiInvalid is an invalid DBI (the max value for the type).  it shouldn't
	// be possible to create a database handle with value dbiInvalid because
	// the process address space would be exhausted.  it is also impractical to
	// have many open databases in an environment.
	const dbiInvalid = ^DBI(0)

	// mdb_cursor_dbi segfaults when passed a nil value
	if c._c == nil {
		return dbiInvalid
	}
	return DBI(C.mdbx_cursor_dbi(c._c))
}

// Get retrieves items from the database. If c.Txn().RawRead is true the slices
// returned by Get reference readonly sections of memory that must not be
// accessed after the transaction has terminated.
//
// In a Txn with RawRead set to true the Set op causes the returned key to
// share its memory with setkey (making it writable memory). In a Txn with
// RawRead set to false the Set op returns key values with memory distinct from
// setkey, as is always the case when using RawRead.
//
// Get ignores setval if setkey is empty.
//
// See mdb_cursor_get.
func (c *Cursor) Get(setkey, setval []byte, op uint) (key, val []byte, err error) {
	var r C.mdbxgo_val_result
	if len(setkey) != 0 || len(setval) != 0 {
		r = c.getVal(setkey, setval, op)
	} else {
		r = c.getValEmpty(op)
	}
	if r.err != success {
		return nil, nil, operrno("mdbx_cursor_get", r.err)
	}

	// For MDBX_SET mdbx makes no promise about the returned key, so as an
	// implementation choice we hand back setkey itself instead of deriving a
	// slice from the C result.
	if op == Set {
		key = setkey
	} else if op != LastDup && op != FirstDup {
		key = castToBytesRaw(unsafe.Pointer(r.kbase), r.klen)
	}
	val = castToBytesRaw(unsafe.Pointer(r.vbase), r.vlen)

	return key, val, nil
}

// getValEmpty retrieves items from the database without using given key or value
// data for reference (Next, First, Last, etc).
//
// See mdb_cursor_get.
//
//nolint:gocritic // false positive on dupSubExpr
func (c *Cursor) getValEmpty(op uint) C.mdbxgo_val_result {
	return C.mdbxgo_cursor_get_empty(c._c, C.MDBX_cursor_op(op))
}

// getVal retrieves items from the database using key and value data for
// reference (GetBoth, GetBothRange, etc).
//
// See mdb_cursor_get.
//
//nolint:gocritic // false positive on dupSubExpr
func (c *Cursor) getVal(setkey, setval []byte, op uint) C.mdbxgo_val_result {
	var k, v *C.char
	if len(setkey) > 0 {
		k = (*C.char)(unsafe.Pointer(&setkey[0]))
	}
	if len(setval) > 0 {
		v = (*C.char)(unsafe.Pointer(&setval[0]))
	}
	return C.mdbxgo_cursor_get_val(
		c._c,
		k, C.size_t(len(setkey)),
		v, C.size_t(len(setval)),
		C.MDBX_cursor_op(op),
	)
}

// Put stores an item in the database.
//
// See mdb_cursor_put.
func (c *Cursor) Put(key, val []byte, flags uint) error {
	if c._c == nil || c.txn == nil {
		return operrno("mdbx_cursor_put", C.MDBX_EINVAL)
	}
	var k, v *C.char
	if len(key) > 0 {
		k = (*C.char)(unsafe.Pointer(&key[0]))
	}
	if len(val) > 0 {
		v = (*C.char)(unsafe.Pointer(&val[0]))
	}
	ret := C.mdbxgo_cursor_put2(
		c._c,
		k, C.size_t(len(key)),
		v, C.size_t(len(val)),
		C.MDBX_put_flags_t(flags),
	)
	return operrno("mdbx_cursor_put", ret)
}

// PutReserve returns a []byte of length n that can be written to, potentially
// avoiding a memcopy.  The returned byte slice is only valid in txn's thread,
// before it has terminated.
//
//nolint:gocritic // false positive on dupSubExpr
func (c *Cursor) PutReserve(key []byte, n int, flags uint) ([]byte, error) {
	var k *C.char
	if len(key) > 0 {
		k = (*C.char)(unsafe.Pointer(&key[0]))
	}
	c.txn.val.iov_len = C.size_t(n)
	ret := C.mdbxgo_cursor_put1(
		c._c,
		k, C.size_t(len(key)),
		&c.txn.val,
		C.MDBX_put_flags_t(flags|C.MDBX_RESERVE),
	)
	err := operrno("mdbx_cursor_put", ret)
	if err != nil {
		c.txn.val = C.MDBX_val{}
		return nil, err
	}
	b := castToBytes(&c.txn.val)
	c.txn.val = C.MDBX_val{}
	return b, nil
}

// PutMulti stores a set of contiguous items with stride size under key.
// PutMulti panics if len(page) is not a multiple of stride.  The cursor's
// database must be DupFixed and DupSort.
//
// See mdb_cursor_put.
func (c *Cursor) PutMulti(key []byte, page []byte, stride int, flags uint) error {
	var k *C.char
	if len(key) > 0 {
		k = (*C.char)(unsafe.Pointer(&key[0]))
	}
	var v *C.char
	if len(page) > 0 {
		v = (*C.char)(unsafe.Pointer(&page[0]))
	}
	vn := WrapMulti(page, stride).Len()
	ret := C.mdbxgo_cursor_putmulti(
		c._c,
		k, C.size_t(len(key)),
		v, C.size_t(vn), C.size_t(stride),
		C.MDBX_put_flags_t(flags|C.MDBX_MULTIPLE),
	)
	return operrno("mdbxgo_cursor_putmulti", ret)
}

// PutCurrent replaces the data of the item at the current cursor position.
// For DupSort databases, this replaces the current duplicate entry in-place,
// avoiding a separate Del+Put round-trip (saves one CGo call per update).
// The cursor must be positioned (e.g. via Get with GetBothRange) before calling.
//
// Equivalent to Put(key, val, Current).
func (c *Cursor) PutCurrent(key, val []byte) error {
	return c.Put(key, val, Current)
}

// Del deletes the item referred to by the cursor from the database.
//
// See mdb_cursor_del.
func (c *Cursor) Del(flags uint) error {
	ret := C.mdbx_cursor_del(c._c, C.MDBX_put_flags_t(flags))
	return operrno("mdbx_cursor_del", ret)
}

// RangeDel deletes a range of items referred to by the cursor from the database.
//
// It returns the number of affected (deleted) items.
// Modes: see mdbx_cursor_bunch_delete.
func (c *Cursor) RangeDel(mode uint) (numberAffected uint64, err error) {
	r := C.mdbxgo_cursor_bunch_delete(c._c, C.MDBX_bunch_action_t(mode))
	if err := operrno("mdbx_cursor_bunch_delete", r.err); err != nil {
		return 0, err
	}
	return uint64(r.val), nil
}

// cptr returns c's underlying C cursor, or NULL when c is nil. The range/
// distribution APIs accept a nil bound cursor to mean "the start/end of the
// table", which maps to a NULL MDBX_cursor* on the C side.
func cptr(c *Cursor) *C.MDBX_cursor {
	if c == nil {
		return nil
	}
	return c._c
}

// DeleteRange performs a mass deletion of the items between the position of the
// receiver cursor (the beginning of the range) and the position of end (the end
// of the range). It is much faster than deleting items one by one because whole
// pages and branches are cut out of the B+ tree.
//
// Both cursors must already be positioned (see Cursor.Get) and bound to the same
// table and write transaction. A nil end deletes up to the last item.
// endIncluding controls whether the item at end's position is itself deleted.
//
// It returns the number of deleted items.
//
// See mdbx_cursor_delete_range.
func (c *Cursor) DeleteRange(end *Cursor, endIncluding bool) (numberAffected uint64, err error) {
	r := C.mdbxgo_cursor_delete_range(c._c, cptr(end), C.bool(endIncluding))
	if err := operrno("mdbx_cursor_delete_range", r.err); err != nil {
		return 0, err
	}
	return uint64(r.val), nil
}

// EstimateDistance estimates the number of elements between the position of the
// receiver cursor and the position of last. Both cursors must be non-nil,
// positioned, and initialized for the same table and transaction. Unlike
// Distance, this estimate has no end-of-table sentinel: a nil last is not a
// valid argument. The result is a rough estimate suitable for building/optimizing
// query plans, not an exact count.
//
// See mdbx_estimate_distance.
func (c *Cursor) EstimateDistance(last *Cursor) (int, error) {
	r := C.mdbxgo_estimate_distance(c._c, cptr(last))
	if err := operrno("mdbx_estimate_distance", r.err); err != nil {
		return 0, err
	}
	return int(r.val), nil
}

// EstimateMove estimates the distance, as a number of elements, that the cursor
// would move if the given key/data + op were applied via Get. The cursor's own
// position and state are left unchanged. The result is a rough estimate suitable
// for building/optimizing query plans, not an exact count.
//
// key/data are interpreted the same way as in Get for the given op; pass nil
// where the op does not require them.
//
// See mdbx_estimate_move.
func (c *Cursor) EstimateMove(key, data []byte, op uint) (int, error) {
	var k, d *C.char
	if len(key) > 0 {
		k = (*C.char)(unsafe.Pointer(&key[0]))
	}
	if len(data) > 0 {
		d = (*C.char)(unsafe.Pointer(&data[0]))
	}
	r := C.mdbxgo_estimate_move(
		c._c,
		k, C.size_t(len(key)),
		d, C.size_t(len(data)),
		C.MDBX_cursor_op(op),
	)
	if err := operrno("mdbx_estimate_move", r.err); err != nil {
		return 0, err
	}
	return int(r.val), nil
}

// Distance calculates the number of elements between the position of the
// receiver cursor and the position of last. Both cursors must be positioned and
// bound to the same table and transaction; a nil last means the end of the
// table.
//
// deepness limits the B-tree level at which the difference is measured: 0 is the
// root (fast but rough), larger values descend toward the leaves (slower but
// exact). For an exact count deepness must be at least the B-tree height (plus
// the nested height for DupSort tables); when in doubt pass a deliberately large
// value such as 42.
//
// See mdbx_cursor_distance.
func (c *Cursor) Distance(last *Cursor, deepness uint) (int, error) {
	r := C.mdbxgo_cursor_distance(c._c, cptr(last), C.unsigned(deepness))
	if err := operrno("mdbx_cursor_distance", r.err); err != nil {
		return 0, err
	}
	return int(r.val), nil
}

// Scroll moves the cursor by amount logical positions at the given B-tree level.
// A positive amount moves forward (toward the end of the table), a negative one
// moves backward. The cursor must already be positioned.
//
// deepness selects the B-tree level the steps are taken at: for the movement to
// match a number of keys/values it must be at least the B-tree height (plus the
// nested height for DupSort tables); when in doubt pass a deliberately large
// value such as 42.
//
// Scroll returns an error for which IsNotFound reports true if the end of the
// data is reached before the cursor moved by the full amount.
//
// See mdbx_cursor_scroll.
func (c *Cursor) Scroll(amount int, deepness uint) error {
	ret := C.mdbx_cursor_scroll(c._c, C.intptr_t(amount), C.unsigned(deepness))
	return operrno("mdbx_cursor_scroll", ret)
}

// DistributeCursors positions each cursor in cursors at an evenly-spaced
// position across the range delimited by first and last, so that consecutive
// cursors delimit approximately equally-sized sub-ranges. This is the intended
// primitive for splitting a key range into balanced chunks for parallel workers
// (e.g. concurrent range deletion or warm-up).
//
// A nil first means the beginning of the table and a nil last means the end;
// at least one of them must be non-nil and positioned. Every entry in cursors
// must be a non-nil open cursor bound to the same table and transaction (a nil
// entry returns an error); these are the cursors that get positioned.
//
// deepness selects the B-tree level at which the distribution is computed: for
// the positions to match a number of keys/values it must be at least the B-tree
// height (plus the nested height for DupSort tables); when in doubt pass a
// deliberately large value such as 42.
//
// allSet reports whether every cursor was positioned. It is false (with a nil
// error) when the range held fewer positions than len(cursors); the surplus
// cursors are left unset (at EOF).
//
// See mdbx_cursor_distribute.
func DistributeCursors(first, last *Cursor, cursors []*Cursor, deepness uint) (allSet bool, err error) {
	if len(cursors) == 0 {
		return false, errors.New("mdbx: DistributeCursors requires at least one cursor")
	}
	arr := make([]*C.MDBX_cursor, len(cursors))
	for i, cur := range cursors {
		if cur == nil || cur._c == nil {
			return false, errors.New("mdbx: DistributeCursors got a nil cursor")
		}
		arr[i] = cur._c
	}
	ret := C.mdbx_cursor_distribute(cptr(first), cptr(last), &arr[0], C.intptr_t(len(arr)), C.unsigned(deepness))
	if ret == C.MDBX_RESULT_TRUE {
		// Not enough positions in the range to set every cursor; the surplus
		// cursors are left at EOF. operrno treats MDBX_RESULT_TRUE as success,
		// so this case is reported via allSet rather than err.
		return false, nil
	}
	if err := operrno("mdbx_cursor_distribute", ret); err != nil {
		return false, err
	}
	return true, nil
}

// Count returns the number of duplicates for the current key.
//
// See mdb_cursor_count.
func (c *Cursor) Count() (uint64, error) {
	r := C.mdbxgo_cursor_count(c._c)
	if r.err != success {
		return 0, operrno("mdbx_cursor_count", r.err)
	}
	return uint64(r.val), nil
}

var cursorPool = sync.Pool{
	New: func() any {
		return CreateCursor()
	},
}

func CursorFromPool() *Cursor { return cursorPool.Get().(*Cursor) }
func CursorToPool(c *Cursor)  { cursorPool.Put(c) }

func CreateCursor() *Cursor {
	c := &Cursor{_c: C.mdbx_cursor_create(nil)}
	if c._c == nil {
		panic(errors.New("mdbx.CreateCursor: OOM"))
	}
	return c
}

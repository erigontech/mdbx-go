package mdbx

/*
#include <stdlib.h>
#include <stdio.h>
#include "mdbxgo.h"
*/
import "C"
import (
	"errors"
	"os"
	"runtime"
	"sync"
	"time"
	"unsafe"

	_ "github.com/erigontech/mdbx-go/dbg"
)

// success is a value returned from the MDBX API to indicate a successful call.
// The functions in this API this behavior and its use is not required.
const success = C.MDBX_SUCCESS

const Major = C.MDBX_VERSION_MAJOR
const Minor = C.MDBX_VERSION_MINOR

const (
	// Flags for Env.Open.
	//
	// See mdbx_env_open

	EnvDefaults = C.MDBX_ENV_DEFAULTS
	LifoReclaim = C.MDBX_LIFORECLAIM
	// FixedMap    = C.MDBX_FIXEDMAP   // Danger zone. Map memory at a fixed address.
	NoSubdir      = C.MDBX_NOSUBDIR // Argument to Open is a file, not a directory.
	Accede        = C.MDBX_ACCEDE
	Readonly      = C.MDBX_RDONLY     // Used in several functions to denote an object as readonly.
	WriteMap      = C.MDBX_WRITEMAP   // Use a writable memory map.
	NoMetaSync    = C.MDBX_NOMETASYNC // Don't fsync metapage after commit.
	UtterlyNoSync = C.MDBX_UTTERLY_NOSYNC
	SafeNoSync    = C.MDBX_SAFE_NOSYNC
	Durable       = C.MDBX_SYNC_DURABLE
	// Deprecated: use NoStickyThreads instead because now they're sharing the same functionality
	NoTLS           = C.MDBX_NOTLS           // Danger zone. When unset reader locktable slots are tied to their thread.
	NoStickyThreads = C.MDBX_NOSTICKYTHREADS // Danger zone. Like MDBX_NOTLS. But also allow move RwTx between threads. Still require to call Begin/Rollback in same thread.
	// NoLock      = C.MDBX_NOLOCK     // Danger zone. MDBX does not use any locks.
	NoReadahead = C.MDBX_NORDAHEAD // Disable readahead. Requires OS support.
	NoMemInit   = C.MDBX_NOMEMINIT // Disable MDBX memory initialization.
	Exclusive   = C.MDBX_EXCLUSIVE // Disable MDBX memory initialization.
)

const (
	MinPageSize = C.MDBX_MIN_PAGESIZE
	MaxPageSize = C.MDBX_MAX_PAGESIZE
	MaxDbi      = C.MDBX_MAX_DBI
)

// These flags are exclusively used in the Env.CopyFlags and Env.CopyFDFlags
// methods.
const (
	// Flags for Env.CopyFlags
	//
	// See mdbx_env_copy2

	CopyCompact = C.MDBX_CP_COMPACT // Perform compaction while copying
)

const (
	AllowTxOverlap = C.MDBX_DBG_LEGACY_OVERLAP
)

type LogLvl = C.MDBX_log_level_t

const (
	LogLvlFatal       = C.MDBX_LOG_FATAL
	LogLvlError       = C.MDBX_LOG_ERROR
	LogLvlWarn        = C.MDBX_LOG_WARN
	LogLvlNotice      = C.MDBX_LOG_NOTICE
	LogLvlVerbose     = C.MDBX_LOG_VERBOSE
	LogLvlDebug       = C.MDBX_LOG_DEBUG
	LogLvlTrace       = C.MDBX_LOG_TRACE
	LogLvlExtra       = C.MDBX_LOG_EXTRA
	LogLvlDoNotChange = C.MDBX_LOG_DONTCHANGE
)

const (
	DbgAssert          = C.MDBX_DBG_ASSERT
	DbgAudit           = C.MDBX_DBG_AUDIT
	DbgJitter          = C.MDBX_DBG_JITTER
	DbgDump            = C.MDBX_DBG_DUMP
	DbgLegacyMultiOpen = C.MDBX_DBG_LEGACY_MULTIOPEN
	DbgLegacyTxOverlap = C.MDBX_DBG_LEGACY_OVERLAP
	DbgDoNotChange     = C.MDBX_DBG_DONTCHANGE
)

const (
	OptMaxDB                        = C.MDBX_opt_max_db
	OptMaxReaders                   = C.MDBX_opt_max_readers
	OptSyncBytes                    = C.MDBX_opt_sync_bytes
	OptSyncPeriod                   = C.MDBX_opt_sync_period
	OptRpAugmentLimit               = C.MDBX_opt_rp_augment_limit
	OptLooseLimit                   = C.MDBX_opt_loose_limit
	OptDpReverseLimit               = C.MDBX_opt_dp_reserve_limit
	OptTxnDpLimit                   = C.MDBX_opt_txn_dp_limit
	OptTxnDpInitial                 = C.MDBX_opt_txn_dp_initial
	OptSpillMaxDenominator          = C.MDBX_opt_spill_max_denominator
	OptSpillMinDenominator          = C.MDBX_opt_spill_min_denominator
	OptSpillParent4ChildDenominator = C.MDBX_opt_spill_parent4child_denominator
	OptMergeThreshold16dot16Percent = C.MDBX_opt_merge_threshold_16dot16_percent
	OptPreferWafInsteadofBalance    = C.MDBX_opt_prefer_waf_insteadof_balance
	OptGCTimeLimit                  = C.MDBX_opt_gc_time_limit
)

var (
	LoggerDoNotChange = C.MDBX_LOGGER_DONTCHANGE
)

// Label - will be added to error messages. For better understanding - which DB has problem.
type Label string

const Default Label = "default"

// DBI is a handle for a database in an Env.
//
// See MDBX_dbi
type DBI C.MDBX_dbi

// Env is opaque structure for a database environment.  A DB environment
// supports multiple databases, all residing in the same shared-memory map.
//
// See MDBX_env.
type Env struct {
	_env  *C.MDBX_env
	label Label

	// closeLock is used to allow the Txn finalizer to check if the Env has
	// been closed, so that it may know if it must abort.
	closeLock sync.RWMutex

	strictThreadCheck bool
}

// NewEnv allocates and initializes a new Env.
//
// See mdbx_env_create.
//
//nolint:gocritic // false positive on dupSubExpr
func NewEnv(label Label) (*Env, error) {
	env := &Env{label: label}
	ret := C.mdbx_env_create(&env._env)
	if ret != success {
		return nil, operrno("mdbx_env_create", ret)
	}
	return env, nil
}

// Open an environment handle. If this function fails Close() must be called to
// discard the Env handle.  Open passes flags|NoTLS to mdbx_env_open.
//
// See mdbx_env_open.
func (env *Env) Open(path string, flags uint, mode os.FileMode) error {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	ret := C.mdbx_env_open(env._env, cpath, C.MDBX_env_flags_t(NoTLS|flags), C.mdbx_mode_t(mode))
	return operrno("mdbx_env_open", ret)
}
func (env *Env) Label() Label { return env.label }

// SetStrictThreadMode in this mode mdbx panics when tx opening and closing are happening in different threads
func (env *Env) SetStrictThreadMode(mode bool) {
	env.strictThreadCheck = mode
}

var errNotOpen = errors.New("enivornment is not open")

/* TODO: fix error: cannot convert *mf (variable of type _Ctype_HANDLE) to type uintptr

// FD returns the open file descriptor (or Windows file handle) for the given
// environment.  An error is returned if the environment has not been
// successfully Opened (where C API just retruns an invalid handle).
//
// See mdbx_env_get_fd.
func (env *Env) FD() (uintptr, error) {
	// fdInvalid is the value -1 as a uintptr, which is used by MDBX in the
	// case that env has not been opened yet.  the strange construction is done
	// to avoid constant value overflow errors at compile time.
	const fdInvalid = ^uintptr(0)

	mf := new(C.mdbx_filehandle_t)
	ret := C.mdbx_env_get_fd(env._env, mf)
	err := operrno("mdbx_env_get_fd", ret)
	if err != nil {
		return 0, err
	}
	fd := uintptr(*mf)

	if fd == fdInvalid {
		return 0, errNotOpen
	}
	return fd, nil
}
*/

// ReaderList dumps the contents of the reader lock table as text.  Readers
// start on the second line as space-delimited fields described by the first
// line.
//
// See mdbx_reader_list.
// func (env *Env) ReaderList(fn func(string) error) error {
//	ctx, done := newMsgFunc(fn)
//	defer done()
//	if fn == nil {
//		ctx = 0
//	}
//
//	ret := C.mdbxgo_reader_list(env._env, C.size_t(ctx))
//	if ret >= 0 {
//		return nil
//	}
//	if ret < 0 && ctx != 0 {
//		err := ctx.get().err
//		if err != nil {
//			return err
//		}
//	}
//	return operrno("mdbx_reader_list", ret)
//}

// ReaderCheck clears stale entries from the reader lock table and returns the
// number of entries cleared.
//
// See mdbx_reader_check()
func (env *Env) ReaderCheck() (int, error) {
	var _dead C.int
	ret := C.mdbx_reader_check(env._env, &_dead)
	return int(_dead), operrno("mdbx_reader_check", ret)
}

// Close shuts down the environment, releases the memory map, and clears the
// finalizer on env.
//
// See mdbx_env_close.
func (env *Env) Close() {
	if env._env == nil {
		return
	}

	env.closeLock.Lock()
	C.mdbx_env_close(env._env)
	env._env = nil
	env.closeLock.Unlock()
}

// CopyFD copies env to the the file descriptor fd.
//
// See mdbx_env_copyfd.
// func (env *Env) CopyFD(fd uintptr) error {
//	ret := C.mdbx_env_copyfd(env._env, C.mdbx_filehandle_t(fd))
//	return operrno("mdbx_env_copyfd", ret)
//}

// CopyFDFlag copies env to the file descriptor fd, with options.
//
// See mdbx_env_copyfd2.
// func (env *Env) CopyFDFlag(fd uintptr, flags uint) error {
//	ret := C.mdbx_env_copyfd2(env._env, C.mdbx_filehandle_t(fd), C.uint(flags))
//	return operrno("mdbx_env_copyfd2", ret)
//}

// Copy copies the data in env to an environment at path.
//
// See mdbx_env_copy.
// func (env *Env) Copy(path string) error {
//	cpath := C.CString(path)
//	defer C.free(unsafe.Pointer(cpath))
//	ret := C.mdbx_env_copy(env._env, cpath)
//	return operrno("mdbx_env_copy", ret)
//}

// CopyFlag copies the data in env to an environment at path created with flags.
//
// See mdbx_env_copy2.
// func (env *Env) CopyFlag(path string, flags uint) error {
//	cpath := C.CString(path)
//	defer C.free(unsafe.Pointer(cpath))
//	ret := C.mdbx_env_copy2(env._env, cpath, C.uint(flags))
//	return operrno("mdbx_env_copy2", ret)
//}

// Stat contains database status information.
//
// See MDBX_stat.
type Stat struct {
	PSize         uint   // Size of a database page. This is currently the same for all databases.
	Depth         uint   // Depth (height) of the B-tree
	BranchPages   uint64 // Number of internal (non-leaf) pages
	LeafPages     uint64 // Number of leaf pages
	OverflowPages uint64 // Number of overflow pages
	Entries       uint64 // Number of data items
	LastTxId      uint64 // Transaction ID of committed last modification
}

// Stat returns statistics about the environment.
//
// See mdbx_env_stat.
func (env *Env) Stat() (*Stat, error) {
	var _stat C.MDBX_stat
	var ret C.int = C.mdbx_env_stat_ex(env._env, nil, &_stat, C.size_t(unsafe.Sizeof(_stat)))
	if ret != success {
		return nil, operrno("mdbx_env_stat_ex", ret)
	}
	stat := Stat{PSize: uint(_stat.ms_psize),
		Depth:         uint(_stat.ms_depth),
		BranchPages:   uint64(_stat.ms_branch_pages),
		LeafPages:     uint64(_stat.ms_leaf_pages),
		OverflowPages: uint64(_stat.ms_overflow_pages),
		Entries:       uint64(_stat.ms_entries),
		LastTxId:      uint64(_stat.ms_mod_txnid)}
	return &stat, nil
}

type EnvInfoGeo struct {
	Lower   uint64
	Upper   uint64
	Current uint64
	Shrink  uint64
	Grow    uint64
}
type EnfInfoPageOps struct {
	Newly    uint64 /**< Quantity of a new pages added */
	Cow      uint64 /**< Quantity of pages copied for update */
	Clone    uint64 /**< Quantity of parent's dirty pages clones for nested transactions */
	Split    uint64 /**< Page splits */
	Merge    uint64 /**< Page merges */
	Spill    uint64 /**< Quantity of spilled dirty pages */
	Unspill  uint64 /**< Quantity of unspilled/reloaded pages */
	Wops     uint64 /**< Number of explicit write operations (not a pages) to a disk */
	Minicore uint64 /**< Number of mincore() calls */
	Prefault uint64 /**< Number of prefault write operations (not a pages) */
	Msync    uint64 /**< Number of explicit write operations (not a pages) to a disk */
	Fsync    uint64 /**< Number of explicit write operations (not a pages) to a disk */
}

// EnvInfo contains information an environment.
//
// See MDBX_envinfo.
type EnvInfo struct {
	MapSize int64 // Size of the data memory map
	LastPNO int64 // ID of the last used page
	Geo     EnvInfoGeo
	/** Statistics of page operations.
	 * \details Overall statistics of page operations of all (running, completed
	 * and aborted) transactions in the current multi-process session (since the
	 * first process opened the database). */
	PageOps           EnfInfoPageOps
	LastTxnID         int64         // ID of the last committed transaction
	MaxReaders        uint          // maximum number of threads for the environment
	NumReaders        uint          // maximum number of threads used in the environment
	PageSize          uint          //
	SystemPageSize    uint          //
	MiLastPgNo        uint64        //
	AutoSyncThreshold uint          //
	UnsyncedBytes     uint          // how many bytes have been committed but not flushed yet to disk
	SinceSync         time.Duration //
	AutosyncPeriod    time.Duration //
	SinceReaderCheck  time.Duration //
	Flags             uint          //
}

// Info returns information about the environment.
//
// See mdbx_env_info.
// txn - can be nil
func (env *Env) Info(txn *Txn) (*EnvInfo, error) {
	if txn == nil {
		var err error
		txn, err = env.BeginTxn(nil, Readonly)
		if err != nil {
			return nil, err
		}
		defer txn.Abort()
	}
	var _info C.MDBX_envinfo
	ret := C.mdbx_env_info_ex(env._env, txn._txn, &_info, C.size_t(unsafe.Sizeof(_info)))
	if ret != success {
		return nil, operrno("mdbx_env_info", ret)
	}
	return castEnvInfo(_info), nil
}

func PreOpenSnapInfo(path string) (*EnvInfo, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	var _info C.MDBX_envinfo
	var bytes C.size_t = C.size_t(unsafe.Sizeof(_info))
	ret := C.mdbx_preopen_snapinfo(cpath, &_info, bytes)
	if ret != success {
		return nil, operrno("mdbx_preopen_snapinfo", ret)
	}
	return castEnvInfo(_info), nil
}

func castEnvInfo(_info C.MDBX_envinfo) *EnvInfo {
	return &EnvInfo{
		MapSize: int64(_info.mi_mapsize),
		Geo: EnvInfoGeo{
			Lower:   uint64(_info.mi_geo.lower),
			Upper:   uint64(_info.mi_geo.upper),
			Current: uint64(_info.mi_geo.current),
			Shrink:  uint64(_info.mi_geo.shrink),
			Grow:    uint64(_info.mi_geo.grow),
		},
		PageOps: EnfInfoPageOps{
			Newly:    uint64(_info.mi_pgop_stat.newly),
			Cow:      uint64(_info.mi_pgop_stat.cow),
			Clone:    uint64(_info.mi_pgop_stat.clone),
			Split:    uint64(_info.mi_pgop_stat.split),
			Merge:    uint64(_info.mi_pgop_stat.merge),
			Spill:    uint64(_info.mi_pgop_stat.spill),
			Unspill:  uint64(_info.mi_pgop_stat.unspill),
			Wops:     uint64(_info.mi_pgop_stat.wops),
			Prefault: uint64(_info.mi_pgop_stat.prefault),
			Minicore: uint64(_info.mi_pgop_stat.mincore),
			Msync:    uint64(_info.mi_pgop_stat.msync),
			Fsync:    uint64(_info.mi_pgop_stat.fsync),
		},
		LastPNO:        int64(_info.mi_last_pgno),
		LastTxnID:      int64(_info.mi_recent_txnid),
		MaxReaders:     uint(_info.mi_maxreaders),
		NumReaders:     uint(_info.mi_numreaders),
		PageSize:       uint(_info.mi_dxb_pagesize),
		SystemPageSize: uint(_info.mi_sys_pagesize),
		MiLastPgNo:     uint64(_info.mi_last_pgno),

		AutoSyncThreshold: uint(_info.mi_autosync_threshold),
		UnsyncedBytes:     uint(_info.mi_unsync_volume),
		SinceSync:         toDuration(_info.mi_since_sync_seconds16dot16),
		AutosyncPeriod:    toDuration(_info.mi_autosync_period_seconds16dot16),
		SinceReaderCheck:  toDuration(_info.mi_since_reader_check_seconds16dot16),
		Flags:             uint(_info.mi_mode),
	}
}

// Sync flushes buffers to disk.  If force is true a synchronous flush occurs
// and ignores any UtterlyNoSync or MapAsync flag on the environment.
//
// See mdbx_env_sync.
func (env *Env) Sync(force bool, nonblock bool) error {
	ret := C.mdbx_env_sync_ex(env._env, C.bool(force), C.bool(nonblock))
	return operrno("mdbx_env_sync_ex", ret)
}

// SetFlags sets flags in the environment.
//
// See mdbx_env_set_flags.
func (env *Env) SetFlags(flags uint) error {
	ret := C.mdbx_env_set_flags(env._env, C.MDBX_env_flags_t(flags), true)
	return operrno("mdbx_env_set_flags", ret)
}

// UnsetFlags clears flags in the environment.
//
// See mdbx_env_set_flags.
func (env *Env) UnsetFlags(flags uint) error {
	ret := C.mdbx_env_set_flags(env._env, C.MDBX_env_flags_t(flags), false)
	return operrno("mdbx_env_set_flags", ret)
}

// Flags returns the flags set in the environment.
//
// See mdbx_env_get_flags.
func (env *Env) Flags() (uint, error) {
	var _flags C.uint
	ret := C.mdbx_env_get_flags(env._env, &_flags)
	if ret != success {
		return 0, operrno("mdbx_env_get_flags", ret)
	}
	return uint(_flags), nil
}

func (env *Env) SetDebug(logLvl LogLvl, dbg int, logger *C.MDBX_debug_func) error {
	_ = C.mdbx_setup_debug(logLvl, C.MDBX_debug_flags_t(dbg), logger)
	return nil
}

func (env *Env) SetOption(option uint, value uint64) error {
	ret := C.mdbx_env_set_option(env._env, C.MDBX_option_t(option), C.uint64_t(value))
	return operrno("mdbx_env_set_option", ret)
}

func (env *Env) GetOption(option uint) (uint64, error) {
	var res C.uint64_t
	ret := C.mdbx_env_get_option(env._env, C.MDBX_option_t(option), &res)
	return uint64(res), operrno("mdbx_env_get_option", ret)
}

func (env *Env) SetSyncPeriod(value time.Duration) error {
	ret := C.mdbx_env_set_syncperiod(env._env, C.uint(NewDuration16dot16(value)))
	return operrno("mdbx_env_set_syncperiod", ret)
}

func (env *Env) GetSyncPeriod() (time.Duration, error) {
	var res C.uint
	ret := C.mdbx_env_get_syncperiod(env._env, &res)
	return Duration16dot16(res).ToDuration(), operrno("mdbx_env_get_syncperiod", ret)
}

func (env *Env) SetSyncBytes(threshold uint) error {
	ret := C.mdbx_env_set_syncbytes(env._env, C.size_t(threshold))
	return operrno("mdbx_env_set_syncbytes", ret)

}

func (env *Env) GetSyncBytes() (uint, error) {
	var res C.size_t
	ret := C.mdbx_env_get_syncbytes(env._env, &res)
	return uint(res), operrno("mdbx_env_get_syncbytes", ret)

}

func (env *Env) SetGeometry(sizeLower int, sizeNow int, sizeUpper int, growthStep int, shrinkThreshold int, pageSize int) error {
	ret := C.mdbx_env_set_geometry(env._env,
		C.intptr_t(sizeLower),
		C.intptr_t(sizeNow),
		C.intptr_t(sizeUpper),
		C.intptr_t(growthStep),
		C.intptr_t(shrinkThreshold),
		C.intptr_t(pageSize))
	return operrno("mdbx_env_set_geometry", ret)
}

// MaxKeySize returns the maximum allowed length for a key.
//
// See mdbx_env_get_maxkeysize.
func (env *Env) MaxKeySize() int {
	if env == nil {
		return int(C.mdbx_env_get_maxkeysize_ex(nil, 0))
	}
	return int(C.mdbx_env_get_maxkeysize_ex(env._env, 0))
}

// BeginTxn is an unsafe, low-level method to initialize a new transaction on
// env.  The Txn returned by BeginTxn is unmanaged and must be terminated by
// calling either its Abort or Commit methods to ensure that its resources are
// released.
//
// BeginTxn does not call runtime.LockOSThread.  Unless the Readonly flag is
// passed goroutines must call runtime.LockOSThread before calling BeginTxn and
// the returned Txn must not have its methods called from another goroutine.
// Failure to meet these restrictions can have undefined results that may
// include deadlocking your application.
//
// Instead of calling BeginTxn users should prefer calling the View and Update
// methods, which assist in management of Txn objects and provide OS thread
// locking required for write transactions.
//
// Unterminated transactions can adversly effect
// database performance and cause the database to grow until the map is full.
//
// See mdbx_txn_begin.
func (env *Env) BeginTxn(parent *Txn, flags uint) (*Txn, error) {
	return beginTxn(env, parent, flags)
}

// RunTxn creates a new Txn and calls fn with it as an argument.  Run commits
// the transaction if fn returns nil otherwise the transaction is aborted.
// Because RunTxn terminates the transaction goroutines should not retain
// references to it or its data after fn returns.
//
// RunTxn does not call runtime.LockOSThread.  Unless the Readonly flag is
// passed the calling goroutine should ensure it is locked to its thread and
// any goroutines started by fn must not call methods on the Txn object it is
// passed.
//
// See mdbx_txn_begin.
func (env *Env) RunTxn(flags uint, fn TxnOp) error {
	return env.run(flags, fn)
}

// View creates a readonly transaction with a consistent view of the
// environment and passes it to fn.  View terminates its transaction after fn
// returns.  Any error encountered by View is returned.
//
// Unlike with Update transactions, goroutines created by fn are free to call
// methods on the Txn passed to fn provided they are synchronized in their
// accesses (e.g. using a mutex or channel).
//
// Any call to Commit, Abort, Reset or Renew on a Txn created by View will
// panic.
func (env *Env) View(fn TxnOp) error {
	return env.run(Readonly, fn)
}

// Update calls fn with a writable transaction.  Update commits the transaction
// if fn returns a nil error otherwise Update aborts the transaction and
// returns the error.
//
// Update calls runtime.LockOSThread to lock the calling goroutine to its
// thread and until fn returns and the transaction has been terminated, at
// which point runtime.UnlockOSThread is called.  If the calling goroutine is
// already known to be locked to a thread, use UpdateLocked instead to avoid
// premature unlocking of the goroutine.
//
// Neither Update nor UpdateLocked cannot be called safely from a goroutine
// where it isn't known if runtime.LockOSThread has been called.  In such
// situations writes must either be done in a newly created goroutine which can
// be safely locked, or through a worker goroutine that accepts updates to
// apply and delivers transaction results using channels.  See the package
// documentation and examples for more details.
//
// Goroutines created by the operation fn must not use methods on the Txn
// object that fn is passed.  Doing so would have undefined and unpredictable
// results for your program (likely including data loss, deadlock, etc).
//
// Any call to Commit, Abort, Reset or Renew on a Txn created by Update will
// panic.
func (env *Env) Update(fn TxnOp) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	return env.run(0, fn)
}

// UpdateLocked behaves like Update but does not lock the calling goroutine to
// its thread.  UpdateLocked should be used if the calling goroutine is already
// locked to its thread for another purpose.
//
// Neither Update nor UpdateLocked cannot be called safely from a goroutine
// where it isn't known if runtime.LockOSThread has been called.  In such
// situations writes must either be done in a newly created goroutine which can
// be safely locked, or through a worker goroutine that accepts updates to
// apply and delivers transaction results using channels.  See the package
// documentation and examples for more details.
//
// Goroutines created by the operation fn must not use methods on the Txn
// object that fn is passed.  Doing so would have undefined and unpredictable
// results for your program (likely including data loss, deadlock, etc).
//
// Any call to Commit, Abort, Reset or Renew on a Txn created by UpdateLocked
// will panic.
func (env *Env) UpdateLocked(fn TxnOp) error {
	return env.run(0, fn)
}

func (env *Env) run(flags uint, fn TxnOp) error {
	txn, err := beginTxn(env, nil, flags)
	if err != nil {
		return err
	}
	return txn.runOpTerm(fn)
}

// CloseDBI closes the database handle, db.  Normally calling CloseDBI
// explicitly is not necessary.
//
// It is the caller's responsibility to serialize calls to CloseDBI.
//
// See mdbx_dbi_close.
func (env *Env) CloseDBI(db DBI) {
	C.mdbx_dbi_close(env._env, C.MDBX_dbi(db))
}

func (env *Env) CHandle() unsafe.Pointer {
	return unsafe.Pointer(env._env)
}

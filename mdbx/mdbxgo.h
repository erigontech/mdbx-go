/* lmdbgo.h
 * Helper utilities for github.com/bmatsuo/lmdb-go/lmdb.  These functions have
 * no compatibility guarantees and may be modified or deleted without warning.
 * */
#ifndef _MDBXGO_H_
#define _MDBXGO_H_

#include "../libmdbx/mdbx.h"

/* Proxy functions for lmdb get/put operations. The functions are defined to
 * take char* values instead of void* to keep cgo from cheking their data for
 * nested pointers and causing a couple of allocations per argument.
 *
 * For more information see github issues for more information about the
 * problem and the decision.
 *      https://github.com/golang/go/issues/14387
 *      https://github.com/golang/go/issues/15048
 *      https://github.com/bmatsuo/lmdb-go/issues/63
 * */
int mdbxgo_del(MDBX_txn *txn, MDBX_dbi dbi, char *kdata, size_t kn, char *vdata, size_t vn);
int mdbxgo_get(MDBX_txn *txn, MDBX_dbi dbi, char *kdata, size_t kn, MDBX_val *val);
int mdbxgo_put1(MDBX_txn *txn, MDBX_dbi dbi, char *kdata, size_t kn, MDBX_val *val, MDBX_put_flags_t flags);
int mdbxgo_put2(MDBX_txn *txn, MDBX_dbi dbi, char *kdata, size_t kn, char *vdata, size_t vn, MDBX_put_flags_t flags);
int mdbxgo_cursor_put1(MDBX_cursor *cur, char *kdata, size_t kn, MDBX_val *val, MDBX_put_flags_t flags);
int mdbxgo_cursor_put2(MDBX_cursor *cur, char *kdata, size_t kn, char *vdata, size_t vn, MDBX_put_flags_t flags);
int mdbxgo_cursor_putmulti(MDBX_cursor *cur, char *kdata, size_t kn, char *vdata, size_t vn, size_t vstride, MDBX_put_flags_t flags);
/* ConstCString wraps a null-terminated (const char *) because Go's type system
 * does not represent the 'cosnt' qualifier directly on a function argument and
 * causes warnings to be emitted during linking.
 * */
typedef struct { const char *p; } mdbxgo_ConstCString;

/* mdbxgo_reader_list is a proxy for mdbx_reader_list that uses a special
 * callback proxy function to relay structured reader records over the
 * mdbxgoMDBReaderListBridge external Go func.
 * */
int mdbxgo_reader_list(MDBX_env *env, size_t ctx);
uint64_t mdbxgo_tid_to_u64(mdbx_tid_t tid);
uint64_t mdbxgo_tid_txn_parked(void);
uint64_t mdbxgo_tid_txn_ousted(void);

int mdbxgo_cmp(MDBX_txn *txn, MDBX_dbi dbi, char *adata, size_t an, char *bdata, size_t bn);
int mdbxgo_dcmp(MDBX_txn *txn, MDBX_dbi dbi, char *adata, size_t an, char *bdata, size_t bn);

/* The mdbxgo_*_result structs bundle an error code with a scalar return value
 * so that helper functions can return both without taking a Go pointer as an
 * out-parameter (which would escape the Go variable to the heap). */
typedef struct { int err; size_t    val; } mdbxgo_size_result;
typedef struct { int err; uint64_t  val; } mdbxgo_u64_result;
typedef struct { int err; unsigned  val; } mdbxgo_uint_result;
typedef struct { int err; int       val; } mdbxgo_int_result;
typedef struct { int err; ptrdiff_t val; } mdbxgo_ptrdiff_result;
typedef struct { int err; intptr_t pageSize, totalPages, availPages; } mdbxgo_sysraminfo_result;

mdbxgo_size_result       mdbxgo_cursor_count(MDBX_cursor *cur);
mdbxgo_u64_result        mdbxgo_cursor_bunch_delete(MDBX_cursor *cur, MDBX_bunch_action_t mode);
mdbxgo_u64_result        mdbxgo_cursor_delete_range(MDBX_cursor *begin, MDBX_cursor *end, bool end_including);
mdbxgo_ptrdiff_result    mdbxgo_estimate_distance(const MDBX_cursor *first, const MDBX_cursor *last);
mdbxgo_ptrdiff_result    mdbxgo_cursor_distance(const MDBX_cursor *first, const MDBX_cursor *last, unsigned deepness);
mdbxgo_ptrdiff_result    mdbxgo_estimate_move(MDBX_cursor *cur, char *kdata, size_t kn, char *vdata, size_t vn, MDBX_cursor_op move_op);
mdbxgo_ptrdiff_result    mdbxgo_estimate_range(MDBX_txn *txn, MDBX_dbi dbi, char *begin_kdata, size_t begin_kn, char *begin_vdata, size_t begin_vn, char *end_kdata, size_t end_kn, char *end_vdata, size_t end_vn);
mdbxgo_u64_result        mdbxgo_dbi_sequence(MDBX_txn *txn, MDBX_dbi dbi, uint64_t increment);
mdbxgo_u64_result        mdbxgo_env_get_option(MDBX_env *env, MDBX_option_t option);
mdbxgo_uint_result       mdbxgo_env_get_syncperiod(MDBX_env *env);
mdbxgo_size_result       mdbxgo_env_get_syncbytes(MDBX_env *env);
mdbxgo_int_result        mdbxgo_reader_check(MDBX_env *env);
mdbxgo_uint_result       mdbxgo_env_get_flags(MDBX_env *env);
mdbxgo_uint_result       mdbxgo_dbi_flags(MDBX_txn *txn, MDBX_dbi dbi);
mdbxgo_sysraminfo_result mdbxgo_get_sysraminfo(void);
mdbxgo_uint_result       mdbxgo_dbi_open(MDBX_txn *txn, const char *name, MDBX_db_flags_t flags);
mdbxgo_uint_result       mdbxgo_dbi_open_ex(MDBX_txn *txn, const char *name, MDBX_db_flags_t flags, MDBX_cmp_func *cmp, MDBX_cmp_func *dcmp);

typedef struct { int err; MDBX_commit_latency lat; } mdbxgo_commit_result;
mdbxgo_commit_result     mdbxgo_txn_commit_ex(MDBX_txn *txn);
mdbxgo_commit_result     mdbxgo_txn_checkpoint(MDBX_txn *txn, MDBX_txn_flags_t weakening);
mdbxgo_commit_result     mdbxgo_txn_commit_embark_read(MDBX_txn **ptxn);

typedef struct { int err; char *kbase; size_t klen; char *vbase; size_t vlen; } mdbxgo_val_result;
mdbxgo_val_result        mdbxgo_cursor_get_empty(MDBX_cursor *cur, MDBX_cursor_op op);
mdbxgo_val_result        mdbxgo_cursor_get_val(MDBX_cursor *cur, char *kdata, size_t kn, char *vdata, size_t vn, MDBX_cursor_op op);

typedef struct {
    int err;
    uint64_t pages_allocated;
    uint64_t pages_backed;
    uint64_t pages_total;
    uint64_t pages_gc;
    uint64_t pages_reclaimable;
    uint64_t pages_retained;
    uint64_t max_reader_lag;
    uint64_t max_retained_pages;
} mdbxgo_gc_info_result;
mdbxgo_gc_info_result    mdbxgo_gc_info(MDBX_txn *txn);

/* mdbxgo_defrag_result is a flat, Go-friendly view of MDBX_defrag_result_t.
 * Wide types (intptr_t, size_t, mdbx_tid_t, mdbx_pid_t) are normalized to
 * int64/uint64 to avoid cgo size mismatches across platforms. */
typedef struct {
    int err;
    int64_t  pages_shrinked;
    uint64_t pages_moved;
    uint64_t pages_scheduled;
    uint64_t pages_retained;
    uint64_t pages_left;
    uint64_t pages_whole;
    uint64_t obstructed_pgno;
    uint64_t obstructed_span;
    uint64_t obstructed_txnid;
    uint64_t obstructor_tid;
    int64_t  obstructor_pid;
    unsigned rough_estimation_cycle_progress_permille;
    unsigned cycles;
    unsigned stopping_reasons;
    uint64_t spent_time_dot16;
} mdbxgo_defrag_result;

/* mdbxgo_env_copy2fd wraps mdbx_env_copy2fd so callers can pass the file
 * descriptor as a uintptr regardless of platform. On POSIX mdbx_filehandle_t
 * is int, on Windows it is HANDLE (a void*); doing the cast in C keeps the
 * Go side identical on every target. */
int mdbxgo_env_copy2fd(MDBX_env *env, uintptr_t fd, MDBX_copy_flags_t flags);

mdbxgo_defrag_result mdbxgo_env_defrag(MDBX_env *env,
                                       size_t defrag_atleast,
                                       size_t time_atleast_dot16,
                                       size_t defrag_enough,
                                       size_t time_limit_dot16,
                                       intptr_t acceptable_backlash,
                                       intptr_t preferred_batch);

#ifndef _WIN32
mdbxgo_int_result        mdbxgo_env_get_fd(MDBX_env *env);
#endif

#endif

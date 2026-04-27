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
typedef struct { int err; size_t   val; } mdbxgo_size_result;
typedef struct { int err; uint64_t val; } mdbxgo_u64_result;
typedef struct { int err; unsigned val; } mdbxgo_uint_result;
typedef struct { int err; int      val; } mdbxgo_int_result;
typedef struct { int err; intptr_t pageSize, totalPages, availPages; } mdbxgo_sysraminfo_result;

mdbxgo_size_result       mdbxgo_cursor_count(MDBX_cursor *cur);
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

#ifndef _WIN32
mdbxgo_int_result        mdbxgo_env_get_fd(MDBX_env *env);
#endif

#endif

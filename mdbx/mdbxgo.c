/* lmdbgo.c
 * Helper utilities for github.com/bmatsuo/lmdb-go/lmdb
 * */
#include <string.h>
#include <stdio.h>
#include "_cgo_export.h"
#include "mdbxgo.h"

#define MDBXGO_SET_VAL(val, size, data) \
    *(val) = (MDBX_val){ .iov_len = (size), .iov_base = (data) }

#define MDBXGO_SET_VAL_RESULT(r, key, val) \
    do { \
        (r).kbase = (key).iov_base; (r).klen = (key).iov_len; \
        (r).vbase = (val).iov_base; (r).vlen = (val).iov_len; \
    } while (0)

int mdbxgo_msg_func_proxy(const char *msg, void *ctx) {
    //  wrap msg and call the bridge function exported from lmdb.go.
    mdbxgo_ConstCString s;
    s.p = msg;
    return mdbxgoMDBMsgFuncBridge(s, (size_t)ctx);
}

// int mdbxgo_reader_list(MDBX_env *env, size_t ctx) {
//     // list readers using a static proxy function that does dynamic dispatch on
//     // ctx.
//	if (ctx)
//		return mdbx_reader_list(env, &mdbxgo_msg_func_proxy, (void *)ctx);
//	return mdbx_reader_list(env, 0, (void *)ctx);
// }

int mdbxgo_del(MDBX_txn *txn, MDBX_dbi dbi, char *kdata, size_t kn, char *vdata, size_t vn) {
    MDBX_val key, val;
    MDBXGO_SET_VAL(&key, kn, kdata);
    if (vdata) {
        MDBXGO_SET_VAL(&val, vn, vdata);
        return mdbx_del(txn, dbi, &key, &val);
    }
    return mdbx_del(txn, dbi, &key, NULL);
}

int mdbxgo_get(MDBX_txn *txn, MDBX_dbi dbi, char *kdata, size_t kn, MDBX_val *val) {
    MDBX_val key;
    MDBXGO_SET_VAL(&key, kn, kdata);
    return mdbx_get(txn, dbi, &key, val);
}

int mdbxgo_put2(MDBX_txn *txn, MDBX_dbi dbi, char *kdata, size_t kn, char *vdata, size_t vn, MDBX_put_flags_t flags) {
    MDBX_val key, val;
    MDBXGO_SET_VAL(&key, kn, kdata);
    MDBXGO_SET_VAL(&val, vn, vdata);
    return mdbx_put(txn, dbi, &key, &val, flags);
}

int mdbxgo_put1(MDBX_txn *txn, MDBX_dbi dbi, char *kdata, size_t kn, MDBX_val *val, MDBX_put_flags_t flags) {
    MDBX_val key;
    MDBXGO_SET_VAL(&key, kn, kdata);
    return mdbx_put(txn, dbi, &key, val, flags);
}

int mdbxgo_cursor_put2(MDBX_cursor *cur, char *kdata, size_t kn, char *vdata, size_t vn, MDBX_put_flags_t flags) {
    MDBX_val key, val;
    MDBXGO_SET_VAL(&key, kn, kdata);
    MDBXGO_SET_VAL(&val, vn, vdata);
    return mdbx_cursor_put(cur, &key, &val, flags);
}

int mdbxgo_cursor_put1(MDBX_cursor *cur, char *kdata, size_t kn, MDBX_val *val, MDBX_put_flags_t flags) {
    MDBX_val key;
    MDBXGO_SET_VAL(&key, kn, kdata);
    return mdbx_cursor_put(cur, &key, val, flags);
}

int mdbxgo_cursor_putmulti(MDBX_cursor *cur, char *kdata, size_t kn, char *vdata, size_t vn, size_t vstride, MDBX_put_flags_t flags) {
    MDBX_val key, val[2];
    MDBXGO_SET_VAL(&key, kn, kdata);
    MDBXGO_SET_VAL(&(val[0]), vstride, vdata);
    MDBXGO_SET_VAL(&(val[1]), vn, 0);
    return mdbx_cursor_put(cur, &key, &val[0], flags);
}

mdbxgo_val_result mdbxgo_cursor_get_empty(MDBX_cursor *cur, MDBX_cursor_op op) {
    mdbxgo_val_result r = {0};
    MDBX_val key = {0}, val = {0};
    r.err = mdbx_cursor_get(cur, &key, &val, op);
    MDBXGO_SET_VAL_RESULT(r, key, val);
    return r;
}

mdbxgo_val_result mdbxgo_cursor_get_val(MDBX_cursor *cur, char *kdata, size_t kn, char *vdata, size_t vn, MDBX_cursor_op op) {
    mdbxgo_val_result r = {0};
    MDBX_val key = {0}, val = {0};
    MDBXGO_SET_VAL(&key, kn, kdata);
    MDBXGO_SET_VAL(&val, vn, vdata);
    r.err = mdbx_cursor_get(cur, &key, &val, op);
    MDBXGO_SET_VAL_RESULT(r, key, val);
    return r;
}

/* Compare two items lexically */
// static int __hot cmp_lexical(const MDBX_val *a, const MDBX_val *b) {
//   if (a->iov_len == b->iov_len)
//     return memcmp(a->iov_base, b->iov_base, a->iov_len);
//
//   const int diff_len = (a->iov_len < b->iov_len) ? -1 : 1;
//   const size_t shortest = (a->iov_len < b->iov_len) ? a->iov_len : b->iov_len;
//   int diff_data = memcmp(a->iov_base, b->iov_base, shortest);
//   return likely(diff_data) ? diff_data : diff_len;
// }

mdbxgo_size_result mdbxgo_cursor_count(MDBX_cursor *cur) {
    mdbxgo_size_result r = {0};
    r.err = mdbx_cursor_count(cur, &r.val);
    return r;
}

mdbxgo_u64_result mdbxgo_dbi_sequence(MDBX_txn *txn, MDBX_dbi dbi, uint64_t increment) {
    mdbxgo_u64_result r = {0};
    r.err = mdbx_dbi_sequence(txn, dbi, &r.val, increment);
    return r;
}

mdbxgo_u64_result mdbxgo_env_get_option(MDBX_env *env, MDBX_option_t option) {
    mdbxgo_u64_result r = {0};
    r.err = mdbx_env_get_option(env, option, &r.val);
    return r;
}

mdbxgo_uint_result mdbxgo_env_get_syncperiod(MDBX_env *env) {
    mdbxgo_uint_result r = {0};
    r.err = mdbx_env_get_syncperiod(env, &r.val);
    return r;
}

mdbxgo_size_result mdbxgo_env_get_syncbytes(MDBX_env *env) {
    mdbxgo_size_result r = {0};
    r.err = mdbx_env_get_syncbytes(env, &r.val);
    return r;
}

mdbxgo_int_result mdbxgo_reader_check(MDBX_env *env) {
    mdbxgo_int_result r = {0};
    r.err = mdbx_reader_check(env, &r.val);
    return r;
}

mdbxgo_uint_result mdbxgo_env_get_flags(MDBX_env *env) {
    mdbxgo_uint_result r = {0};
    r.err = mdbx_env_get_flags(env, &r.val);
    return r;
}

mdbxgo_uint_result mdbxgo_dbi_flags(MDBX_txn *txn, MDBX_dbi dbi) {
    mdbxgo_uint_result r = {0};
    r.err = mdbx_dbi_flags(txn, dbi, &r.val);
    return r;
}

mdbxgo_sysraminfo_result mdbxgo_get_sysraminfo(void) {
    mdbxgo_sysraminfo_result r = {0};
    r.err = mdbx_get_sysraminfo(&r.pageSize, &r.totalPages, &r.availPages);
    return r;
}

mdbxgo_uint_result mdbxgo_dbi_open(MDBX_txn *txn, const char *name, MDBX_db_flags_t flags) {
    mdbxgo_uint_result r = {0};
    MDBX_dbi dbi = 0;
    r.err = mdbx_dbi_open(txn, name, flags, &dbi);
    if (r.err == MDBX_SUCCESS) {
        r.val = dbi;
    }
    return r;
}

mdbxgo_uint_result mdbxgo_dbi_open_ex(MDBX_txn *txn, const char *name, MDBX_db_flags_t flags, MDBX_cmp_func *cmp, MDBX_cmp_func *dcmp) {
    mdbxgo_uint_result r = {0};
    MDBX_dbi dbi = 0;
    r.err = mdbx_dbi_open_ex(txn, name, flags, &dbi, cmp, dcmp);
    if (r.err == MDBX_SUCCESS) {
        r.val = dbi;
    }
    return r;
}

mdbxgo_commit_result mdbxgo_txn_commit_ex(MDBX_txn *txn) {
    mdbxgo_commit_result r = {0};
    r.err = mdbx_txn_commit_ex(txn, &r.lat);
    return r;
}

#ifndef _WIN32
mdbxgo_int_result mdbxgo_env_get_fd(MDBX_env *env) {
    mdbxgo_int_result r = {0};
    r.err = mdbx_env_get_fd(env, &r.val);
    return r;
}
#endif
int mdbxgo_cmp(MDBX_txn *txn, MDBX_dbi dbi, char *adata, size_t an, char *bdata, size_t bn) {
    MDBX_val a;
    MDBXGO_SET_VAL(&a, an, adata);
    MDBX_val b;
    MDBXGO_SET_VAL(&b, bn, bdata);
    return mdbx_cmp(txn, dbi, &a, &b);
}

int mdbxgo_dcmp(MDBX_txn *txn, MDBX_dbi dbi, char *adata, size_t an, char *bdata, size_t bn) {
    MDBX_val a;
    MDBXGO_SET_VAL(&a, an, adata);
    MDBX_val b;
    MDBXGO_SET_VAL(&b, bn, bdata);
    return mdbx_dcmp(txn, dbi, &a, &b);
}

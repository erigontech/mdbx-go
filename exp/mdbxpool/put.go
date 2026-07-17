//go:build !race

package mdbxpool

// In general we want Txn objects to be returned to the sync.Pool.  But the
// default behavior of Pool.Put under race detection is to drop everything on
// the floor; dropped Txns are never terminated and benchmarks issuing
// repeated reads would quickly blow the environment's reader limit.  So under
// race detection Txns are aborted eagerly instead of pooled.
const returnTxnToPool = true

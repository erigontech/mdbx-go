//go:build !race

package mdbxpool

// Normal (non-race) configuration: Txn objects are returned to the sync.Pool
// for reuse.  Under -race, putrace.go flips this constant to false: there
// Pool.Put drops everything on the floor, dropped Txns are never terminated,
// and repeated reads would blow the environment's reader limit, so Txns are
// aborted eagerly instead of pooled.
const returnTxnToPool = true

//go:build !race

package mdbxpool

// Normal (non-race) configuration: Txn objects are returned to the sync.Pool
// for reuse.  putrace.go flips this constant to false under -race, where
// Pool.Put randomly drops objects on the floor (and pre-go1.13 race builds
// never reused them at all): a dropped Txn was Reset but never aborted, so
// its C handle and reader-table slot leak, and enough drops exhaust the
// environment's reader limit.  With the constant false the pool aborts Txns
// eagerly instead of pooling them.
const returnTxnToPool = true

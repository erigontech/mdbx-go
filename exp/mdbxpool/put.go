//go:build !race

package mdbxpool

// Normal (non-race) configuration: Txn objects are returned to the sync.Pool
// for reuse.  putrace.go flips this constant to false under -race, where
// Pool.Put randomly drops most objects on the floor: pooled Txns would be
// lost without ever being terminated and repeated reads would blow the
// environment's reader limit.  With the constant false the pool aborts Txns
// eagerly instead of pooling them.
const returnTxnToPool = true

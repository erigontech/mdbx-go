//go:build !race

package mdbxpool

// Normal (non-race) configuration: Txn objects are returned to the sync.Pool
// for reuse.  Under -race, putrace.go flips this constant to false: there
// Pool.Put randomly drops most objects on the floor, so pooled Txns would be
// lost without ever being terminated and repeated reads would blow the
// environment's reader limit.  With the constant false the pool aborts Txns
// eagerly instead of pooling them.
const returnTxnToPool = true

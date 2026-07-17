//go:build race

package mdbxpool

// transactions abort immediately instead of "being put in the pool" when race
// detection is enabled to prevent benchmarks with race enabled from forcing
// applications to allow ridiculously large maximum numbers of readers.
//
// Under race detection sync.Pool.Put randomly drops objects on the floor
// (older Go versions never reused them at all), so pooling cannot be relied
// on and requires this bypass, unfortunately.
const returnTxnToPool = false

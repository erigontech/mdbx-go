/*
Package mdbxpool reuses read-only mdbx transactions across goroutines.

Beginning a read-only transaction costs a reader-table registration and two
Go allocations. A Pool keeps finished transactions on a bounded free list and
hands them back out, replacing that work with a reset/renew pair:

	p := mdbxpool.New(env)
	defer p.Close()

	err := p.View(func(txn *mdbx.Txn) error {
		v, err := txn.Get(dbi, key)
		...
	})

# What pooling is and is not worth

Measured on an Apple M4 Max against libmdbx 0.14.3, one small read per
iteration, by BenchmarkPool:

	              serial     16 goroutines
	env.View      293.2n     445.9n
	Pool.View     187.2n     448.1n
	per read      88 B, 2 allocs  ->  8 B, 1 alloc

Skipping the reader-table registration is worth about a third of a small read
serially. Under 16-way read concurrency with no writer it is a wash on time:
BeginTxn is cheap when nothing contends with it, and the free list's mutex
costs about what it saves. The allocation win holds throughout.

Where reuse pays off again is under write pressure, because a writer makes
beginning a transaction more expensive while leaving renewal largely alone.
With a writer committing continuously, recycling beats begin/abort by 17%
(BenchmarkStrategies, below).

So: worth it for serial or bursty readers, for read-heavy work alongside a
busy writer, and anywhere the per-read allocation matters. Not worth much for
saturated read concurrency against an idle database -- though it does no harm
there either.

# Idle transactions do not pin snapshots

A running read transaction holds an MVCC snapshot, and every page superseded
since that snapshot began is unreclaimable until it ends. An idle pooled
reader that kept its snapshot would therefore make writes allocate new pages
instead of recycling old ones, and the database would grow.

Pool.Put calls Txn.Reset before returning a transaction to the free list.
Reset releases the snapshot unconditionally while keeping the reader-table
slot and the handle's memory, which is what makes renewal cheap. Idle pooled
readers consequently retain nothing: measured with Env.ReaderStats over six
rounds of rewriting a 2048-key table, a pooled transaction retained 0 bytes
where a live reader over the same churn retained about 21 MB
(TestPoolIdleTransactionsRetainNoPages).

This is why the pool needs no tracking of commit ids, and no background
sweeps of the free list, to stay safe against database bloat.

# Why not Park or Refresh

libmdbx 0.13 added mdbx_txn_park, and 0.14 added mdbx_txn_refresh. Neither
one improves on Reset/Renew for pooling, so the pool does not use them
(BenchmarkStrategies):

	                        serial      16 readers + a writer
	BeginAbort              246.0n      563.4n
	ResetRenew  (Pool)      179.6n      466.6n
	ResetRefresh            178.6n      463.1n
	ParkUnpark              161.4n      483.5n
	ParkUnpark + Refresh    184.2n      509.0n

Park keeps the snapshot but lets a writer oust the reader if it obstructs
garbage recycling, so it releases pages only under writer pressure, whereas
Reset releases them immediately. Its serial edge comes from handing back a
stale snapshot: a parked transaction that was not ousted resumes on the view
it had when it was parked, which is rarely what a pooled reader wants.
Advancing it to the tip costs a Refresh, and that combination is the slowest
of the four. Under a concurrent writer -- the workload a pool exists for --
ousting makes plain Park/Unpark slower than Reset/Renew as well.

Park's real use is a long-lived reader that must keep one snapshot across
idle gaps, for example a consistent backup or a long scan. That is not what a
pool does. BenchmarkStrategies reproduces both tables, so this choice can be
rechecked against future libmdbx releases rather than taken on faith.

# Reader slots

Every live or idle transaction occupies one slot in the reader lock table,
whose size is fixed at MDBX_opt_max_readers. A Pool holds at most MaxIdle
idle transactions -- Put aborts rather than growing past that -- so a pool
adds no more than MaxIdle slots beyond the transactions its callers have
checked out. Sizing max_readers for peak read concurrency plus MaxIdle is
sufficient.

Pool.Close aborts every idle transaction, releasing their slots. This is
deterministic, unlike a sync.Pool, which cannot be drained: objects sitting
in per-P caches are unreachable to the draining goroutine, and a GC discards
them outright. Because mdbx installs no Txn finalizers, a discarded read-only
Txn leaks its reader slot and C memory for the lifetime of the Env.

# Concurrency

A Pool is safe for concurrent use. Transactions it returns are not: a
*mdbx.Txn checked out of the pool belongs to one goroutine until it is
returned, exactly as if that goroutine had called Env.BeginTxn itself.

The pool moves read-only transactions between goroutines and OS threads by
design, which is incompatible with Env.SetStrictThreadMode(true). Under that
mode Txn.Abort panics when it runs on a thread other than the one that began
the transaction, so Pool.Close panics as soon as it drains a transaction
begun elsewhere. Strict thread mode is a debugging aid and is off by default;
libmdbx itself permits read-only transactions to move between threads.
*/
package mdbxpool

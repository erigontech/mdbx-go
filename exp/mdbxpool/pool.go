package mdbxpool

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/erigontech/mdbx-go/mdbx"
)

// ErrClosed is returned by Get and View after Close.
var ErrClosed = errors.New("mdbxpool: pool is closed")

// Option configures a Pool. See New.
type Option func(*Pool)

// WithMaxIdle sets how many finished transactions the pool keeps for reuse.
// Put aborts a transaction rather than growing the free list beyond n, so n
// bounds how many reader-table slots the pool occupies on top of the
// transactions its callers hold. n <= 0 keeps none, which turns Get and Put
// into Env.BeginTxn and Txn.Abort.
//
// The default is runtime.GOMAXPROCS(0), on the assumption that reads run on
// at most that many goroutines at once.
func WithMaxIdle(n int) Option {
	return func(p *Pool) {
		p.maxIdle = max(n, 0)
	}
}

// Pool reuses read-only transactions on a bounded free list.
//
// The zero Pool is not usable; call New. A Pool is safe for concurrent use,
// but the transactions it hands out are not: each belongs to one goroutine
// until it is returned.
type Pool struct {
	env     *mdbx.Env
	maxIdle int

	mu     sync.Mutex
	idle   []*mdbx.Txn
	closed bool

	hits       atomic.Uint64
	misses     atomic.Uint64
	discards   atomic.Uint64
	renewFails atomic.Uint64
}

// New returns a Pool that draws read-only transactions from env.
//
// The pool does not take ownership of env. Close the pool before closing env,
// so that pooled transactions release their reader slots first.
func New(env *mdbx.Env, opts ...Option) *Pool {
	p := &Pool{
		env:     env,
		maxIdle: runtime.GOMAXPROCS(0),
	}
	for _, opt := range opts {
		opt(p)
	}
	// Allocated once at full capacity: Put never grows the free list past
	// maxIdle, so this is the only allocation it will ever need.
	p.idle = make([]*mdbx.Txn, 0, p.maxIdle)
	return p
}

// Get returns a read-only transaction viewing the most recent committed
// snapshot, reusing a pooled one when the free list is not empty.
//
// The caller must return it with Put, which is what makes reuse work; a
// transaction dropped instead of returned leaks its reader slot, since mdbx
// installs no Txn finalizers.
func (p *Pool) Get() (*mdbx.Txn, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrClosed
	}
	var txn *mdbx.Txn
	if n := len(p.idle); n > 0 {
		// Pop from the end: the most recently returned transaction is the
		// one whose pages are most likely still cached.
		txn, p.idle[n-1] = p.idle[n-1], nil
		p.idle = p.idle[:n-1]
	}
	p.mu.Unlock()

	if txn == nil {
		p.misses.Add(1)
		return p.env.BeginTxn(nil, mdbx.Readonly)
	}

	// Renew re-acquires a reader lock on the current snapshot, which is what
	// releases the pages the previous use held.
	if err := txn.Renew(); err != nil {
		// The handle is unusable, and Renew leaves nothing to salvage.
		// Discard it and begin a fresh transaction rather than failing the
		// caller for a pooling problem.
		txn.Abort()
		p.renewFails.Add(1)
		p.misses.Add(1)
		return p.env.BeginTxn(nil, mdbx.Readonly)
	}
	p.hits.Add(1)
	return txn, nil
}

// Put returns a transaction obtained from Get.
//
// The transaction is reset -- releasing its MVCC snapshot but keeping its
// reader slot -- and kept for reuse. It is aborted instead when the free list
// is full, when the pool is closed, or when the reset fails, since a
// transaction that failed to reset is still a live reader and pooling it
// would pin its snapshot for as long as it stayed idle.
//
// Put must only be passed transactions returned by Get. Passing nil is a
// no-op; passing a write transaction, or one begun elsewhere, has undefined
// results.
func (p *Pool) Put(txn *mdbx.Txn) {
	if txn == nil {
		return
	}

	// Reset outside the lock: it is a cgo call, and nothing about it depends
	// on the free list.
	if err := txn.Reset(); err != nil {
		p.discards.Add(1)
		txn.Abort()
		return
	}

	p.mu.Lock()
	if !p.closed && len(p.idle) < p.maxIdle {
		p.idle = append(p.idle, txn)
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	p.discards.Add(1)
	txn.Abort()
}

// View runs fn inside a pooled read-only transaction and returns the transaction
// to the pool afterwards, whether fn returns an error or panics.
//
// fn must not call Commit, Abort, Reset or Renew on the transaction; those
// panic, as they do inside Env.View.
func (p *Pool) View(fn mdbx.TxnOp) error {
	txn, err := p.Get()
	if err != nil {
		return err
	}
	defer p.Put(txn)
	return txn.RunOp(fn, false)
}

// Close aborts every idle transaction, releasing their reader slots, and
// makes subsequent Get calls return ErrClosed. Transactions checked out when
// Close runs are unaffected; passing one to Put afterwards aborts it.
//
// Close is idempotent and always returns nil; the error is for future use and
// so that callers may defer it uniformly.
func (p *Pool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	idle := p.idle
	p.idle = nil
	p.mu.Unlock()

	for _, txn := range idle {
		txn.Abort()
	}
	return nil
}

// Stats reports a Pool's counters.
type Stats struct {
	// Idle is the number of transactions on the free list right now.
	Idle int
	// MaxIdle is the free list's capacity.
	MaxIdle int
	// Hits counts Get calls served from the free list.
	Hits uint64
	// Misses counts Get calls that had to begin a new transaction, including
	// those that fell back after a failed renewal.
	Misses uint64
	// Discards counts transactions Put aborted instead of pooling, because
	// the free list was full, the pool was closed, or the reset failed.
	Discards uint64
	// RenewFails counts pooled transactions Get discarded because renewing
	// them failed.
	RenewFails uint64
}

// Stats returns a snapshot of the pool's counters.
//
// For the reader lock table itself -- how many slots are in use, how much
// data readers retain -- use Env.ReaderStats.
func (p *Pool) Stats() Stats {
	p.mu.Lock()
	idle := len(p.idle)
	p.mu.Unlock()

	return Stats{
		Idle:       idle,
		MaxIdle:    p.maxIdle,
		Hits:       p.hits.Load(),
		Misses:     p.misses.Load(),
		Discards:   p.discards.Load(),
		RenewFails: p.renewFails.Load(),
	}
}

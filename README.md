# mdbx-go

Go bindings to the libmdbx: https://erthink.github.io/libmdbx/

Most of articles in internet about LMDB are applicable to MDBX. But mdbx has more features.

For deeper DB understanding please read through [mdbx.h](https://github.com/erthink/libmdbx/blob/master/mdbx.h)

## Min Requirements

C language Compilers compatible with GCC or CLANG (mingw 10 on windows)
Golang: 1.15

## Packages

Functionality is logically divided into several packages. Applications will usually need to import **mdbx** but may
import other packages on an as needed basis.

Packages in the `exp/` directory are not stable and may change without warning. That said, they are generally usable if
application dependencies are managed and pinned by tag/commit.

Developers concerned with package stability should consult the documentation.

#### mdbx [![GoDoc](https://godoc.org/github.com/torquem-ch/mdbx-go/mdbx?status.svg)](https://godoc.org/github.com/github.com/torquem-ch/mdbx-go/mdbx) [![stable](https://img.shields.io/badge/stability-stable-brightgreen.svg)](#user-content-versioning-and-stability)

```go
import "github.com/torquem-ch/mdbx-go/mdbx"
```

Core bindings allowing low-level access to MDBX.

#### exp/mdbxpool [![GoDoc](https://godoc.org/github.com/torquem-ch/mdbx-go/mdbx/exp/mdbxpool?status.svg)](https://godoc.org/github.com/torquem-ch/mdbx-go/mdbx/exp/mdbxpool) [![experimental](https://img.shields.io/badge/stability-experimental-red.svg)](#user-content-versioning-and-stability)

```go
import "github.com/torquem-ch/mdbx-go/exp/mdbxpool"
```

A utility package which facilitates reuse of mdbx.Txn objects using a sync.Pool. Naively storing mdbx.Txn objects in
sync.Pool can be troublesome. And the mdbxpool.TxnPool type has been defined as a complete pooling solution and as
reference for applications attempting to write their own pooling implementation.

The mdbxpool package is relatively new. But it has a lot of potential utility. And once the mdbxpool API has been ironed
out, and the implementation hardened through use by real applications it can be integrated directly into the mdbx
package for more transparent integration. Please test this package and provide feedback to speed this process up.

## Key Features

### Idiomatic API

API inspired by [BoltDB](https://github.com/boltdb/bolt) with automatic commit/rollback of transactions. The goal of
mdbx-go is to provide idiomatic database interactions without compromising the flexibility of the C API.

**NOTE:** While the mdbx package tries hard to make MDBX as easy to use as possible there are compromises, gotchas, and
caveats that application developers must be aware of when relying on MDBX to store their data. All users are encouraged
to fully read the [documentation](https://erthink.github.io/libmdbx/) so they are aware of these caveats. And even
better if read through [mdbx.h](https://github.com/erthink/libmdbx/blob/master/mdbx.h)

### High Performance notices

Applications with high performance requirements can opt-in to fast, zero-copy reads at the cost of runtime safety.
Zero-copy behavior is specified at the transaction level to reduce instrumentation overhead.

```
err := mdbx.View(func(txn *mdbx.Txn) error {
    // RawRead enables zero-copy behavior with some serious caveats.
    // Read the documentation carefully before using.
    txn.RawRead = true

    val, err := txn.Get(dbi, []byte("largevalue"), 0)
    // ...
})
```

Use NoReadahead if Data > RAM

### Advantages of BoltDB

- Nested databases allow for hierarchical data organization.

- Far more databases can be accessed concurrently.

- No `Bucket` object - means less allocations and higher performance

- Operating systems that do not support sparse files do not use up excessive space due to a large pre-allocation of file
  space.

- As a pure Go package bolt can be easily cross-compiled using the `go`
  toolchain and `GOOS`/`GOARCH` variables.

- Its simpler design and implementation in pure Go mean it is free of many caveats and gotchas which are present using
  the MDBX package. For more information about caveats with the MDBX package, consult its
  [documentation](https://erthink.github.io/libmdbx/) so they are aware of these caveats. And even better if read
  through [mdbx.h](https://github.com/erthink/libmdbx/blob/master/mdbx.h).

### Advantages of MDBX

- Keys can contain multiple values using the DupSort flag.

- Updates can have sub-updates for atomic batching of changes.

- Databases typically remain open for the application lifetime. This limits the number of concurrently accessible
  databases. But, this minimizes the overhead of database accesses and typically produces cleaner code than an
  equivalent BoltDB implementation.

- Significantly faster than BoltDB. The raw speed of MDBX easily surpasses BoltDB. Additionally, MDBX provides
  optimizations ranging from safe, feature-specific optimizations to generally unsafe, extremely situational ones.
  Applications are free to enable any optimizations that fit their data, access, and reliability models.

- MDBX allows multiple applications to access a database simultaneously. Updates from concurrent processes are
  synchronized using a database lock file.

- As a C library, applications in any language can interact with MDBX databases. Mission critical Go applications can
  use a database while Python scripts perform analysis on the side.

## Build

On FreeBSD 10, you must explicitly set `CC` (otherwise it will fail with a cryptic error), for example:

    CC=clang go test -v ./...

## Documentation

- Examples see in *_test.go files of this repo
- [The MDBX](https://erthink.github.io/libmdbx/) And even better if read
  through [mdbx.h](https://github.com/erthink/libmdbx/blob/master/mdbx.h).
- [godoc.org](https://godoc.org/github.com/torquem-ch/mdbx-go)
- [The LMDB](http://symas.com/lmdb/)

### Versioning and Stability

The mdbx-go project makes regular releases with IDs `X.Y.Z`. All packages outside of the `exp/` directory are considered
stable and adhere to the guidelines of [semantic versioning](http://semver.org/).

Experimental packages (those packages in `exp/`) are not required to adhere to semantic versioning. However packages
specifically declared to merely be
"unstable" can be relied on more for long term use with less concern.

The API of an unstable package may change in subtle ways between minor release versions. But deprecations will be
indicated at least one release in advance and all functionality will remain available through some method.

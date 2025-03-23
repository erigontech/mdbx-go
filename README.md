- [mdbx-go](#mdbx-go)
  - [Min Requirements](#min-requirements)
  - [Packages](#packages)
      - [mdbx  ](#mdbx--)
      - [exp/mdbxpool  ](#expmdbxpool--)
  - [Key Features](#key-features)
    - [Idiomatic API](#idiomatic-api)
    - [High-Performance notices](#high-performance-notices)
    - [Advantages of BoltDB](#advantages-of-boltdb)
    - [Advantages of LMDB over BoltDB](#advantages-of-lmdb-over-boltdb)
    - [Advantages of MDBX over LMDB](#advantages-of-mdbx-over-lmdb)
  - [Build](#build)
  - [Update C code](#update-c-code)
  - [Build binaries](#build-binaries)
  - [Documentation](#documentation)
    - [Versioning and Stability](#versioning-and-stability)

# mdbx-go

Go bindings to the libmdbx: <https://libmdbx.dqdkfa.ru>

**Notice**: page `./mdbx` contains only `mdbx.h` and `mdbx.c` - to minimize go build time/size.
But full version of libmdbx (produced by it's `make dist` command) is in `./../libmdbx/`.
License is also there.

Most of articles in internet about LMDB are applicable to MDBX. But mdbx has more features.

For deeper DB understanding please read through [mdbx.h](https://gitflic.ru/project/erthink/libmdbx/blob?file=mdbx.h)

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
to fully read the [documentation](https://libmdbx.dqdkfa.ru) so they are aware of these caveats. And even
better if read through [mdbx.h](https://gitflic.ru/project/erthink/libmdbx/blob?file=mdbx.h)

### High-Performance notices

Applications with high-performance requirements can opt-in to fast, zero-copy reads at the cost of runtime safety.
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
  [documentation](https://libmdbx.dqdkfa.ru) so they are aware of these caveats. And even better if read
  through [mdbx.h](https://gitflic.ru/project/erthink/libmdbx/blob?file=mdbx.h).

### Advantages of LMDB over BoltDB

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

### Advantages of MDBX over LMDB

See in mdbx's readme.md

## Build

On FreeBSD 10, you must explicitly set `CC` (otherwise it will fail with a cryptic error), for example:

    CC=clang go test -v ./...

## Update C code

In `libmdbx` repo: `make dist && cp -R ./dist/* ./../mdbx-go/libmdbx/`. Then in mdbx-go repo: `make cp`

On mac: 
```
brew install --default-names gnu-sed
PATH="/usr/local/opt/gnu-sed/libexec/gnubin:$PATH" make cp
```

## Build binaries

In mdbx-go repo: `MDBX_BUILD_TIMESTAMP=unknown make tools`

Or if use mdbx-go as a library:

```sh
go mod vendor && cd vendor/github.com/torquem-ch/mdbx-go && make tools 
rm -rf vendor
```

## Documentation

- Examples see in *_test.go files of this repo
- [The MDBX](https://libmdbx.dqdkfa.ru) And even better if read
  through [mdbx.h](https://gitflic.ru/project/erthink/libmdbx/blob?file=mdbx.h).
- [godoc.org](https://godoc.org/github.com/torquem-ch/mdbx-go)
- [The LMDB](http://symas.com/lmdb/)

### Versioning and Stability

The mdbx-go project makes regular releases with IDs `X.Y.Z`. All packages outside of the `exp/` directory are considered
stable and adhere to the guidelines of [semantic versioning](http://semver.org/).

Experimental packages (those packages in `exp/`) are not required to adhere to semantic versioning. However packages
specifically declared to merely be
"unstable" can be relied on more for long-term use with less concern.

The API of an unstable package may change in subtle ways between minor release versions. But deprecations will be
indicated at least one release in advance and all functionality will remain available through some method.

## Benchmark Notice
It's noticed that GODEBUG=cgocheck=0 significantly increase mdbx-go perfomance (but be aware of misuse, it's 
cgoCheckPointer disable, so of course it could be dangerous DIOR)
```shell
goos: darwin
goarch: arm64
pkg: github.com/erigontech/mdbx-go/mdbx
cpu: Apple M3 Max
                         │  master.txt   │        master_cgocheck0.txt         │
                         │    sec/op     │   sec/op     vs base                │
Cursor-16                   107.2n ±  0%   103.5n ± 1%   -3.40% (p=0.000 n=10)
Cursor_Renew/1-16           37.23n ±  2%   35.54n ± 1%   -4.54% (p=0.000 n=10)
Cursor_Renew/2-16           36.11n ±  2%   34.54n ± 0%   -4.36% (p=0.000 n=10)
Cursor_Renew/3-16           112.4n ±  2%   102.8n ± 0%   -8.54% (p=0.000 n=10)
Cursor_Renew/4-16           43.85n ±  2%   41.12n ± 1%   -6.21% (p=0.000 n=10)
Cursor_Set_OneKey-16        59.98n ±  1%   42.81n ± 1%  -28.63% (p=0.000 n=10)
Cursor_Set_Sequence-16     106.50n ±  0%   90.60n ± 1%  -14.93% (p=0.000 n=10)
Cursor_Set_Random-16        475.6n ±  2%   461.6n ± 8%   -2.95% (p=0.034 n=10)
Errno_Error-16              207.2n ±  2%   202.4n ± 0%   -2.32% (p=0.000 n=10)
Txn_abort-16                170.3n ±  0%   158.9n ± 1%   -6.75% (p=0.000 n=10)
Txn_commit-16               49.10µ ±  3%   51.04µ ± 9%        ~ (p=0.105 n=10)
Txn_ro-16                   207.3n ±  0%   196.8n ± 1%   -5.04% (p=0.000 n=10)
Txn_unmanaged_abort-16      164.6n ±  1%   152.3n ± 1%   -7.44% (p=0.000 n=10)
Txn_unmanaged_commit-16     164.3n ±  1%   152.1n ± 1%   -7.45% (p=0.000 n=10)
Txn_unmanaged_ro-16         156.1n ±  3%   147.0n ± 1%   -5.83% (p=0.000 n=10)
Txn_renew-16                85.16n ±  0%   80.72n ± 0%   -5.21% (p=0.000 n=10)
Txn_Put_append-16           195.7n ±  0%   200.0n ± 3%        ~ (p=0.159 n=10)
Txn_Put_append_noflag-16    226.8n ±  0%   225.9n ± 1%        ~ (p=0.092 n=10)
Txn_Get_OneKey-16           55.67n ±  0%   44.70n ± 1%  -19.70% (p=0.000 n=10)
Txn_Get_Sequence-16         149.1n ±  1%   135.1n ± 0%   -9.42% (p=0.000 n=10)
Txn_Get_Random-16           476.8n ± 11%   445.6n ± 1%   -6.53% (p=0.000 n=10)
geomean                     167.2n         155.3n        -7.12%

                         │  master.txt  │        master_cgocheck0.txt         │
                         │     B/op     │    B/op     vs base                 │
Cursor-16                  16.00 ± 0%     16.00 ± 0%       ~ (p=1.000 n=10) ¹
Cursor_Renew/1-16          0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Cursor_Renew/2-16          0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Cursor_Renew/3-16          16.00 ± 0%     16.00 ± 0%       ~ (p=1.000 n=10) ¹
Cursor_Renew/4-16          0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Cursor_Set_OneKey-16       0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Cursor_Set_Sequence-16     0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Cursor_Set_Random-16       0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Errno_Error-16             320.0 ± 0%     320.0 ± 0%       ~ (p=1.000 n=10) ¹
Txn_abort-16               80.00 ± 0%     80.00 ± 0%       ~ (p=1.000 n=10) ¹
Txn_commit-16              248.0 ± 0%     248.0 ± 0%       ~ (p=1.000 n=10) ¹
Txn_ro-16                  240.0 ± 0%     240.0 ± 0%       ~ (p=1.000 n=10) ¹
Txn_unmanaged_abort-16     80.00 ± 0%     80.00 ± 0%       ~ (p=1.000 n=10) ¹
Txn_unmanaged_commit-16    80.00 ± 0%     80.00 ± 0%       ~ (p=1.000 n=10) ¹
Txn_unmanaged_ro-16        80.00 ± 0%     80.00 ± 0%       ~ (p=1.000 n=10) ¹
Txn_renew-16               0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_Put_append-16          8.000 ± 0%     8.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_Put_append_noflag-16   0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_Get_OneKey-16          0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_Get_Sequence-16        0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_Get_Random-16          0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
geomean                               ²               +0.00%                ²
¹ all samples are equal
² summaries must be >0 to compute geomean

                         │  master.txt  │        master_cgocheck0.txt         │
                         │  allocs/op   │ allocs/op   vs base                 │
Cursor-16                  1.000 ± 0%     1.000 ± 0%       ~ (p=1.000 n=10) ¹
Cursor_Renew/1-16          0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Cursor_Renew/2-16          0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Cursor_Renew/3-16          1.000 ± 0%     1.000 ± 0%       ~ (p=1.000 n=10) ¹
Cursor_Renew/4-16          0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Cursor_Set_OneKey-16       0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Cursor_Set_Sequence-16     0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Cursor_Set_Random-16       0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Errno_Error-16             6.000 ± 0%     6.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_abort-16               1.000 ± 0%     1.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_commit-16              3.000 ± 0%     3.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_ro-16                  2.000 ± 0%     2.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_unmanaged_abort-16     1.000 ± 0%     1.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_unmanaged_commit-16    1.000 ± 0%     1.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_unmanaged_ro-16        1.000 ± 0%     1.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_renew-16               0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_Put_append-16          1.000 ± 0%     1.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_Put_append_noflag-16   0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_Get_OneKey-16          0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_Get_Sequence-16        0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
Txn_Get_Random-16          0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=10) ¹
geomean                               ²               +0.00%                ²
¹ all samples are equal
² summaries must be >0 to compute geomean
```
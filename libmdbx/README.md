<!-- Required extensions: pymdownx.betterem, pymdownx.tilde, pymdownx.emoji, pymdownx.tasklist, pymdownx.superfences -->

libmdbx
========

<!-- section-begin overview -->

_libmdbx_ is an extremely fast, compact, powerful, embedded, transactional
[key-value database](https://en.wikipedia.org/wiki/Key-value_database),
with [Apache 2.0 license](https://gitflic.ru/project/erthink/libmdbx/blob?file=LICENSE).
_libmdbx_ has a specific set of properties and capabilities,
focused on creating unique lightweight solutions.

1. Allows **a swarm of multi-threaded processes to
[ACID](https://en.wikipedia.org/wiki/ACID)ly read and update** several
key-value [maps](https://en.wikipedia.org/wiki/Associative_array) and
[multimaps](https://en.wikipedia.org/wiki/Multimap) in a locally-shared
database.

2. Provides **extraordinary performance**, minimal overhead through
[Memory-Mapping](https://en.wikipedia.org/wiki/Memory-mapped_file) and
`Olog(N)` operations costs by virtue of [B+
tree](https://en.wikipedia.org/wiki/B%2B_tree).

3. Requires **no maintenance and no crash recovery** since it doesn't use
[WAL](https://en.wikipedia.org/wiki/Write-ahead_logging), but that might
be a caveat for write-intensive workloads with durability requirements.

4. Enforces [serializability](https://en.wikipedia.org/wiki/Serializability) for
writers just by single
[mutex](https://en.wikipedia.org/wiki/Mutual_exclusion) and affords
[wait-free](https://en.wikipedia.org/wiki/Non-blocking_algorithm#Wait-freedom)
for parallel readers without atomic/interlocked operations, while
**writing and reading transactions do not block each other**.

5. **Guarantee data integrity** after crash unless this was explicitly
neglected in favour of write performance.

6. Supports Linux, Windows, MacOS, Android, iOS, FreeBSD, DragonFly, Solaris,
OpenSolaris, OpenIndiana, NetBSD, OpenBSD and other systems compliant with
**POSIX.1-2008**.

7. **Compact and friendly for fully embedding**. Only ≈25KLOC of `C11`,
≈64K x86 binary code of core, no internal threads neither server process(es),
but implements a simplified variant of the [Berkeley
DB](https://en.wikipedia.org/wiki/Berkeley_DB) and
[dbm](https://en.wikipedia.org/wiki/DBM_(computing)) API.

<!-- section-end -->

Historically, _libmdbx_ is a deeply revised and extended descendant of the legendary
[Lightning Memory-Mapped Database](https://en.wikipedia.org/wiki/Lightning_Memory-Mapped_Database).
_libmdbx_ inherits all benefits from _LMDB_, but resolves some issues and adds [a set of improvements](#improvements-beyond-lmdb).

[![Telergam: Support | Discussions | News](https://img.shields.io/endpoint?color=scarlet&logo=telegram&label=Support%20%7C%20Discussions%20%7C%20News&url=https%3A%2F%2Ftg.sumanjay.workers.dev%2Flibmdbx)](https://t.me/libmdbx)

> Please refer to the online [documentation](https://libmdbx.dqdkfa.ru)
> with [`C` API description](https://libmdbx.dqdkfa.ru/group__c__api.html)
> and pay attention to the [`C++` API](https://gitflic.ru/project/erthink/libmdbx/blob?file=mdbx.h%2B%2B#line-num-1).
> Donations are welcome to the Ethereum/ERC-20 `0xD104d8f8B2dC312aaD74899F83EBf3EEBDC1EA3A`.
> Всё будет хорошо!

Telegram Group archive: [1](https://libmdbx.dqdkfa.ru/tg-archive/messages1.html),
[2](https://libmdbx.dqdkfa.ru/tg-archive/messages2.html), [3](https://libmdbx.dqdkfa.ru/tg-archive/messages3.html), [4](https://libmdbx.dqdkfa.ru/tg-archive/messages4.html),
[5](https://libmdbx.dqdkfa.ru/tg-archive/messages5.html), [6](https://libmdbx.dqdkfa.ru/tg-archive/messages6.html), [7](https://libmdbx.dqdkfa.ru/tg-archive/messages7.html).

## Github

### на Русском (мой родной язык)

Весной 2022, без каких-либо предупреждений или пояснений, администрация
Github удалила мой аккаунт и все проекты. Через несколько месяцев, без
какого-либо моего участия или уведомления, проекты были
восстановлены/открыты в статусе "public read-only archive" из какой-то
неполноценной резервной копии. Эти действия Github я расцениваю как
злонамеренный саботаж, а сам сервис Github считаю навсегда утратившим
какое-либо доверие.

Вследствие произошедшего, никогда и ни при каких условиях, я не буду
размещать на Github первоисточники (aka origins) моих проектов, либо
как-либо полагаться на инфраструктуру Github.

Тем не менее, понимая что пользователям моих проектов удобнее получать к
ним доступ именно на Github, я не хочу ограничивать их свободу или
создавать неудобство, и поэтому размещаю на Github зеркала (aka mirrors)
репозиториев моих проектов. При этом ещё раз акцентирую внимание, что
это только зеркала, которые могут быть заморожены, заблокированы или
удалены в любой момент, как это уже было в 2022.

### in English

In the spring of 2022, without any warnings or explanations, the Github
administration deleted my account and all projects. A few months later,
without any involvement or notification from me, the projects were
restored/opened in the "public read-only archive" status from some kind
of incomplete backup. I regard these actions of Github as malicious
sabotage, and I consider the Github service itself to have lost any
trust forever.

As a result of what has happened, I will never, under any circumstances,
post the primary sources (aka origins) of my projects on Github, or rely
in any way on the Github infrastructure.

Nevertheless, realizing that it is more convenient for users of my
projects to access them on Github, I do not want to restrict their
freedom or create inconvenience, and therefore I place mirrors of my
project repositories on Github. At the same time, I would like to
emphasize once again that these are only mirrors that can be frozen,
blocked or deleted at any time, as was the case in 2022.

## MithrilDB and Future

<!-- section-begin mithril -->

The next version is under non-public development from scratch and will be
released as **MithrilDB** and `libmithrildb` for libraries & packages.
Admittedly mythical [Mithril](https://en.wikipedia.org/wiki/Mithril) is
resembling silver but being stronger and lighter than steel. Therefore
_MithrilDB_ is a rightly relevant name.

_MithrilDB_ is radically different from _libmdbx_ by the new database
format and API based on C++20. The goal of this revolution is to provide
a clearer and robust API, add more features and new valuable properties
of the database. All fundamental architectural problems of libmdbx/LMDB
have been solved there, but now the active development has been
suspended for top-three reasons:

1. For now _libmdbx_ mostly enough and I’m busy for scalability.
2. Waiting for fresh [Elbrus CPU](https://wiki.elbrus.ru/) of [e2k architecture](https://en.wikipedia.org/wiki/Elbrus_2000),
especially with hardware acceleration of [Streebog](https://en.wikipedia.org/wiki/Streebog) and
[Kuznyechik](https://en.wikipedia.org/wiki/Kuznyechik), which are required for Merkle tree, etc.
3. The expectation of needs and opportunities due to the wide use of NVDIMM (aka persistent memory),
modern NVMe and [Ангара](https://ru.wikipedia.org/wiki/Ангара_(интерконнект)).

However, _MithrilDB_ will not be available for countries unfriendly to
Russia (i.e. acceded the sanctions, devil adepts and/or NATO). But it is
not yet known whether such restriction will be implemented only through
a license and support, either the source code will not be open at all.
Basically I am not inclined to allow my work to contribute to the
profit that goes to weapons that kill my relatives and friends.
NO OPTIONS.

Nonetheless, I try not to make any promises regarding _MithrilDB_ until release.

Contrary to _MithrilDB_, _libmdbx_ will forever free and open source.
Moreover with high-quality support whenever possible. Tu deviens
responsable pour toujours de ce que tu as apprivois. So I will continue
to comply with the original open license and the principles of
constructive cooperation, in spite of outright Github sabotage and
sanctions. I will also try to keep (not drop) Windows support, despite
it is an unused obsolete technology for us.

<!-- section-end -->

```
$ objdump -f -h -j .text libmdbx.so

  libmdbx.so:     формат файла elf64-e2k
  архитектура: elbrus-v6:64, флаги 0x00000150:
  HAS_SYMS, DYNAMIC, D_PAGED
  начальный адрес 0x00000000??????00

  Разделы:
  Idx Name          Разм      VMA               LMA               Фа  смещ.  Выр.  Флаги
   10 .text         000e7460  0000000000025c00  0000000000025c00  00025c00  2**10  CONTENTS, ALLOC, LOAD, READONLY, CODE

$ cc --version
  lcc:1.27.14:Jan-31-2024:e2k-v6-linux
  gcc (GCC) 9.3.0 compatible
```

-----

## Table of Contents
- [Characteristics](#characteristics)
    - [Features](#features)
    - [Limitations](#limitations)
    - [Gotchas](#gotchas)
    - [Comparison with other databases](#comparison-with-other-databases)
    - [Improvements beyond LMDB](#improvements-beyond-lmdb)
    - [History & Acknowledgments](#history)
- [Usage](#usage)
    - [Building and Testing](#building-and-testing)
    - [API description](#api-description)
    - [Bindings](#bindings)
- [Performance comparison](#performance-comparison)
    - [Integral performance](#integral-performance)
    - [Read scalability](#read-scalability)
    - [Sync-write mode](#sync-write-mode)
    - [Lazy-write mode](#lazy-write-mode)
    - [Async-write mode](#async-write-mode)
    - [Cost comparison](#cost-comparison)

# Characteristics

<!-- section-begin characteristics -->

## Features

- Key-value data model, keys are always sorted.

- Fully [ACID](https://en.wikipedia.org/wiki/ACID)-compliant, through to
[MVCC](https://en.wikipedia.org/wiki/Multiversion_concurrency_control)
and [CoW](https://en.wikipedia.org/wiki/Copy-on-write).

- Multiple key-value tables/sub-databases within a single datafile.

- Range lookups, including range query estimation.

- Efficient support for short fixed length keys, including native 32/64-bit integers.

- Ultra-efficient support for [multimaps](https://en.wikipedia.org/wiki/Multimap). Multi-values sorted, searchable and iterable. Keys stored without duplication.

- Data is [memory-mapped](https://en.wikipedia.org/wiki/Memory-mapped_file) and accessible directly/zero-copy. Traversal of database records is extremely-fast.

- Transactions for readers and writers, ones do not block others.

- Writes are strongly serialized. No transaction conflicts nor deadlocks.

- Readers are [non-blocking](https://en.wikipedia.org/wiki/Non-blocking_algorithm), notwithstanding [snapshot isolation](https://en.wikipedia.org/wiki/Snapshot_isolation).

- Nested write transactions.

- Reads scale linearly across CPUs.

- Continuous zero-overhead database compactification.

- Automatic on-the-fly database size adjustment.

- Customizable database page size.

- `Olog(N)` cost of lookup, insert, update, and delete operations by virtue of [B+ tree characteristics](https://en.wikipedia.org/wiki/B%2B_tree#Characteristics).

- Online hot backup.

- Append operation for efficient bulk insertion of pre-sorted data.

- No [WAL](https://en.wikipedia.org/wiki/Write-ahead_logging) nor any transaction journal. No crash recovery needed. No maintenance is required.

- No internal cache and/or memory management, all done by basic OS services.

## Limitations

- **Page size**: a power of 2, minimum `256` (mostly for testing), maximum `65536` bytes, default `4096` bytes.
- **Key size**: minimum `0`, maximum ≈½ pagesize (`2022` bytes for default 4K pagesize, `32742` bytes for 64K pagesize).
- **Value size**: minimum `0`, maximum `2146435072` (`0x7FF00000`) bytes for maps, ≈½ pagesize for multimaps (`2022` bytes for default 4K pagesize, `32742` bytes for 64K pagesize).
- **Write transaction size**: up to `1327217884` pages (`4.944272` TiB for default 4K pagesize, `79.108351` TiB for 64K pagesize).
- **Database size**: up to `2147483648` pages (≈`8.0` TiB for default 4K pagesize, ≈`128.0` TiB for 64K pagesize).
- **Maximum tables/sub-databases**: `32765`.

## Gotchas

1. There cannot be more than one writer at a time, i.e. no more than one write transaction at a time.

2. _libmdbx_ is based on [B+ tree](https://en.wikipedia.org/wiki/B%2B_tree), so access to database pages is mostly random.
Thus SSDs provide a significant performance boost over spinning disks for large databases.

3. _libmdbx_ uses [shadow paging](https://en.wikipedia.org/wiki/Shadow_paging) instead of [WAL](https://en.wikipedia.org/wiki/Write-ahead_logging).
Thus syncing data to disk might be a bottleneck for write intensive workload.

4. _libmdbx_ uses [copy-on-write](https://en.wikipedia.org/wiki/Copy-on-write) for [snapshot isolation](https://en.wikipedia.org/wiki/Snapshot_isolation) during updates,
but read transactions prevents recycling an old retired/freed pages, since it read ones. Thus altering of data during a parallel
long-lived read operation will increase the process work set, may exhaust entire free database space,
the database can grow quickly, and result in performance degradation.
Try to avoid long running read transactions, otherwise use [transaction parking](https://libmdbx.dqdkfa.ru/group__c__transactions.html#ga2c2c97730ff35cadcedfbd891ac9b12f)
and/or [Handle-Slow-Readers callback](https://libmdbx.dqdkfa.ru/group__c__err.html#ga2cb11b56414c282fe06dd942ae6cade6).

5. _libmdbx_ is extraordinarily fast and provides minimal overhead for data access,
so you should reconsider using brute force techniques and double check your code.
On the one hand, in the case of _libmdbx_, a simple linear search may be more profitable than complex indexes.
On the other hand, if you make something suboptimally, you can notice detrimentally only on sufficiently large data.

## Comparison with other databases
For now please refer to [chapter of "BoltDB comparison with other
databases"](https://github.com/coreos/bbolt#comparison-with-other-databases)
which is also (mostly) applicable to _libmdbx_ with minor clarification:
 - a database could shared by multiple processes, i.e. no multi-process issues;
 - no issues with moving a cursor(s) after the deletion;
 - _libmdbx_ provides zero-overhead database compactification, so a database file could be shrinked/truncated in particular cases;
 - excluding disk I/O time _libmdbx_ could be ≈3 times faster than BoltDB and up to 10-100K times faster than both BoltDB and LMDB in particular extreme cases;
 - _libmdbx_ provides more features compared to BoltDB and/or LMDB.

<!-- section-end -->

<!-- section-begin improvements -->

Improvements beyond LMDB
========================

_libmdbx_ is superior to legendary _[LMDB](https://symas.com/lmdb/)_ in
terms of features and reliability, not inferior in performance. In
comparison to _LMDB_, _libmdbx_ make things "just work" perfectly and
out-of-the-box, not silently and catastrophically break down. The list
below is pruned down to the improvements most notable and obvious from
the user's point of view.

## Some Added Features

1. Keys could be more than 2 times longer than _LMDB_.
   > For DB with default page size _libmdbx_ support keys up to 2022 bytes
   > and up to 32742 bytes for 64K page size. _LMDB_ allows key size up to
   > 511 bytes and may silently loses data with large values.

2. Up to 30% faster than _LMDB_ in [CRUD](https://en.wikipedia.org/wiki/Create,_read,_update_and_delete) benchmarks.
   > Benchmarks of the in-[tmpfs](https://en.wikipedia.org/wiki/Tmpfs) scenarios,
   > that tests the speed of the engine itself, showned that _libmdbx_ 10-20% faster than _LMDB_,
   > and up to 30% faster when _libmdbx_ compiled with specific build options
   > which downgrades several runtime checks to be match with LMDB behaviour.
   >
   > However, libmdbx may be slower than LMDB on Windows, since uses native file locking API.
   > These locks are really slow, but they prevent an inconsistent backup from being obtained by copying the DB file during an ongoing write transaction.
   > So I think this is the right decision, and for speed, it's better to use Linux, or ask Microsoft to fix up file locks.
   >
   > Noted above and other results could be easily reproduced with [ioArena](https://abf.io/erthink/ioarena) just by `make bench-quartet` command,
   > including comparisons with [RockDB](https://en.wikipedia.org/wiki/RocksDB)
   > and [WiredTiger](https://en.wikipedia.org/wiki/WiredTiger).

3. Automatic on-the-fly database size adjustment, both increment and reduction.
   > _libmdbx_ manages the database size according to parameters specified
   > by `mdbx_env_set_geometry()` function,
   > ones include the growth step and the truncation threshold.
   >
   > Unfortunately, on-the-fly database size adjustment doesn't work under [Wine](https://en.wikipedia.org/wiki/Wine_(software))
   > due to its internal limitations and unimplemented functions, i.e. the `MDBX_UNABLE_EXTEND_MAPSIZE` error will be returned.

4. Automatic continuous zero-overhead database compactification.
   > During each commit _libmdbx_ merges a freeing pages which adjacent with the unallocated area
   > at the end of file, and then truncates unused space when a lot enough of.

5. The same database format for 32- and 64-bit builds.
   > _libmdbx_ database format depends only on the [endianness](https://en.wikipedia.org/wiki/Endianness) but not on the [bitness](https://en.wiktionary.org/wiki/bitness).

6. The "Big Foot" feature than solves speific performance issues with huge transactions and extra-large page-number-lists.

7. LIFO policy for Garbage Collection recycling. This can significantly increase write performance due write-back disk cache up to several times in a best case scenario.
   > LIFO means that for reuse will be taken the latest becomes unused pages.
   > Therefore the loop of database pages circulation becomes as short as possible.
   > In other words, the set of pages, that are (over)written in memory and on disk during a series of write transactions, will be as small as possible.
   > Thus creates ideal conditions for the battery-backed or flash-backed disk cache efficiency.

8. Parking of read transactions with ousting and auto-restart, [Handle-Slow-Readers callback](https://libmdbx.dqdkfa.ru/group__c__err.html#ga2cb11b56414c282fe06dd942ae6cade6) to resolve an issues due to long-lived read transactions.

9. Fast estimation of range query result volume, i.e. how many items can
be found between a `KEY1` and a `KEY2`. This is a prerequisite for build
and/or optimize query execution plans.
   > _libmdbx_ performs a rough estimate based on common B-tree pages of the paths from root to corresponding keys.

10. Database integrity check API both with standalone `mdbx_chk` utility.

11. Support for opening databases in the exclusive mode, including on a network share.

12. Extended information of whole-database, tables/sub-databases, transactions, readers enumeration.
   > _libmdbx_ provides a lot of information, including dirty and leftover pages
   > for a write transaction, reading lag and holdover space for read transactions.

13. Support of Zero-length for keys and values.

14. Useful runtime options for tuning engine to application's requirements and use cases specific.

15. Automated steady sync-to-disk upon several thresholds and/or timeout via cheap polling.

16. Ability to determine whether the particular data is on a dirty page
or not, that allows to avoid copy-out before updates.

17. Extended update and delete operations.
   > _libmdbx_ allows one _at once_ with getting previous value
   > and addressing the particular item from multi-value with the same key.

18. Sequence generation and three persistent 64-bit vector-clock like markers.

## Other fixes and specifics

1. Fixed more than 10 significant errors, in particular: page leaks,
wrong table/sub-database statistics, segfault in several conditions,
nonoptimal page merge strategy, updating an existing record with
a change in data size (including for multimap), etc.

2. All cursors can be reused and should be closed explicitly,
regardless ones were opened within a write or read transaction.

3. Opening database handles are spared from race conditions and
pre-opening is not needed.

4. Returning `MDBX_EMULTIVAL` error in case of ambiguous update or delete.

5. Guarantee of database integrity even in asynchronous unordered write-to-disk mode.
   > _libmdbx_ propose additional trade-off by `MDBX_SAFE_NOSYNC` with append-like manner for updates,
   > that avoids database corruption after a system crash contrary to LMDB.
   > Nevertheless, the `MDBX_UTTERLY_NOSYNC` mode is available to match LMDB's behaviour for `MDB_NOSYNC`.

6. On **MacOS & iOS** the `fcntl(F_FULLFSYNC)` syscall is used _by
default_ to synchronize data with the disk, as this is [the only way to
guarantee data
durability](https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man2/fsync.2.html)
in case of power failure. Unfortunately, in scenarios with high write
intensity, the use of `F_FULLFSYNC` significantly degrades performance
compared to LMDB, where the `fsync()` syscall is used. Therefore,
_libmdbx_ allows you to override this behavior by defining the
`MDBX_OSX_SPEED_INSTEADOF_DURABILITY=1` option while build the library.

7. On **Windows** the `LockFileEx()` syscall is used for locking, since
it allows place the database on network drives, and provides protection
against incompetent user actions (aka
[poka-yoke](https://en.wikipedia.org/wiki/Poka-yoke)). Therefore
_libmdbx_ may be a little lag in performance tests from LMDB where the
named mutexes are used.

<!-- section-end -->
<!-- section-begin history -->

# History

Historically, _libmdbx_ is a deeply revised and extended descendant of the
[Lightning Memory-Mapped Database](https://en.wikipedia.org/wiki/Lightning_Memory-Mapped_Database).
At first the development was carried out within the
[ReOpenLDAP](https://web.archive.org/web/https://github.com/erthink/ReOpenLDAP) project. About a
year later _libmdbx_ was separated into a standalone project, which was
[presented at Highload++ 2015
conference](http://www.highload.ru/2015/abstracts/1831.html).

Since 2017 _libmdbx_ is used in [Fast Positive Tables](https://gitflic.ru/project/erthink/libfpta),
and until 2025 development was funded by [Positive Technologies](https://www.ptsecurity.com).
Since 2020 _libmdbx_ is used in Ethereum: [Erigon](https://github.com/erigontech/erigon), [Akula](https://github.com/akula-bft/akula),
[Silkworm](https://github.com/erigontech/silkworm), [Reth](https://github.com/paradigmxyz/reth), etc.

On 2022-04-15 the Github administration, without any warning nor
explanation, deleted _libmdbx_ along with a lot of other projects,
simultaneously blocking access for many developers. Therefore on
2022-04-21 I have migrated to a reliable trusted infrastructure.
The origin for now is at [GitFlic](https://gitflic.ru/project/erthink/libmdbx)
with backup at [ABF by ROSA Лаб](https://abf.rosalinux.ru/erthink/libmdbx).
For the same reason ~~Github~~ is blacklisted forever.

Since May 2024 and version 0.13 _libmdbx_ was re-licensed under Apache-2.0 license.
Please refer to the [`COPYRIGHT` file](https://gitflic.ru/project/erthink/libmdbx/blob/raw?file=COPYRIGHT) for license change explanations.

## Acknowledgments
Howard Chu <hyc@openldap.org> and Hallvard Furuseth
<hallvard@openldap.org> are the authors of _LMDB_, from which _libmdbx_
was forked in 2015.

Martin Hedenfalk <martin@bzero.se> is the author of `btree.c` code, which
was used to begin development of _LMDB_.

<!-- section-end -->

--------------------------------------------------------------------------------

Usage
=====

<!-- section-begin usage -->

Currently, libmdbx is only available in a
[source code](https://en.wikipedia.org/wiki/Source_code) form.
Packages support for common Linux distributions is planned in the future,
since release the version 1.0.

## Source code embedding

_libmdbx_ provides three official ways for integration in source code form:

1. Using an amalgamated source code which available in the [releases section](https://gitflic.ru/project/erthink/libmdbx/release) on GitFlic.
   > An amalgamated source code includes all files required to build and
   > use _libmdbx_, but not for testing _libmdbx_ itself.
   > Beside the releases an amalgamated sources could be created any time from the original clone of git
   > repository on Linux by executing `make dist`. As a result, the desired
   > set of files will be formed in the `dist` subdirectory.

2. Using [Conan Package Manager](https://conan.io/):
    - optional: Setup your own conan-server;
    - Create conan-package by `conan create .` inside the _libmdbx_' repo subdirectory;
    - optional: Upload created recipe and/or package to the conan-server by `conan upload -r SERVER 'mdbx/*'`;
    - Consume libmdbx-package from the local conan-cache or from conan-server in accordance with the [Conan tutorial](https://docs.conan.io/2/tutorial/consuming_packages.html).

3. Adding the complete source code as a `git submodule` from the [origin git repository](https://gitflic.ru/project/erthink/libmdbx) on GitFlic.
   > This allows you to build as _libmdbx_ and testing tool.
   >  On the other hand, this way requires you to pull git tags, and use C++11 compiler for test tool.

_**Please, avoid using any other techniques.**_ Otherwise, at least
don't ask for support and don't name such chimeras `libmdbx`.

## Building and Testing

Both amalgamated and original source code provides build through the use
[CMake](https://cmake.org/) or [GNU
Make](https://www.gnu.org/software/make/) with
[bash](https://en.wikipedia.org/wiki/Bash_(Unix_shell)).

All build ways
are completely traditional and have minimal prerequirements like
`build-essential`, i.e. the non-obsolete C/C++ compiler and a
[SDK](https://en.wikipedia.org/wiki/Software_development_kit) for the
target platform.
Obviously you need building tools itself, i.e. `git`,
`cmake` or GNU `make` with `bash`. For your convenience, `make help`
and `make options` are also available for listing existing targets
and build options respectively.

The only significant specificity is that git' tags are required
to build from complete (not amalgamated) source codes.
Executing **`git fetch --tags --force --prune`** is enough to get ones,
and `--unshallow` or `--update-shallow` is required for shallow cloned case.

So just using CMake or GNU Make in your habitual manner and feel free to
fill an issue or make pull request in the case something will be
unexpected or broken down.

### Testing
The amalgamated source code does not contain any tests for or several reasons.
Please read [the explanation](https://libmdbx.dqdkfa.ru/dead-github/issues/214#issuecomment-870717981) and don't ask to alter this.
So for testing _libmdbx_ itself you need a full source code, i.e. the clone of a git repository, there is no option.

The full source code of _libmdbx_ has a [`test` subdirectory](https://gitflic.ru/project/erthink/libmdbx/tree/master/test) with minimalistic test "framework".
Actually yonder is a source code of the `mdbx_test` – console utility which has a set of command-line options that allow construct and run a reasonable enough test scenarios.
This test utility is intended for _libmdbx_'s developers for testing library itself, but not for use by users.
Therefore, only basic information is provided:

   - There are few CRUD-based test cases (hill, TTL, nested, append, jitter, etc),
     which can be combined to test the concurrent operations within shared database in a multi-processes environment.
     This is the `basic` test scenario.
   - The `Makefile` provide several self-described targets for testing: `smoke`, `test`, `check`, `memcheck`, `test-valgrind`,
     `test-asan`, `test-leak`, `test-ubsan`, `cross-gcc`, `cross-qemu`, `gcc-analyzer`, `smoke-fault`, `smoke-singleprocess`,
     `test-singleprocess`, `long-test`. Please run `make --help` if doubt.
   - In addition to the `mdbx_test` utility, there is the script [`stochastic.sh`](https://gitflic.ru/project/erthink/libmdbx/blob/master/test/stochastic.sh),
     which calls `mdbx_test` by going through set of modes and options, with gradually increasing the number of operations and the size of transactions.
     This script is used for mostly of all automatic testing, including `Makefile` targets and Continuous Integration.
   - Brief information of available command-line options is available by `--help`.
     However, you should dive into source code to get all, there is no option.

Anyway, no matter how thoroughly the _libmdbx_ is tested, you should rely only on your own tests for a few reasons:

1. Mostly of all use cases are unique.
   So it is no warranty that your use case was properly tested, even the _libmdbx_'s tests engages stochastic approach.
2. If there are problems, then your test on the one hand will help to verify whether you are using _libmdbx_ correctly,
   on the other hand it will allow to reproduce the problem and insure against regression in a future.
3. Actually you should rely on than you checked by yourself or take a risk.

### Common important details

#### Build reproducibility
By default _libmdbx_ track build time via `MDBX_BUILD_TIMESTAMP` build option and macro.
So for a [reproducible builds](https://en.wikipedia.org/wiki/Reproducible_builds) you should predefine/override it to known fixed string value.
For instance:

 - for reproducible build with make: `make MDBX_BUILD_TIMESTAMP=unknown ` ...
 - or during configure by CMake: `cmake -DMDBX_BUILD_TIMESTAMP:STRING=unknown ` ...

Of course, in addition to this, your toolchain must ensure the reproducibility of builds.
For more information please refer to [reproducible-builds.org](https://reproducible-builds.org/).

#### Containers
There are no special traits nor quirks if you use _libmdbx_ ONLY inside
the single container. But in a cross-container(s) or with a host-container(s)
interoperability cases the three major things MUST be guaranteed:

1. Coherence of memory mapping content and unified page cache inside OS
kernel for host and all container(s) operated with a DB. Basically this
means must be only a single physical copy of each memory mapped DB' page
in the system memory.

2. Uniqueness of [PID](https://en.wikipedia.org/wiki/Process_identifier) values and/or a common space for ones:
    - for POSIX systems: PID uniqueness for all processes operated with a DB.
      I.e. the `--pid=host` is required for run DB-aware processes inside Docker,
      either without host interaction a `--pid=container:<name|id>` with the same name/id.
    - for non-POSIX (i.e. Windows) systems: inter-visibility of processes handles.
      I.e. the `OpenProcess(SYNCHRONIZE, ..., PID)` must return reasonable error,
      including `ERROR_ACCESS_DENIED`,
      but not the `ERROR_INVALID_PARAMETER` as for an invalid/non-existent PID.

3. The versions/builds of _libmdbx_ and `libc`/`pthreads` (`glibc`, `musl`, etc) must be be compatible.
   - Basically, the `options:` string in the output of `mdbx_chk -V` must be the same for host and container(s).
     See `MDBX_LOCKING`, `MDBX_USE_OFDLOCKS` and other build options for details.
   - Avoid using different versions of `libc`, especially mixing different implementations, i.e. `glibc` with `musl`, etc.
     Prefer to use the same LTS version, or switch to full virtualization/isolation if in doubt.

#### DSO/DLL unloading and destructors of Thread-Local-Storage objects
When building _libmdbx_ as a shared library or use static _libmdbx_ as a
part of another dynamic library, it is advisable to make sure that your
system ensures the correctness of the call destructors of
Thread-Local-Storage objects when unloading dynamic libraries.

If this is not the case, then unloading a dynamic-link library with
_libmdbx_ code inside, can result in either a resource leak or a crash
due to calling destructors from an already unloaded DSO/DLL object. The
problem can only manifest in a multithreaded application, which makes
the unloading of shared dynamic libraries with _libmdbx_ code inside,
after using _libmdbx_. It is known that TLS-destructors are properly
maintained in the following cases:

- On all modern versions of Windows (Windows 7 and later).

- On systems with the
[`__cxa_thread_atexit_impl()`](https://sourceware.org/glibc/wiki/Destructor%20support%20for%20thread_local%20variables)
function in the standard C library, including systems with GNU libc
version 2.18 and later.

- On systems with libpthread/ntpl from GNU libc with bug fixes
[#21031](https://sourceware.org/bugzilla/show_bug.cgi?id=21031) and
[#21032](https://sourceware.org/bugzilla/show_bug.cgi?id=21032), or
where there are no similar bugs in the pthreads implementation.

### Linux and other platforms with GNU Make
To build the library it is enough to execute `make all` in the directory
of source code, and `make check` to execute the basic tests.

If the `make` installed on the system is not GNU Make, there will be a
lot of errors from make when trying to build. In this case, perhaps you
should use `gmake` instead of `make`, or even `gnu-make`, etc.

### FreeBSD and related platforms
As a rule on BSD and it derivatives the default is to use Berkeley Make and
[Bash](https://en.wikipedia.org/wiki/Bash_(Unix_shell)) is not installed.

So you need to install the required components: GNU Make, Bash, C and C++
compilers compatible with GCC or CLANG. After that, to build the
library, it is enough to execute `gmake all` (or `make all`) in the
directory with source code, and `gmake check` (or `make check`) to run
the basic tests.

### Windows
For build _libmdbx_ on Windows the _original_ CMake and [Microsoft Visual
Studio 2019](https://en.wikipedia.org/wiki/Microsoft_Visual_Studio) are
recommended. Please use the recent versions of CMake, Visual Studio and Windows
SDK to avoid troubles with C11 support and `alignas()` feature.

For build by MinGW the 10.2 or recent version coupled with a modern CMake are required.
So it is recommended to use [chocolatey](https://chocolatey.org/) to install and/or update the ones.

Another ways to build is potentially possible but not supported and will not.
The `CMakeLists.txt` or `GNUMakefile` scripts will probably need to be modified accordingly.
Using other methods do not forget to add the `ntdll.lib` to linking.

It should be noted that in _libmdbx_ was efforts to avoid
runtime dependencies from CRT and other MSVC libraries.
For this is enough to pass the `-DMDBX_WITHOUT_MSVC_CRT:BOOL=ON` option
during configure by CMake.

To run the [long stochastic test scenario](test/stochastic.sh),
[bash](https://en.wikipedia.org/wiki/Bash_(Unix_shell)) is required, and
such testing is recommended with placing the test data on the
[RAM-disk](https://en.wikipedia.org/wiki/RAM_drive).

### Windows Subsystem for Linux
_libmdbx_ could be used in [WSL2](https://en.wikipedia.org/wiki/Windows_Subsystem_for_Linux#WSL_2)
but NOT in [WSL1](https://en.wikipedia.org/wiki/Windows_Subsystem_for_Linux#WSL_1) environment.
This is a consequence of the fundamental shortcomings of _WSL1_ and cannot be fixed.
To avoid data loss, _libmdbx_ returns the `ENOLCK` (37, "No record locks available")
error when opening the database in a _WSL1_ environment.

### MacOS
Current [native build tools](https://en.wikipedia.org/wiki/Xcode) for
MacOS include GNU Make, CLANG and an outdated version of Bash.
However, the build script uses GNU-kind of `sed` and `tar`.
So the easiest way to install all prerequirements is to use [Homebrew](https://brew.sh/),
just by `brew install bash make cmake ninja gnu-sed gnu-tar --with-default-names`.

Next, to build the library, it is enough to run `make all` in the
directory with source code, and run `make check` to execute the base
tests. If something goes wrong, it is recommended to install
[Homebrew](https://brew.sh/) and try again.

To run the [long stochastic test scenario](test/stochastic.sh), you
will need to install the current (not outdated) version of
[Bash](https://en.wikipedia.org/wiki/Bash_(Unix_shell)).
Just install it as noted above.

### Android
I recommend using CMake to build _libmdbx_ for Android.
Please refer to the [official guide](https://developer.android.com/studio/projects/add-native-code).

### iOS
To build _libmdbx_ for iOS, I recommend using CMake with the
["toolchain file"](https://cmake.org/cmake/help/latest/variable/CMAKE_TOOLCHAIN_FILE.html)
from the [ios-cmake](https://github.com/leetal/ios-cmake) project.

<!-- section-end -->

## API description

Please refer to the online [_libmdbx_ API reference](https://libmdbx.dqdkfa.ru/docs)
and/or see the [mdbx.h++](mdbx.h%2B%2B) and [mdbx.h](mdbx.h) headers.

<!-- section-begin bindings -->

Bindings
========

| Runtime |  Repo  | Author |
| ------- | ------ | ------ |
| Rust    | [libmdbx-rs](https://github.com/vorot93/libmdbx-rs)   | [Artem Vorotnikov](https://github.com/vorot93) |
| Python  | [PyPi/libmdbx](https://pypi.org/project/libmdbx/)     | [Lazymio](https://github.com/wtdcode) |
| Java    | [mdbxjni](https://github.com/castortech/mdbxjni)      | [Castor Technologies](https://castortech.com/) |
| Go      | [mdbx-go](https://github.com/torquem-ch/mdbx-go)      | [Alex Sharov](https://github.com/AskAlexSharov) |
| Ruby    | [ruby-mdbx](https://rubygems.org/gems/mdbx/)          | [Mahlon E. Smith](https://github.com/mahlonsmith) |
| Zig     | [mdbx-zig](https://github.com/theseyan/lmdbx-zig)     | [Sayan J. Das](https://github.com/theseyan) |

##### Obsolete/Outdated/Unsupported:

| Runtime |  Repo  | Author |
| ------- | ------ | ------ |
| .NET    | [mdbx.NET](https://github.com/wangjia184/mdbx.NET) | [Jerry Wang](https://github.com/wangjia184) |
| Scala   | [mdbx4s](https://github.com/david-bouyssie/mdbx4s) | [David Bouyssié](https://github.com/david-bouyssie) |
| Rust    | [mdbx](https://crates.io/crates/mdbx)                 | [gcxfd](https://github.com/gcxfd) |
| Haskell | [libmdbx-hs](https://hackage.haskell.org/package/libmdbx) | [Francisco Vallarino](https://github.com/fjvallarino) |
| Lua     | [lua-libmdbx](https://github.com/mah0x211/lua-libmdbx) | [Masatoshi Fukunaga](https://github.com/mah0x211) |
| NodeJS, [Deno](https://deno.land/) | [lmdbx-js](https://github.com/kriszyp/lmdbx-js) | [Kris Zyp](https://github.com/kriszyp/)
| NodeJS  | [node-mdbx](https://www.npmjs.com/package/node-mdbx/) | [Сергей Федотов](mailto:sergey.fedotov@corp.mail.ru) |
| Nim     | [NimDBX](https://github.com/snej/nimdbx) | [Jens Alfke](https://github.com/snej)

<!-- section-end -->

--------------------------------------------------------------------------------

<!-- section-begin performance -->

Performance comparison
======================

Over the past 10 years, _libmdbx_ has had a lot of significant
improvements and innovations. _libmdbx_ has become a slightly faster in
simple cases and many times faster in complex scenarios, especially with
a huge transactions in gigantic databases. Therefore, on the one hand,
the results below are outdated. However, on the other hand, these simple
benchmarks are evident, easy to reproduce, and are close to the most
common use cases.

The following all benchmark results were obtained in 2015 by [IOArena](https://abf.io/erthink/ioarena)
and multiple [scripts](https://github.com/pmwkaa/ioarena/tree/HL%2B%2B2015)
runs on my laptop (i7-4600U 2.1 GHz, SSD MZNTD512HAGL-000L1).

## Integral performance

Here showed sum of performance metrics in 3 benchmarks:

 - Read/Search on the machine with 4 logical CPUs in HyperThreading mode (i.e. actually 2 physical CPU cores);

 - Transactions with [CRUD](https://en.wikipedia.org/wiki/CRUD)
 operations in sync-write mode (fdatasync is called after each
 transaction);

 - Transactions with [CRUD](https://en.wikipedia.org/wiki/CRUD)
 operations in lazy-write mode (moment to sync data to persistent storage
 is decided by OS).

*Reasons why asynchronous mode isn't benchmarked here:*

  1. It doesn't make sense as it has to be done with DB engines, oriented
  for keeping data in memory e.g. [Tarantool](https://tarantool.io/),
  [Redis](https://redis.io/)), etc.

  2. Performance gap is too high to compare in any meaningful way.

![Comparison #1: Integral Performance](https://libmdbx.dqdkfa.ru/img/perf-slide-1.png)

--------------------------------------------------------------------------------

## Read Scalability

Summary performance with concurrent read/search queries in 1-2-4-8
threads on the machine with 4 logical CPUs in HyperThreading mode (i.e.
actually 2 physical CPU cores).

![Comparison #2: Read Scalability](https://libmdbx.dqdkfa.ru/img/perf-slide-2.png)

--------------------------------------------------------------------------------

## Sync-write mode

 - Linear scale on left and dark rectangles mean arithmetic mean
 transactions per second;

 - Logarithmic scale on right is in seconds and yellow intervals mean
 execution time of transactions. Each interval shows minimal and maximum
 execution time, cross marks standard deviation.

**10,000 transactions in sync-write mode**. In case of a crash all data
is consistent and conforms to the last successful transaction. The
[fdatasync](https://linux.die.net/man/2/fdatasync) syscall is used after
each write transaction in this mode.

In the benchmark each transaction contains combined CRUD operations (2
inserts, 1 read, 1 update, 1 delete). Benchmark starts on an empty database
and after full run the database contains 10,000 small key-value records.

![Comparison #3: Sync-write mode](https://libmdbx.dqdkfa.ru/img/perf-slide-3.png)

--------------------------------------------------------------------------------

## Lazy-write mode

 - Linear scale on left and dark rectangles mean arithmetic mean of
 thousands transactions per second;

 - Logarithmic scale on right in seconds and yellow intervals mean
 execution time of transactions. Each interval shows minimal and maximum
 execution time, cross marks standard deviation.

**100,000 transactions in lazy-write mode**. In case of a crash all data
is consistent and conforms to the one of last successful transactions, but
transactions after it will be lost. Other DB engines use
[WAL](https://en.wikipedia.org/wiki/Write-ahead_logging) or transaction
journal for that, which in turn depends on order of operations in the
journaled filesystem. _libmdbx_ doesn't use WAL and hands I/O operations
to filesystem and OS kernel (mmap).

In the benchmark each transaction contains combined CRUD operations (2
inserts, 1 read, 1 update, 1 delete). Benchmark starts on an empty database
and after full run the database contains 100,000 small key-value
records.

![Comparison #4: Lazy-write mode](https://libmdbx.dqdkfa.ru/img/perf-slide-4.png)

--------------------------------------------------------------------------------

## Async-write mode

 - Linear scale on left and dark rectangles mean arithmetic mean of
 thousands transactions per second;

 - Logarithmic scale on right in seconds and yellow intervals mean
 execution time of transactions. Each interval shows minimal and maximum
 execution time, cross marks standard deviation.

**1,000,000 transactions in async-write mode**.
In case of a crash all data is consistent and conforms to the one of
last successful transactions, but lost transaction count is much higher
than in lazy-write mode. All DB engines in this mode do as little writes
as possible on persistent storage. _libmdbx_ uses
[msync(MS_ASYNC)](https://linux.die.net/man/2/msync) in this mode.

In the benchmark each transaction contains combined CRUD operations (2
inserts, 1 read, 1 update, 1 delete). Benchmark starts on an empty database
and after full run the database contains 10,000 small key-value records.

![Comparison #5: Async-write mode](https://libmdbx.dqdkfa.ru/img/perf-slide-5.png)

--------------------------------------------------------------------------------

## Cost comparison

Summary of used resources during lazy-write mode benchmarks:

 - Read and write IOPs;

 - Sum of user CPU time and sys CPU time;

 - Used space on persistent storage after the test and closed DB, but not
 waiting for the end of all internal housekeeping operations (LSM
 compactification, etc).

_ForestDB_ is excluded because benchmark showed it's resource
consumption for each resource (CPU, IOPs) much higher than other engines
which prevents to meaningfully compare it with them.

All benchmark data is gathered by
[getrusage()](http://man7.org/linux/man-pages/man2/getrusage.2.html)
syscall and by scanning the data directory.

![Comparison #6: Cost comparison](https://libmdbx.dqdkfa.ru/img/perf-slide-6.png)

<!-- section-end -->

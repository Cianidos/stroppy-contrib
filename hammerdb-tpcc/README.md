# HammerDB-derived PostgreSQL TPROC-C

Standalone Stroppy SDK workload. See [workload/README.md](workload/README.md)
for modes, flags, semantics, and known upstream quirks. This entire module is
GPL-3.0, not the repository's default Apache-2.0; see [LICENSE](LICENSE) and
[NOTICE](workload/NOTICE).

```sh
go test -race ./...
go run . -d pg -D url='postgres://tpcc:tpcc@localhost:5432/tpcc' \
  --mode timed --vus 8 --rampup 2m --duration 5m
```

No setup runs by default, so the same command accepts a database created by
HammerDB v6.1. To create a dedicated empty database's full one-warehouse dataset:

```sh
go run . -d pg -D url='postgres://tpcc:tpcc@localhost:5432/new_tpcc' \
  --create=true --warehouses 1 \
  --steps create_schema,load_items,load_data,create_indexes,create_routines,analyze
```

Use `--storedprocs=true` on both setup and run to select PostgreSQL procedures.
Without it, the default is HammerDB's function mode. `--mode test --iterations 100`
runs 100 transactions **per client**. `--async-scale=true --async-client 10`
creates ten concurrent clients per active worker. `--keyandthink=true` enables
HammerDB's pacing. Other parameters and process-environment names appear in
`go run . --help`.

## Side-by-side comparisons

Use the pinned upstream reference:

```sh
git clone https://github.com/TPC-Council/HammerDB ../HammerDB
git -C ../HammerDB switch --detach 2442b32756a484de4206bb0061d9bb28869beaf0
```

Create and snapshot the dataset once. Restore it before each run; do not let both
clients modify the same live database concurrently. Match PostgreSQL version,
settings, storage, schema/routines, warehouse count, warm cache policy, and host
resources. Use the same function/procedure profile, pacing, warehouse mode,
iteration ceiling, and measurement window. The Stroppy `--vus 8` setting matches
**nine** configured HammerDB VUs in a timed run: eight active workers plus one
monitor. For async mode, also match clients per worker and login delay.

Compare `hammerdb.tpcc` report NOPM/TPM (or `hammerdb_nopm`/`hammerdb_tpm` gauges),
not Stroppy's native TPS. PostgreSQL TPM is cluster-wide, includes rollbacks and
monitor queries, and can include unrelated activity. Native SDK metrics include
ramp-up and client drain. One side's native summary is not the other's throughput
score. Whole-minute results retain HammerDB's integer division; shorter durations
are supported for smoke testing.

SQL routines remain upstream SQL. Client input generation uses Tcl's Park-Miller
stream with explicit `--seed`; server-side `random()` is not seeded or replaced
in production. Client scheduling, binary/text result transport, SDK telemetry,
and Go/Tcl runtime overhead differ. Those are performance comparison subjects,
not claims of identical wire traffic or scheduling.

## Verification

Unit/recording tests require no database. The Tcl generator oracle runs when
`tclsh` is installed; CI installs it. Tagged PostgreSQL tests require CREATEDB
permission, create uniquely named disposable databases, and remove only those:

```sh
HAMMERDB_PG_URL='host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable' \
  go test -race -tags=integration ./workload
```

`tools/reference.tcl` invokes the actual pinned HammerDB loader without its GUI;
it needs Tcl, Thread, and Pgtcl. `tools/fingerprint.sql` hashes all nine tables
excluding wall-clock timestamps. Running both loaders with seed 1 produced equal
row counts and non-timestamp hashes for every table, including 298,934 order lines.

```sh
tclsh tools/reference.tcl ../HammerDB \
  'host=localhost user=postgres dbname=reference_empty' functions
psql 'postgres://postgres@localhost/reference_empty' -Atf tools/fingerprint.sql
psql 'postgres://postgres@localhost/stroppy_empty' -Atf tools/fingerprint.sql
```

`tools/drive.tcl` and `TestDifferentialClient` provide a controlled 100-transaction
oracle. Restore two copies of the same snapshot first. Both use client seed 1
and database `setseed(0.5)` **only in this oracle**, plus a fixed procedure input
timestamp. Function and procedure runs produced equal nine-table non-timestamp
fingerprints after 100 transactions.

```sh
tclsh tools/drive.tcl ../HammerDB \
  'host=localhost user=postgres dbname=reference_copy' functions 100
HAMMERDB_DIFFERENTIAL_URL='host=localhost user=postgres dbname=stroppy_copy' \
  go test -tags=integration ./workload -run '^TestDifferentialClient$'
```

For procedures, use `procedures` in the Tcl command and add
`HAMMERDB_DIFFERENTIAL_PROCEDURES=true` to the Go command. Both copies must have
procedure routines installed. Debian Pgtcl 3.0.1 exposes a shorter prepared-call
signature than HammerDB's bundled binding; append `legacy-pgtcl` to the Tcl
command for that binding. The adapter removes only the two empty format lists.
It does not alter SQL or transaction inputs.

SQL extraction is reproducible and does not execute downloaded Tcl:

```sh
python3 -I tools/extract.py ../HammerDB workload
git diff -- workload/schema.sql workload/functions.sql workload/procedures.sql
```

## Scope

Native PostgreSQL only: functions/procedures, finite/timed runs, fixed/all-warehouse
terminals, pacing, async clients, HammerDB XML connection-routing policies,
partitioned order-line setup, and optional post-run vacuum/checkpoint. The loader
uses SDK COPY batches with HammerDB values and ordering; it does not reproduce
HammerDB's exact load-time commit/transfer chunking or implicit timestamp timing.
Load performance is not part of the runtime parity claim. Multi-loader warehouse
ranges match HammerDB; independent worker seeds are explicit in the Go workload.

Citus, EDB/Oracle compatibility, DRITA, remote primary/replica orchestration,
GUI/Tcl control commands, and automatic user/database/tablespace provisioning
are not implemented. The workload never drops existing tables. Peak performance
comparison has not been benchmarked as part of the parity validation.

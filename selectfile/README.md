# selectfile

Loads one named query from `workload/queries.sql` and checks its answer against a
declared parameter. Requires Go 1.27 or newer.

```sh
go test ./...
go run . -d pg -D url=postgres://user@localhost:5432/postgres --iterations 1
go run . -d pg -D url=... --expected 2 --iterations 1   # fails: the file returns 1
go run . -d pg -D url=... my-queries.sql                # load a different file
```

Parameters:

| Flag | Meaning |
|------|---------|
| `--expected` | Answer the loaded statement must return (default 1). |
| `--label` | Copied into the run report data. |
| `--sql-file` | SQL file to load instead of the embedded `queries.sql`. |
| `[sql_file]` | Positional form of `--sql-file`. |

The query is resolved by section and name (`--+ query`, `--= select_one`), so a
replacement file must define the same pair. A missing file or missing query fails
the run before any action executes.

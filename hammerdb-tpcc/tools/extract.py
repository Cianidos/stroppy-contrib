#!/usr/bin/env python3
"""Extract native PostgreSQL SQL from a pinned HammerDB checkout, without executing Tcl."""

import argparse
import hashlib
from pathlib import Path
import re
import subprocess

COMMIT = "2442b32756a484de4206bb0061d9bb28869beaf0"


def routines(text, start, end):
    result = []
    for match in re.finditer(r"set sql\(\d+\) \{", text[start:end]):
        offset = start + match.end()
        depth = 1
        cursor = offset
        while depth:
            if text[cursor] == "{":
                depth += 1
            elif text[cursor] == "}":
                depth -= 1
            cursor += 1
        sql = text[offset:cursor - 1].strip()
        if sql.startswith(("CREATE OR REPLACE FUNCTION", "CREATE OR REPLACE PROCEDURE")):
            result.append(sql)
    if len(result) != 6:
        raise ValueError(f"expected six routines, found {len(result)}")
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("hammerdb", type=Path)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    revision = subprocess.check_output(["git", "-C", str(args.hammerdb), "rev-parse", "HEAD"], text=True).strip()
    if revision != COMMIT:
        raise ValueError(f"expected HammerDB {COMMIT}, got {revision}")
    path = args.hammerdb / "src/postgresql/pgoltp.tcl"
    text = path.read_text()
    procedure_start = text.index('        if { $pg_storedprocs eq "true" } {')
    procedure_end = text.index('        } else {', procedure_start)
    function_start = text.index('            set sql(1) { CREATE OR REPLACE FUNCTION', procedure_end)
    function_end = text.index('        if { $citus_compatible eq "true" } {', function_start)
    header = (
        "-- HammerDB v6.1, GPL-3.0; see LICENSE and NOTICE.\n"
        f"-- Source commit: {COMMIT}\n"
        f"-- Source sha256: {hashlib.sha256(path.read_bytes()).hexdigest()}\n\n"
    )
    for name, start, end in (
        ("procedures", procedure_start, procedure_end),
        ("functions", function_start, function_end),
    ):
        sql = header + "--+ routines\n"
        for routine, body in zip(("dbms_random", "neword", "delivery", "payment", "ostat", "slev"), routines(text, start, end)):
            sql += f"--= {routine}\n{body}\n\n"
        (args.output / f"{name}.sql").write_text(sql.rstrip() + "\n")
    start = text.index('        set sql(1) "CREATE TABLE CUSTOMER (C_SINCE')
    end = text.index('        if { $citus_compatible', start)
    statements = re.findall(r'set sql\(\d+\) "(CREATE TABLE [^"\n]+)"', text[start:end])
    statements = [sql for sql in statements if "PARTITION BY" not in sql]
    if len(statements) != 9:
        raise ValueError(f"expected nine tables, found {len(statements)}")
    sql = header + "--+ create_schema\n"
    for statement in statements:
        sql += f"--= {statement.split()[2].lower()}\n{statement}\n\n"
    sql += "--+ create_indexes\n"
    start = text.index('proc CreateIndexes')
    end = text.index('proc gettimestamp', start)
    for statement in re.findall(r'set sql\(\d+\) "(CREATE UNIQUE INDEX [^"\n]+)"', text[start:end]):
        sql += f"--= {statement.split()[3].lower()}\n{statement}\n\n"
    sql += "--+ analyze\n--= analyze\nANALYZE\n"
    (args.output / "schema.sql").write_text(sql)


if __name__ == "__main__":
    main()

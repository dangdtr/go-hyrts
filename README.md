# go-hyrts

Hybrid regression test selection for Go modules: after a change, run only the
tests that can be affected by it.

```sh
go install github.com/dangdtr/go-hyrts@latest

hyrts collect                 # once (e.g. nightly): run every test, record its dependencies
hyrts select -explain         # what would run, and why
hyrts run -- -race -count=1   # run only the affected tests (flags after -- go to go test)
```

## How it works

`hyrts collect` runs each top-level test (`Test*`, `Fuzz*`, `Example*`) once,
in its own process, from a test binary built with `-cover -coverpkg=<module>/...`.
For every test it records:

- **the functions it executed**, from the coverage profile (dynamic, so it
  follows interfaces, closures, goroutines and reflection);
- **the files and directories it opened or stat'ed and the environment
  variables it read**, from the `testing` package's own access log
  (`-test.testlogfile`, the log `go test` uses for its cache), so a test that
  reads `testdata/`, `migrations/` or `deploy/*.yaml` at run time depends on
  them;
- whether it passed, failed or skipped, and how long it took.

It also snapshots the module: a checksum of every file and, for Go files, of
every function and declaration. Checksums are taken over the token stream, so
comments and formatting are not changes.

`hyrts select` compares the current module with that snapshot (no git needed;
uncommitted changes count) and applies these rules:

| Change | Selected |
|---|---|
| function or method body changed or removed | tests that executed it |
| test function changed / added | that test |
| test helper, `TestMain` or test-file declaration changed | every test of that package |
| type, var, const or blank import changed/removed; var with initializer, `init`, or method on an existing type added | every test whose binary links the package (static fallback) |
| new function, new type and its methods | nothing (only changed code can call them) |
| non-Go file | tests that opened it or a directory containing it; every linker if it is embedded or compiled into a package |
| `go.mod`, `go.sum`, `go.work`, `vendor/`, Go version, build tags | everything |
| test that failed or skipped during collection, or is new | always |

Selection is always against the collected snapshot, not the previous run, so
consecutive changes stay safe; precision only decreases as the snapshot ages.
Re-run `hyrts collect` regularly (nightly, or after merges to main).

`hyrts run` runs everything selected in one `go test` invocation, keeping the
go command's package parallelism.

## Limits

- Code executed in **another process** (a binary the test builds and execs, a
  container) is not in the coverage. Use `-always 'pkg\.TestName'` for such tests.
- Behaviour that depends on **unrecorded state** (time, network, database
  contents written by other tests) is outside any selection technique.
- Tests that read files outside the module are not tracked.

The database lives in `.hyrts/db.json` (git-ignored automatically).

## History

The first version (2024) was a thesis prototype matching test call sites to
changed function names. This rewrite keeps the hybrid idea from HyRTS
(Zhang, ICSE 2018: method-level dynamic dependencies, file-level fallback) and
replaces the analysis with per-test coverage, runtime file tracking and a
static package-graph fallback.

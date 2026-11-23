# go-hyrts

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.22-blue.svg)](https://golang.org)
[![Build Status](https://github.com/dangdtr/go-hyrts/actions/workflows/go.yml/badge.svg)](https://github.com/dangdtr/go-hyrts/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/dangdtr/go-hyrts)](https://goreportcard.com/report/github.com/dangdtr/go-hyrts)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**go-hyrts** is a **Hybrid Regression Test Selection (RTS)** tool for Go modules. After a change, it automatically selects and runs **only the tests that can be affected by that change**, drastically shortening the developer feedback loop.

---

## ⚡ Quick Benchmark

In real-world testing on a complex production service (`shield`):

| Action | Standard `go test ./...` | `hyrts run -update` | Speedup |
| :--- | :---: | :---: | :---: |
| **1-function change** | **120s** *(2m00s)* | **8.5s** | **~14x faster** |

*Note: The 8.5s above is end-to-end, including running the affected tests and incrementally re-collecting their dependencies.*

---

## 💡 Why go-hyrts?

The standard Go test cache operates at the **package level**:
- If package `A` changes, every test in `A` and every test in packages importing `A` must re-run.
- If package `A` has 200 tests and you only modified 1 private helper function used by 2 tests, standard Go runs all 200 tests.

**go-hyrts** solves this at the **function and method level**:
- It tracks which tests actually executed which functions at runtime.
- Modifying a single function runs **only the tests that called that function**.
- Updating comments, documentation, or running `gofmt` triggers **zero tests**.

---

## ✨ Key Features

- **Dynamic Per-Test Coverage (`-cover -coverpkg`)**: Traces runtime execution precisely, effortlessly following interfaces, dynamic dispatch, closures, goroutines, and reflection.
- **Runtime Resource Tracking (`-test.testlogfile`)**: Leverages Go's internal test log to record accessed files (`testdata/`, migrations, YAML) and environment variables. If a test fixture file changes, only tests reading that file are re-run.
- **Token-Level AST Checksums**: Checksums are computed over the Go token stream. Modifying comments, whitespace, or formatting never triggers a test run.
- **Incremental Collection (`-update`)**: After running selected tests, only those tests are re-collected and merged into the database. Unselected tests retain their recorded dependencies safely.
- **Concurrent Pipeline**: Builds test binaries in parallel (slowest packages first) while concurrently executing tests in finished binaries.
- **Zero External Dependencies**: Built entirely with Go's standard library (`go/parser`, `go/ast`, `os/exec`, `crypto/sha256`).

---

## 🚀 Installation

Requires Go 1.22 or later.

```sh
go install github.com/dangdtr/go-hyrts@latest
```

---

## 📖 Everyday Workflow

### 1. Initial Collection (Run Once)
Run every test once and record per-test function and file dependencies into `.hyrts/db.json`:
```sh
hyrts collect
```
*Tip: Speed up collection on multi-core machines by increasing package concurrency:*
```sh
hyrts collect -j 8 -p 4
```

### 2. Inspect Affected Tests
After making changes to your code:
```sh
hyrts select -explain
```
Sample output:
```text
hyrts: 1 changed files, selected 2/184 tests in 1 packages (~0.4s of 45.2s, 1%) in 12ms
  example.com/demo/calc.TestAdd
      executes Add changed (calc/calc.go)
  example.com/demo/app.TestSum
      executes Add changed (calc/calc.go)
```

### 3. Run Affected Tests & Update
Run only the selected tests, passing flags to `go test` after `--`:
```sh
hyrts run -- -race -count=1
```

To run the selected tests and **immediately update the database** with their new dependencies:
```sh
hyrts run -update -- -race -count=1
```

---

## 🛠 Selection Rules

`hyrts select` compares the current module against the recorded snapshot (no `git commit` required; dirty working trees are fully supported):

| Code Change | Tests Selected |
| :--- | :--- |
| **Function or method body changed/removed** | Only tests that actually executed that function/method |
| **Test function modified or added** | Only that specific test |
| **Test helper, `TestMain`, or `_test.go` declaration changed** | All tests in that package |
| **Type, const, package var, `init()`, or blank import changed** | All tests linking that package (safe static fallback) |
| **New function or new type added** | **None** (unmodified existing code cannot call it yet) |
| **Non-Go file changed (`testdata/*`, `.sql`, `.yaml`)** | Tests that opened/stat'ed the file or its directory |
| **`go.mod`, `go.sum`, `go.work`, Go version, build tags** | **All tests** |
| **Test that previously failed or skipped** | **Always selected** |

---

## ⚙️ CLI Reference

### Common Flags
- `-C <dir>`: Module directory (default: `.`)
- `-db <path>`: Database location (default: `<module>/.hyrts/db.json`)
- `-tags <list>`: Build tags, identical to `go test -tags`

### `hyrts collect`
- `-update`: Re-collect only the tests affected since the last collect (incremental mode).
- `-j <n>`: Maximum concurrent test processes across all packages (default: `GOMAXPROCS`).
- `-p <n>`: Maximum concurrent test processes within one package (default: `1`, safe for shared DB fixtures).
- `-timeout <duration>`: Per-test execution timeout (default: `10m`).
- `-pkg <list>`: Comma-separated list of import paths to collect (default: all).

### `hyrts select` / `hyrts run`
- `-explain`: Print the exact reason why each test was selected.
- `-json`: Output the selection report in JSON format (ideal for CI scripting).
- `-always <regex>`: Regex pattern matching `importpath.TestName` to always run (e.g. for blackbox tests).
- `-n`: Dry-run mode; print the generated `go test` command without executing it.
- `-update`: *(run only)* After tests pass, re-collect the selected tests and update `.hyrts/db.json`.
- `-- <flags>`: Any flags following `--` are forwarded directly to `go test` (e.g. `-- -race -v -count=1`).

---

## 🔄 CI/CD Integration

You can cache `.hyrts/db.json` across CI runs to dramatically accelerate pull request checks:

```yaml
# Example: GitHub Actions
- name: Restore hyrts database cache
  uses: actions/cache@v4
  with:
    path: .hyrts/db.json
    key: hyrts-${{ runner.os }}-${{ hashFiles('go.sum') }}-${{ github.base_ref }}
    restore-keys: |
      hyrts-${{ runner.os }}-${{ hashFiles('go.sum') }}-

- name: Run affected tests
  run: |
    if [ ! -f .hyrts/db.json ]; then
      hyrts collect -j 4 -p 2
    else
      hyrts run -update -- -race
    fi
```

---

## ⚠️ Known Limitations

- **External Processes**: Code executed in separate binaries (e.g. `exec.Command("./cli")` or external Docker containers) cannot be traced by Go's in-process coverage. Use `-always <regex>` for those tests.
- **Unrecorded Non-Deterministic State**: Dependencies on system clocks (`time.Now()`), network sockets, or unisolated database states written by external services cannot be tracked.
- **External Files**: Files read outside of the Go module root are not monitored.

---

## 📜 Background & History

- The original version (2024) was a thesis prototype that used static AST inspection to match test call sites to modified function names.
- This modern rewrite preserves the hybrid principle from **HyRTS** (*Zhang et al., ICSE 2018: Hybrid Regression Test Selection*) and implements it natively for Go using dynamic per-test coverage, `-test.testlogfile` runtime tracing, token-level checksums, and static package graph fallback.

---

## 📄 License

MIT License. See [LICENSE](LICENSE) for details.

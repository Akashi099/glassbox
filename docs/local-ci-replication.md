# Local CI Replication

This document explains how to replicate CI checks on your machine before
pushing, what each check does, and how to interpret failures.

The entry point is `scripts/test-ci-locally.sh`. It runs the same Go and Rust
checks that the CI pipeline runs, in the same order, so a clean local run is a
strong signal that CI will pass.

---

## Prerequisites

The following tools must be on your `PATH`:

| Tool | Purpose |
|---|---|
| `go` | Go compiler and toolchain |
| `gofmt` | Go source formatter (bundled with Go) |
| `cargo` | Rust toolchain (required only if the `simulator/` directory is present) |

The script checks for each tool at startup and exits with an error if any are
missing.

---

## Running the full check suite

From the repository root:

```bash
bash scripts/test-ci-locally.sh
```

A passing run ends with:

```
[OK] All CI checks passed! Safe to push.
```

Exit code is `0` on success, `1` on any failure.

---

## What the script checks

### Go checks (always run)

| Step | Command | What it validates |
|---|---|---|
| Dependency verification | `go mod verify` | Module download checksums match `go.sum` |
| Formatting | `gofmt -l` over `*.go` (excluding `simulator/target/`, `.gocache/`, `vendor/`) | No unformatted Go source files |
| Static analysis | `go vet ./...` | Common correctness issues |
| Build | `go build -v ./...` | All packages compile |
| Binary build | `go build -o Glassbox ./cmd/glassbox` | Integration-test binary builds |
| Unit tests | `go test -v -race ./...` | All Go tests pass with the race detector |

### Rust checks (run only when `simulator/` exists)

| Step | Command | What it validates |
|---|---|---|
| Formatting | `cargo fmt --check` | No unformatted Rust source files |
| Linting | `cargo clippy --all-targets --all-features -- -D warnings` | No Clippy warnings |
| Tests | `cargo test --verbose` | All Rust tests pass |
| Build | `cargo build --verbose` | Simulator compiles |

---

## Fixing common failures

### Formatting failure (Go)

```
[FAIL] The following Go files are not formatted:
./internal/session/session.go
Run 'go fmt ./...' to fix.
```

Run:

```bash
go fmt ./...
```

Then re-run the script. The check excludes `simulator/target/`, `.gocache/`,
and `vendor/` automatically.

### Formatting failure (Rust)

```
[FAIL] Rust files are not formatted. Run 'cargo fmt' to fix.
```

Run:

```bash
cd simulator && cargo fmt
```

### Dependency verification failure

```
verifying github.com/some/module@v1.2.3: checksum mismatch
```

Your local module cache may be stale or corrupted. Try:

```bash
go clean -modcache
go mod download
go mod verify
```

### Build failure

Compilation errors appear inline from `go build` or `cargo build`. Fix the
errors, then re-run the script.

### Test failure

Failed tests print directly to stdout from `go test -v`. Note the test name
and package, then run that test in isolation for a tighter feedback loop:

```bash
go test -v -run TestYourTestName ./internal/your/package/...
```

For Rust:

```bash
cd simulator && cargo test your_test_name -- --nocapture
```

---

## Relationship to CI

The local script mirrors the Go and Rust steps in the `ci-test.yml` workflow.
It does not replicate TypeScript tests (run `npm ci && npm test` separately),
mutation tests, or release-signing steps — those are covered by their own
workflows. For CI artifact retrieval and failure reproduction from a downloaded
artifact, see [CI Artifacts Guide](ci-artifacts.md).

---

## Make targets

The Makefile provides convenience wrappers for individual CI steps if you want
to run them without the full script:

```bash
make lint          # golangci-lint run
make lint-strict   # golangci-lint with zero-issue tolerance + go vet
make test          # go test ./...
make build         # go build ./cmd/glassbox
make rust-test     # cargo test
make rust-lint     # cargo clippy
```

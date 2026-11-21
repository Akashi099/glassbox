# Strict Linting Escalation

This document describes the strict linting rules for Go and Rust, the Make
targets and scripts that enforce them, and how to interpret and fix failures.

---

## Overview

Glassbox runs two levels of linting:

| Level | Go command | Rust command | Tolerance |
|---|---|---|---|
| Standard | `golangci-lint run` | `cargo clippy --all-targets --all-features` | Default issue limits |
| Strict | `golangci-lint run --config=.golangci.yml --max-issues-per-linter=0 --max-same-issues=0` | `cargo clippy … -D warnings -D clippy::all -D unused-variables …` | Zero issues; any warning is a failure |

Strict linting is used in CI and before merging to the main branch.

---

## Go strict linting

### Configuration

The linter configuration lives in `.golangci.yml`. The active linters are:

| Linter | What it catches |
|---|---|
| `ineffassign` | Assignments whose value is never used |
| `govet` | Suspicious Go constructs (misuse of `sync/atomic`, struct tag mistakes, etc.) |
| `staticcheck` | A broad set of correctness and performance issues |

The `run` section sets a 5-minute timeout and `readonly` module download mode
so the linter does not modify `go.sum` during CI.

### Running strict Go linting

```bash
# Via Make (recommended):
make lint-strict

# Via script:
bash scripts/lint-strict.sh

# Directly:
golangci-lint run --config=.golangci.yml --max-issues-per-linter=0 --max-same-issues=0
go vet ./...
```

A passing run ends with:

```
Strict linting passed
```

If `golangci-lint` is not installed, `scripts/lint-strict.sh` falls back to
`go vet ./...` only and prints a warning.

### Failure output example

```
internal/session/session.go:42:7: ineffassign: assignment to `err` is ineffective (ineffassign)
```

Fix the issue reported by the linter, then re-run. With
`--max-issues-per-linter=0 --max-same-issues=0` there is no suppression of
repeated issues — every occurrence must be resolved.

### Verifying strict linting detects violations

`scripts/test-strict-linting.sh` confirms that the linter is correctly
configured to catch violations. It creates a temporary Go module with a
deliberate unused-variable issue and asserts that `golangci-lint` reports it:

```bash
bash scripts/test-strict-linting.sh
```

Expected output:

```
[OK] Strict linting caught unused variable
Strict linting verification passed
```

If the script prints `[FAIL] Strict linting failed to catch unused variable`,
the linter is either not installed or not configured to catch `ineffassign`.
Check that `.golangci.yml` enables `ineffassign` and that `golangci-lint` is
at a compatible version for golangci-lint v2.

---

## Rust strict linting

### Configuration

Strict Rust linting is configured entirely via Clippy flags (no separate config
file). The flags used are:

| Flag | Effect |
|---|---|
| `-D warnings` | Treat all warnings as errors |
| `-D clippy::all` | Enable and deny all Clippy lints |
| `-D unused-variables` | Deny unused variables |
| `-D unused-imports` | Deny unused imports |
| `-D unused-mut` | Deny `mut` on bindings that are never mutated |
| `-D dead-code` | Deny unreachable code |
| `-D unused-assignments` | Deny assignments whose value is never read |
| `-W clippy::pedantic` | Warn on pedantic lints |
| `-W clippy::nursery` | Warn on experimental lints |

### Running strict Rust linting

```bash
# Via Make:
make rust-lint-strict

# Directly (from repo root):
cd simulator && cargo clippy --all-targets --all-features -- \
  -D warnings \
  -D clippy::all \
  -D unused-variables \
  -D unused-imports \
  -D unused-mut \
  -D dead-code \
  -D unused-assignments \
  -W clippy::pedantic \
  -W clippy::nursery
```

A passing run ends with:

```
 Strict Rust linting passed
```

### Failure output example

```
error[unused_variables]: unused variable: `result`
  --> simulator/src/runner.rs:88:9
   |
88 |         let result = compute();
   |             ^^^^^^ help: if this is intentional, prefix it with an underscore: `_result`
```

**Fix:** either use the variable, rename it with an `_` prefix if intentional,
or remove it.

---

## Running all strict linting (Go + Rust)

```bash
make lint-all-strict
```

This runs `lint-strict` (Go) followed by `rust-lint-strict` (Rust) and exits
non-zero if either fails.

---

## Standard linting

If you only want to run the standard (non-strict) linter before local commits:

```bash
# Go:
make lint           # golangci-lint run (default limits)

# Rust:
make rust-lint      # cargo clippy --all-targets --all-features (warnings allowed)
```

---

## Escalation rules summary

| Trigger | Action |
|---|---|
| Any `golangci-lint` finding in CI | Build fails; must fix before merge |
| Any Clippy warning under `-D warnings` | Build fails; must fix before merge |
| `golangci-lint` not installed locally | Falls back to `go vet`; CI always uses the full linter |
| Linter version mismatch | Pin `golangci-lint` to the version specified in `Makefile` |

To check the pinned version:

```bash
grep GOLANGCI_LINT_VERSION Makefile
```

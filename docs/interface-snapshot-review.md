# Interface Snapshot Review

This document covers two related checks: the simulator interface validation
(`scripts/validate-interface.sh`) and the API compatibility snapshot tool
(`scripts/api-snapshot.sh`). Together they guard against unintended changes to
the public-facing contract of the simulator and CLI.

---

## Simulator interface validation

`scripts/validate-interface.sh` checks that the `RunnerInterface` contract in
`internal/simulator/runner.go` is correctly defined, implemented, and
test-covered.

### What it checks

| Check | Location | Expected value |
|---|---|---|
| Interface defined | `internal/simulator/runner.go` | `type RunnerInterface interface` present |
| `Run` method signature | `internal/simulator/runner.go` | `Run(req *SimulationRequest) (*SimulationResponse, error)` |
| Compile-time guard | `internal/simulator/runner.go` | `var _ RunnerInterface = (*Runner)(nil)` present |
| `NewDebugCommand` accepts interface | `internal/cmd/debug.go` | `func NewDebugCommand(runner simulator.RunnerInterface)` |
| `MockRunner` defined in tests | `internal/cmd/debug_test.go` | `type MockRunner struct` present |
| Backward-compatibility cobra command | `internal/cmd/debug.go` | `var debugCmd = &cobra.Command` present |

### Running the check

```bash
# Via Make:
make validate-interface

# Directly:
bash scripts/validate-interface.sh
```

A passing run ends with:

```
All validation checks passed!
```

Exit code is `0` on success, `1` on any failure.

**Failure output example:**

```
[FAIL] Compile-time interface check missing
```

**Fix:** add the missing compile-time guard to `internal/simulator/runner.go`:

```go
var _ RunnerInterface = (*Runner)(nil)
```

---

## API compatibility snapshots

`scripts/api-snapshot.sh` captures the full public API surface and fails CI
when that surface changes unexpectedly. The snapshot covers TypeScript exports,
CLI help output, Go package symbols, and JSON schemas.

For the full reference, see [API Compatibility Checks](api-compatibility.md).
This section focuses on the CLI snapshot files that the interface review uses.

### Snapshot files

| File | Contents |
|---|---|
| `.api-snapshots/cli-root.txt` | `glassbox --help` output (normalised) |
| `.api-snapshots/cli-debug.txt` | `glassbox debug --help` output |
| `.api-snapshots/cli-session.txt` | `glassbox session --help` output |
| `.api-snapshots/cli-audit:sign.txt` | `glassbox audit:sign --help` output |
| `.api-snapshots/cli-audit:verify.txt` | `glassbox audit:verify --help` output |
| `.api-snapshots/cli-cache.txt` | `glassbox cache --help` output |
| `.api-snapshots/cli-version.txt` | `glassbox version --help` output |
| `.api-snapshots/cli-protocol:register.txt` | `glassbox protocol:register --help` output |
| `.api-snapshots/cli-protocol:handle.txt` | `glassbox protocol:handle --help` output |
| `.api-snapshots/ts-main.txt` | Exported symbols from `src/index.ts` |
| `.api-snapshots/ts-browser.txt` | Exported symbols from `src/audit/browser/index.ts` |
| `.api-snapshots/go-*.txt` | Go public symbols per package |
| `.api-snapshots/schemas.txt` | SHA-256 checksums of `docs/schema/` files |

CLI help text is normalised before storage: version strings are replaced with
`X.X.X` and binary paths are stripped, so snapshot diffs reflect only
meaningful flag and subcommand changes.

### Generating snapshots

Snapshots must be generated from a built binary:

```bash
# Build the binary first
go build -o glassbox ./cmd/glassbox

# Generate all snapshots
scripts/api-snapshot.sh generate
```

The script writes to `.api-snapshots/` and prints a line per file:

```
[OK]  cli-root: 42 help lines
[OK]  cli-debug: 38 help lines
...
[OK]  Snapshots written to .api-snapshots/
```

### Checking for regressions

```bash
scripts/api-snapshot.sh check
```

The check command regenerates snapshots into a temp directory, diffs each file
against the stored version, and exits `1` if any differ:

```
[FAIL] cli-debug.txt: API changed — diff:
--- stored
+++ current
-  --trace-output string   write trace to this path
+  --trace-out string      write trace to this path
```

### Accepting an intentional change

1. Make the intentional change to the interface.
2. Regenerate snapshots:
   ```bash
   go build -o glassbox ./cmd/glassbox
   scripts/api-snapshot.sh generate
   ```
3. Review the diff:
   ```bash
   git diff .api-snapshots/
   ```
4. Write a migration note in your PR description explaining what changed and
   what, if anything, consumers need to update.
5. Commit the updated snapshot files alongside the code change.

### Environment variables

| Variable | Default | Effect |
|---|---|---|
| `GLASSBOX_BIN` | `./glassbox` | Path to the binary for CLI snapshot generation |
| `SNAPSHOT_DIR` | `.api-snapshots` | Directory where snapshots are stored |

---

## Procedure for a clean checkout

The following sequence covers both checks from scratch:

```bash
# 1. Build the binary
go build -o glassbox ./cmd/glassbox

# 2. Validate the simulator interface
make validate-interface

# 3. Generate API snapshots
scripts/api-snapshot.sh generate

# 4. Verify snapshots match (should pass immediately after generate)
scripts/api-snapshot.sh check
```

Expected final output:

```
All validation checks passed!       ← from validate-interface
[OK]  All API snapshots match.      ← from api-snapshot.sh check
```

# Fuzz Crash and Timeout Triage Guide

This document provides instructions for triaging fuzz test failures, crashes, and timeouts.

## Crash Triage

### When a Crash is Detected

1. **Locate the Crash Input**

   When `go test -fuzz` finds a failure, it minimizes the input
   **automatically** before stopping. It then writes the result to
   `testdata/fuzz/<FuzzTargetName>/<hash>` inside the **package directory**,
   not the repository root. For example:
   ```bash
   ls internal/ipc/testdata/fuzz/FuzzUnmarshalSimulationRequest/
   ```
   The fuzzer's last lines name the file and print the exact command to
   re-run it:
   ```
   Failing input written to testdata/fuzz/FuzzUnmarshalSimulationRequest/<hash>
   To re-run:
   go test -run=FuzzUnmarshalSimulationRequest/<hash>
   ```

2. **Reproduce the Crash** (deterministic; no fuzzing engine)
   ```bash
   go test -run='^FuzzTargetName$/<hash>' -v ./path/to/package
   ```
   This fails with the same panic or `t.Fatal` every time. Running
   `-fuzz` again is **not** a reliable way to reproduce, because the engine
   may never hit the same input.

3. **Keep the Crash Input**

   The `<hash>` file is a Go corpus file. It starts with a
   `go test fuzz v1` line followed by one Go literal per fuzz argument. Commit
   it in place: plain `go test ./path/to/package` runs every file under
   `testdata/fuzz/<FuzzTargetName>/` as a regression case without fuzzing.
   See [Fixture maintenance](#fixture-maintenance).

4. **Add a Named Regression Test** (optional but recommended)

   Write a `TestRegression_*` that decodes the same bytes, as shown in
   [Adding Regression Tests](#adding-regression-tests).

5. **Analyze the Root Cause**
   - Review the stack trace
   - Check for buffer overflows, null pointer dereferences, or integer overflows
   - Verify input validation is sufficient
   - Check for resource exhaustion (memory, CPU)

6. **Fix the Issue**
   - Add proper input validation
   - Add bounds checking
   - Add error handling for edge cases
   - Consider adding fuzzing-specific assertions

7. **Verify the Fix**
   ```bash
   # The committed crash input must now pass
   go test -run='^FuzzTargetName$' -v ./path/to/package

   # Then fuzz again briefly to check the fix did not move the crash
   go test -run='^$' -fuzz='^FuzzTargetName$' -fuzztime=30s ./path/to/package
   ```
   There is no `make test-fuzz` target. CI runs the per-target commands in
   `.github/workflows/fuzz.yml`.

### Crash Classification

- **Critical**: Panic, segmentation fault, or memory safety violation
- **High**: Unhandled error that should be caught
- **Medium**: Timeout or resource exhaustion
- **Low**: Expected error for malformed input (not a crash)

## Timeout Triage

### When a Timeout Occurs

1. **Identify the Slow Path**
   ```bash
   # Run with verbose output to see where it hangs
   go test -v -fuzz=FuzzTargetName -fuzztime=10s ./path/to/package
   ```

2. **Profile the Fuzz Target**
   ```bash
   # Add CPU profiling to the fuzz target
   go test -cpuprofile=cpu.prof -fuzz=FuzzTargetName -fuzztime=10s ./path/to/package
   go tool pprof cpu.prof
   ```

3. **Add Execution Bounds**
   ```go
   // In the fuzz function, add timeout checks
   func FuzzTarget(f *testing.F) {
       f.Fuzz(func(t *testing.T, data []byte) {
           // Add early exit for large inputs
           if len(data) > 1_000_000 {
               return
           }
           
           // Add timeout check if needed
           // ...
       })
   }
   ```

4. **Optimize the Target**
   - Reduce algorithmic complexity
   - Add early validation to reject obviously invalid inputs
   - Limit recursion depth
   - Add size limits on parsed structures

5. **Adjust Fuzz Timeouts**
   ```bash
   # Increase timeout for complex targets
   go test -fuzz=FuzzTargetName -fuzztime=60s -timeout=120s ./path/to/package
   ```

### Timeout Classification

- **Critical**: Infinite loop or unbounded recursion
- **High**: Exponential time complexity
- **Medium**: Slow but bounded execution
- **Low**: Expected slowness for large inputs

## Regression Prevention

### Adding Regression Tests

When a crash or timeout is fixed, add a regression test:

```go
func TestRegression_Crash_IssueXXX(t *testing.T) {
    // Load the crash input from corpus
    crashInput := loadCrashInput("testdata/fuzz/corpus/target/crash_input.bin")
    
    // This should not panic
    result, err := ParseFunction(crashInput)
    
    // Expect an error for malformed input, but no panic
    if err == nil {
        t.Log("Input was unexpectedly valid")
    }
    _ = result
}
```

### Updating the Corpus

See [Fixture maintenance](#fixture-maintenance) below.

## Fixture maintenance

### Two different corpus locations

| Location | Read by `go test`? | Purpose |
|----------|--------------------|---------|
| `<package>/testdata/fuzz/<FuzzTargetName>/<file>` | **Yes**. Every file runs as a seed during `go test`, and during `-fuzz` | Executable regression inputs. Crash inputs go here |
| `testdata/fuzz/corpus/<area>/` (repo root) | **No**. No Go code loads this directory | Human-readable reference seeds and per-area `README.md` notes |

A file copied into `testdata/fuzz/corpus/<area>/` is never executed. To keep a
crash from regressing, the file must stay in the package-relative directory.

### Adding a crash input as a fixture

1. Leave the fuzzer's `<hash>` file where it was written:
   `<package>/testdata/fuzz/<FuzzTargetName>/<hash>`. You may rename it to a
   descriptive name such as `issue1234_truncated_header`. Go reads every file
   in that directory, whatever its name.
2. Confirm it is in valid corpus format. The first line must be exactly
   `go test fuzz v1`:
   ```bash
   head -1 internal/ipc/testdata/fuzz/FuzzUnmarshalSimulationRequest/issue1234_truncated_header
   ```
3. Confirm it fails **before** the fix and passes **after** it:
   ```bash
   go test -run='^FuzzUnmarshalSimulationRequest$' -v ./internal/ipc/
   ```
   Before the fix this ends in `--- FAIL: FuzzUnmarshalSimulationRequest/issue1234_truncated_header`
   and `FAIL`. After the fix it ends in `ok  github.com/dotandev/glassbox/internal/ipc`.
4. Add a line to the matching `testdata/fuzz/corpus/<area>/README.md`. Give
   the file path, the issue number, and a one-line description of the
   failure class.
5. Commit the corpus file, the fix, and the README line together.

Before committing, check that the input contains no secrets, real keys, or
real transaction data. Fuzz inputs are committed verbatim.

### Which targets `go test` can actually run

Go only discovers `Fuzz*` functions in `_test.go` files. The following files
define fuzz targets in **non-test** files, so `go test -list '^Fuzz'` does not
show them and `go test -fuzz=<name>` has nothing to fuzz:

- `internal/simulator/xdr_decoder_fuzz.go`
- `internal/decoder/decoder_fuzz.go`
- `internal/trace/event_decoder_fuzz.go`
- `internal/session/archive_fuzz.go`
- `internal/abi/wasm_fuzz.go`
- `internal/cmd/audit_verify_fuzz.go`
- `internal/demangle/demangle_fuzz.go`

Before you triage or add a fixture, check that the target is discoverable:

```bash
go test -list '^Fuzz' ./internal/ipc/
```

For a discoverable package this prints the target names and then `ok`, for
example `FuzzUnmarshalSimulationRequest`. For `./internal/abi/` it prints only
`ok`, with no names.

**Known baseline (pre-existing):** in `.github/workflows/fuzz.yml`, each fuzz
command ends in `|| true`, and a job fails only if its output log contains
the word `crash`. That means CI jobs for undiscoverable targets pass without
fuzzing anything. Separately, `internal/decoder`, `internal/trace`,
`internal/session`, and `internal/cmd` do not currently build. Their fuzz jobs
report `[build failed]`, which the `crash` grep does not catch. Packages
whose targets are discoverable and build today include `internal/ipc`,
`internal/dwarf`, `internal/wasmvalidate`, and `internal/simulator` (only
`FuzzFootprintDecode`).

## CI Integration

### Fuzz Failures in CI

When CI reports a fuzz failure:

1. **Download the Artifacts**
   - CI should save crash inputs as artifacts
   - Download the failing corpus

2. **Reproduce Locally**
   ```bash
   # CI artifacts hold the output log and profiles. If a crash file is
   # included, copy it into the package-relative corpus directory
   mkdir -p path/to/package/testdata/fuzz/FuzzTargetName/
   cp <ci_crash_file> path/to/package/testdata/fuzz/FuzzTargetName/
   go test -run='^FuzzTargetName$' -v ./path/to/package
   ```

3. **Triage Following the Steps Above**
   - Follow crash or timeout triage as appropriate
   - Document the findings

4. **Update CI Configuration**
   - If the failure is a false positive, add an exclusion
   - If it's a real bug, fix it and update the corpus

## Reporting

### Bug Report Template

```markdown
## Fuzz Failure Report

**Target**: FuzzTargetName
**Package**: path/to/package
**Severity**: Critical/High/Medium/Low

### Reproduction
```bash
go test -run='^FuzzTargetName$/<hash>' -v ./path/to/package
```

### Crash Input
Committed at: path/to/package/testdata/fuzz/FuzzTargetName/<hash>

### Stack Trace
```
[stack trace here]
```

### Root Cause
[description of the vulnerability]

### Fix
[description of the fix]

### Regression Test
[link to regression test]
```

## Best Practices

1. **Never Ignore Crashes**: All crashes must be investigated and fixed
2. **Minimize Before Analysis**: Always minimize crash inputs first
3. **Document Everything**: Keep records of all triage decisions
4. **Update Corpus**: Add all crash inputs to the corpus
5. **Add Regression Tests**: Ensure bugs don't reoccur
6. **Review Regularly**: Periodically review timeout thresholds
7. **Share Knowledge**: Document common patterns and solutions

## Resources

- [Go Fuzzing Documentation](https://go.dev/doc/fuzz/)
- [OSS-Fuzz Triage Guide](https://google.github.io/oss-fuzz/getting-started/bug-handling/)
- [Fuzzing Book](https://www.fuzzingbook.org/)

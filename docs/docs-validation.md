# Documentation Validation

This document describes the three checks run by `scripts/validate-docs.sh`,
how to invoke them, what each one does, and how to fix failures.

---

## Overview

`scripts/validate-docs.sh` runs three independent checks against the `docs/`
directory:

| Check | Flag | What it verifies |
|---|---|---|
| Deterministic build comparison | `--determinism` | Docs content is byte-identical across two back-to-back snapshot builds (after normalisation) |
| Broken internal link detection | `--links` | Every local Markdown link target exists on disk |
| Command-flag smoke check | `--flags` | Every `--flag` in a `glassbox …` code block matches a flag in the CLI snapshots |

---

## Running the checks

### All checks (default)

```bash
# Via Make:
make validate-docs

# Directly:
bash scripts/validate-docs.sh
```

A passing run ends with:

```
Result: all documentation validation checks passed.
```

Exit code is `0` on success, `1` on any failure.

### Individual checks

```bash
# Determinism only:
make validate-docs-determinism
# or:
bash scripts/validate-docs.sh --determinism

# Internal links only:
make validate-docs-links
# or:
bash scripts/validate-docs.sh --links

# Flag smoke check only:
make validate-docs-flags
# or:
bash scripts/validate-docs.sh --flags
```

---

## Check 1: Deterministic build comparison

The script builds a normalised snapshot of every `docs/*.md` file twice from
the same `SOURCE_DATE_EPOCH` and asserts the SHA-256 hashes match.

Before hashing, variable content is stripped:

| Pattern replaced | Replaced with |
|---|---|
| ISO 8601 timestamps (`2026-01-02T03:04:05Z`) | `TIMESTAMP` |
| Absolute paths under `/home/`, `/tmp/`, `/runner/` | `PATH_REDACTED` |
| `git describe --dirty` suffix (`-dirty`) | `-DIRTY` |
| Lines containing `Generated at:` or `generated_at` | (removed) |

**Failure output:**

```
  [FAIL] Documentation snapshots differ between builds — non-deterministic content detected
```

Followed by a `diff` of the first 50 diverging lines.

**Fix:** remove content that varies by run environment (absolute paths,
wall-clock timestamps, etc.) or add it to the normalisation patterns in
`scripts/validate-docs.sh`.

**Controlling `SOURCE_DATE_EPOCH`:**

```bash
# Use a specific epoch:
SOURCE_DATE_EPOCH=1700000000 bash scripts/validate-docs.sh --determinism

# Default: taken from the HEAD commit timestamp
SOURCE_DATE_EPOCH=$(git log -1 --format=%ct)
```

---

## Check 2: Broken internal link detection

The script scans all `docs/**/*.md` files for Markdown links of the form
`[text](target)` and checks that each local target file or directory exists.
HTTP/HTTPS links and anchor-only links (`#section`) are skipped — no network
access is required.

**Passing output example:**

```
  [PASS] No broken internal links found (checked 142 link(s))
```

**Failure output example:**

```
  [FAIL] Broken link in docs/session-reference.md: '../examples/session-fixture.json' → /path/to/examples/session-fixture.json (not found)
```

**Fix:** correct the link target, create the missing file, or remove the link.

---

## Check 3: Command-flag smoke check

The script reads CLI snapshot files from `.api-snapshots/cli-*.txt` to build
a known-flag set, then scans fenced code blocks in `docs/**/*.md` for lines
beginning with `glassbox `. Any `--flag` on those lines that is absent from
the known-flag set is reported as a failure.

**Prerequisite:** CLI snapshots must exist. Generate them with:

```bash
scripts/api-snapshot.sh generate
```

If the snapshot directory is missing, the check is skipped with:

```
  [SKIP] API snapshot directory not found: .api-snapshots — run: scripts/api-snapshot.sh generate
```

**Passing output example:**

```
  [INFO] Known flags from snapshots: 87
  [PASS] All flags in docs examples are recognised (checked 34 flag occurrence(s))
```

**Failure output example:**

```
  [FAIL] Unknown flag in docs/debug-command.md: '--legacy-trace'
  [INFO] Tip: run 'scripts/api-snapshot.sh generate' to regenerate snapshots after adding flags
```

**Fix options:**

- If the flag exists but the snapshot is stale: `scripts/api-snapshot.sh generate`
- If the flag was removed from the binary: update the doc example to use the
  current flag name
- To skip this check without deleting snapshots (e.g. during a build where the
  binary is unavailable): `SKIP_FLAG_CHECK=1 bash scripts/validate-docs.sh --flags`

---

## Environment variables

| Variable | Default | Effect |
|---|---|---|
| `SOURCE_DATE_EPOCH` | `git log -1 --format=%ct` | Epoch used in determinism check |
| `DOCS_DIR` | `docs` | Docs directory to scan |
| `SNAPSHOT_DIR` | `.doc-snapshots` | Temporary snapshot cache for determinism check |
| `GLASSBOX_BIN` | `bin/glassbox` | Binary path (used by flag check if needed) |
| `SKIP_FLAG_CHECK` | `0` | Set to `1` to skip the flag smoke check |

---

## CI integration

`make validate-docs` runs in CI as part of the documentation quality gate. All
three sub-checks run; a failure in any one fails the job. The check runs after
`scripts/api-snapshot.sh generate` so CLI snapshots are always current.

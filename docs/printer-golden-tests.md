# Printer Golden Tests

The Text, Markdown, JSON, and HTML trace printers can drift in *meaning* even
while each compiles and keeps its own isolated tests green — a field renamed in
one template, silently dropped in another, formatted differently in a third.
The printer golden tests catch this by rendering one set of canonical trace
fixtures through **every** supported output mode and comparing the results
against checked-in golden files.

## What is covered

| Piece | Location |
|-------|----------|
| Canonical fixtures | `internal/trace/golden_fixtures_test.go` |
| Golden tests | `internal/trace/printer_golden_test.go` |
| Golden files | `internal/trace/testdata/printer_golden/*.golden` |

**Output modes** (one golden file per fixture × mode):

- `terminal` — the interactive tree printer (`PrintExecutionTrace`, color off, width 100)
- `text` — plain-text export (`GenerateTracePlainTextWithOptions`)
- `markdown` — Markdown export (`GenerateTraceMarkdownWithOptions`)
- `html` — HTML export (`GenerateTraceHTMLWithOptions`)
- `json` — raw `ExecutionTrace` JSON embedded by the export envelope (the test marshals the trace directly)

**Fixtures** (one per semantic category from issue #547):

| Fixture | Exercises |
|---------|-----------|
| `calls` | Contract calls, host functions, auth steps, return values |
| `errors` | Failed calls, traps, final failure status |
| `source_locations` | Source file/line and GitHub links (plus an unmapped step) |
| `gas` | Cost annotations and per-component breakdowns |
| `comments` | Trace comments and session metadata |
| `empty` | Zero-step traces and their diagnostic messaging |

Reviewer-comment examples have separate golden fixtures under
`internal/trace/testdata/annotations/` (see
[trace-export-annotations.md](trace-export-annotations.md)); they are not part
of this canonical fixture set.

Beyond byte comparison, `TestPrinterGolden_SemanticConsistency` asserts that
each fixture's semantic tokens (hashes, function names, errors, cost values,
comments, …) survive into every mode that is supposed to carry them, and
`TestPrinterGolden_ExportPathCoversAllFormats` +
`TestPrinterGolden_NoOrphanGoldens` gate coverage: a format added to
`trace.SupportedExportFormats()` fails CI until it has a renderer and golden
files, and goldens orphaned by a renamed fixture fail CI until deleted.

## Running the tests

```bash
go test ./internal/trace -run TestPrinterGolden
```

## Updating goldens after an intentional formatting change

1. Make your printer/template change.
2. Regenerate the golden files:

   ```bash
   go test ./internal/trace -run TestPrinterGolden -update
   ```

3. Inspect `git diff internal/trace/testdata/printer_golden/` — **the golden
   diff is the review artifact**. Every changed line should be a change you
   intended; anything else is drift the tests just caught.
4. Commit the golden files together with the code change so reviewers see the
   output change alongside its cause.

Never hand-edit a `.golden` file; always regenerate with `-update`.

## Normalization policy

Golden comparison must be deterministic, so:

- **Fixtures pin all volatile fields at build time.** `AddState` stamps
  wall-clock timestamps on states and snapshots; the fixture builder
  (`newGoldenTrace`) overwrites them with fixed times derived from the
  canonical `2026-01-01T00:00:00Z` stub.
- **Comparison normalizes only line endings** (CRLF → LF), the one
  environment-volatile byte sequence left. `.golden` files are also pinned to
  LF via `.gitattributes`.

Do **not** widen normalization (e.g. scrubbing hashes or numbers): masking
stable fields hides exactly the drift these tests exist to catch. If you add a
fixture containing a genuinely volatile field, pin it in the builder instead.

## Golden output refresh checklist

This checklist covers every checked-in golden or snapshot set that has an
update flag. Each flag is defined only in its own package's test binary, so
always pass a **single package path**. `go test ./... -update` fails with
`flag provided but not defined: -update` in every package that does not
define the flag.

| Golden set | Location | Refresh command | Files |
|------------|----------|-----------------|-------|
| Trace printer goldens | `internal/trace/testdata/printer_golden/` | `go test ./internal/trace -run TestPrinterGolden -update` | `<fixture>_<mode>.golden` (6 fixtures × 5 modes = 30 files) |
| CLI help snapshots | `internal/cmd/testdata/help/` | `go test ./internal/cmd -run TestHelpSnapshot -update-help` | `<command>.txt`, listed in that directory's `README.md` |
| Shell completion snapshots | `internal/cmd/testdata/completion/` | `go test ./internal/cmd -run TestCompletionSnapshot -update` | `glassbox.bash`, `glassbox.zsh`, `glassbox.fish`, `glassbox.ps1` |

`internal/cmd` uses **two different flags**: `-update-help` for help
snapshots and `-update` for completion snapshots. Running
`-run TestHelpSnapshot -update` regenerates nothing. The help tests ignore
`-update`, so they still compare against the existing files.

### Steps

1. **Start from a clean tree.** Run `git status --short` and check that
   nothing under the golden directories is already modified.
2. **Confirm the failure is the one you expect.** Run the comparison without
   the update flag (for example `go test ./internal/trace -run TestPrinterGolden`).
   Every failing test should trace back to your intended change.
3. **Regenerate** with the refresh command from the table. Each rewritten file
   is logged (`updated snapshot: …` / `updated help snapshot: …` in
   `internal/cmd`). Update mode always passes, so it proves nothing on its own.
4. **Re-run without the flag.** The output must end in `ok`. This is the real
   check.
5. **Review the golden diff line by line**, for example with
   `git diff --stat internal/trace/testdata/printer_golden/` and then the full
   diff. Look for:
   - changes you did not intend (drift the tests just caught)
   - timestamps, absolute paths (`/home/…`, `/Users/…`), hostnames, or
     tokens. Fix these in the fixture builder. Do not hand-edit them out of the
     golden file.
   - CRLF line endings. `.gitattributes` pins only `*.golden` to LF. The
     `internal/cmd` `.txt`, `.bash`, `.zsh`, `.fish`, and `.ps1` snapshots
     are not pinned.
6. **Check for orphans.** If you renamed or removed a printer fixture,
   `TestPrinterGolden_NoOrphanGoldens` fails until you delete the old
   `.golden` files. The `internal/cmd` snapshots have no orphan check, so
   delete stale files by hand.
7. **Commit the goldens together with the code change** that caused them.

### Failure messages you will see

| Message | Meaning | Fix |
|---------|---------|-----|
| `help snapshot testdata/help/<name>.txt not found — run '…-update-help' to generate it` | No snapshot is checked in | Run the help refresh command, review, commit |
| `snapshot testdata/completion/glassbox.<ext> not found — run '…-update' to generate it` | Same, for completion | Run the completion refresh command |
| `completion script for <shell> is stale … Run: go test ./internal/cmd -run TestCompletionSnapshot_BASH -update` | The output changed | The suggested test name is upper-cased and matches **nothing**, because `-run` is case-sensitive. Use `-run TestCompletionSnapshot_Bash` (or `_Zsh`, `_Fish`, `_PowerShell`), or the whole-set command in the table |

### Known baseline (pre-existing)

- `internal/cmd/testdata/help/` and `internal/cmd/testdata/completion/`
  contain only `.gitkeep` (plus `README.md` in `help/`). On a clean checkout
  every `TestHelpSnapshot_*` and `TestCompletionSnapshot_*` test fails with the
  `not found` message until the snapshots are generated and committed.
- On current `main`, neither `go test ./internal/trace` nor
  `go test ./internal/cmd` builds (`[build failed]`). The cause is compile
  errors in packages they depend on, such as `internal/decoder` and
  `internal/demangle`. You cannot run any refresh command until those are
  fixed. Record this as the baseline rather than as a golden failure.

## Adding a fixture or output mode

- **New fixture**: add it to `goldenFixtures()` with `commonTokens` (must
  appear in every mode) and `exportTokens` (export formats only — the terminal
  printer intentionally omits source locations and annotations), run `-update`,
  and commit the new goldens. Follow the stub-value and secret-avoidance rules
  in [regression-test-guide.md](regression-test-guide.md).
- **New output mode**: add it to `trace.SupportedExportFormats()`, wire the
  export switch, add a case in `renderGoldenMode`, then `-update`. CI fails
  until all three are done.

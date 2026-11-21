# Release-note fragments

This directory holds pending release-note fragments. Each fragment is a single
TOML file describing one user-facing change. The `make changelog-generate`
target assembles them into `CHANGELOG.md` and the `make changelog-check` target
validates them in CI.

See `docs/changelog-fragments.md` for the full authoring guide.

---

## Validating fragments locally

Run the validator against all pending fragments before pushing:

```bash
make changelog-check
```

To validate a single file:

```bash
bash scripts/validate-fragments.sh changelog/fragments/597-api-compat-checks.toml
```

A passing run prints one `[OK]` line per fragment and exits 0:

```
[OK]  597-api-compat-checks.toml
[OK]  All fragments valid.
```

---

## Duplicate PR number detection

Each fragment's `pr` field must be unique across all pending fragments.
Reusing the same PR number produces an error and exits 1:

```
[FAIL] 598-another-change.toml: duplicate pr=597 already used in 597-api-compat-checks.toml

1 error(s) found. Fix fragments before releasing.
```

To resolve it, correct the `pr` field to match the actual GitHub PR number.

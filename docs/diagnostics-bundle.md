# Diagnostics Bundle (`doctor --bundle`)

## Overview

`glassbox doctor --bundle` generates a **portable, redacted diagnostics archive** that can be shared for support or issue reproduction without containing any private key, authentication token, or personally-identifying material.

The command runs the same checks as `glassbox doctor` and then writes a deterministic ZIP archive containing a `manifest.json` and a `README.txt`.

## Usage

```sh
# Generate a bundle in the OS temp directory (path is printed)
glassbox doctor --bundle

# Generate at a specific path
glassbox doctor --bundle --bundle-output ./glassbox-diag.gbdiag

# Combine with --verbose for richer path information in check results
glassbox doctor --verbose --bundle --bundle-output ./diag.gbdiag
```

## Archive Layout

```
glassbox-diag-YYYYMMDD-HHMMSS.gbdiag   (ZIP)
├── manifest.json    ← machine-readable diagnostics
└── README.txt       ← human-readable summary
```

### `manifest.json` Schema (version 1)

```jsonc
{
  "schema_version": 1,          // incremented on breaking layout changes
  "meta": {
    "glassbox_version": "1.2.3",
    "commit_sha": "abc123de",
    "build_date": "unknown",
    "go_version": "go1.26"
  },
  "platform": {
    "os": "linux",
    "arch": "amd64",
    "num_cpu": 8,
    "go_os": "linux",
    "go_arch": "amd64",
    "hostname": "myhost",       // first DNS label only (no domain)
    "protocol_registration_state": "registered"  // registered | not-registered | unknown
  },
  "config": {
    "rpc_url": "https://soroban-testnet.stellar.org",
    "network": "testnet",
    "log_level": "info",
    "request_timeout": 15,
    "max_trace_depth": 50,
    "failure_threshold": 5,
    "retry_timeout": 60,
    "failover_strategy": "",
    "telemetry": false,
    "crash_reporting": false,
    "rpc_token": "[REDACTED]",        // always redacted
    "crash_sentry_dsn": "[REDACTED]", // always redacted
    "crash_endpoint": "[REDACTED]",   // always redacted
    "cache_path": "~/.Glassbox/cache" // home dir replaced with ~
  },
  "checks": [
    {
      "id": "go",
      "name": "Go",
      "ok": true,
      "version": "go1.26.0",
      "path": "/usr/local/go/bin/go"
    },
    {
      "id": "simulator",
      "name": "Simulator Binary (glassbox-sim)",
      "ok": false,
      "fix_hint": "Build the simulator: cd simulator && cargo build --release"
    }
    // … one entry per doctor check
  ],
  "policy": {
    "collector_version": "1.0.0",
    "verbose_mode": false,
    "inventory": [
      {
        "name": "meta.glassbox_version",
        "classification": "safe",
        "description": "binary version string"
      },
      {
        "name": "platform.hostname",
        "classification": "masked",
        "description": "first hostname label only (domain stripped)"
      },
      {
        "name": "config.rpc_url",
        "classification": "masked",
        "description": "RPC URL; query string redacted"
      },
      {
        "name": "config.rpc_token",
        "classification": "redacted",
        "description": "always replaced with [REDACTED]"
      },
      {
        "name": "config.cache_path",
        "classification": "masked",
        "description": "home dir replaced with ~"
      }
    ]
  },
  "generated_at": "2026-07-24T12:00:00Z"
}
```

## Redaction Policy

The following values are **always replaced with `[REDACTED]`**:

| Category | Fields / Env vars |
|----------|------------------|
| RPC authentication | `rpc_token`, `GLASSBOX_RPC_TOKEN` |
| Crash reporting | `crash_sentry_dsn`, `crash_endpoint`, `GLASSBOX_SENTRY_DSN`, `GLASSBOX_CRASH_ENDPOINT` |
| Config passphrase | `GLASSBOX_CONFIG_PASSPHRASE` |
| Ed25519 private keys | Any 64-hex-character value |
| Stellar secret keys | Any `S`-prefixed 56-character base32 string |
| Filesystem paths | Home directory prefix replaced with `~` |

## Acceptance Criteria

- The bundle can be generated **offline** (no network requests).
- The archive **contains no private key or token material**.
- The archive has a **manifest with a schema version**.
- The archive is **readable on another machine** (standard ZIP format, `.gbdiag` or `.zip` extension).
- Tests assert redaction of representative environment values (`GLASSBOX_RPC_TOKEN`, `GLASSBOX_SENTRY_DSN`).

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.gbdiag` | Canonical Glassbox diagnostics archive |
| `.zip` | Accepted for interoperability with generic ZIP tools |

Any other extension is rejected with an exit code 1 and error message:
```
unsupported diagnostics archive extension ".tar.gz" — use .gbdiag or .zip
```

---

## Doctor Dependency Checks Inventory

Every diagnostics bundle captures the exact health status and diagnostic details of development dependencies:

| Check ID | Component Name | Description & Verification |
|----------|----------------|----------------------------|
| `go` | Go Toolchain | Verifies Go is installed, matches the minimum required version in `go.mod` (Go 1.26), and reports binary path. |
| `rust` | Rust Compiler | Verifies `rustc` compiler availability and active toolchain edition. |
| `cargo` | Cargo Package Manager | Verifies `cargo` executable presence for compiling simulator components. |
| `simulator` | Simulator Binary (`glassbox-sim`) | Checks if the native Soroban simulation engine binary is present in `~/.glassbox/cache` or system `PATH`. |
| `cache_dir` | Cache Directory | Checks read/write permissions for `~/.glassbox/cache`. |
| `protocol_registry` | Protocol Registry | Checks if the `glassbox://` custom desktop protocol is registered on the OS. |
| `go_mod` | Go Module Dependencies | Verifies all packages in `go.mod` and `go.sum` are downloaded and verified. |
| `config_toml` | Configuration File | Validates syntax and schema correctness of `glassbox.toml`. |
| `rpc` | Stellar RPC Endpoint | Probes the configured Soroban RPC URL for reachability and protocol version. |
| `deep_link` | Desktop Deep Linking | Confirms system URL handler binding for IDE integration. |
| `telemetry_queue` | Telemetry Spool | Checks size and dispatch status of local offline telemetry events. |

---

## Embedded `README.txt`

Every `.gbdiag` archive includes a human-readable `README.txt` accompanying `manifest.json`:

```text
Glassbox Diagnostics Archive
============================

Generated by: glassbox doctor --bundle
Glassbox version: 1.2.3 (d7da2ca)
Generated at:     2026-11-20T18:45:00Z

Contents
--------
  manifest.json  — machine-readable diagnostics (schema version, platform,
                   config shape, and doctor check results)
  README.txt     — this file

Privacy
-------
Sensitive values (RPC tokens, private keys, Sentry DSN, crash endpoints)
have been replaced with "[REDACTED]".  Home-directory paths appear as ~.

This archive contains NO private key or token material.
```

---

## Offline Inspection & Verification Procedure

Recipients can inspect and verify bundle contents using standard tools without network connectivity:

```bash
# 1. View archive member list
unzip -l glassbox-diag-20261120-184500.gbdiag

# 2. Print human-readable README summary
unzip -p glassbox-diag-20261120-184500.gbdiag README.txt

# 3. View machine-readable manifest (pretty-printed JSON)
unzip -p glassbox-diag-20261120-184500.gbdiag manifest.json | jq .

# 4. Verify privacy contract: confirm zero secrets or unmasked paths remain
unzip -p glassbox-diag-20261120-184500.gbdiag manifest.json | grep -iE 'tok_|sentry\.io|password|secret'

# 5. Inspect the redaction inventory and policy
unzip -p glassbox-diag-20261120-184500.gbdiag manifest.json | jq '.policy.inventory'
```

---

## Related Documentation

- [Audit Bundle Contents](audit-bundle-contents.md) — portable artifact definitions (debug bundles, session archives, and diagnostics bundles)
- [Repository Health Report](health-report.md) — code hygiene and stale suppression scanning via `scripts/health-report.sh`

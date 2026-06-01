# Telemetry Metadata

Glassbox includes environment metadata in telemetry events to help diagnose usage patterns and troubleshoot issues.

## Environment Metadata

When telemetry is enabled (non-anonymized mode), the following metadata is included in command usage events:

| Field | Description | Example |
|-------|-------------|---------|
| `env.version` | CLI version | `"1.2.3"` or `"dev"` |
| `env.platform` | Operating system | `"linux"`, `"darwin"`, `"windows"` |
| `env.arch` | CPU architecture | `"amd64"`, `"arm64"` |
| `env.feature_flags` | Enabled feature flags | `["telemetry", "failover"]` |

## Privacy

Sensitive values are automatically excluded from telemetry:

- Configuration values containing tokens, passwords, or keys are redacted
- Transaction hashes are fingerprinted (hashed client-side) rather than transmitted raw
- File paths are reduced to basename only
- Long strings are truncated to 128 characters

## Anonymized Mode

When `--telemetry-anonymized` is enabled (or `GLASSBOX_TELEMETRY_ANONYMIZED=true`), environment metadata is not included in telemetry events. Only the command name is recorded.

## Enabling Telemetry

```bash
# Enable basic telemetry
glassbox --telemetry

# Enable with anonymization
glassbox --telemetry --telemetry-anonymized
```

Or via environment variables:

```bash
GLASSBOX_TELEMETRY=true
GLASSBOX_TELEMETRY_ANONYMIZED=false
```

## Feature Flags

The following feature flags are currently tracked:

- `telemetry` - When telemetry is enabled
- `crash_reporting` - When crash reporting is enabled
- `failover` - When multi-RPC failover is configured
- `multi_rpc` - When multiple Soroban RPC URLs are configured
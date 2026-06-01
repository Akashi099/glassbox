# Config Show Command

The `config show` command displays the effective runtime configuration as read from flags, environment variables, and config files.

## Usage

```bash
# Human-readable output
glassbox config show

# JSON output with source annotations
glassbox config show --json
```

## Output

### Human-Readable Format

```
Active configuration:
{
  "rpc_url": "https://...",
  "network": "testnet",
  ...
}

Loaded from: /home/user/.glassbox/config.toml

Configuration sources:
  ✓ Environment variables (active)
  ✓ User config (active)
```

### JSON Format

```json
{
  "values": {
    "rpc_url": {
      "value": "https://test.example.com",
      "source": "environment"
    },
    "network": {
      "value": "testnet",
      "source": "file"
    }
  },
  "source_file": "/home/user/.glassbox/config.toml"
}
```

## Configuration Precedence

Configuration is resolved in the following order (highest wins):

1. CLI flags
2. Environment variables (`GLASSBOX_*`)
3. Repository-local config (`.glassbox.toml` or `.Glassbox.toml`)
4. Home directory config (`~/.glassbox/config.toml` or `~/.Glassbox.toml`)
5. Built-in defaults

## Config File Locations

The following locations are searched (first match wins):

- `.glassbox.toml` (current directory)
- `.Glassbox.toml` (current directory)
- `~/.glassbox/config.toml` (home directory, XDG-style)
- `~/.Glassbox.toml` (home directory, legacy)
- `/etc/Glassbox/config.toml` (system-wide)

## Source Annotations

In JSON mode, each config value includes a `source` field:

| Source | Description |
|--------|-------------|
| `environment` | Set via `GLASSBOX_*` environment variable |
| `file` | Set in a config file |
| `default` | Built-in default value |

Sensitive values (tokens, passwords) are redacted as `[redacted]` in the output.
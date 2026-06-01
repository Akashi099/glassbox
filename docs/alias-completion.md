# Shell Completion with Alias Support

Glassbox shell completion scripts now include command aliases as completions, in addition to the main command names.

## Supported Aliases

The following aliases are included in shell completions:

| Alias | Command |
|-------|---------|
| `sim` | `simulate` |
| `sign` | `audit:sign` |
| `verify` | `audit:verify` |
| `gen` | `generate-xdr-snapshot` |
| `tx` | `transaction` |
| `contract` | `contract` |

## Usage

Generate completion scripts for your shell:

```bash
# Bash
glassbox completion bash > /etc/bash_completion.d/glassbox
source ~/.bashrc

# Zsh
glassbox completion zsh > "${fpath[1]}/_glassbox"

# Fish
glassbox completion fish > ~/.config/fish/completions/glassbox.fish

# PowerShell
glassbox completion powershell | Out-String | Invoke-Expression
```

## Verification

To verify aliases are included in completions, generate the script and search for alias patterns:

```bash
glassbox completion bash | grep -E "^\s*(sim|sign|verify|gen|tx|contract)"
```

This should output completion entries for each alias.
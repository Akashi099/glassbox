// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script for your shell",
	Long: `Generate shell completion scripts for Glassbox commands.

The completion script must be evaluated to provide interactive completion of Glassbox commands.
This can be done by sourcing it from your shell profile or piping it to the appropriate location.

Installation instructions:

  Bash:
    $ glassbox completion bash > /etc/bash_completion.d/glassbox
    $ source ~/.bashrc

  Zsh:
    $ glassbox completion zsh > "${fpath[1]}/_glassbox"
    # or place in your custom completions directory:
    $ mkdir -p ~/.zsh/completions
    $ glassbox completion zsh > ~/.zsh/completions/_glassbox
    # then add to your ~/.zshrc: fpath=(~/.zsh/completions $fpath)

  Fish:
    $ glassbox completion fish > ~/.config/fish/completions/glassbox.fish
    $ source ~/.config/fish/config.fish

  PowerShell:
    PS> glassbox completion powershell | Out-String | Invoke-Expression
    # To load completions for every new session, add to your PowerShell profile:
    PS> glassbox completion powershell >> $PROFILE

For detailed instructions on setting up completions for your shell, consult your shell's documentation.

Aliases:
  Glassbox supports command aliases that are also included in shell completions.
  Common aliases include:
    sim    -> simulate
    sign   -> audit:sign
    verify -> audit:verify
    gen    -> generate-xdr-snapshot`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(_ *cobra.Command, args []string) error {
		shell := args[0]

		switch shell {
		case "bash":
			return rootCmd.GenBashCompletionV2(os.Stdout, true)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell type %q. Valid shells: bash, zsh, fish, powershell", shell)
		}
	},
}

// commandAliases maps alias names to their primary command paths.
// These are included in shell completion scripts.
var commandAliases = map[string]string{
	"sim":     "simulate",
	"sign":    "audit:sign",
	"verify":  "audit:verify",
	"gen":     "generate-xdr-snapshot",
	"tx":      "transaction",
	"contract": "contract",
}

// GetCommandAliases returns the map of command aliases for external use.
func GetCommandAliases() map[string]string {
	return commandAliases
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
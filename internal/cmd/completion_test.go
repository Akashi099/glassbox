// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"
)

func TestGetCommandAliases(t *testing.T) {
	aliases := GetCommandAliases()

	// Verify expected aliases are present
	expectedAliases := map[string]string{
		"sim":     "simulate",
		"sign":    "audit:sign",
		"verify":  "audit:verify",
		"gen":     "generate-xdr-snapshot",
		"tx":      "transaction",
		"contract": "contract",
	}

	for alias, expectedCmd := range expectedAliases {
		if cmd, ok := aliases[alias]; !ok {
			t.Errorf("Expected alias %q not found", alias)
		} else if cmd != expectedCmd {
			t.Errorf("Alias %q: expected command %q, got %q", alias, expectedCmd, cmd)
		}
	}
}

func TestCompletionCmd_ValidShells(t *testing.T) {
	validShells := []string{"bash", "zsh", "fish", "powershell"}

	for _, shell := range validShells {
		cmd := completionCmd
		args := []string{shell}

		// Verify the command accepts valid shells
		if err := cmd.ValidateArgs(args); err != nil {
			t.Errorf("Valid shell %q should not produce validation error: %v", shell, err)
		}
	}
}

func TestCompletionCmd_InvalidShell(t *testing.T) {
	cmd := completionCmd
	args := []string{"invalid-shell"}

	// Verify invalid shell is rejected
	if err := cmd.ValidateArgs(args); err == nil {
		t.Error("Expected validation error for invalid shell")
	}
}
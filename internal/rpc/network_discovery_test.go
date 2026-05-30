// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package rpc

import (
	"os"
	"strings"
	"testing"
)

func TestValidateNetworkDiscovery_KnownNetworks(t *testing.T) {
	knownNets := []Network{Testnet, Mainnet, Futurenet}
	for _, net := range knownNets {
		t.Run(string(net), func(t *testing.T) {
			if err := ValidateNetworkDiscovery(net, ""); err != nil {
				t.Errorf("expected no error for built-in network %q, got: %v", net, err)
			}
		})
	}
}

func TestValidateNetworkDiscovery_KnownNetworkWithValidURL(t *testing.T) {
	err := ValidateNetworkDiscovery(Testnet, "https://custom-rpc.example.com")
	if err != nil {
		t.Errorf("expected no error for known network with valid custom URL, got: %v", err)
	}
}

func TestValidateNetworkDiscovery_KnownNetworkWithInvalidURL(t *testing.T) {
	err := ValidateNetworkDiscovery(Mainnet, "not-a-url")
	if err == nil {
		t.Error("expected error for known network with invalid URL, got nil")
	}
	if !strings.Contains(err.Error(), "invalid Soroban RPC endpoint") {
		t.Errorf("error should mention invalid endpoint, got: %v", err)
	}
}

func TestValidateNetworkDiscovery_EmptyNetworkNoURL(t *testing.T) {
	os.Unsetenv("GLASSBOX_RPC_URL")
	os.Unsetenv("GLASSBOX_SOROBAN_RPC_URLS")

	err := ValidateNetworkDiscovery("", "")
	if err == nil {
		t.Error("expected error for empty network with no URL, got nil")
	}
	if !strings.Contains(err.Error(), "--network") {
		t.Errorf("error should suggest --network flag, got: %v", err)
	}
	if !strings.Contains(err.Error(), "GLASSBOX_RPC_URL") {
		t.Errorf("error should mention GLASSBOX_RPC_URL, got: %v", err)
	}
}

func TestValidateNetworkDiscovery_EmptyNetworkWithURL(t *testing.T) {
	err := ValidateNetworkDiscovery("", "https://my-rpc.example.com")
	if err != nil {
		t.Errorf("expected no error for empty network with valid URL, got: %v", err)
	}
}

func TestValidateNetworkDiscovery_EmptyNetworkWithInvalidURL(t *testing.T) {
	err := ValidateNetworkDiscovery("", "ftp://not-allowed.com")
	if err == nil {
		t.Error("expected error for empty network with invalid URL scheme, got nil")
	}
}

func TestValidateNetworkDiscovery_UnknownNetworkNoURL(t *testing.T) {
	os.Unsetenv("GLASSBOX_RPC_URL")
	os.Unsetenv("GLASSBOX_SOROBAN_RPC_URLS")

	err := ValidateNetworkDiscovery("staging", "")
	if err == nil {
		t.Error("expected error for unknown network with no URL, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "staging") {
		t.Errorf("error should name the unknown network alias, got: %v", msg)
	}
	if !strings.Contains(msg, "GLASSBOX_RPC_URL") {
		t.Errorf("error should mention GLASSBOX_RPC_URL as a remedy, got: %v", msg)
	}
	if !strings.Contains(msg, "testnet") {
		t.Errorf("error should suggest built-in network alternatives, got: %v", msg)
	}
}

func TestValidateNetworkDiscovery_UnknownNetworkWithValidURL(t *testing.T) {
	err := ValidateNetworkDiscovery("staging", "https://staging-rpc.example.com")
	if err != nil {
		t.Errorf("expected no error for unknown network with valid URL, got: %v", err)
	}
}

func TestValidateNetworkDiscovery_UnknownNetworkWithInvalidURL(t *testing.T) {
	err := ValidateNetworkDiscovery("staging", "://bad-url")
	if err == nil {
		t.Error("expected error for unknown network with malformed URL, got nil")
	}
	if !strings.Contains(err.Error(), "staging") {
		t.Errorf("error should reference the network name, got: %v", err)
	}
}

func TestValidateNetworkDiscovery_EnvURLResolution(t *testing.T) {
	t.Run("GLASSBOX_RPC_URL", func(t *testing.T) {
		os.Setenv("GLASSBOX_RPC_URL", "https://env-rpc.example.com")
		defer os.Unsetenv("GLASSBOX_RPC_URL")

		// Unknown network but env provides a URL — should succeed.
		err := ValidateNetworkDiscovery("custom", "")
		if err != nil {
			t.Errorf("expected env URL to satisfy unknown network, got: %v", err)
		}
	})

	t.Run("GLASSBOX_SOROBAN_RPC_URLS", func(t *testing.T) {
		os.Unsetenv("GLASSBOX_RPC_URL")
		os.Setenv("GLASSBOX_SOROBAN_RPC_URLS", "https://soroban-env.example.com,https://fallback.example.com")
		defer os.Unsetenv("GLASSBOX_SOROBAN_RPC_URLS")

		err := ValidateNetworkDiscovery("", "")
		if err != nil {
			t.Errorf("expected GLASSBOX_SOROBAN_RPC_URLS to satisfy empty network, got: %v", err)
		}
	})
}

func TestValidateNetworkDiscovery_InvalidEnvURL(t *testing.T) {
	os.Setenv("GLASSBOX_RPC_URL", "not-a-valid-url")
	defer os.Unsetenv("GLASSBOX_RPC_URL")

	err := ValidateNetworkDiscovery("", "")
	if err == nil {
		t.Error("expected error for invalid URL from env, got nil")
	}
}

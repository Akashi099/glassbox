// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package rpc

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/dotandev/glassbox/internal/errors"
)

func isValidURL(urlStr string) error {
	if urlStr == "" {
		return errors.WrapValidationError("URL cannot be empty")
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return errors.WrapValidationError(fmt.Sprintf("invalid URL format: %v", err))
	}

	if parsed.Scheme == "" {
		return errors.WrapValidationError("URL must include scheme (http:// or https://)")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.WrapValidationError(fmt.Sprintf("URL scheme must be http or https, got %q", parsed.Scheme))
	}

	if parsed.Host == "" {
		return errors.WrapValidationError("URL must include a host")
	}

	return nil
}

// knownNetworks lists the built-in network aliases that carry default endpoints.
var knownNetworks = map[Network]bool{
	Testnet:   true,
	Mainnet:   true,
	Futurenet: true,
}

// ValidateNetworkDiscovery checks that the network name and Soroban endpoint
// are sufficient to reach the Stellar network before any RPC call is made.
//
// Rules:
//   - Built-in networks (testnet, mainnet, futurenet) always have default
//     endpoints, so no explicit URL is required.
//   - An empty network name is allowed only when a Soroban URL is provided
//     via the sorobanURL argument or GLASSBOX_RPC_URL / GLASSBOX_SOROBAN_RPC_URLS
//     environment variables.
//   - An unrecognised non-empty network alias requires an explicit URL.
//   - Any provided URL is validated for scheme and host.
func ValidateNetworkDiscovery(net Network, sorobanURL string) error {
	// Resolve URL from environment when not explicitly supplied.
	if sorobanURL == "" {
		if v := os.Getenv("GLASSBOX_RPC_URL"); v != "" {
			sorobanURL = strings.TrimSpace(v)
		} else if v := os.Getenv("GLASSBOX_SOROBAN_RPC_URLS"); v != "" {
			parts := strings.SplitN(v, ",", 2)
			sorobanURL = strings.TrimSpace(parts[0])
		}
	}

	switch {
	case net == "" && sorobanURL == "":
		return errors.WrapValidationError(
			"no network or RPC endpoint configured; set --network (testnet, mainnet, futurenet) " +
				"or provide GLASSBOX_RPC_URL in the environment")

	case net == "":
		// Anonymous endpoint — validate the URL.
		if err := isValidURL(sorobanURL); err != nil {
			return errors.WrapValidationError(fmt.Sprintf(
				"invalid Soroban RPC endpoint: %v; endpoint must be a valid http or https URL", err))
		}
		return nil

	case knownNetworks[net]:
		// Well-known networks always have bundled default endpoints; no URL required.
		if sorobanURL != "" {
			if err := isValidURL(sorobanURL); err != nil {
				return errors.WrapValidationError(fmt.Sprintf(
					"invalid Soroban RPC endpoint for network %q: %v", net, err))
			}
		}
		return nil

	default:
		// Unknown network alias — an explicit endpoint is mandatory.
		if sorobanURL == "" {
			return errors.WrapValidationError(fmt.Sprintf(
				"unknown network %q: custom networks require an explicit RPC endpoint; "+
					"set GLASSBOX_RPC_URL or use a built-in network (testnet, mainnet, futurenet)", net))
		}
		if err := isValidURL(sorobanURL); err != nil {
			return errors.WrapValidationError(fmt.Sprintf(
				"invalid Soroban RPC endpoint for network %q: %v; "+
					"endpoint must be a valid http or https URL", net, err))
		}
		return nil
	}
}

func ValidateNetworkConfig(config NetworkConfig) error {
	if config.Name == "" {
		return errors.WrapValidationError("network name is required")
	}

	if config.NetworkPassphrase == "" {
		return errors.WrapValidationError("network passphrase is required")
	}

	if config.HorizonURL == "" && config.SorobanRPCURL == "" {
		return errors.WrapValidationError("at least one of HorizonURL or SorobanRPCURL is required")
	}

	if config.HorizonURL != "" {
		if err := isValidURL(config.HorizonURL); err != nil {
			return errors.WrapValidationError(fmt.Sprintf("invalid HorizonURL: %v", err))
		}
	}

	if config.SorobanRPCURL != "" {
		if err := isValidURL(config.SorobanRPCURL); err != nil {
			return errors.WrapValidationError(fmt.Sprintf("invalid SorobanRPCURL: %v", err))
		}
	}

	if config.HorizonURL == "" && config.SorobanRPCURL == "" {
		return errors.WrapValidationError("at least one of HorizonURL or SorobanRPCURL must be provided")
	}

	if config.NetworkPassphrase == "" {
		return errors.WrapValidationError("network passphrase is required")
	}

	return nil
}

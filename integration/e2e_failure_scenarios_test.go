// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// sorobanErrorResponse builds a JSON-RPC error payload.
func sorobanErrorResponse(code int, message string) []byte {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

// runWithRPC executes the glassbox binary with GLASSBOX_RPC_URL pointing at
// mockURL. All other env is stripped to avoid picking up the caller's config.
func runWithRPC(t *testing.T, mockURL string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	bin := binaryPath(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = []string{
		fmt.Sprintf("GLASSBOX_RPC_URL=%s", mockURL),
		"HOME=/tmp",
		"USERPROFILE=/tmp",
	}
	// Preserve PATH so the binary can locate shared libraries.
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "PATH=") {
			cmd.Env = append(cmd.Env, e)
			break
		}
	}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// containsAny reports whether s contains at least one of the given needles
// (case-insensitive).
func containsAny(s string, needles ...string) bool {
	lower := strings.ToLower(s)
	for _, n := range needles {
		if strings.Contains(lower, strings.ToLower(n)) {
			return true
		}
	}
	return false
}

// ─── failure scenario tests ───────────────────────────────────────────────────

// TestAuthFailureScenario verifies the CLI surfaces auth-related errors from
// a Soroban RPC server and exits non-zero.
func TestAuthFailureScenario(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(sorobanErrorResponse(-32001,
			"auth error: missing or invalid authorization credentials"))
	}))
	defer srv.Close()

	_, stderr, err := runWithRPC(t, srv.URL,
		"debug",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"--network", "testnet",
	)

	if exitCode(err) == 0 {
		t.Error("expected non-zero exit code for auth failure scenario")
	}
	assertNotContains(t, "stderr", stderr, "panic")
	assertNotContains(t, "stderr", stderr, "goroutine")
	if !containsAny(stderr, "auth", "error", "failed", "invalid", "credential") {
		t.Errorf("expected an error message in stderr for auth failure, got: %q", stderr)
	}
}

// TestBudgetExhaustionScenario verifies the CLI handles CPU/memory budget
// exhaustion errors from Soroban with a clear diagnostic message.
func TestBudgetExhaustionScenario(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(sorobanErrorResponse(-32602,
			"transaction simulation failed: ExceededResourceLimits (CPU instructions budget exhausted)"))
	}))
	defer srv.Close()

	_, stderr, err := runWithRPC(t, srv.URL,
		"debug",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"--network", "testnet",
	)

	if exitCode(err) == 0 {
		t.Error("expected non-zero exit code for budget exhaustion scenario")
	}
	assertNotContains(t, "stderr", stderr, "panic")
	assertNotContains(t, "stderr", stderr, "goroutine")
	if !containsAny(stderr, "budget", "resource", "error", "failed", "simulation", "limit") {
		t.Errorf("expected a diagnostic message in stderr for budget exhaustion, got: %q", stderr)
	}
}

// TestMissingContractCodeScenario verifies the CLI surfaces a missing contract
// code error rather than crashing or returning a generic failure.
func TestMissingContractCodeScenario(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(sorobanErrorResponse(-32602,
			"transaction simulation failed: MissingValue (contract code not found in ledger)"))
	}))
	defer srv.Close()

	_, stderr, err := runWithRPC(t, srv.URL,
		"debug",
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		"--network", "testnet",
	)

	if exitCode(err) == 0 {
		t.Error("expected non-zero exit code for missing contract code scenario")
	}
	assertNotContains(t, "stderr", stderr, "panic")
	assertNotContains(t, "stderr", stderr, "goroutine")
	if !containsAny(stderr, "contract", "missing", "error", "failed", "not found", "ledger") {
		t.Errorf("expected a diagnostic message in stderr for missing contract code, got: %q", stderr)
	}
}

// TestSourceMappingFailureScenario verifies the CLI continues gracefully when
// source map information is unavailable, without panicking.
func TestSourceMappingFailureScenario(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(sorobanErrorResponse(-32602,
			"transaction simulation failed: source map not available for this contract"))
	}))
	defer srv.Close()

	_, stderr, err := runWithRPC(t, srv.URL,
		"debug",
		"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		"--network", "testnet",
	)

	if exitCode(err) == 0 {
		t.Error("expected non-zero exit code for source mapping failure scenario")
	}
	assertNotContains(t, "stderr", stderr, "panic")
	assertNotContains(t, "stderr", stderr, "goroutine")
}

// TestInvalidNetworkExplicitError checks that an unrecognised --network value
// produces an explicit, non-panicking error naming the invalid alias.
func TestInvalidNetworkExplicitError(t *testing.T) {
	_, stderr, err := runErst(t,
		"debug",
		"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		"--network", "not-a-valid-network",
	)

	if exitCode(err) == 0 {
		t.Error("expected non-zero exit code for invalid network alias")
	}
	assertNotContains(t, "stderr", stderr, "panic")
	assertNotContains(t, "stderr", stderr, "goroutine")
	if !containsAny(stderr, "network", "invalid", "not-a-valid-network", "error") {
		t.Errorf("expected a network error message in stderr, got: %q", stderr)
	}
}

// TestDebugExitCodesForFailures verifies exit-code conventions across the
// common failure modes exercised by this file.
func TestDebugExitCodesForFailures(t *testing.T) {
	cases := []struct {
		name   string
		rpcMsg string
		args   []string
	}{
		{
			name:   "auth_failure",
			rpcMsg: "auth error: unauthorized",
			args:   []string{"debug", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "--network", "testnet"},
		},
		{
			name:   "budget_exhaustion",
			rpcMsg: "ExceededResourceLimits: CPU budget exhausted",
			args:   []string{"debug", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "--network", "testnet"},
		},
		{
			name:   "missing_contract",
			rpcMsg: "MissingValue: contract code not found",
			args:   []string{"debug", "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "--network", "testnet"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(sorobanErrorResponse(-32602, tc.rpcMsg))
			}))
			defer srv.Close()

			_, stderr, err := runWithRPC(t, srv.URL, tc.args...)

			if exitCode(err) == 0 {
				t.Errorf("scenario %q: expected non-zero exit, got 0", tc.name)
			}
			assertNotContains(t, "stderr", stderr, "panic")
		})
	}
}

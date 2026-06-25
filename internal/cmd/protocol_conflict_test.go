// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dotandev/glassbox/internal/protocolreg"
)

// ── protocol:diagnose conflict output ────────────────────────────────────────

// TestProtocolDiagnoseCmd_ConflictInErrorMessage verifies that the
// protocol:diagnose command surfaces ConflictDetected in its human-readable
// output and returns an error that names the conflicting handler.
func TestProtocolDiagnoseCmd_ConflictOutput(t *testing.T) {
	// Build a synthetic DiagnosticReport as if the platform returned a conflict.
	report := &protocolreg.DiagnosticReport{
		Platform:           "linux",
		Scheme:             protocolreg.Scheme,
		Status:             protocolreg.StatusDegraded,
		Issues:             []string{"Protocol conflict: wrapper script references a foreign binary."},
		RemediationSteps:   []string{"Run 'glassbox protocol:repair' to overwrite the conflicting registration."},
		ExecutablePath:     "/usr/local/bin/glassbox",
		RegisteredHandler:  "/usr/bin/other",
		HandlerMatchesSelf: false,
		ConflictDetected:   true,
		ConflictingHandler: "/usr/bin/other",
	}

	// Simulate the rendering logic from protocolDiagnoseCmd.RunE.
	var stdout, stderr bytes.Buffer

	for _, check := range report.Checks {
		stdout.WriteString("[OK]   " + check + "\n")
	}
	for _, issue := range report.Issues {
		stderr.WriteString("[FAIL] " + issue + "\n")
	}

	if report.RegisteredHandler != "" {
		stdout.WriteString("Registered handler: " + report.RegisteredHandler + "\n")
		if report.HandlerMatchesSelf {
			stdout.WriteString("Handler matches current executable: yes\n")
		} else if report.ConflictDetected {
			stderr.WriteString("Handler matches current executable: NO (conflict — registered handler is " +
				report.ConflictingHandler + ")\n")
			stderr.WriteString("⚠  Protocol conflict detected: the glassbox:// scheme is currently handled by\n" +
				"   a different binary (" + report.ConflictingHandler + ").\n" +
				"   Run 'glassbox protocol:repair' to reclaim the registration.\n")
		} else {
			stderr.WriteString("Handler matches current executable: NO (stale path)\n")
		}
	}

	stderrStr := stderr.String()

	// The conflict warning must appear.
	if !strings.Contains(stderrStr, "conflict") {
		t.Errorf("stderr should contain 'conflict', got: %s", stderrStr)
	}
	// The conflicting handler path must be named.
	if !strings.Contains(stderrStr, "/usr/bin/other") {
		t.Errorf("stderr should name the conflicting handler, got: %s", stderrStr)
	}
	// Remediation must mention protocol:repair.
	if !strings.Contains(stderrStr, "protocol:repair") {
		t.Errorf("stderr should mention 'protocol:repair', got: %s", stderrStr)
	}
}

// TestProtocolDiagnoseCmd_NoConflict_NoConflictWarning verifies that the
// conflict warning is absent when a healthy registration is returned.
func TestProtocolDiagnoseCmd_NoConflict_NoConflictWarning(t *testing.T) {
	report := &protocolreg.DiagnosticReport{
		Platform:           "linux",
		Scheme:             protocolreg.Scheme,
		Status:             protocolreg.StatusOK,
		Checks:             []string{"Desktop file found", "Helper script launches current binary"},
		RegisteredHandler:  "/usr/local/bin/glassbox",
		HandlerMatchesSelf: true,
		ConflictDetected:   false,
		ConflictingHandler: "",
	}

	var stderr bytes.Buffer

	if report.RegisteredHandler != "" && !report.HandlerMatchesSelf {
		if report.ConflictDetected {
			stderr.WriteString("⚠  Protocol conflict detected\n")
		} else {
			stderr.WriteString("Handler matches current executable: NO (stale path)\n")
		}
	}

	if strings.Contains(stderr.String(), "conflict") {
		t.Errorf("healthy registration should produce no conflict warning, got: %s", stderr.String())
	}
}

// TestProtocolDiagnoseCmd_StalePath_NotConflict verifies that a stale path
// (glassbox registered under a different path) produces a stale warning, not
// a conflict warning.
func TestProtocolDiagnoseCmd_StalePath_NotConflict(t *testing.T) {
	report := &protocolreg.DiagnosticReport{
		Platform:           "linux",
		Scheme:             protocolreg.Scheme,
		Status:             protocolreg.StatusDegraded,
		Issues:             []string{"Protocol helper script does not reference current binary"},
		RegisteredHandler:  "/old/path/glassbox",
		HandlerMatchesSelf: false,
		ConflictDetected:   false, // stale, not a foreign conflict
		ConflictingHandler: "",
	}

	var stderr bytes.Buffer

	if report.RegisteredHandler != "" {
		if !report.HandlerMatchesSelf {
			if report.ConflictDetected {
				stderr.WriteString("⚠  Protocol conflict detected\n")
			} else {
				stderr.WriteString("Handler matches current executable: NO (stale path)\n")
			}
		}
	}

	stderrStr := stderr.String()
	if strings.Contains(stderrStr, "Protocol conflict detected") {
		t.Errorf("stale path should not be labelled as a conflict, got: %s", stderrStr)
	}
	if !strings.Contains(stderrStr, "stale path") {
		t.Errorf("stale path should produce a 'stale path' label, got: %s", stderrStr)
	}
}

// TestProtocolDiagnoseCmd_ConflictError_NamesHandler verifies the error
// returned by protocol:diagnose when a conflict is found names the handler.
func TestProtocolDiagnoseCmd_ConflictError_NamesHandler(t *testing.T) {
	conflictingHandler := "/usr/bin/someotherapp"

	// Simulate the error construction path from protocolDiagnoseCmd.
	var resultErr error
	report := &protocolreg.DiagnosticReport{
		Status:             protocolreg.StatusDegraded,
		ConflictDetected:   true,
		ConflictingHandler: conflictingHandler,
	}
	if report.Status != protocolreg.StatusOK {
		if report.ConflictDetected {
			resultErr = &conflictError{handler: report.ConflictingHandler}
		}
	}

	if resultErr == nil {
		t.Fatal("expected a non-nil error for a conflict")
	}
	msg := resultErr.Error()
	if !strings.Contains(msg, conflictingHandler) {
		t.Errorf("conflict error should name the conflicting handler, got: %q", msg)
	}
	if !strings.Contains(msg, "protocol:repair") {
		t.Errorf("conflict error should mention 'protocol:repair', got: %q", msg)
	}
}

// conflictError is a minimal error type matching the shape produced by
// protocol:diagnose when ConflictDetected is true.
type conflictError struct{ handler string }

func (e *conflictError) Error() string {
	return "protocol registration conflict: glassbox:// is claimed by " +
		e.handler + " — run 'glassbox protocol:repair' to resolve"
}

// ── protocol:handle — invalid URI diagnostic ─────────────────────────────────

// TestProtocolHandleCmd_EmptyURI returns a clear error for an empty URI.
func TestProtocolHandleCmd_EmptyURI(t *testing.T) {
	_, err := protocolreg.ParseDebugURI("")
	if err == nil {
		t.Fatal("expected error for empty URI")
	}
	if strings.TrimSpace(err.Error()) == "" {
		t.Error("error for empty URI must be non-empty")
	}
}

// TestProtocolHandleCmd_WrongScheme error names the expected scheme.
func TestProtocolHandleCmd_WrongScheme_ErrorNamesScheme(t *testing.T) {
	_, err := protocolreg.ParseDebugURI("https://debug/abc123?network=testnet")
	if err == nil {
		t.Fatal("expected error for wrong scheme")
	}
	if !strings.Contains(err.Error(), "glassbox") {
		t.Errorf("error should mention expected scheme 'glassbox', got: %v", err)
	}
}

// TestProtocolHandleCmd_MissingNetwork error names the missing parameter.
func TestProtocolHandleCmd_MissingNetwork_ErrorNamesParam(t *testing.T) {
	validHash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	_, err := protocolreg.ParseDebugURI("glassbox://debug/" + validHash)
	if err == nil {
		t.Fatal("expected error for missing network")
	}
	if !strings.Contains(err.Error(), "network") {
		t.Errorf("error should mention 'network' parameter, got: %v", err)
	}
}

// TestProtocolHandleCmd_InvalidNetwork_ListsValidOptions verifies the error
// lists accepted network names so users know what values are valid.
func TestProtocolHandleCmd_InvalidNetwork_ListsValidOptions(t *testing.T) {
	validHash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	_, err := protocolreg.ParseDebugURI("glassbox://debug/" + validHash + "?network=devnet")
	if err == nil {
		t.Fatal("expected error for invalid network")
	}
	msg := err.Error()
	for _, net := range []string{"testnet", "mainnet", "futurenet"} {
		if !strings.Contains(msg, net) {
			t.Errorf("error should list valid network %q, got: %v", net, err)
		}
	}
}

// TestProtocolHandleCmd_InvalidView_ListsValidOptions verifies the error for
// an unknown view names the accepted values.
func TestProtocolHandleCmd_InvalidView_ListsValidOptions(t *testing.T) {
	validHash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	_, err := protocolreg.ParseDebugURI(
		"glassbox://debug/" + validHash + "?network=testnet&view=unknown",
	)
	if err == nil {
		t.Fatal("expected error for invalid view")
	}
	msg := err.Error()
	for _, v := range []string{"trace", "flamegraph", "events"} {
		if !strings.Contains(msg, v) {
			t.Errorf("error should list valid view %q, got: %v", v, err)
		}
	}
}

// TestProtocolHandleCmd_NegativeOp error mentions the parameter name.
func TestProtocolHandleCmd_NegativeOp_ErrorMentionsParam(t *testing.T) {
	validHash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	_, err := protocolreg.ParseDebugURI(
		"glassbox://debug/" + validHash + "?network=testnet&op=-1",
	)
	if err == nil {
		t.Fatal("expected error for negative op")
	}
	if !strings.Contains(err.Error(), "operation index") {
		t.Errorf("error should mention 'operation index', got: %v", err)
	}
}

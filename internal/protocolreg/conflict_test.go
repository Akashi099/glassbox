// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package protocolreg

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ── extractExecPath ───────────────────────────────────────────────────────────

func TestExtractExecPath_SingleQuoted(t *testing.T) {
	script := "#!/bin/sh\nexec '/usr/local/bin/glassbox' protocol-handler \"$1\"\n"
	got := extractExecPath(script)
	if got != "/usr/local/bin/glassbox" {
		t.Errorf("expected '/usr/local/bin/glassbox', got %q", got)
	}
}

func TestExtractExecPath_Unquoted(t *testing.T) {
	script := "#!/bin/sh\nexec /home/user/bin/glassbox protocol-handler \"$1\"\n"
	got := extractExecPath(script)
	if got != "/home/user/bin/glassbox" {
		t.Errorf("expected '/home/user/bin/glassbox', got %q", got)
	}
}

func TestExtractExecPath_ForeignBinary(t *testing.T) {
	script := "#!/bin/sh\nexec /usr/bin/othertool protocol-handler \"$1\"\n"
	got := extractExecPath(script)
	if got != "/usr/bin/othertool" {
		t.Errorf("expected '/usr/bin/othertool', got %q", got)
	}
}

func TestExtractExecPath_EmptyScript(t *testing.T) {
	got := extractExecPath("")
	if got != "" {
		t.Errorf("expected empty string for empty script, got %q", got)
	}
}

func TestExtractExecPath_NoExecLine(t *testing.T) {
	script := "#!/bin/sh\necho hello\n"
	got := extractExecPath(script)
	if got != "" {
		t.Errorf("expected empty string when no exec line, got %q", got)
	}
}

// ── DiagnosticReport conflict fields ─────────────────────────────────────────

func TestDiagnosticReport_ConflictFields_DefaultFalse(t *testing.T) {
	r := newTestRegistrar(t)
	report := r.Diagnose()
	// On a clean system with no registration the conflict fields should be false/empty.
	if report.ConflictDetected {
		t.Error("ConflictDetected should default to false when nothing is registered")
	}
	if report.ConflictingHandler != "" {
		t.Errorf("ConflictingHandler should be empty when no conflict, got %q", report.ConflictingHandler)
	}
}

func TestDiagnosticReport_ConflictFields_PresentInStruct(t *testing.T) {
	// Verify the fields are settable — catches accidental removal.
	r := &DiagnosticReport{
		ConflictDetected:   true,
		ConflictingHandler: "/usr/bin/other",
	}
	if !r.ConflictDetected {
		t.Error("ConflictDetected field must be settable")
	}
	if r.ConflictingHandler != "/usr/bin/other" {
		t.Errorf("ConflictingHandler field must be settable, got %q", r.ConflictingHandler)
	}
}

// ── Linux conflict detection via wrapper script ───────────────────────────────

func TestDiagnose_Linux_WrapperForeignBinary_SetsConflict(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	r := newTestRegistrar(t)

	// Write desktop file pointing to the wrapper path.
	if err := os.MkdirAll(filepath.Dir(r.linuxDesktopPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(r.linuxWrapperPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	// Desktop file references the expected wrapper path.
	if err := os.WriteFile(r.linuxDesktopPath(), []byte(r.linuxDesktopEntry()), 0o644); err != nil {
		t.Fatal(err)
	}
	// Wrapper script references a completely foreign binary (not glassbox).
	foreignScript := "#!/bin/sh\nexec /usr/bin/completelydifferentapp protocol-handler \"$1\"\n"
	if err := os.WriteFile(r.linuxWrapperPath(), []byte(foreignScript), 0o755); err != nil {
		t.Fatal(err)
	}

	report := r.Diagnose()

	if !report.ConflictDetected {
		t.Error("expected ConflictDetected=true when wrapper references a foreign binary")
	}
	if report.ConflictingHandler != "/usr/bin/completelydifferentapp" {
		t.Errorf("ConflictingHandler should be the foreign path, got %q", report.ConflictingHandler)
	}
	// Issue message should mention "conflict" and the conflicting path.
	combined := strings.Join(report.Issues, " ")
	if !strings.Contains(combined, "conflict") {
		t.Errorf("issues should mention 'conflict', got: %v", report.Issues)
	}
	if !strings.Contains(combined, "/usr/bin/completelydifferentapp") {
		t.Errorf("issues should include the conflicting path, got: %v", report.Issues)
	}
	// Remediation must mention protocol:repair.
	remSteps := strings.Join(report.RemediationSteps, " ")
	if !strings.Contains(remSteps, "protocol:repair") {
		t.Errorf("remediation steps should mention 'protocol:repair', got: %v", report.RemediationSteps)
	}
}

func TestDiagnose_Linux_WrapperStalePath_NoConflict(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	r := newTestRegistrar(t)

	// Set up valid desktop file.
	if err := os.MkdirAll(filepath.Dir(r.linuxDesktopPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(r.linuxWrapperPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.linuxDesktopPath(), []byte(r.linuxDesktopEntry()), 0o644); err != nil {
		t.Fatal(err)
	}
	// Wrapper references a different *glassbox* binary (stale self-path, not a foreign conflict).
	staleScript := "#!/bin/sh\nexec /old/path/glassbox protocol-handler \"$1\"\n"
	if err := os.WriteFile(r.linuxWrapperPath(), []byte(staleScript), 0o755); err != nil {
		t.Fatal(err)
	}

	report := r.Diagnose()

	// Stale glassbox path: HandlerMatchesSelf=false but ConflictDetected=false.
	if report.HandlerMatchesSelf {
		t.Error("HandlerMatchesSelf should be false for a stale path")
	}
	if report.ConflictDetected {
		t.Error("ConflictDetected should be false for a stale glassbox path (not a foreign binary)")
	}
}

// ── Repair conflict path ──────────────────────────────────────────────────────

func TestRepair_AfterConflict_RecordsConflictAction(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test for conflict repair path")
	}

	r := newTestRegistrar(t)

	// Write a foreign-binary wrapper to trigger conflict detection.
	if err := os.MkdirAll(filepath.Dir(r.linuxDesktopPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(r.linuxWrapperPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.linuxDesktopPath(), []byte(r.linuxDesktopEntry()), 0o644); err != nil {
		t.Fatal(err)
	}
	foreignScript := "#!/bin/sh\nexec /usr/bin/otherapp protocol-handler \"$1\"\n"
	if err := os.WriteFile(r.linuxWrapperPath(), []byte(foreignScript), 0o755); err != nil {
		t.Fatal(err)
	}

	result := r.Repair()

	// Repair must record at least one action (whether it succeeds or fails
	// depends on xdg-mime availability in the test environment).
	if len(result.Actions) == 0 {
		t.Fatal("Repair must record at least one action")
	}
}

// ── defaultRemediationSteps mentions protocol:repair ─────────────────────────

func TestDefaultRemediationSteps_MentionsProtocolRepair(t *testing.T) {
	r := newTestRegistrar(t)
	steps := r.defaultRemediationSteps()
	combined := strings.Join(steps, " ")
	if !strings.Contains(combined, "protocol:repair") {
		t.Errorf("defaultRemediationSteps should mention 'protocol:repair', got: %v", steps)
	}
}

// ── StatusDegraded set when conflict present ──────────────────────────────────

func TestDiagnose_Linux_ConflictSetsDegradedStatus(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	r := newTestRegistrar(t)

	if err := os.MkdirAll(filepath.Dir(r.linuxDesktopPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(r.linuxWrapperPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.linuxDesktopPath(), []byte(r.linuxDesktopEntry()), 0o644); err != nil {
		t.Fatal(err)
	}
	foreignScript := "#!/bin/sh\nexec /usr/bin/foreignapp protocol-handler \"$1\"\n"
	if err := os.WriteFile(r.linuxWrapperPath(), []byte(foreignScript), 0o755); err != nil {
		t.Fatal(err)
	}

	report := r.Diagnose()

	// A conflict with an existing registration artefact → StatusDegraded (not StatusNotRegistered).
	if report.Status == StatusNotRegistered {
		t.Errorf("expected StatusDegraded for a conflict, got StatusNotRegistered")
	}
}

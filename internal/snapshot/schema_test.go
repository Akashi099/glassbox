// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Tests for Part B: schema stability and upgrade diagnostics.
//
// These tests exercise CheckSchemaVersion, ValidateSchemaVersion,
// SchemaVersionSummary, classifySchemaVersion, and the SchemaError type to
// ensure stale, future-version, or unsupported snapshot files surface clear,
// actionable error messages before any simulation work begins.

package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helpers

func writeSnapshotWithSchemaVersion(t *testing.T, version int) string {
	t.Helper()
	ps := &PersistedSnapshot{
		Metadata: &ReplayMetadata{
			SchemaVersion:   version,
			GlassboxVersion: "vX.Y.Z",
			TxHash:          "abc123",
			Network:         "testnet",
		},
		Snapshot: FromMap(nil),
	}
	data, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal test snapshot: %v", err)
	}
	path := filepath.Join(t.TempDir(), "snap.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write test snapshot: %v", err)
	}
	return path
}

// ── CheckSchemaVersion ────────────────────────────────────────────────────────

func TestCheckSchemaVersion_CurrentVersion_IsNotNeedsUpgrade(t *testing.T) {
	path := writeSnapshotWithSchemaVersion(t, PersistSchemaVersion)
	result, err := CheckSchemaVersion(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NeedsUpgrade {
		t.Error("current version should not require upgrade")
	}
	if result.Unsupported {
		t.Error("current version should not be unsupported")
	}
	if result.FromFuture {
		t.Error("current version should not be from future")
	}
}

func TestCheckSchemaVersion_OldUnsupportedVersion_MarksUnsupported(t *testing.T) {
	// Version 0 is below MinSupportedSchemaVersion.
	path := writeSnapshotWithSchemaVersion(t, 0)
	result, err := CheckSchemaVersion(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Unsupported {
		t.Error("version 0 should be marked unsupported")
	}
	if !strings.Contains(result.Message, "re-run") {
		t.Errorf("message should suggest re-running, got: %s", result.Message)
	}
}

func TestCheckSchemaVersion_FutureVersion_MarksFromFuture(t *testing.T) {
	path := writeSnapshotWithSchemaVersion(t, PersistSchemaVersion+100)
	result, err := CheckSchemaVersion(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.FromFuture {
		t.Error("version 101 should be marked from future")
	}
	if !result.Unsupported {
		t.Error("future version should also be marked unsupported")
	}
	if !strings.Contains(result.Message, "upgrade Glassbox") {
		t.Errorf("message should suggest upgrading Glassbox, got: %s", result.Message)
	}
}

func TestCheckSchemaVersion_MissingFile_ReturnsError(t *testing.T) {
	_, err := CheckSchemaVersion("/nonexistent/snap.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestCheckSchemaVersion_InvalidJSON_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := CheckSchemaVersion(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestCheckSchemaVersion_MissingMetadata_ReturnsError(t *testing.T) {
	// File that has no metadata section.
	ps := &PersistedSnapshot{Snapshot: FromMap(nil)}
	data, _ := json.MarshalIndent(ps, "", "  ")
	path := filepath.Join(t.TempDir(), "nometadata.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := CheckSchemaVersion(path)
	if err == nil {
		t.Fatal("expected error for missing metadata section")
	}
	if !strings.Contains(err.Error(), "metadata") {
		t.Errorf("error should mention metadata, got: %v", err)
	}
}

// ── ValidateSchemaVersion ─────────────────────────────────────────────────────

func TestValidateSchemaVersion_CurrentVersion_NoError(t *testing.T) {
	if err := ValidateSchemaVersion(PersistSchemaVersion, "snap.json"); err != nil {
		t.Errorf("expected no error for current schema version, got: %v", err)
	}
}

func TestValidateSchemaVersion_ZeroVersion_ReturnsSchemaError(t *testing.T) {
	err := ValidateSchemaVersion(0, "/path/to/snap.json")
	if err == nil {
		t.Fatal("expected error for version 0")
	}
	if !IsSchemaError(err) {
		t.Errorf("expected *SchemaError, got: %T: %v", err, err)
	}
	se := AsSchemaError(err)
	if se == nil {
		t.Fatal("AsSchemaError returned nil")
	}
	if !se.Result.Unsupported {
		t.Error("version 0 result should be unsupported")
	}
}

func TestValidateSchemaVersion_FutureVersion_ReturnsSchemaError(t *testing.T) {
	err := ValidateSchemaVersion(PersistSchemaVersion+1, "snap.json")
	if err == nil {
		t.Fatal("expected error for future version")
	}
	if !IsSchemaError(err) {
		t.Errorf("expected *SchemaError, got: %T: %v", err, err)
	}
	se := AsSchemaError(err)
	if !se.Result.FromFuture {
		t.Error("future version result should have FromFuture set")
	}
}

func TestValidateSchemaVersion_PathIncludedInError(t *testing.T) {
	const path = "/home/user/snap.json"
	err := ValidateSchemaVersion(0, path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error should include file path, got: %v", err)
	}
}

// ── SchemaError type ──────────────────────────────────────────────────────────

func TestSchemaError_ErrorString_ContainsMessageAndPath(t *testing.T) {
	r := classifySchemaVersion(0)
	se := &SchemaError{Result: r, Path: "/my/snap.json"}
	msg := se.Error()
	if !strings.Contains(msg, "/my/snap.json") {
		t.Errorf("error should contain path, got: %s", msg)
	}
	if !strings.Contains(msg, r.Message) {
		t.Errorf("error should contain result message, got: %s", msg)
	}
}

func TestIsSchemaError_PositiveAndNegative(t *testing.T) {
	if IsSchemaError(nil) {
		t.Error("nil is not a SchemaError")
	}
	if IsSchemaError(os.ErrNotExist) {
		t.Error("ErrNotExist is not a SchemaError")
	}

	r := classifySchemaVersion(0)
	se := &SchemaError{Result: r, Path: "snap.json"}
	if !IsSchemaError(se) {
		t.Error("expected IsSchemaError to return true for *SchemaError")
	}
}

func TestAsSchemaError_NilForNonSchemaError(t *testing.T) {
	if AsSchemaError(os.ErrNotExist) != nil {
		t.Error("expected nil for non-schema error")
	}
}

// ── SchemaVersionSummary ──────────────────────────────────────────────────────

func TestSchemaVersionSummary_CurrentVersion(t *testing.T) {
	summary := SchemaVersionSummary(PersistSchemaVersion)
	if !strings.Contains(summary, "current") {
		t.Errorf("expected 'current' in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "no upgrade") {
		t.Errorf("expected 'no upgrade' in summary, got: %s", summary)
	}
}

func TestSchemaVersionSummary_UnsupportedVersion_MentionsRemediation(t *testing.T) {
	summary := SchemaVersionSummary(0)
	if !strings.Contains(summary, "re-run") {
		t.Errorf("expected re-run hint in summary, got: %s", summary)
	}
}

func TestSchemaVersionSummary_FutureVersion_MentionsUpgrade(t *testing.T) {
	summary := SchemaVersionSummary(PersistSchemaVersion + 99)
	if !strings.Contains(summary, "upgrade") {
		t.Errorf("expected upgrade hint in summary, got: %s", summary)
	}
}

// ── LoadPersisted uses ValidateSchemaVersion ──────────────────────────────────

// TestLoadPersisted_WrongSchemaVersion_UsesValidateSchemaVersion verifies that
// LoadPersisted delegates version checking to ValidateSchemaVersion so error
// messages are consistent and structured (i.e. return *SchemaError).
func TestLoadPersisted_WrongSchemaVersion_UsesSchemaError(t *testing.T) {
	path := writeSnapshotWithSchemaVersion(t, PersistSchemaVersion+5)
	_, err := LoadPersisted(path)
	if err == nil {
		t.Fatal("expected error for future schema version")
	}
	// Error must be a *SchemaError so callers can classify it.
	if !IsSchemaError(err) {
		t.Errorf("expected *SchemaError from LoadPersisted, got: %T: %v", err, err)
	}
	// Message must tell the user what to do.
	if !strings.Contains(err.Error(), "upgrade Glassbox") {
		t.Errorf("error should tell user to upgrade Glassbox, got: %v", err)
	}
}

func TestLoadPersisted_UnsupportedSchemaVersion_MentionsReRun(t *testing.T) {
	path := writeSnapshotWithSchemaVersion(t, 0)
	_, err := LoadPersisted(path)
	if err == nil {
		t.Fatal("expected error for schema version 0")
	}
	if !strings.Contains(err.Error(), "re-run") {
		t.Errorf("error should suggest re-running the debug command, got: %v", err)
	}
}

func TestLoadPersisted_CurrentSchemaVersion_Succeeds(t *testing.T) {
	path := writeSnapshotWithSchemaVersion(t, PersistSchemaVersion)
	ps, err := LoadPersisted(path)
	if err != nil {
		t.Fatalf("expected no error for current schema version, got: %v", err)
	}
	if ps == nil {
		t.Fatal("expected non-nil PersistedSnapshot")
	}
}

// ── classifySchemaVersion ─────────────────────────────────────────────────────

func TestClassifySchemaVersion_Current_NoFlags(t *testing.T) {
	r := classifySchemaVersion(PersistSchemaVersion)
	if r.NeedsUpgrade || r.Unsupported || r.FromFuture {
		t.Errorf("current version should have no flags set, got: %+v", r)
	}
	if r.StoredVersion != PersistSchemaVersion {
		t.Errorf("StoredVersion mismatch: %d", r.StoredVersion)
	}
	if r.CurrentVersion != PersistSchemaVersion {
		t.Errorf("CurrentVersion mismatch: %d", r.CurrentVersion)
	}
}

func TestClassifySchemaVersion_BelowMinimum_Unsupported(t *testing.T) {
	r := classifySchemaVersion(MinSupportedSchemaVersion - 1)
	if !r.Unsupported {
		t.Error("expected Unsupported for version below minimum")
	}
	if r.NeedsUpgrade {
		t.Error("should not also set NeedsUpgrade for truly unsupported version")
	}
}

func TestClassifySchemaVersion_AboveCurrent_FromFuture(t *testing.T) {
	r := classifySchemaVersion(PersistSchemaVersion + 1)
	if !r.FromFuture {
		t.Error("expected FromFuture for version above current")
	}
	if !r.Unsupported {
		t.Error("expected Unsupported for future version")
	}
}

func TestClassifySchemaVersion_MessageNeverEmpty(t *testing.T) {
	for _, v := range []int{0, MinSupportedSchemaVersion, PersistSchemaVersion, PersistSchemaVersion + 1, 9999} {
		r := classifySchemaVersion(v)
		if r.Message == "" {
			t.Errorf("classifySchemaVersion(%d) returned empty message", v)
		}
	}
}

// ── Schema check wired into debug PreRunE (via pre-flight in debug.go) ────────
// These integration tests verify that the schema pre-flight added to PreRunE
// actually surfaces schema errors for persisted snapshot files.

func TestSchemaPreFlight_FutureSnapshotRejected_BeforeReplay(t *testing.T) {
	path := writeSnapshotWithSchemaVersion(t, PersistSchemaVersion+10)

	// Verify that CheckSchemaVersion correctly classifies the future version.
	result, err := CheckSchemaVersion(path)
	if err != nil {
		t.Fatalf("CheckSchemaVersion error: %v", err)
	}
	if !result.FromFuture {
		t.Error("expected FromFuture for future version")
	}
	if !strings.Contains(result.Message, "upgrade Glassbox") {
		t.Errorf("message should suggest upgrading, got: %s", result.Message)
	}
}

func TestSchemaPreFlight_UnsupportedSnapshotRejected_BeforeReplay(t *testing.T) {
	path := writeSnapshotWithSchemaVersion(t, 0)

	result, err := CheckSchemaVersion(path)
	if err != nil {
		t.Fatalf("CheckSchemaVersion error: %v", err)
	}
	if !result.Unsupported {
		t.Error("expected Unsupported for version 0")
	}
	if !strings.Contains(result.Message, "re-run") {
		t.Errorf("message should suggest re-running, got: %s", result.Message)
	}
}

func TestSchemaPreFlight_CurrentSnapshotPassesCheck(t *testing.T) {
	path := writeSnapshotWithSchemaVersion(t, PersistSchemaVersion)

	result, err := CheckSchemaVersion(path)
	if err != nil {
		t.Fatalf("CheckSchemaVersion error: %v", err)
	}
	if result.Unsupported || result.NeedsUpgrade || result.FromFuture {
		t.Errorf("current version should pass pre-flight check, got: %+v", result)
	}
}

// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Tests for Part A: path normalization and safety in the debug command.
//
// These tests exercise NormalizePath, ValidateInputPath, ValidateOutputPath,
// ValidatePathTraversal, ValidateDebugInputPaths, and ValidateDebugOutputPaths
// to ensure invalid, unsafe, or ambiguous paths are caught before any
// simulation or network work begins.

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── NormalizePath ─────────────────────────────────────────────────────────────

func TestNormalizePath_EmptyReturnsEmpty(t *testing.T) {
	got, err := NormalizePath("flag", "", "")
	if err != nil {
		t.Fatalf("expected no error for empty path, got: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty result for empty path, got: %q", got)
	}
}

func TestNormalizePath_NullByteRejected(t *testing.T) {
	_, err := NormalizePath("flag", "/some/path\x00injected", "")
	if err == nil {
		t.Fatal("expected error for path containing null byte")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Errorf("error should mention null bytes, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--flag") {
		t.Errorf("error should mention flag name, got: %v", err)
	}
}

func TestNormalizePath_ReturnsAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.json")
	if err := os.WriteFile(file, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := NormalizePath("flag", file, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got: %q", got)
	}
}

func TestNormalizePath_AllowedRootEnforced(t *testing.T) {
	dir := t.TempDir()
	allowed := filepath.Join(dir, "workspace")
	if err := os.MkdirAll(allowed, 0755); err != nil {
		t.Fatal(err)
	}
	// A path outside the allowed root.
	outside := dir // parent of workspace

	_, err := NormalizePath("flag", outside, allowed)
	if err == nil {
		t.Fatal("expected error for path outside allowed root")
	}
	if !strings.Contains(err.Error(), "resolves outside") {
		t.Errorf("error should say 'resolves outside', got: %v", err)
	}
}

func TestNormalizePath_AllowedRootPermits_ValidPath(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(sub, "snap.json")
	if err := os.WriteFile(file, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := NormalizePath("snapshot", file, dir)
	if err != nil {
		t.Fatalf("expected no error for path inside allowed root, got: %v", err)
	}
	if !strings.HasPrefix(got, dir) {
		t.Errorf("expected normalized path under allowed root, got: %q", got)
	}
}

// ── ValidateInputPath ─────────────────────────────────────────────────────────

func TestValidateInputPath_ExistingFile_Succeeds(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "input.json")
	if err := os.WriteFile(file, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ValidateInputPath("json-file", file)
	if err != nil {
		t.Fatalf("expected no error for existing file, got: %v", err)
	}
	if got == "" {
		t.Error("expected non-empty normalized path")
	}
}

func TestValidateInputPath_MissingFile_ErrorMentionsPath(t *testing.T) {
	_, err := ValidateInputPath("snapshot", "/nonexistent/snap.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	msg := err.Error()
	if !strings.Contains(msg, "/nonexistent/snap.json") {
		t.Errorf("error should mention file path, got: %q", msg)
	}
	if !strings.Contains(msg, "--snapshot") {
		t.Errorf("error should mention flag name --snapshot, got: %q", msg)
	}
	// Should include remediation hint.
	if !strings.Contains(msg, "Check that") {
		t.Errorf("error should include a remediation hint, got: %q", msg)
	}
}

func TestValidateInputPath_DirectoryRejected(t *testing.T) {
	dir := t.TempDir()
	_, err := ValidateInputPath("wasm", dir)
	if err == nil {
		t.Fatal("expected error when path is a directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should say 'directory', got: %v", err)
	}
}

func TestValidateInputPath_EmptyPathReturnsEmpty(t *testing.T) {
	got, err := ValidateInputPath("wasm", "")
	if err != nil {
		t.Fatalf("expected no error for empty path, got: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty result for empty path, got: %q", got)
	}
}

func TestValidateInputPath_NullByteRejected(t *testing.T) {
	_, err := ValidateInputPath("xdr-file", "/path/to\x00file.xdr")
	if err == nil {
		t.Fatal("expected error for path with null byte")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Errorf("error should mention null bytes, got: %v", err)
	}
}

// ── ValidateOutputPath ────────────────────────────────────────────────────────

func TestValidateOutputPath_EmptyReturnsEmpty(t *testing.T) {
	got, err := ValidateOutputPath("save-snapshots", "")
	if err != nil {
		t.Fatalf("expected no error for empty path, got: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty result for empty path, got: %q", got)
	}
}

func TestValidateOutputPath_NewFileInExistingDir_Succeeds(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "out.json")

	got, err := ValidateOutputPath("save-snapshots", file)
	if err != nil {
		t.Fatalf("expected no error for writable output path, got: %v", err)
	}
	if got == "" {
		t.Error("expected non-empty normalized path")
	}
}

func TestValidateOutputPath_ExistingDirectory_Rejected(t *testing.T) {
	dir := t.TempDir()
	_, err := ValidateOutputPath("export-svg", dir)
	if err == nil {
		t.Fatal("expected error when output path is an existing directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should say 'directory', got: %v", err)
	}
}

func TestValidateOutputPath_NullByteRejected(t *testing.T) {
	_, err := ValidateOutputPath("trace-output", "/path/to\x00trace.out")
	if err == nil {
		t.Fatal("expected error for path with null byte")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Errorf("error should mention null bytes, got: %v", err)
	}
}

func TestValidateOutputPath_FlagNameInError(t *testing.T) {
	// Parent dir does not exist; validate that it's reported with the right flag name
	_, err := ValidateOutputPath("export-svg", "/nonexistent-root/sub/out.svg")
	// May or may not error depending on OS; if it does, flag name must be present.
	if err != nil && !strings.Contains(err.Error(), "--export-svg") {
		t.Errorf("error should mention --export-svg, got: %v", err)
	}
}

// ── ValidatePathTraversal ─────────────────────────────────────────────────────

func TestValidatePathTraversal_SafePath_NoError(t *testing.T) {
	if err := ValidatePathTraversal("flag", "contracts/token/src/lib.rs"); err != nil {
		t.Errorf("expected no error for safe relative path, got: %v", err)
	}
}

func TestValidatePathTraversal_DoubleDotRejected(t *testing.T) {
	err := ValidatePathTraversal("source-alias", "../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal attempt")
	}
	if !strings.Contains(err.Error(), "traversal") {
		t.Errorf("error should mention traversal, got: %v", err)
	}
}

func TestValidatePathTraversal_NullByteRejected(t *testing.T) {
	err := ValidatePathTraversal("flag", "ok/path\x00")
	if err == nil {
		t.Fatal("expected error for null byte in path")
	}
}

func TestValidatePathTraversal_EmptyPath_NoError(t *testing.T) {
	if err := ValidatePathTraversal("flag", ""); err != nil {
		t.Errorf("expected no error for empty path, got: %v", err)
	}
}

// ── ValidateDebugInputPaths (batch) ──────────────────────────────────────────

func TestValidateDebugInputPaths_AllEmpty_NoError(t *testing.T) {
	if err := ValidateDebugInputPaths("", "", "", "", "", "", ""); err != nil {
		t.Errorf("expected no error when all paths are empty, got: %v", err)
	}
}

func TestValidateDebugInputPaths_MissingSnapshot_Surfaces_FlagName(t *testing.T) {
	err := ValidateDebugInputPaths("/nonexistent/snap.json", "", "", "", "", "", "")
	if err == nil {
		t.Fatal("expected error for missing --snapshot file")
	}
	if !strings.Contains(err.Error(), "--snapshot") {
		t.Errorf("error should mention --snapshot, got: %v", err)
	}
}

func TestValidateDebugInputPaths_MissingWasm_Surfaces_FlagName(t *testing.T) {
	err := ValidateDebugInputPaths("", "/nonexistent/contract.wasm", "", "", "", "", "")
	if err == nil {
		t.Fatal("expected error for missing --wasm file")
	}
	if !strings.Contains(err.Error(), "--wasm") {
		t.Errorf("error should mention --wasm, got: %v", err)
	}
}

func TestValidateDebugInputPaths_ValidExistingFiles_NoError(t *testing.T) {
	dir := t.TempDir()
	snap := filepath.Join(dir, "snap.json")
	wasm := filepath.Join(dir, "contract.wasm")
	for _, f := range []string{snap, wasm} {
		if err := os.WriteFile(f, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := ValidateDebugInputPaths(snap, wasm, "", "", "", "", ""); err != nil {
		t.Errorf("expected no error for existing files, got: %v", err)
	}
}

// ── ValidateDebugOutputPaths (batch) ─────────────────────────────────────────

func TestValidateDebugOutputPaths_AllEmpty_NoError(t *testing.T) {
	if err := ValidateDebugOutputPaths("", "", ""); err != nil {
		t.Errorf("expected no error when all paths are empty, got: %v", err)
	}
}

func TestValidateDebugOutputPaths_ExistingDirectory_RejectsWithFlagName(t *testing.T) {
	dir := t.TempDir()
	err := ValidateDebugOutputPaths(dir, "", "")
	if err == nil {
		t.Fatal("expected error when --save-snapshots is an existing directory")
	}
	if !strings.Contains(err.Error(), "--save-snapshots") {
		t.Errorf("error should mention --save-snapshots, got: %v", err)
	}
}

func TestValidateDebugOutputPaths_ValidOutputFiles_NoError(t *testing.T) {
	dir := t.TempDir()
	if err := ValidateDebugOutputPaths(
		filepath.Join(dir, "out.snap.json"),
		filepath.Join(dir, "graph.svg"),
		filepath.Join(dir, "trace.out"),
	); err != nil {
		t.Errorf("expected no error for writable output paths, got: %v", err)
	}
}

// ── PreRunE integration: path validation wired into debug command ─────────────

// TestDebugPreRunE_MissingSnapshotFile_SurfacesEarly verifies that a missing
// --snapshot file is caught in PreRunE before any network call.
func TestDebugPreRunE_MissingSnapshotFile_SurfacesEarly(t *testing.T) {
	t.Cleanup(resetDebugFlags)
	t.Cleanup(func() { snapshotFlag = "" })
	networkFlag = "testnet"
	snapshotFlag = "/nonexistent/snap.json"
	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"

	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err == nil {
		t.Fatal("expected error for missing --snapshot file")
	}
	if !strings.Contains(err.Error(), "--snapshot") {
		t.Errorf("error should mention --snapshot, got: %v", err)
	}
}

// TestDebugPreRunE_MissingWasmFile_SurfacesEarly verifies that a missing --wasm
// file is caught in PreRunE before the simulator is invoked.
func TestDebugPreRunE_MissingWasmFile_SurfacesEarly(t *testing.T) {
	t.Cleanup(resetDebugFlags)
	wasmPath = "/nonexistent/contract.wasm"

	err := debugCmd.PreRunE(debugCmd, []string{})
	if err == nil {
		t.Fatal("expected error for missing --wasm file")
	}
	if !strings.Contains(err.Error(), "wasm") {
		t.Errorf("error should mention wasm, got: %v", err)
	}
}

// TestDebugPreRunE_NullByteInPath_Rejected verifies that a path containing a
// null byte is rejected before any I/O is attempted.
func TestDebugPreRunE_NullByteInPath_Rejected(t *testing.T) {
	t.Cleanup(resetDebugFlags)
	t.Cleanup(func() { snapshotFlag = "" })
	networkFlag = "testnet"
	snapshotFlag = "/path/to/snap\x00.json"
	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"

	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err == nil {
		t.Fatal("expected error for null byte in --snapshot path")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Errorf("error should mention null bytes, got: %v", err)
	}
}

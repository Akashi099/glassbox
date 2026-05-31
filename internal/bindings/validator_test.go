// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package bindings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dotandev/glassbox/internal/abi"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// writeBindingDir generates a full set of binding files into dir using the
// provided spec and metadata.  Returns the ABI hash that was embedded.
func writeBindingDir(t *testing.T, dir string, spec *abi.ContractSpec, meta ArtifactMetadata) string {
	t.Helper()

	hash, err := HashABI(spec)
	if err != nil {
		t.Fatalf("HashABI: %v", err)
	}
	meta.ABIHash = hash

	header := RenderMetadataHeader(meta) + "\n"
	for _, name := range generatedFileNames {
		content := header + "// generated content for " + name + "\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	return hash
}

// specToJSONBytes serialises a ContractSpec to canonical JSON bytes.
func specToJSONBytes(t *testing.T, spec *abi.ContractSpec) []byte {
	t.Helper()
	s, err := abi.FormatJSON(spec)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}
	return []byte(s)
}

// freshMeta returns an ArtifactMetadata with a fixed timestamp for tests.
func freshMeta() ArtifactMetadata {
	return ArtifactMetadata{
		ContractID:      "CTEST",
		GeneratedAt:     time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC),
		GlassboxVersion: "test",
	}
}

// ── Validate – all fresh ──────────────────────────────────────────────────────

func TestValidate_AllFresh(t *testing.T) {
	dir := t.TempDir()
	spec := minimalSpec()
	writeBindingDir(t, dir, spec, freshMeta())

	report, err := Validate(ValidatorConfig{
		OutputDir: dir,
		SpecBytes: specToJSONBytes(t, spec),
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if report.IsStale {
		t.Error("expected IsStale=false for up-to-date bindings")
	}
	if report.StaleCount != 0 {
		t.Errorf("expected StaleCount=0, got %d", report.StaleCount)
	}
	for _, f := range report.Files {
		if f.Status != StatusFresh {
			t.Errorf("file %s: expected fresh, got %s (%s)", f.Path, f.Status, f.Reason)
		}
	}
}

// ── Validate – stale (ABI changed) ───────────────────────────────────────────

func TestValidate_StaleWhenABIChanges(t *testing.T) {
	dir := t.TempDir()
	oldSpec := minimalSpec()
	writeBindingDir(t, dir, oldSpec, freshMeta())

	// Now validate against a *different* spec (extra function added).
	newSpec := minimalSpec()
	newSpec.Functions = append(newSpec.Functions, xdr.ScSpecFunctionV0{
		Name:    "burn",
		Inputs:  []xdr.ScSpecFunctionInputV0{{Name: "amount", Type: xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeU128}}},
		Outputs: []xdr.ScSpecTypeDef{{Type: xdr.ScSpecTypeScSpecTypeVoid}},
	})

	report, err := Validate(ValidatorConfig{
		OutputDir: dir,
		SpecBytes: specToJSONBytes(t, newSpec),
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if !report.IsStale {
		t.Error("expected IsStale=true when ABI has changed")
	}
	if report.StaleCount != len(generatedFileNames) {
		t.Errorf("expected all %d files stale, got %d", len(generatedFileNames), report.StaleCount)
	}
	for _, f := range report.Files {
		if f.Status != StatusStale {
			t.Errorf("file %s: expected stale, got %s", f.Path, f.Status)
		}
	}
}

// ── Validate – missing files ──────────────────────────────────────────────────

func TestValidate_MissingFiles(t *testing.T) {
	dir := t.TempDir()
	spec := minimalSpec()
	// Write only some of the files.
	hash, _ := HashABI(spec)
	meta := freshMeta()
	meta.ABIHash = hash
	header := RenderMetadataHeader(meta) + "\n"

	partial := []string{"types.ts", "client.ts"}
	for _, name := range partial {
		content := header + "// content\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	report, err := Validate(ValidatorConfig{
		OutputDir: dir,
		SpecBytes: specToJSONBytes(t, spec),
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if !report.IsStale {
		t.Error("expected IsStale=true when files are missing")
	}

	missing := 0
	for _, f := range report.Files {
		if f.Status == StatusMissing {
			missing++
		}
	}
	expected := len(generatedFileNames) - len(partial)
	if missing != expected {
		t.Errorf("expected %d missing files, got %d", expected, missing)
	}
}

// ── Validate – no metadata header ────────────────────────────────────────────

func TestValidate_NoMetadataHeader(t *testing.T) {
	dir := t.TempDir()
	// Write files without any metadata header (old-style generation).
	for _, name := range generatedFileNames {
		content := "// Auto-generated TypeScript\nexport const x = 1;\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	spec := minimalSpec()
	report, err := Validate(ValidatorConfig{
		OutputDir: dir,
		SpecBytes: specToJSONBytes(t, spec),
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if !report.IsStale {
		t.Error("expected IsStale=true for files without metadata headers")
	}
	for _, f := range report.Files {
		if f.Status != StatusNoMetadata {
			t.Errorf("file %s: expected no-metadata, got %s", f.Path, f.Status)
		}
	}
}

// ── Validate – mixed statuses ─────────────────────────────────────────────────

func TestValidate_MixedStatuses(t *testing.T) {
	dir := t.TempDir()
	spec := minimalSpec()
	hash, _ := HashABI(spec)
	meta := freshMeta()
	meta.ABIHash = hash
	header := RenderMetadataHeader(meta) + "\n"

	// Write types.ts as fresh, metadata.ts without header, leave the rest missing.
	if err := os.WriteFile(filepath.Join(dir, "types.ts"), []byte(header+"// types\n"), 0644); err != nil {
		t.Fatalf("writing types.ts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "metadata.ts"), []byte("// no header\n"), 0644); err != nil {
		t.Fatalf("writing metadata.ts: %v", err)
	}

	report, err := Validate(ValidatorConfig{
		OutputDir: dir,
		SpecBytes: specToJSONBytes(t, spec),
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if !report.IsStale {
		t.Error("expected IsStale=true for mixed statuses")
	}

	statusMap := make(map[string]ValidationStatus)
	for _, f := range report.Files {
		statusMap[f.Path] = f.Status
	}

	if statusMap["types.ts"] != StatusFresh {
		t.Errorf("types.ts: expected fresh, got %s", statusMap["types.ts"])
	}
	if statusMap["metadata.ts"] != StatusNoMetadata {
		t.Errorf("metadata.ts: expected no-metadata, got %s", statusMap["metadata.ts"])
	}
	if statusMap["client.ts"] != StatusMissing {
		t.Errorf("client.ts: expected missing, got %s", statusMap["client.ts"])
	}
}

// ── Validate – no source provided ────────────────────────────────────────────

func TestValidate_NoSource_ReturnsError(t *testing.T) {
	_, err := Validate(ValidatorConfig{OutputDir: t.TempDir()})
	if err == nil {
		t.Error("expected error when no ABI source is provided")
	}
}

// ── Validate – report fields ──────────────────────────────────────────────────

func TestValidate_ReportContainsSourceHash(t *testing.T) {
	dir := t.TempDir()
	spec := minimalSpec()
	writeBindingDir(t, dir, spec, freshMeta())

	report, err := Validate(ValidatorConfig{
		OutputDir: dir,
		SpecBytes: specToJSONBytes(t, spec),
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if report.SourceABIHash == "" {
		t.Error("SourceABIHash should not be empty")
	}
	if report.OutputDir != dir {
		t.Errorf("OutputDir: got %q, want %q", report.OutputDir, dir)
	}
	if len(report.Files) != len(generatedFileNames) {
		t.Errorf("expected %d file results, got %d", len(generatedFileNames), len(report.Files))
	}
}

// ── Validate – JSON serialisability ──────────────────────────────────────────

func TestValidate_ReportIsJSONSerialisable(t *testing.T) {
	dir := t.TempDir()
	spec := minimalSpec()
	writeBindingDir(t, dir, spec, freshMeta())

	report, err := Validate(ValidatorConfig{
		OutputDir: dir,
		SpecBytes: specToJSONBytes(t, spec),
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	b, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal(report): %v", err)
	}
	if len(b) == 0 {
		t.Error("marshalled report is empty")
	}

	// Round-trip: unmarshal back and check key fields.
	var decoded ValidationReport
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if decoded.SourceABIHash != report.SourceABIHash {
		t.Errorf("round-trip SourceABIHash mismatch: %q vs %q", decoded.SourceABIHash, report.SourceABIHash)
	}
}

// ── Full Generate() → Validate() pipeline ────────────────────────────────────

func TestGenerateThenValidate_Fresh(t *testing.T) {
	spec := minimalSpec()
	jsonBytes := specToJSONBytes(t, spec)

	dir := t.TempDir()
	cfg := GeneratorConfig{
		SpecBytes:   jsonBytes,
		SpecFormat:  abi.ImportFormatJSON,
		OutputDir:   dir,
		PackageName: "test-contract",
		Network:     "testnet",
	}
	gen := NewGenerator(cfg)
	files, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Write files to disk.
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.Path), []byte(f.Content), 0644); err != nil {
			t.Fatalf("writing %s: %v", f.Path, err)
		}
	}

	// Validate against the same spec → should be fresh.
	report, err := Validate(ValidatorConfig{
		OutputDir: dir,
		SpecBytes: jsonBytes,
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if report.IsStale {
		for _, f := range report.Files {
			if f.Status != StatusFresh {
				t.Logf("  %s: %s – %s", f.Path, f.Status, f.Reason)
			}
		}
		t.Error("expected fresh bindings immediately after generation")
	}
}

func TestGenerateThenValidate_StaleAfterABIChange(t *testing.T) {
	oldSpec := minimalSpec()
	jsonBytes := specToJSONBytes(t, oldSpec)

	dir := t.TempDir()
	cfg := GeneratorConfig{
		SpecBytes:   jsonBytes,
		SpecFormat:  abi.ImportFormatJSON,
		OutputDir:   dir,
		PackageName: "test-contract",
		Network:     "testnet",
	}
	gen := NewGenerator(cfg)
	files, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.Path), []byte(f.Content), 0644); err != nil {
			t.Fatalf("writing %s: %v", f.Path, err)
		}
	}

	// Simulate an ABI change.
	newSpec := minimalSpec()
	newSpec.Functions = append(newSpec.Functions, xdr.ScSpecFunctionV0{
		Name:    "approve",
		Inputs:  []xdr.ScSpecFunctionInputV0{{Name: "spender", Type: xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeAddress}}},
		Outputs: []xdr.ScSpecTypeDef{{Type: xdr.ScSpecTypeScSpecTypeVoid}},
	})

	report, err := Validate(ValidatorConfig{
		OutputDir: dir,
		SpecBytes: specToJSONBytes(t, newSpec),
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !report.IsStale {
		t.Error("expected stale bindings after ABI change")
	}
	if report.StaleCount != len(generatedFileNames) {
		t.Errorf("expected all %d files stale, got %d", len(generatedFileNames), report.StaleCount)
	}
}

// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dotandev/glassbox/internal/abi"
	"github.com/dotandev/glassbox/internal/bindings"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// writeTestBindingDir writes a complete set of generated binding files into dir
// using the provided spec.  Returns the ABI hash that was embedded.
func writeTestBindingDir(t *testing.T, dir string, spec *abi.ContractSpec) string {
	t.Helper()
	hash, err := bindings.HashABI(spec)
	if err != nil {
		t.Fatalf("HashABI: %v", err)
	}
	meta := bindings.ArtifactMetadata{
		ABIHash:         hash,
		ContractID:      "",
		GeneratedAt:     time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC),
		GlassboxVersion: "test",
	}
	header := bindings.RenderMetadataHeader(meta) + "\n"
	for _, name := range []string{
		"types.ts", "metadata.ts", "client.ts",
		"Glassbox-integration.ts", "index.ts", "package.json", "README.md",
	} {
		content := header + "// content for " + name + "\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	return hash
}

// minimalTestSpec returns a minimal ContractSpec for CLI tests.
func minimalTestSpec() *abi.ContractSpec {
	return &abi.ContractSpec{
		Functions: []xdr.ScSpecFunctionV0{
			{
				Name: "transfer",
				Inputs: []xdr.ScSpecFunctionInputV0{
					{Name: "to", Type: xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeAddress}},
					{Name: "amount", Type: xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeU128}},
				},
				Outputs: []xdr.ScSpecTypeDef{{Type: xdr.ScSpecTypeScSpecTypeVoid}},
			},
		},
	}
}

// specJSONFile writes a spec as JSON to a temp file and returns its path.
func specJSONFile(t *testing.T, spec *abi.ContractSpec) string {
	t.Helper()
	jsonStr, err := abi.FormatJSON(spec)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}
	f, err := os.CreateTemp(t.TempDir(), "spec-*.json")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := f.WriteString(jsonStr); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	f.Close()
	return f.Name()
}

// runCheckBindingsCmd executes the check-bindings command with the given args
// and returns stdout, stderr, and the error.
func runCheckBindingsCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()

	// Reset package-level flag variables between test runs.
	checkBindingsOutput = ""
	checkBindingsSpecFile = ""
	checkBindingsSpecFormat = ""
	checkBindingsJSON = false
	checkBindingsRegenerate = false
	checkBindingsNetwork = "testnet"
	checkBindingsPackage = ""
	checkBindingsRuntime = "node"
	checkBindingsContractID = ""
	checkBindingsDebugMeta = false

	var buf bytes.Buffer
	checkBindingsCmd.SetOut(&buf)
	checkBindingsCmd.SetErr(&buf)
	checkBindingsCmd.SetArgs(args)

	err := checkBindingsCmd.Execute()
	return buf.String(), err
}

// ── validateCheckBindingsFlags ────────────────────────────────────────────────

func TestValidateCheckBindingsFlags_NoSource_ReturnsError(t *testing.T) {
	err := validateCheckBindingsFlags(nil, "", "", "")
	if err == nil {
		t.Error("expected error when no source is provided")
	}
}

func TestValidateCheckBindingsFlags_BothSources_ReturnsError(t *testing.T) {
	err := validateCheckBindingsFlags([]string{"contract.wasm"}, "", "spec.json", "")
	if err == nil {
		t.Error("expected error when both wasm-file and --spec-file are provided")
	}
}

func TestValidateCheckBindingsFlags_InvalidSpecFormat_ReturnsError(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "spec-*.json")
	f.Close()
	err := validateCheckBindingsFlags(nil, "", f.Name(), "yaml")
	if err == nil {
		t.Error("expected error for unsupported spec-format")
	}
}

func TestValidateCheckBindingsFlags_OutputIsFile_ReturnsError(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "notadir-*")
	f.Close()
	specFile, _ := os.CreateTemp(t.TempDir(), "spec-*.json")
	specFile.Close()
	err := validateCheckBindingsFlags(nil, f.Name(), specFile.Name(), "")
	if err == nil {
		t.Error("expected error when --output points to a file, not a directory")
	}
}

func TestValidateCheckBindingsFlags_ValidWasm(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "contract-*.wasm")
	f.Close()
	err := validateCheckBindingsFlags([]string{f.Name()}, "", "", "")
	if err != nil {
		t.Errorf("unexpected error for valid wasm path: %v", err)
	}
}

func TestValidateCheckBindingsFlags_ValidSpecFile(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "spec-*.json")
	f.Close()
	err := validateCheckBindingsFlags(nil, "", f.Name(), "json")
	if err != nil {
		t.Errorf("unexpected error for valid spec-file: %v", err)
	}
}

// ── check-bindings command – text output ──────────────────────────────────────

func TestCheckBindingsCmd_FreshBindings_ExitsZero(t *testing.T) {
	dir := t.TempDir()
	spec := minimalTestSpec()
	writeTestBindingDir(t, dir, spec)
	specFile := specJSONFile(t, spec)

	out, err := runCheckBindingsCmd(t, "--spec-file", specFile, "--output", dir)
	if err != nil {
		t.Errorf("expected exit 0 for fresh bindings, got error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "[OK]") {
		t.Errorf("expected [OK] in output, got:\n%s", out)
	}
}

func TestCheckBindingsCmd_StaleBindings_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	oldSpec := minimalTestSpec()
	writeTestBindingDir(t, dir, oldSpec)

	// Validate against a changed spec.
	newSpec := minimalTestSpec()
	newSpec.Functions = append(newSpec.Functions, xdr.ScSpecFunctionV0{
		Name:    "burn",
		Inputs:  []xdr.ScSpecFunctionInputV0{{Name: "amount", Type: xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeU128}}},
		Outputs: []xdr.ScSpecTypeDef{{Type: xdr.ScSpecTypeScSpecTypeVoid}},
	})
	specFile := specJSONFile(t, newSpec)

	out, err := runCheckBindingsCmd(t, "--spec-file", specFile, "--output", dir)
	if err == nil {
		t.Errorf("expected non-zero exit for stale bindings\noutput: %s", out)
	}
	if !strings.Contains(out, "[STALE]") {
		t.Errorf("expected [STALE] in output, got:\n%s", out)
	}
}

func TestCheckBindingsCmd_MissingFiles_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	spec := minimalTestSpec()
	specFile := specJSONFile(t, spec)
	// Don't write any binding files – all should be missing.

	out, err := runCheckBindingsCmd(t, "--spec-file", specFile, "--output", dir)
	if err == nil {
		t.Errorf("expected non-zero exit for missing files\noutput: %s", out)
	}
	if !strings.Contains(out, "missing") {
		t.Errorf("expected 'missing' in output, got:\n%s", out)
	}
}

// ── check-bindings command – JSON output ──────────────────────────────────────

func TestCheckBindingsCmd_JSONOutput_FreshBindings(t *testing.T) {
	dir := t.TempDir()
	spec := minimalTestSpec()
	writeTestBindingDir(t, dir, spec)
	specFile := specJSONFile(t, spec)

	out, err := runCheckBindingsCmd(t, "--spec-file", specFile, "--output", dir, "--json")
	if err != nil {
		t.Errorf("expected exit 0 for fresh bindings with --json: %v\noutput: %s", err, out)
	}
	// Output should be valid JSON containing isStale=false.
	if !strings.Contains(out, `"isStale"`) {
		t.Errorf("expected JSON output with isStale field, got:\n%s", out)
	}
	if !strings.Contains(out, `"isStale": false`) {
		t.Errorf("expected isStale=false in JSON output, got:\n%s", out)
	}
}

func TestCheckBindingsCmd_JSONOutput_StaleBindings(t *testing.T) {
	dir := t.TempDir()
	oldSpec := minimalTestSpec()
	writeTestBindingDir(t, dir, oldSpec)

	newSpec := minimalTestSpec()
	newSpec.Functions = append(newSpec.Functions, xdr.ScSpecFunctionV0{
		Name:    "approve",
		Inputs:  []xdr.ScSpecFunctionInputV0{{Name: "spender", Type: xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeAddress}}},
		Outputs: []xdr.ScSpecTypeDef{{Type: xdr.ScSpecTypeScSpecTypeVoid}},
	})
	specFile := specJSONFile(t, newSpec)

	out, err := runCheckBindingsCmd(t, "--spec-file", specFile, "--output", dir, "--json")
	if err == nil {
		t.Error("expected non-zero exit for stale bindings with --json")
	}
	// JSON output should still be written even when stale.
	if !strings.Contains(out, `"isStale": true`) {
		t.Errorf("expected isStale=true in JSON output, got:\n%s", out)
	}
}

// ── check-bindings command – --regenerate ────────────────────────────────────

func TestCheckBindingsCmd_Regenerate_WritesFiles(t *testing.T) {
	dir := t.TempDir()
	oldSpec := minimalTestSpec()
	writeTestBindingDir(t, dir, oldSpec)

	// Change the ABI.
	newSpec := minimalTestSpec()
	newSpec.Functions = append(newSpec.Functions, xdr.ScSpecFunctionV0{
		Name:    "burn",
		Inputs:  []xdr.ScSpecFunctionInputV0{{Name: "amount", Type: xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeU128}}},
		Outputs: []xdr.ScSpecTypeDef{{Type: xdr.ScSpecTypeScSpecTypeVoid}},
	})
	specFile := specJSONFile(t, newSpec)

	out, err := runCheckBindingsCmd(t,
		"--spec-file", specFile,
		"--output", dir,
		"--regenerate",
		"--package", "test-contract",
	)
	if err != nil {
		t.Errorf("expected exit 0 after regeneration, got: %v\noutput: %s", err, out)
	}

	// After regeneration the files should be fresh.
	report, valErr := bindings.Validate(bindings.ValidatorConfig{
		OutputDir: dir,
		SpecBytes: func() []byte {
			s, _ := abi.FormatJSON(newSpec)
			return []byte(s)
		}(),
	})
	if valErr != nil {
		t.Fatalf("post-regeneration Validate: %v", valErr)
	}
	if report.IsStale {
		for _, f := range report.Files {
			if f.Status != bindings.StatusFresh {
				t.Logf("  %s: %s – %s", f.Path, f.Status, f.Reason)
			}
		}
		t.Error("expected fresh bindings after --regenerate")
	}
}

func TestCheckBindingsCmd_Regenerate_AlreadyFresh_ExitsZero(t *testing.T) {
	dir := t.TempDir()
	spec := minimalTestSpec()
	writeTestBindingDir(t, dir, spec)
	specFile := specJSONFile(t, spec)

	out, err := runCheckBindingsCmd(t,
		"--spec-file", specFile,
		"--output", dir,
		"--regenerate",
		"--package", "test-contract",
	)
	if err != nil {
		t.Errorf("expected exit 0 when already fresh with --regenerate: %v\noutput: %s", err, out)
	}
}

// ── ValidationReport JSON structure ──────────────────────────────────────────

func TestValidationReport_JSONFields(t *testing.T) {
	dir := t.TempDir()
	spec := minimalTestSpec()
	writeTestBindingDir(t, dir, spec)

	jsonStr, _ := abi.FormatJSON(spec)
	report, err := bindings.Validate(bindings.ValidatorConfig{
		OutputDir: dir,
		SpecBytes: []byte(jsonStr),
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	b, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	for _, field := range []string{"outputDir", "sourceABIHash", "isStale", "files", "staleCount"} {
		if _, ok := m[field]; !ok {
			t.Errorf("JSON report missing field %q", field)
		}
	}
}

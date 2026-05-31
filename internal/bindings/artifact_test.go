// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package bindings

import (
	"strings"
	"testing"
	"time"

	"github.com/dotandev/glassbox/internal/abi"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// ── HashABI ───────────────────────────────────────────────────────────────────

func TestHashABI_Deterministic(t *testing.T) {
	spec := minimalSpec()
	h1, err := HashABI(spec)
	if err != nil {
		t.Fatalf("HashABI: %v", err)
	}
	h2, err := HashABI(spec)
	if err != nil {
		t.Fatalf("HashABI second call: %v", err)
	}
	if h1 != h2 {
		t.Errorf("HashABI is not deterministic: %q != %q", h1, h2)
	}
}

func TestHashABI_ChangesWhenABIChanges(t *testing.T) {
	spec1 := minimalSpec()
	h1, err := HashABI(spec1)
	if err != nil {
		t.Fatalf("HashABI spec1: %v", err)
	}

	// Add a second function to the spec.
	spec2 := minimalSpec()
	spec2.Functions = append(spec2.Functions, xdr.ScSpecFunctionV0{
		Name: "burn",
		Inputs: []xdr.ScSpecFunctionInputV0{
			{Name: "amount", Type: xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeU128}},
		},
		Outputs: []xdr.ScSpecTypeDef{{Type: xdr.ScSpecTypeScSpecTypeVoid}},
	})
	h2, err := HashABI(spec2)
	if err != nil {
		t.Fatalf("HashABI spec2: %v", err)
	}

	if h1 == h2 {
		t.Error("HashABI should differ when the ABI changes")
	}
}

func TestHashABI_IsHexString(t *testing.T) {
	h, err := HashABI(minimalSpec())
	if err != nil {
		t.Fatalf("HashABI: %v", err)
	}
	if len(h) != 64 {
		t.Errorf("expected 64-char hex SHA-256, got %d chars: %q", len(h), h)
	}
	for _, c := range h {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("non-hex character %q in hash %q", c, h)
		}
	}
}

// ── HashABIBytes ──────────────────────────────────────────────────────────────

func TestHashABIBytes_JSONMatchesHashABI(t *testing.T) {
	spec := minimalSpec()
	jsonBytes, err := abi.FormatJSON(spec)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	hFromBytes, err := HashABIBytes([]byte(jsonBytes), abi.ImportFormatJSON)
	if err != nil {
		t.Fatalf("HashABIBytes: %v", err)
	}
	hFromSpec, err := HashABI(spec)
	if err != nil {
		t.Fatalf("HashABI: %v", err)
	}
	if hFromBytes != hFromSpec {
		t.Errorf("HashABIBytes(JSON) != HashABI: %q vs %q", hFromBytes, hFromSpec)
	}
}

func TestHashABIBytes_AutoDetect(t *testing.T) {
	spec := minimalSpec()
	jsonBytes, err := abi.FormatJSON(spec)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	// Empty format → auto-detect.
	h, err := HashABIBytes([]byte(jsonBytes), "")
	if err != nil {
		t.Fatalf("HashABIBytes auto-detect: %v", err)
	}
	expected, _ := HashABI(spec)
	if h != expected {
		t.Errorf("auto-detect hash mismatch: %q vs %q", h, expected)
	}
}

// ── RenderMetadataHeader / ParseMetadataHeader round-trip ────────────────────

func TestRenderAndParseMetadataHeader_RoundTrip(t *testing.T) {
	ts := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	original := ArtifactMetadata{
		ABIHash:         "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		ContractID:      "CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQAHHAGCN4B2",
		GeneratedAt:     ts,
		GlassboxVersion: "1.2.3",
	}

	header := RenderMetadataHeader(original)
	// Simulate a full file: header + some TypeScript content.
	fileContent := header + "\n// some TypeScript\nexport const x = 1;\n"

	parsed, err := ParseMetadataHeader(fileContent)
	if err != nil {
		t.Fatalf("ParseMetadataHeader: %v", err)
	}
	if parsed == nil {
		t.Fatal("ParseMetadataHeader returned nil for a file with a header")
	}

	if parsed.ABIHash != original.ABIHash {
		t.Errorf("ABIHash: got %q, want %q", parsed.ABIHash, original.ABIHash)
	}
	if parsed.ContractID != original.ContractID {
		t.Errorf("ContractID: got %q, want %q", parsed.ContractID, original.ContractID)
	}
	if !parsed.GeneratedAt.Equal(original.GeneratedAt) {
		t.Errorf("GeneratedAt: got %v, want %v", parsed.GeneratedAt, original.GeneratedAt)
	}
	if parsed.GlassboxVersion != original.GlassboxVersion {
		t.Errorf("GlassboxVersion: got %q, want %q", parsed.GlassboxVersion, original.GlassboxVersion)
	}
}

func TestParseMetadataHeader_NoHeader_ReturnsNil(t *testing.T) {
	content := "// Auto-generated TypeScript types\nexport type Address = string;\n"
	meta, err := ParseMetadataHeader(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta != nil {
		t.Errorf("expected nil for file without metadata header, got %+v", meta)
	}
}

func TestParseMetadataHeader_EmptyFile_ReturnsNil(t *testing.T) {
	meta, err := ParseMetadataHeader("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta != nil {
		t.Errorf("expected nil for empty file, got %+v", meta)
	}
}

func TestParseMetadataHeader_InvalidTimestamp_ReturnsError(t *testing.T) {
	content := "/* @glassbox-bindings-meta\n * abi-hash:    abc123\n * contract-id: \n * generated:   not-a-timestamp\n * glassbox:    1.0.0\n */\n"
	_, err := ParseMetadataHeader(content)
	if err == nil {
		t.Error("expected error for invalid timestamp, got nil")
	}
}

func TestRenderMetadataHeader_ContainsAllFields(t *testing.T) {
	meta := ArtifactMetadata{
		ABIHash:         "deadbeef",
		ContractID:      "CTEST",
		GeneratedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		GlassboxVersion: "0.9.0",
	}
	header := RenderMetadataHeader(meta)

	for _, want := range []string{
		"@glassbox-bindings-meta",
		"abi-hash:",
		"deadbeef",
		"contract-id:",
		"CTEST",
		"generated:",
		"2026-01-01T00:00:00Z",
		"glassbox:",
		"0.9.0",
	} {
		if !strings.Contains(header, want) {
			t.Errorf("header missing %q\nheader:\n%s", want, header)
		}
	}
}

// ── Generate() embeds metadata header ────────────────────────────────────────

func TestGenerate_EmbedArtifactMetadata_HeaderPresent(t *testing.T) {
	g := buildTestGeneratorWithMeta(t)
	files, err := generateFromSpec(g)
	if err != nil {
		t.Fatalf("generateFromSpec: %v", err)
	}

	for _, f := range files {
		if !strings.Contains(f.Content, "@glassbox-bindings-meta") {
			t.Errorf("file %s is missing @glassbox-bindings-meta header", f.Path)
		}
		if !strings.Contains(f.Content, "abi-hash:") {
			t.Errorf("file %s is missing abi-hash field", f.Path)
		}
	}
}

func TestGenerate_EmbedArtifactMetadata_HashIsConsistent(t *testing.T) {
	g := buildTestGeneratorWithMeta(t)
	files, err := generateFromSpec(g)
	if err != nil {
		t.Fatalf("generateFromSpec: %v", err)
	}

	// All files should carry the same ABI hash.
	var firstHash string
	for _, f := range files {
		meta, err := ParseMetadataHeader(f.Content)
		if err != nil {
			t.Fatalf("ParseMetadataHeader(%s): %v", f.Path, err)
		}
		if meta == nil {
			t.Fatalf("file %s has no metadata header", f.Path)
		}
		if firstHash == "" {
			firstHash = meta.ABIHash
		} else if meta.ABIHash != firstHash {
			t.Errorf("file %s has different ABI hash: %q vs %q", f.Path, meta.ABIHash, firstHash)
		}
	}
}

func TestGenerate_NoEmbedArtifactMetadata_NoHeader(t *testing.T) {
	g := buildTestGenerator()
	g.config.NoEmbedArtifactMetadata = true
	g.config.artifactMeta = nil

	files, err := generateFromSpec(g)
	if err != nil {
		t.Fatalf("generateFromSpec: %v", err)
	}
	for _, f := range files {
		if strings.Contains(f.Content, "@glassbox-bindings-meta") {
			t.Errorf("file %s should not have metadata header when EmbedArtifactMetadata=false", f.Path)
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// minimalSpec returns a ContractSpec with a single function for use in tests.
func minimalSpec() *abi.ContractSpec {
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

// buildTestGeneratorWithMeta returns a Generator that has EmbedArtifactMetadata
// enabled and a fixed generation time so tests are deterministic.
func buildTestGeneratorWithMeta(t *testing.T) *Generator {
	t.Helper()
	g := buildTestGenerator()
	g.config.fixedGenerationTime = time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)

	// Pre-compute the artifact metadata (normally done by Generate()).
	hash, err := HashABI(g.spec)
	if err != nil {
		t.Fatalf("HashABI: %v", err)
	}
	g.config.artifactMeta = &ArtifactMetadata{
		ABIHash:         hash,
		ContractID:      g.config.ContractID,
		GeneratedAt:     g.config.fixedGenerationTime,
		GlassboxVersion: "test",
	}
	return g
}

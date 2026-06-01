// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	ctx := context.Background()

	// Test with tracing disabled
	cleanup, err := Init(ctx, Config{
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("Failed to initialize telemetry with disabled config: %v", err)
	}
	cleanup()

	// Graceful degradation: Init must never fail even when collector is unreachable
	cleanup, err = Init(ctx, Config{
		Enabled:     true,
		ExporterURL: "http://localhost:4318",
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Init must not fail when collector is down (graceful degradation): %v", err)
	}
	cleanup()

	// Tracer is always available (no-op if collector was unreachable)
	tracer := GetTracer()
	if tracer == nil {
		t.Fatal("Tracer should not be nil after initialization")
	}
	_, span := tracer.Start(ctx, "test-span")
	span.End()
}

func TestGetTracer(t *testing.T) {
	// Should not panic even if not initialized
	tracer := GetTracer()
	if tracer == nil {
		t.Fatal("GetTracer should never return nil")
	}

	// Should be able to create spans (no-op if not initialized)
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	span.End()
}

func TestSanitizeValueAndAttr(t *testing.T) {
	// Hash-like key should be fingerprinted, not include raw value
	raw := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	sv := SanitizeValue("transaction.hash", raw)
	if !strings.HasPrefix(sv, "sha256:") {
		t.Fatalf("expected fingerprint prefix, got: %s", sv)
	}
	if strings.Contains(sv, raw[:8]) {
		t.Fatalf("sanitized value must not contain raw prefix")
	}

	kv := Attr("transaction.hash", raw)
	if !strings.HasPrefix(kv.Value.AsString(), "sha256:") {
		t.Fatalf("Attr did not sanitize value: %v", kv)
	}
}

// TestInit_UnreachableCollector proves graceful degradation: with tracing enabled
// and an unreachable OTLP endpoint, Init succeeds and core paths (GetTracer, spans)
// work without blocking or error. Run with: go test ./internal/telemetry/... -v -run TestInit_UnreachableCollector
func TestInit_UnreachableCollector(t *testing.T) {
	ctx := context.Background()
	// Use a port that nothing listens on so the collector is "down"
	cleanup, err := Init(ctx, Config{
		Enabled:     true,
		ExporterURL: "http://127.0.0.1:37999",
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("graceful degradation: Init must not fail when collector is down, got: %v", err)
	}
	defer cleanup()

	tracer := GetTracer()
	if tracer == nil {
		t.Fatal("GetTracer must never return nil")
	}
	_, span := tracer.Start(ctx, "telemetry-test-span")
	span.End()
	// If we get here without blocking or panic, telemetry fails silently as intended
}
func TestEnvMetadata(t *testing.T) {
	ctx := context.Background()

	// Initialize with anonymization disabled to capture env metadata
	cleanup, err := Init(ctx, Config{
		Enabled:     true,
		ExporterURL: "http://localhost:4318",
		ServiceName: "test-service",
		Anonymized:  false,
	})
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	defer cleanup()

	// Verify env metadata was captured
	meta := GetEnvMetadata()
	if meta.Version == "" {
		t.Error("Expected version to be set")
	}
	if meta.Platform == "" {
		t.Error("Expected platform to be set")
	}
	if meta.Arch == "" {
		t.Error("Expected arch to be set")
	}
	// Feature flags may be empty but should not panic
	_ = meta.FeatureFlags
}

func TestEnvMetadata_Anonymized(t *testing.T) {
	ctx := context.Background()

	// Initialize with anonymization enabled
	cleanup, err := Init(ctx, Config{
		Enabled:     true,
		ExporterURL: "http://localhost:4318",
		ServiceName: "test-service",
		Anonymized:  true,
	})
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	defer cleanup()

	// When anonymized, sensitive env data should not be exposed in traces
	meta := GetEnvMetadata()
	if !meta.Anonymized {
		t.Error("Expected anonymized flag to be true")
	}
}

func TestRecordCommandUsage_WithMetadata(t *testing.T) {
	ctx := context.Background()

	// Initialize with metadata collection
	cleanup, err := Init(ctx, Config{
		Enabled:     true,
		ExporterURL: "http://localhost:4318",
		ServiceName: "test-service",
		Anonymized:  false,
	})
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	defer cleanup()

	// Record a command usage event - should not panic
	RecordCommandUsage(ctx, "test-command")

	// If we get here without panic, the metadata was handled correctly
}

func TestGetVersion(t *testing.T) {
	// Test that getVersion returns non-empty string
	// In dev mode it returns "dev", in production it would be the actual version
	version := getVersion()
	if version == "" {
		t.Error("Expected non-empty version")
	}
}

func TestGetFeatureFlags(t *testing.T) {
	// Test feature flag detection
	// Note: This test may behave differently based on environment
	flags := getFeatureFlags()
	// Should return a slice (may be empty)
	if flags == nil {
		t.Error("Expected non-nil feature flags slice")
	}
}
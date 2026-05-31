// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package snapshot

import (
	"strings"
	"testing"
)

func TestDiffSnapshots_Identical(t *testing.T) {
	snap := FromMap(map[string]string{
		"keyA": "valueA",
		"keyB": "valueB",
	})

	diff := DiffSnapshots(snap, snap)

	if diff.TotalChanges() != 0 {
		t.Errorf("identical snapshots should produce no changes, got %d", diff.TotalChanges())
	}
	if len(diff.Added) != 0 {
		t.Errorf("expected 0 added, got %d", len(diff.Added))
	}
	if len(diff.Removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(diff.Removed))
	}
	if len(diff.Modified) != 0 {
		t.Errorf("expected 0 modified, got %d", len(diff.Modified))
	}
}

func TestDiffSnapshots_AddedEntries(t *testing.T) {
	base := FromMap(map[string]string{
		"keyA": "valueA",
	})
	target := FromMap(map[string]string{
		"keyA": "valueA",
		"keyB": "valueB",
		"keyC": "valueC",
	})

	diff := DiffSnapshots(base, target)

	if len(diff.Added) != 2 {
		t.Errorf("expected 2 added, got %d", len(diff.Added))
	}
	if len(diff.Removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(diff.Removed))
	}
	if len(diff.Modified) != 0 {
		t.Errorf("expected 0 modified, got %d", len(diff.Modified))
	}

	// Verify sort order
	if diff.Added[0].Key != "keyB" || diff.Added[1].Key != "keyC" {
		t.Errorf("added entries not sorted: %v", diff.Added)
	}
	for _, ch := range diff.Added {
		if ch.Kind != ChangeAdded {
			t.Errorf("expected kind %q, got %q", ChangeAdded, ch.Kind)
		}
	}
}

func TestDiffSnapshots_RemovedEntries(t *testing.T) {
	base := FromMap(map[string]string{
		"keyA": "valueA",
		"keyB": "valueB",
	})
	target := FromMap(map[string]string{
		"keyA": "valueA",
	})

	diff := DiffSnapshots(base, target)

	if len(diff.Removed) != 1 {
		t.Errorf("expected 1 removed, got %d", len(diff.Removed))
	}
	if diff.Removed[0].Key != "keyB" {
		t.Errorf("expected removed key %q, got %q", "keyB", diff.Removed[0].Key)
	}
	if diff.Removed[0].OldValue != "valueB" {
		t.Errorf("expected removed old value %q, got %q", "valueB", diff.Removed[0].OldValue)
	}
	if diff.Removed[0].Kind != ChangeRemoved {
		t.Errorf("expected kind %q, got %q", ChangeRemoved, diff.Removed[0].Kind)
	}
}

func TestDiffSnapshots_ModifiedEntries(t *testing.T) {
	base := FromMap(map[string]string{
		"keyA": "oldValue",
		"keyB": "unchanged",
	})
	target := FromMap(map[string]string{
		"keyA": "newValue",
		"keyB": "unchanged",
	})

	diff := DiffSnapshots(base, target)

	if len(diff.Modified) != 1 {
		t.Errorf("expected 1 modified, got %d", len(diff.Modified))
	}
	ch := diff.Modified[0]
	if ch.Key != "keyA" {
		t.Errorf("expected modified key %q, got %q", "keyA", ch.Key)
	}
	if ch.OldValue != "oldValue" {
		t.Errorf("expected old value %q, got %q", "oldValue", ch.OldValue)
	}
	if ch.NewValue != "newValue" {
		t.Errorf("expected new value %q, got %q", "newValue", ch.NewValue)
	}
	if ch.Kind != ChangeModified {
		t.Errorf("expected kind %q, got %q", ChangeModified, ch.Kind)
	}
}

func TestDiffSnapshots_MixedChanges(t *testing.T) {
	base := FromMap(map[string]string{
		"contract:AAAA": "state1",
		"contract:BBBB": "state2",
		"contract:CCCC": "state3",
	})
	target := FromMap(map[string]string{
		"contract:AAAA": "state1-updated",
		"contract:CCCC": "state3",
		"contract:DDDD": "state4-new",
	})

	diff := DiffSnapshots(base, target)

	if len(diff.Added) != 1 {
		t.Errorf("expected 1 added, got %d", len(diff.Added))
	}
	if len(diff.Removed) != 1 {
		t.Errorf("expected 1 removed, got %d", len(diff.Removed))
	}
	if len(diff.Modified) != 1 {
		t.Errorf("expected 1 modified, got %d", len(diff.Modified))
	}
}

func TestDiffSnapshots_EmptyBase(t *testing.T) {
	base := FromMap(nil)
	target := FromMap(map[string]string{
		"keyA": "valueA",
	})

	diff := DiffSnapshots(base, target)

	if len(diff.Added) != 1 {
		t.Errorf("expected 1 added, got %d", len(diff.Added))
	}
	if diff.TotalChanges() != 1 {
		t.Errorf("expected 1 total change, got %d", diff.TotalChanges())
	}
}

func TestDiffSnapshots_EmptyTarget(t *testing.T) {
	base := FromMap(map[string]string{
		"keyA": "valueA",
	})
	target := FromMap(nil)

	diff := DiffSnapshots(base, target)

	if len(diff.Removed) != 1 {
		t.Errorf("expected 1 removed, got %d", len(diff.Removed))
	}
}

func TestDiffSnapshots_FingerprintsPreserved(t *testing.T) {
	base := FromMap(map[string]string{"k": "v1"})
	target := FromMap(map[string]string{"k": "v2"})

	diff := DiffSnapshots(base, target)

	if diff.BaseFingerprint == "" {
		t.Error("base fingerprint should be set")
	}
	if diff.TargetFingerprint == "" {
		t.Error("target fingerprint should be set")
	}
	if diff.BaseFingerprint == diff.TargetFingerprint {
		t.Error("different snapshots should have different fingerprints")
	}
}

func TestFormatDiff_Identical(t *testing.T) {
	snap := FromMap(map[string]string{"k": "v"})
	diff := DiffSnapshots(snap, snap)
	output := FormatDiff(diff)

	if !strings.Contains(output, "identical") {
		t.Errorf("expected 'identical' in output for no-change diff, got: %s", output)
	}
}

func TestFormatDiff_ShowsSummaryLine(t *testing.T) {
	base := FromMap(map[string]string{
		"k1": "v1",
		"k2": "v2",
	})
	target := FromMap(map[string]string{
		"k1": "v1-new",
		"k3": "v3",
	})

	diff := DiffSnapshots(base, target)
	output := FormatDiff(diff)

	if !strings.Contains(output, "1 added") {
		t.Errorf("expected '1 added' in output, got: %s", output)
	}
	if !strings.Contains(output, "1 removed") {
		t.Errorf("expected '1 removed' in output, got: %s", output)
	}
	if !strings.Contains(output, "1 modified") {
		t.Errorf("expected '1 modified' in output, got: %s", output)
	}
}

func TestFormatDiff_PrefixSymbols(t *testing.T) {
	base := FromMap(map[string]string{
		"gone": "old",
	})
	target := FromMap(map[string]string{
		"new": "fresh",
	})

	diff := DiffSnapshots(base, target)
	output := FormatDiff(diff)

	if !strings.Contains(output, "+") {
		t.Errorf("expected '+' prefix for added entry, got: %s", output)
	}
	if !strings.Contains(output, "-") {
		t.Errorf("expected '-' prefix for removed entry, got: %s", output)
	}
}

func TestFormatDiff_ModifiedShowsOldAndNew(t *testing.T) {
	base := FromMap(map[string]string{"key": "old-value"})
	target := FromMap(map[string]string{"key": "new-value"})

	diff := DiffSnapshots(base, target)
	output := FormatDiff(diff)

	if !strings.Contains(output, "~") {
		t.Errorf("expected '~' prefix for modified entry, got: %s", output)
	}
	if !strings.Contains(output, "old:") {
		t.Errorf("expected 'old:' label in modified output, got: %s", output)
	}
	if !strings.Contains(output, "new:") {
		t.Errorf("expected 'new:' label in modified output, got: %s", output)
	}
}

func TestFormatDiff_IncludesFingerprints(t *testing.T) {
	base := FromMap(map[string]string{"k": "v1"})
	target := FromMap(map[string]string{"k": "v2"})

	diff := DiffSnapshots(base, target)
	output := FormatDiff(diff)

	if !strings.Contains(output, "fingerprint") {
		t.Errorf("expected fingerprints in diff output, got: %s", output)
	}
}

func TestTotalChanges(t *testing.T) {
	diff := &SnapshotDiff{
		Added:    []EntryChange{{}, {}},
		Removed:  []EntryChange{{}},
		Modified: []EntryChange{{}, {}, {}},
	}
	if diff.TotalChanges() != 6 {
		t.Errorf("expected TotalChanges()=6, got %d", diff.TotalChanges())
	}
}

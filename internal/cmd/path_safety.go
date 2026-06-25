// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dotandev/glassbox/internal/errors"
)

// PathKind distinguishes input files from output paths so validation rules
// can differ (inputs must exist; outputs must have a writable parent dir).
type PathKind int

const (
	// PathKindInput — the file must already exist and be readable.
	PathKindInput PathKind = iota
	// PathKindOutput — the file may not exist yet; its parent dir must be
	// reachable and writable.
	PathKindOutput
)

// NormalizePath cleans and resolves a user-supplied path to an absolute,
// symlink-resolved form. It rejects:
//   - empty paths (returned as-is with no error — callers check optionality)
//   - paths containing null bytes (shell injection risk)
//   - paths that resolve outside the allowed root when root != ""
//
// Returns the normalized absolute path and nil on success.
func NormalizePath(flag, rawPath, allowedRoot string) (string, error) {
	if rawPath == "" {
		return "", nil
	}

	// Null bytes in paths are a shell injection vector.
	if strings.ContainsRune(rawPath, 0) {
		return "", errors.WrapValidationError(fmt.Sprintf(
			"--%s: path contains null bytes and cannot be used: %q", flag, rawPath,
		))
	}

	// Clean the path to resolve . and .. components before stat.
	cleaned := filepath.Clean(rawPath)

	// Resolve to absolute so relative paths like ../../etc/passwd are caught.
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", errors.WrapValidationError(fmt.Sprintf(
			"--%s: cannot resolve path %q to an absolute path: %v", flag, rawPath, err,
		))
	}

	// Resolve symlinks to detect indirect traversal.
	// For output paths the target may not exist yet, so we resolve the parent.
	resolved := abs
	if _, statErr := os.Stat(abs); statErr == nil {
		if r, err := filepath.EvalSymlinks(abs); err == nil {
			resolved = r
		}
	} else {
		// Target doesn't exist — resolve the parent directory instead.
		parent := filepath.Dir(abs)
		if r, err := filepath.EvalSymlinks(parent); err == nil {
			resolved = filepath.Join(r, filepath.Base(abs))
		}
	}

	// Enforce allowed root boundary when specified.
	if allowedRoot != "" {
		rootAbs, err := filepath.Abs(allowedRoot)
		if err != nil {
			return "", errors.WrapValidationError(fmt.Sprintf(
				"cannot resolve allowed root path %q: %v", allowedRoot, err,
			))
		}
		// Ensure the resolved path starts with the root (add separator to
		// prevent "/allowed/subpath" matching "/allowedother").
		rootWithSep := rootAbs
		if !strings.HasSuffix(rootWithSep, string(filepath.Separator)) {
			rootWithSep += string(filepath.Separator)
		}
		if resolved != rootAbs && !strings.HasPrefix(resolved, rootWithSep) {
			return "", errors.WrapValidationError(fmt.Sprintf(
				"--%s: path %q resolves outside the allowed root %q — "+
					"remove any path traversal sequences (../) from the value",
				flag, rawPath, allowedRoot,
			))
		}
	}

	return resolved, nil
}

// ValidateInputPath validates that a user-supplied input path exists, is
// readable, and does not contain unsafe sequences. It normalizes the path
// before checking existence.
//
// Returns the normalized absolute path and nil on success, or a descriptive
// error with the flag name included.
func ValidateInputPath(flag, rawPath string) (string, error) {
	if rawPath == "" {
		return "", nil
	}

	normalized, err := NormalizePath(flag, rawPath, "")
	if err != nil {
		return "", err
	}

	info, err := os.Stat(normalized)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.WrapValidationError(fmt.Sprintf(
				"--%s: file not found: %q\n"+
					"Check that the path is correct and the file exists",
				flag, rawPath,
			))
		}
		return "", errors.WrapValidationError(fmt.Sprintf(
			"--%s: cannot access %q: %v", flag, rawPath, err,
		))
	}
	if info.IsDir() {
		return "", errors.WrapValidationError(fmt.Sprintf(
			"--%s: %q is a directory, not a file; provide the path to the file directly",
			flag, rawPath,
		))
	}

	return normalized, nil
}

// ValidateOutputPath validates that a user-supplied output path is safe to
// write to. It normalizes the path and checks that the parent directory
// exists (or can be created). It does not require the file itself to exist.
//
// Returns the normalized absolute path and nil on success.
func ValidateOutputPath(flag, rawPath string) (string, error) {
	if rawPath == "" {
		return "", nil
	}

	normalized, err := NormalizePath(flag, rawPath, "")
	if err != nil {
		return "", err
	}

	// The parent directory must be a directory (not a file).
	parentDir := filepath.Dir(normalized)
	if info, statErr := os.Stat(parentDir); statErr == nil {
		if !info.IsDir() {
			return "", errors.WrapValidationError(fmt.Sprintf(
				"--%s: parent path %q exists but is not a directory",
				flag, parentDir,
			))
		}
	}
	// Parent dir may not exist yet — that's fine; we create it at write time.

	// Reject paths that already point to a directory — the caller wants to
	// write a file, not into a directory.
	if info, statErr := os.Stat(normalized); statErr == nil && info.IsDir() {
		return "", errors.WrapValidationError(fmt.Sprintf(
			"--%s: %q is a directory; provide a full file path (e.g. %q)",
			flag, rawPath, filepath.Join(rawPath, "output.json"),
		))
	}

	return normalized, nil
}

// ValidatePathTraversal checks a path for directory traversal sequences that
// could escape an intended root. Returns an error when traversal is detected.
// This is a lightweight check for contexts where full normalization is not
// appropriate (e.g. checking embedded resource paths from external input).
func ValidatePathTraversal(flag, rawPath string) error {
	if rawPath == "" {
		return nil
	}

	cleaned := filepath.Clean(rawPath)

	// After cleaning, a traversal attempt will produce a path that starts
	// with ".." (relative) or is a different absolute path than expected.
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return errors.WrapValidationError(fmt.Sprintf(
			"--%s: path %q contains directory traversal sequences (../) which are not allowed",
			flag, rawPath,
		))
	}

	// Reject null bytes.
	if strings.ContainsRune(rawPath, 0) {
		return errors.WrapValidationError(fmt.Sprintf(
			"--%s: path contains null bytes: %q", flag, rawPath,
		))
	}

	return nil
}

// ValidateDebugOutputPaths validates all output-path flags used by the debug
// command before any simulation or network fetch begins. It checks:
//   - --save-snapshots
//   - --export-svg
//   - --trace-output
//
// Each flag is validated as an output path (parent dir exists; not a dir itself).
// Returns the first error encountered with the flag name and the bad value.
func ValidateDebugOutputPaths(saveSnapshots, exportSVG, traceOutput string) error {
	type check struct {
		flag string
		path string
	}
	for _, c := range []check{
		{"save-snapshots", saveSnapshots},
		{"export-svg", exportSVG},
		{"trace-output", traceOutput},
	} {
		if _, err := ValidateOutputPath(c.flag, c.path); err != nil {
			return err
		}
	}
	return nil
}

// ValidateDebugInputPaths validates all input-path flags used by the debug
// command before any simulation or network fetch begins. It checks:
//   - --snapshot
//   - --wasm
//   - --xdr-file
//   - --json-file
//   - --result-meta-file
//   - --load-snapshots
//   - --source-alias
//   - --contract-source (directory — checked separately)
//
// Returns the first error encountered.
func ValidateDebugInputPaths(snapshot, wasm, xdrFile, jsonFile, resultMetaFile, loadSnapshots, sourceAlias string) error {
	type check struct {
		flag string
		path string
	}
	for _, c := range []check{
		{"snapshot", snapshot},
		{"wasm", wasm},
		{"xdr-file", xdrFile},
		{"json-file", jsonFile},
		{"result-meta-file", resultMetaFile},
		{"load-snapshots", loadSnapshots},
		{"source-alias", sourceAlias},
	} {
		if _, err := ValidateInputPath(c.flag, c.path); err != nil {
			return err
		}
	}
	return nil
}

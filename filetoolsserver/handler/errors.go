// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import "github.com/dimitar-grigorov/mcp-file-tools/internal/errkind"

// Sentinel errors for handler operations.
// Match with errors.Is(); the attached Kind is what mapOperationError classifies on.

// Input validation errors
var (
	// ErrPathRequired is returned when a required path parameter is empty.
	ErrPathRequired = errkind.New(errkind.InvalidPath, "path is required and must be a non-empty string")

	// ErrPatternRequired is returned when a required pattern parameter is empty.
	ErrPatternRequired = errkind.New(errkind.InvalidInput, "pattern is required and must be a non-empty string")

	// ErrPathMustBeDirectory is returned when a directory is expected but a file was provided.
	ErrPathMustBeDirectory = errkind.New(errkind.InvalidPath, "path must be a directory")
)

// Encoding errors
var (
	// ErrEncodingUnsupported is returned when an unsupported encoding is specified.
	// Wrap this error to include the encoding name: fmt.Errorf("%w: %s", ErrEncodingUnsupported, name)
	ErrEncodingUnsupported = errkind.New(errkind.Encoding, "unsupported encoding")

	// ErrBOMEncodingConflict is returned when a file's BOM contradicts the requested encoding.
	ErrBOMEncodingConflict = errkind.New(errkind.Encoding, "BOM conflicts with requested encoding")

	// ErrBOMPolicyInvalid is returned for an unknown bom policy, or one the encoding cannot satisfy.
	ErrBOMPolicyInvalid = errkind.New(errkind.InvalidInput, "invalid bom policy")

	// ErrLineEndingPolicyInvalid is returned for an unknown lineEndings policy.
	ErrLineEndingPolicyInvalid = errkind.New(errkind.InvalidInput, "invalid lineEndings policy")

	// ErrLineEndingActionInvalid is returned for an unknown manage_line_endings action.
	ErrLineEndingActionInvalid = errkind.New(errkind.InvalidInput, `invalid action — valid: "detect", "convert"`)

	// ErrLineEndingStyleRequired is returned when action="convert" arrives without a style.
	ErrLineEndingStyleRequired = errkind.New(errkind.InvalidInput, `style is required for action="convert" — "lf" or "crlf"`)
)

// Edit operation errors
var (
	// ErrEditNoMatch is returned when old_text cannot be found in the file.
	// Wrap this error to include context: fmt.Errorf("%w:\n%s", ErrEditNoMatch, oldText)
	ErrEditNoMatch = errkind.New(errkind.InvalidInput, "could not find exact match for edit")

	// ErrOldTextEmpty is returned when an edit operation has an empty old_text field.
	ErrOldTextEmpty = errkind.New(errkind.InvalidInput, "edit old_text cannot be empty")
)

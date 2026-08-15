// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import "github.com/dimitar-grigorov/mcp-file-tools/v4/internal/errkind"

// Sentinel errors for handler operations.
// Match with errors.Is(); the attached Kind is what mapOperationError classifies on.

// Input validation errors
var (
	ErrPathRequired        = errkind.New(errkind.InvalidPath, "path is required and must be a non-empty string")
	ErrPatternRequired     = errkind.New(errkind.InvalidInput, "pattern is required and must be a non-empty string")
	ErrPathMustBeDirectory = errkind.New(errkind.InvalidPath, "path must be a directory")
)

// Encoding errors
var (
	// Wrap to include the encoding name: fmt.Errorf("%w: %s", ErrEncodingUnsupported, name)
	ErrEncodingUnsupported = errkind.New(errkind.Encoding, "unsupported encoding")

	ErrBOMEncodingConflict     = errkind.New(errkind.Encoding, "BOM conflicts with requested encoding")
	ErrBOMPolicyInvalid        = errkind.New(errkind.InvalidInput, "invalid bom policy")
	ErrLineEndingPolicyInvalid = errkind.New(errkind.InvalidInput, "invalid lineEndings policy")
	ErrLineEndingActionInvalid = errkind.New(errkind.InvalidInput, `invalid action — valid: "detect", "convert"`)
	ErrLineEndingStyleRequired = errkind.New(errkind.InvalidInput, `style is required for action="convert" — "lf" or "crlf"`)
)

// Edit operation errors
var (
	// Wrap to include context: fmt.Errorf("%w:\n%s", ErrEditNoMatch, oldText)
	ErrEditNoMatch = errkind.New(errkind.InvalidInput, "could not find exact match for edit")

	ErrOldTextEmpty  = errkind.New(errkind.InvalidInput, "edit old_text cannot be empty")
	ErrEditAmbiguous = errkind.New(errkind.InvalidInput, "oldText matches more than one place")
)

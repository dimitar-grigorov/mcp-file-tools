package handler

import "errors"

// Sentinel errors for handler operations.
// Use errors.Is() to check for specific error types.

// Input validation errors
var (
	// ErrPathRequired is returned when a required path parameter is empty.
	ErrPathRequired = errors.New("path is required and must be a non-empty string")

	// ErrPatternRequired is returned when a required pattern parameter is empty.
	ErrPatternRequired = errors.New("pattern is required and must be a non-empty string")

	// ErrEditsRequired is returned when the edits array is missing or empty.
	ErrEditsRequired = errors.New("edits array is required and must not be empty")

	// ErrPathMustBeDirectory is returned when a directory is expected but a file was provided.
	ErrPathMustBeDirectory = errors.New("path must be a directory")
)

// Encoding errors
var (
	// ErrEncodingUnsupported is returned when an unsupported encoding is specified.
	// Wrap this error to include the encoding name: fmt.Errorf("%w: %s", ErrEncodingUnsupported, name)
	ErrEncodingUnsupported = errors.New("unsupported encoding")
)

// Edit operation errors
var (
	// ErrEditNoMatch is returned when old_text cannot be found in the file.
	// Wrap this error to include context: fmt.Errorf("%w:\n%s", ErrEditNoMatch, oldText)
	ErrEditNoMatch = errors.New("could not find exact match for edit")

	// ErrOldTextEmpty is returned when an edit operation has an empty old_text field.
	ErrOldTextEmpty = errors.New("edit old_text cannot be empty")
)

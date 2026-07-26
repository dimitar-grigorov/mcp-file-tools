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

	// ErrEditsRequired is returned when the edits array is missing or empty.
	ErrEditsRequired = errkind.New(errkind.InvalidInput, "edits array is required and must not be empty")

	// ErrPathMustBeDirectory is returned when a directory is expected but a file was provided.
	ErrPathMustBeDirectory = errkind.New(errkind.InvalidPath, "path must be a directory")
)

// Encoding errors
var (
	// ErrEncodingUnsupported is returned when an unsupported encoding is specified.
	// Wrap this error to include the encoding name: fmt.Errorf("%w: %s", ErrEncodingUnsupported, name)
	ErrEncodingUnsupported = errkind.New(errkind.Encoding, "unsupported encoding")
)

// Edit operation errors
var (
	// ErrEditNoMatch is returned when old_text cannot be found in the file.
	// Wrap this error to include context: fmt.Errorf("%w:\n%s", ErrEditNoMatch, oldText)
	ErrEditNoMatch = errkind.New(errkind.InvalidInput, "could not find exact match for edit")

	// ErrOldTextEmpty is returned when an edit operation has an empty old_text field.
	ErrOldTextEmpty = errkind.New(errkind.InvalidInput, "edit old_text cannot be empty")
)

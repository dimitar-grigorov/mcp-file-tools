// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import "github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"

// ReadTextFileInput for reading files with encoding support; Offset/Limit are 1-indexed line numbers for a partial read.
type ReadTextFileInput struct {
	Path          string `json:"path"`
	Encoding      string `json:"encoding,omitempty"`
	Offset        *int   `json:"offset,omitempty"`
	Limit         *int   `json:"limit,omitempty"`
	MaxCharacters *int   `json:"maxCharacters,omitempty"`
	LineNumbers   bool   `json:"lineNumbers,omitempty"` // prefix lines with "N<tab>", absolute numbering
}

type ReadTextFileOutput struct {
	Content            string `json:"content"`
	TotalLines         int    `json:"totalLines"`
	FileSizeBytes      int64  `json:"fileSizeBytes"`
	StartLine          int    `json:"startLine,omitempty"`
	EndLine            int    `json:"endLine,omitempty"`
	Truncated          bool   `json:"truncated,omitempty"`
	DetectedEncoding   string `json:"detectedEncoding,omitempty"`
	EncodingConfidence int    `json:"encodingConfidence,omitempty"`
	Hint               string `json:"hint,omitempty"`
}

// WriteFileInput - encoding defaults to the existing file's encoding, else utf-8.
// BOM: "auto" (default), "always", "never", "preserve"; LineEndings: "preserve" (default), "crlf", "lf", "asis".
type WriteFileInput struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding,omitempty"`
	BOM         string `json:"bom,omitempty"`
	LineEndings string `json:"lineEndings,omitempty"`
}

type WriteFileOutput struct {
	Message     string `json:"message"`
	HasBOM      bool   `json:"hasBom"`
	BOMType     string `json:"bomType,omitempty"`
	LineEndings string `json:"lineEndings,omitempty"` // set when content was normalised
}

// ListDirectoryInput - SortBy is "name" (default), "mtime" or "size".
type ListDirectoryInput struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern,omitempty"` // glob pattern, e.g. *.pas
	SortBy  string `json:"sortBy,omitempty"`
	Reverse bool   `json:"reverse,omitempty"`
}

type ListDirectoryOutput struct {
	Files []string `json:"files"`
}

type ListEncodingsInput struct{}

type ListEncodingsOutput struct {
	Encodings []encoding.EncodingListItem `json:"encodings"`
}

// DetectEncodingInput supports three modes: "sample" (default), "chunked", "full"
type DetectEncodingInput struct {
	Path string `json:"path"`
	Mode string `json:"mode,omitempty"`
}

type DetectEncodingOutput struct {
	Encoding   string              `json:"encoding"`
	Confidence int                 `json:"confidence"`
	HasBOM     bool                `json:"has_bom"`
	Candidates []EncodingCandidate `json:"candidates,omitempty"` // only when the verdict is in doubt
}

// EncodingCandidate is one ranked alternative for a file detection could not settle.
type EncodingCandidate struct {
	Encoding   string `json:"encoding"`
	Confidence int    `json:"confidence"`
	Supported  bool   `json:"supported"` // false: cannot be passed as an encoding parameter
}

type ListAllowedDirectoriesInput struct{}

type ListAllowedDirectoriesOutput struct {
	Directories []string `json:"directories"`
	Message     string   `json:"message,omitempty"`
}

type GetFileInfoInput struct {
	Path string `json:"path"`
}

type GetFileInfoOutput struct {
	Size        int64  `json:"size"`
	Created     string `json:"created"`
	Modified    string `json:"modified"`
	Accessed    string `json:"accessed"`
	IsDirectory bool   `json:"isDirectory"`
	IsFile      bool   `json:"isFile"`
	Permissions string `json:"permissions"`
}

type CreateDirectoryInput struct {
	Path string `json:"path"`
}

type CreateDirectoryOutput struct {
	Message string `json:"message"`
}

type MoveFileInput struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type MoveFileOutput struct {
	Message string `json:"message"`
}

// SearchFilesInput - pattern supports *.ext and **/*.ext syntax; SortBy is "name" (default), "mtime" or "size".
type SearchFilesInput struct {
	Path             string   `json:"path"`
	Pattern          string   `json:"pattern"`
	ExcludePatterns  []string `json:"excludePatterns,omitempty"`
	RespectGitignore *bool    `json:"respectGitignore,omitempty"` // default true
	MaxResults       int      `json:"maxResults,omitempty"`
	SortBy           string   `json:"sortBy,omitempty"`
	Reverse          bool     `json:"reverse,omitempty"`
}

type SearchFilesOutput struct {
	Files     []string `json:"files"`
	Truncated bool     `json:"truncated,omitempty"`
}

type EditOperation struct {
	OldText    string   `json:"oldText"`
	NewText    string   `json:"newText"`
	Similarity *float64 `json:"similarity,omitempty"`
	ReplaceAll bool     `json:"replaceAll,omitempty"` // required when oldText matches more than once
}

// EditFileInput applies text replacements with whitespace-flexible matching; DryRun previews without writing.
type EditFileInput struct {
	Path          string          `json:"path"`
	Edits         []EditOperation `json:"edits,omitempty"`
	Patch         string          `json:"patch,omitempty"`
	DryRun        bool            `json:"dryRun,omitempty"`
	Encoding      string          `json:"encoding,omitempty"`
	ForceWritable *bool           `json:"forceWritable,omitempty"` // default: false - fail on read-only files
}

type EditFileOutput struct {
	Diff            string `json:"diff"`
	ReadOnlyCleared bool   `json:"readOnlyCleared,omitempty"` // true if read-only flag was cleared
	Replacements    int    `json:"replacements,omitempty"`    // set when replaceAll changed more than one place
}

type ReadMultipleFilesInput struct {
	Paths    []string `json:"paths"`
	Encoding string   `json:"encoding,omitempty"`
}

// Error codes for programmatic error handling
const (
	ErrCodeNone            = ""                 // No error
	ErrCodeNotFound        = "NOT_FOUND"        // File does not exist
	ErrCodePermission      = "PERMISSION"       // Permission denied
	ErrCodeAccessDenied    = "ACCESS_DENIED"    // Path outside allowed directories
	ErrCodeEncoding        = "ENCODING"         // Encoding detection/conversion failed
	ErrCodeIO              = "IO_ERROR"         // General I/O error
	ErrCodeInvalidPath     = "INVALID_PATH"     // Path validation failed
	ErrCodeSymlinkEscape   = "SYMLINK_ESCAPE"   // Symlink target outside allowed dirs
	ErrCodeOperationFailed = "OPERATION_FAILED" // Generic operation failure
)

type FileReadResult struct {
	Path               string `json:"path"`
	Content            string `json:"content,omitempty"`
	Error              string `json:"error,omitempty"`
	ErrorCode          string `json:"errorCode,omitempty"` // Machine-readable error code
	DetectedEncoding   string `json:"detectedEncoding,omitempty"`
	EncodingConfidence int    `json:"encodingConfidence,omitempty"`
	Hint               string `json:"hint,omitempty"`
}

type ReadMultipleFilesOutput struct {
	Results      []FileReadResult `json:"results"`
	SuccessCount int              `json:"successCount"`
	ErrorCount   int              `json:"errorCount"`
	Errors       []string         `json:"errors,omitempty"` // Summary of all errors
}

// TreeInput for compact tree view. MaxFiles defaults to 1000.
type TreeInput struct {
	Path             string   `json:"path"`
	MaxDepth         int      `json:"maxDepth,omitempty"`
	MaxFiles         int      `json:"maxFiles,omitempty"`
	DirsOnly         bool     `json:"dirsOnly,omitempty"`
	Exclude          []string `json:"exclude,omitempty"`
	ShowEncoding     bool     `json:"showEncoding,omitempty"`
	RespectGitignore *bool    `json:"respectGitignore,omitempty"` // default true
}

type TreeOutput struct {
	Tree      string `json:"tree"`
	FileCount int    `json:"fileCount"`
	DirCount  int    `json:"dirCount"`
	Truncated bool   `json:"truncated,omitempty"`
}

type DeleteFileInput struct {
	Path string `json:"path"`
}

type DeleteFileOutput struct {
	Message string `json:"message"`
}

type CopyFileInput struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type CopyFileOutput struct {
	Message string `json:"message"`
}

// ConvertEncodingInput converts between encodings; From is auto-detected if empty, BOM is "auto" (default), "always", "never" or "preserve".
// Pass either Path (one file) or Paths (a batch), never both.
type ConvertEncodingInput struct {
	Path   string   `json:"path,omitempty"`
	Paths  []string `json:"paths,omitempty"`
	From   string   `json:"from,omitempty"`
	To     string   `json:"to"`
	Backup bool     `json:"backup,omitempty"`
	BOM    string   `json:"bom,omitempty"`
	DryRun bool     `json:"dryRun,omitempty"`
	// AllowLowConfidence converts on a detection below the trust threshold.
	AllowLowConfidence bool `json:"allowLowConfidence,omitempty"`
}

// ConvertFileResult is one file's outcome within a batch conversion.
type ConvertFileResult struct {
	Path             string                     `json:"path"`
	SourceEncoding   string                     `json:"sourceEncoding,omitempty"`
	Changed          bool                       `json:"changed"`
	Message          string                     `json:"message,omitempty"`
	Error            string                     `json:"error,omitempty"`
	BackupPath       string                     `json:"backupPath,omitempty"`
	HasBOM           bool                       `json:"hasBom,omitempty"`
	BOMType          string                     `json:"bomType,omitempty"`
	Unsupported      []encoding.UnsupportedRune `json:"unsupported,omitempty"`
	UnsupportedCount int                        `json:"unsupportedCount,omitempty"`
}

// ConvertEncodingOutput keeps the flat fields for a single Path; Results and the counts are filled only for a Paths batch.
type ConvertEncodingOutput struct {
	Message        string `json:"message"`
	SourceEncoding string `json:"sourceEncoding,omitempty"`
	TargetEncoding string `json:"targetEncoding"`
	BackupPath     string `json:"backupPath,omitempty"`
	HasBOM         bool   `json:"hasBom"`
	BOMType        string `json:"bomType,omitempty"`
	Changed        bool   `json:"changed"`
	DryRun         bool   `json:"dryRun,omitempty"`

	Results      []ConvertFileResult `json:"results,omitempty"`
	SuccessCount int                 `json:"successCount,omitempty"`
	ErrorCount   int                 `json:"errorCount,omitempty"`
	Errors       []string            `json:"errors,omitempty"`
}

// GrepInput for searching file contents with regex; OutputMode is "content" (default), "files_with_matches" or "count".
type GrepInput struct {
	Pattern          string   `json:"pattern"`
	Patterns         []string `json:"patterns,omitempty"` // match any of these, in one pass
	Paths            []string `json:"paths"`
	CaseSensitive    *bool    `json:"caseSensitive,omitempty"` // defaults to true
	ContextBefore    int      `json:"contextBefore,omitempty"`
	ContextAfter     int      `json:"contextAfter,omitempty"`
	MaxMatches       int      `json:"maxMatches,omitempty"` // defaults to 1000
	Include          string   `json:"include,omitempty"`
	Exclude          string   `json:"exclude,omitempty"`
	Includes         []string `json:"includes,omitempty"`
	Excludes         []string `json:"excludes,omitempty"`
	Encoding         string   `json:"encoding,omitempty"`
	OutputMode       string   `json:"outputMode,omitempty"`
	MatchesOnly      bool     `json:"matchesOnly,omitempty"`      // text is the matched substring, not the line
	Offset           int      `json:"offset,omitempty"`           // skip the first N results, for paging
	RespectGitignore *bool    `json:"respectGitignore,omitempty"` // default true
}

type GrepMatch struct {
	Path     string   `json:"path"`
	Line     int      `json:"line"`
	Column   int      `json:"column"`
	Text     string   `json:"text"`
	Before   []string `json:"before,omitempty"`
	After    []string `json:"after,omitempty"`
	Encoding string   `json:"encoding,omitempty"`
}

// GrepFileCount is one file's matching-line count in "count" mode.
type GrepFileCount struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

// GrepOutput fills one field per output mode: Matches, Files or Counts.
type GrepOutput struct {
	Matches       []GrepMatch     `json:"matches"`
	Files         []string        `json:"files,omitempty"`
	Counts        []GrepFileCount `json:"counts,omitempty"`
	TotalMatches  int             `json:"totalMatches"`
	FilesSearched int             `json:"filesSearched"`
	FilesMatched  int             `json:"filesMatched"`
	Truncated     bool            `json:"truncated,omitempty"`
	NextOffset    int             `json:"nextOffset,omitempty"` // pass as offset to page on
	Hint          string          `json:"hint,omitempty"`
}

type DetectLineEndingsInput struct {
	Path     string `json:"path"`
	Encoding string `json:"encoding,omitempty"`
}

// ChangeLineEndingsInput converts line endings in a file; Style must be "lf" or "crlf".
type ChangeLineEndingsInput struct {
	Path     string `json:"path"`
	Style    string `json:"style"`
	Encoding string `json:"encoding,omitempty"`
}

type ChangeLineEndingsOutput struct {
	Message       string `json:"message"`
	OriginalStyle string `json:"originalStyle"`
	NewStyle      string `json:"newStyle"`
	LinesChanged  int    `json:"linesChanged"`
}

// ManageBomInput manages a Unicode BOM; Action is "detect", "strip" or "add".
// Encoding is required for "add": utf-8, utf-16-le, utf-16-be, utf-32-le, utf-32-be.
type ManageBomInput struct {
	Path     string `json:"path"`
	Action   string `json:"action"`
	Encoding string `json:"encoding,omitempty"`
}

type ManageBomOutput struct {
	Message  string `json:"message"`
	HasBOM   bool   `json:"hasBom"`
	BOMType  string `json:"bomType,omitempty"`  // e.g. "utf-8", "utf-16-le"
	BOMBytes int    `json:"bomBytes,omitempty"` // size of BOM in bytes (2, 3, or 4)
	Changed  bool   `json:"changed"`
}

// ManageLineEndingsInput manages line endings; Action is "detect" (report style) or "convert" (rewrite to Style, "lf" or "crlf").
type ManageLineEndingsInput struct {
	Path     string `json:"path"`
	Action   string `json:"action"`
	Style    string `json:"style,omitempty"`
	Encoding string `json:"encoding,omitempty"`
}

type ManageLineEndingsOutput struct {
	Style             string `json:"style"` // detect: dominant style; convert: the new style
	TotalLines        int    `json:"totalLines,omitempty"`
	InconsistentLines []int  `json:"inconsistentLines,omitempty"`
	OriginalStyle     string `json:"originalStyle,omitempty"`
	LinesChanged      int    `json:"linesChanged,omitempty"`
	Message           string `json:"message,omitempty"`
	Changed           bool   `json:"changed,omitempty"`
}

// DetectLineEndingsOutput - Style is "crlf", "lf", "mixed", or "none"
type DetectLineEndingsOutput struct {
	Style             string `json:"style"`
	TotalLines        int    `json:"totalLines"`
	InconsistentLines []int  `json:"inconsistentLines"`
}

package handler

// ReadTextFileInput for reading files with encoding support.
// Offset/Limit are 1-indexed line numbers for partial reads.
type ReadTextFileInput struct {
	Path     string `json:"path"`
	Encoding string `json:"encoding,omitempty"`
	Offset   *int   `json:"offset,omitempty"`
	Limit    *int   `json:"limit,omitempty"`
}

type ReadTextFileOutput struct {
	Content            string `json:"content"`
	TotalLines         int    `json:"totalLines"`
	StartLine          int    `json:"startLine,omitempty"`
	EndLine            int    `json:"endLine,omitempty"`
	DetectedEncoding   string `json:"detectedEncoding,omitempty"`
	EncodingConfidence int    `json:"encodingConfidence,omitempty"`
}

// WriteFileInput - encoding defaults to cp1251 for legacy codebases
type WriteFileInput struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding,omitempty"`
}

type WriteFileOutput struct {
	Message string `json:"message"`
}

type ListDirectoryInput struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern,omitempty"` // glob pattern, e.g. *.pas
}

type ListDirectoryOutput struct {
	Files []string `json:"files"`
}

type ListEncodingsInput struct{}

type EncodingItem struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Aliases     []string `json:"aliases"`
	Description string   `json:"description"`
}

type ListEncodingsOutput struct {
	Encodings []EncodingItem `json:"encodings"`
}

// DetectEncodingInput supports three modes: "sample" (default), "chunked", "full"
type DetectEncodingInput struct {
	Path string `json:"path"`
	Mode string `json:"mode,omitempty"`
}

type DetectEncodingOutput struct {
	Encoding   string `json:"encoding"`
	Confidence int    `json:"confidence"`
	HasBOM     bool   `json:"has_bom"`
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

// DirectoryTreeInput - deprecated, use TreeInput instead
type DirectoryTreeInput struct {
	Path            string   `json:"path"`
	ExcludePatterns []string `json:"excludePatterns,omitempty"`
}

type DirectoryTreeOutput struct {
	Tree string `json:"tree"`
}

type TreeEntry struct {
	Name     string       `json:"name"`
	Type     string       `json:"type"`
	Children *[]TreeEntry `json:"children,omitempty"`
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

// SearchFilesInput - pattern supports *.ext and **/*.ext syntax
type SearchFilesInput struct {
	Path            string   `json:"path"`
	Pattern         string   `json:"pattern"`
	ExcludePatterns []string `json:"excludePatterns,omitempty"`
}

type SearchFilesOutput struct {
	Files []string `json:"files"`
}

type EditOperation struct {
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
}

// EditFileInput applies text replacements with whitespace-flexible matching.
// Set DryRun to preview changes without writing.
// Set ForceWritable to false to fail on read-only files instead of clearing the flag.
type EditFileInput struct {
	Path          string          `json:"path"`
	Edits         []EditOperation `json:"edits"`
	DryRun        bool            `json:"dryRun,omitempty"`
	Encoding      string          `json:"encoding,omitempty"`
	ForceWritable *bool           `json:"forceWritable,omitempty"` // default: true - clear read-only flag if set
}

type EditFileOutput struct {
	Diff            string `json:"diff"`
	ReadOnlyCleared bool   `json:"readOnlyCleared,omitempty"` // true if read-only flag was cleared
}

type ReadMultipleFilesInput struct {
	Paths    []string `json:"paths"`
	Encoding string   `json:"encoding,omitempty"`
}

type FileReadResult struct {
	Path               string `json:"path"`
	Content            string `json:"content,omitempty"`
	Error              string `json:"error,omitempty"`
	DetectedEncoding   string `json:"detectedEncoding,omitempty"`
	EncodingConfidence int    `json:"encodingConfidence,omitempty"`
}

type ReadMultipleFilesOutput struct {
	Results      []FileReadResult `json:"results"`
	SuccessCount int              `json:"successCount"`
	ErrorCount   int              `json:"errorCount"`
}

// TreeInput for compact tree view. MaxFiles defaults to 1000.
type TreeInput struct {
	Path     string   `json:"path"`
	MaxDepth int      `json:"maxDepth,omitempty"`
	MaxFiles int      `json:"maxFiles,omitempty"`
	DirsOnly bool     `json:"dirsOnly,omitempty"`
	Exclude  []string `json:"exclude,omitempty"`
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

// ConvertEncodingInput converts between encodings. From is auto-detected if empty.
type ConvertEncodingInput struct {
	Path   string `json:"path"`
	From   string `json:"from,omitempty"`
	To     string `json:"to"`
	Backup bool   `json:"backup,omitempty"`
}

type ConvertEncodingOutput struct {
	Message        string `json:"message"`
	SourceEncoding string `json:"sourceEncoding"`
	TargetEncoding string `json:"targetEncoding"`
	BackupPath     string `json:"backupPath,omitempty"`
}

// GrepInput for searching file contents with regex
type GrepInput struct {
	Pattern       string   `json:"pattern"`
	Paths         []string `json:"paths"`
	CaseSensitive *bool    `json:"caseSensitive,omitempty"` // defaults to true
	ContextBefore int      `json:"contextBefore,omitempty"`
	ContextAfter  int      `json:"contextAfter,omitempty"`
	MaxMatches    int      `json:"maxMatches,omitempty"` // defaults to 1000
	Include       string   `json:"include,omitempty"`
	Exclude       string   `json:"exclude,omitempty"`
	Encoding      string   `json:"encoding,omitempty"`
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

type GrepOutput struct {
	Matches       []GrepMatch `json:"matches"`
	TotalMatches  int         `json:"totalMatches"`
	FilesSearched int         `json:"filesSearched"`
	FilesMatched  int         `json:"filesMatched"`
	Truncated     bool        `json:"truncated,omitempty"`
}

type DetectLineEndingsInput struct {
	Path string `json:"path"`
}

// DetectLineEndingsOutput - Style is "crlf", "lf", "mixed", or "none"
type DetectLineEndingsOutput struct {
	Style             string `json:"style"`
	TotalLines        int    `json:"totalLines"`
	InconsistentLines []int  `json:"inconsistentLines"`
}


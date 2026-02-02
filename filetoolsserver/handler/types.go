package handler

// ReadTextFileInput defines input parameters for read_text_file tool.
// Path: Path to the file to read (required)
// Encoding: File encoding - auto-detected if not specified (optional)
// Offset: Start reading from this line number, 1-indexed (optional)
// Limit: Maximum number of lines to read (optional)
// Head: Read only the first N lines - deprecated, use Limit (optional)
// Tail: Read only the last N lines (optional)
type ReadTextFileInput struct {
	Path     string `json:"path"`
	Encoding string `json:"encoding,omitempty"`
	Offset   *int   `json:"offset,omitempty"`
	Limit    *int   `json:"limit,omitempty"`
	Head     *int   `json:"head,omitempty"`
	Tail     *int   `json:"tail,omitempty"`
}

// ReadTextFileOutput defines output for read_text_file tool
type ReadTextFileOutput struct {
	Content            string `json:"content"`
	TotalLines         int    `json:"totalLines"`
	StartLine          int    `json:"startLine,omitempty"`
	EndLine            int    `json:"endLine,omitempty"`
	DetectedEncoding   string `json:"detectedEncoding,omitempty"`
	EncodingConfidence int    `json:"encodingConfidence,omitempty"`
}

// WriteFileInput defines input parameters for write_file tool.
// Path: Absolute path to the file to write (required)
// Content: Content to write to the file (required)
// Encoding: Target encoding - utf-8 (no conversion) or cp1251/windows-1251 (converts from UTF-8). Default: cp1251
type WriteFileInput struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding,omitempty"`
}

// WriteFileOutput defines output for write_file tool
type WriteFileOutput struct {
	Message string `json:"message"`
}

// ListDirectoryInput defines input parameters for list_directory tool.
// Path: Absolute path to the directory to list (required)
// Pattern: Optional glob pattern to filter files (e.g. *.pas *.dfm). Default: *
type ListDirectoryInput struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern,omitempty"`
}

// ListDirectoryOutput defines output for list_directory tool
type ListDirectoryOutput struct {
	Files []string `json:"files"`
}

// ListEncodingsInput defines input parameters for list_encodings tool
type ListEncodingsInput struct{}

// EncodingItem represents a single encoding in the list
type EncodingItem struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Aliases     []string `json:"aliases"`
	Description string   `json:"description"`
}

// ListEncodingsOutput defines output for list_encodings tool
type ListEncodingsOutput struct {
	Encodings []EncodingItem `json:"encodings"`
}

// DetectEncodingInput defines input parameters for detect_encoding tool.
// Path: Path to the file to detect encoding for (required)
type DetectEncodingInput struct {
	Path string `json:"path"`
}

// DetectEncodingOutput defines output for detect_encoding tool
type DetectEncodingOutput struct {
	Encoding   string `json:"encoding"`
	Confidence int    `json:"confidence"`
	HasBOM     bool   `json:"has_bom"`
}

// ListAllowedDirectoriesInput defines input parameters for list_allowed_directories tool
type ListAllowedDirectoriesInput struct{}

// ListAllowedDirectoriesOutput defines output for list_allowed_directories tool
type ListAllowedDirectoriesOutput struct {
	Directories []string `json:"directories"`
}

// GetFileInfoInput defines input parameters for get_file_info tool.
// Path: Path to the file or directory to get info for (required)
type GetFileInfoInput struct {
	Path string `json:"path"`
}

// GetFileInfoOutput defines output for get_file_info tool
type GetFileInfoOutput struct {
	Size        int64  `json:"size"`
	Created     string `json:"created"`
	Modified    string `json:"modified"`
	Accessed    string `json:"accessed"`
	IsDirectory bool   `json:"isDirectory"`
	IsFile      bool   `json:"isFile"`
	Permissions string `json:"permissions"`
}

// DirectoryTreeInput defines input parameters for directory_tree tool.
// Path: Root directory to generate tree from (required)
// ExcludePatterns: Glob patterns to exclude from the tree (optional)
type DirectoryTreeInput struct {
	Path            string   `json:"path"`
	ExcludePatterns []string `json:"excludePatterns,omitempty"`
}

// DirectoryTreeOutput defines output for directory_tree tool
type DirectoryTreeOutput struct {
	Tree string `json:"tree"`
}

// TreeEntry represents a single entry in the directory tree
type TreeEntry struct {
	Name     string       `json:"name"`
	Type     string       `json:"type"`
	Children *[]TreeEntry `json:"children,omitempty"`
}

// CreateDirectoryInput defines input parameters for create_directory tool.
// Path: Absolute path to the directory to create (required)
type CreateDirectoryInput struct {
	Path string `json:"path"`
}

// CreateDirectoryOutput defines output for create_directory tool
type CreateDirectoryOutput struct {
	Message string `json:"message"`
}

// MoveFileInput defines input parameters for move_file tool.
// Source: Absolute path to the file or directory to move (required)
// Destination: Absolute path to the destination location (required)
type MoveFileInput struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// MoveFileOutput defines output for move_file tool
type MoveFileOutput struct {
	Message string `json:"message"`
}

// SearchFilesInput defines input parameters for search_files tool.
// Path: Root directory to search from (required)
// Pattern: Glob pattern to match files (required). Use *.ext for current dir, **/*.ext for recursive
// ExcludePatterns: Glob patterns to exclude from results (optional)
type SearchFilesInput struct {
	Path            string   `json:"path"`
	Pattern         string   `json:"pattern"`
	ExcludePatterns []string `json:"excludePatterns,omitempty"`
}

// SearchFilesOutput defines output for search_files tool
type SearchFilesOutput struct {
	Files []string `json:"files"`
}

// EditOperation defines a single edit operation for edit_file tool.
// OldText: Text to search for - must match exactly (or with whitespace flexibility)
// NewText: Text to replace with
type EditOperation struct {
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
}

// EditFileInput defines input parameters for edit_file tool.
// Path: Absolute path to the file to edit (required)
// Edits: Array of edit operations to apply sequentially (required)
// DryRun: If true, return diff without writing changes (default: false)
// Encoding: File encoding (optional, auto-detected if not specified)
type EditFileInput struct {
	Path     string          `json:"path"`
	Edits    []EditOperation `json:"edits"`
	DryRun   bool            `json:"dryRun,omitempty"`
	Encoding string          `json:"encoding,omitempty"`
}

// EditFileOutput defines output for edit_file tool
type EditFileOutput struct {
	Diff string `json:"diff"`
}

// ReadMultipleFilesInput defines input parameters for read_multiple_files tool.
// Paths: Array of file paths to read (required, min 1)
// Encoding: Encoding for all files - auto-detected per file if not specified (optional)
type ReadMultipleFilesInput struct {
	Paths    []string `json:"paths"`
	Encoding string   `json:"encoding,omitempty"`
}

// FileReadResult represents the result of reading a single file
type FileReadResult struct {
	Path               string `json:"path"`
	Content            string `json:"content,omitempty"`
	Error              string `json:"error,omitempty"`
	DetectedEncoding   string `json:"detectedEncoding,omitempty"`
	EncodingConfidence int    `json:"encodingConfidence,omitempty"`
}

// ReadMultipleFilesOutput defines output for read_multiple_files tool
type ReadMultipleFilesOutput struct {
	Results      []FileReadResult `json:"results"`
	SuccessCount int              `json:"successCount"`
	ErrorCount   int              `json:"errorCount"`
}

// TreeInput defines input parameters for tree tool.
// Path: Root directory to display (required)
// MaxDepth: Maximum recursion depth, 0 = unlimited (optional, default: 0)
// MaxFiles: Maximum entries to return, 0 = unlimited (optional, default: 1000)
// DirsOnly: Only show directories, not files (optional, default: false)
// Exclude: Patterns to exclude (optional)
type TreeInput struct {
	Path     string   `json:"path"`
	MaxDepth int      `json:"maxDepth,omitempty"`
	MaxFiles int      `json:"maxFiles,omitempty"`
	DirsOnly bool     `json:"dirsOnly,omitempty"`
	Exclude  []string `json:"exclude,omitempty"`
}

// TreeOutput defines output for tree tool
type TreeOutput struct {
	Tree      string `json:"tree"`
	FileCount int    `json:"fileCount"`
	DirCount  int    `json:"dirCount"`
	Truncated bool   `json:"truncated,omitempty"`
}


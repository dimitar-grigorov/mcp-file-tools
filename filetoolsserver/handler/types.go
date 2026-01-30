package handler

// ReadTextFileInput defines input parameters for read_text_file tool.
// Path: Path to the file to read (required)
// Encoding: File encoding - utf-8 (default) or cp1251/windows-1251 (converts to UTF-8). Omit for UTF-8.
// Head: Read only the first N lines (optional)
// Tail: Read only the last N lines (optional)
type ReadTextFileInput struct {
	Path     string `json:"path"`
	Encoding string `json:"encoding,omitempty"`
	Head     *int   `json:"head,omitempty"`
	Tail     *int   `json:"tail,omitempty"`
}

// ReadTextFileOutput defines output for read_text_file tool
type ReadTextFileOutput struct {
	Content            string `json:"content"`
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

// ListEncodingsOutput defines output for list_encodings tool
type ListEncodingsOutput struct {
	Encodings []string `json:"encodings"`
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


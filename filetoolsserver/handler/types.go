package handler

// ReadTextFileInput defines input parameters for read_text_file tool
type ReadTextFileInput struct {
	Path     string `json:"path" jsonschema:"required,description=Path to the file to read"`
	Encoding string `json:"encoding,omitempty" jsonschema:"description=File encoding: utf-8 (default) or cp1251/windows-1251 (converts to UTF-8). Omit for UTF-8."`
	Head     *int   `json:"head,omitempty" jsonschema:"description=Read only the first N lines"`
	Tail     *int   `json:"tail,omitempty" jsonschema:"description=Read only the last N lines"`
}

// ReadTextFileOutput defines output for read_text_file tool
type ReadTextFileOutput struct {
	Content string `json:"content"`
}

// WriteFileInput defines input parameters for write_file tool
type WriteFileInput struct {
	Path     string `json:"path" jsonschema:"required,description=Absolute path to the file to write"`
	Content  string `json:"content" jsonschema:"required,description=Content to write to the file"`
	Encoding string `json:"encoding,omitempty" jsonschema:"description=Target encoding: utf-8 (no conversion) or cp1251/windows-1251 (converts from UTF-8). Default: cp1251,default=cp1251"`
}

// WriteFileOutput defines output for write_file tool
type WriteFileOutput struct {
	Message string `json:"message"`
}

// ListDirectoryInput defines input parameters for list_directory tool
type ListDirectoryInput struct {
	Path    string `json:"path" jsonschema:"required,description=Absolute path to the directory to list"`
	Pattern string `json:"pattern,omitempty" jsonschema:"description=Optional glob pattern to filter files (e.g. *.pas *.dfm),default=*"`
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

// DetectEncodingInput defines input parameters for detect_encoding tool
type DetectEncodingInput struct {
	Path string `json:"path" jsonschema:"required,description=Path to the file to detect encoding for"`
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


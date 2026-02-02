# Tools Reference

## File Operations

### read_text_file

Read file contents with automatic encoding detection and optional partial reading. UTF-8 files pass through unchanged; other encodings convert to UTF-8.

**Parameters:**
- `path` (required): Path to the file
- `encoding` (optional): Encoding name (auto-detects if omitted)
- `offset` (optional): Start reading from this line number (1-indexed)
- `limit` (optional): Maximum number of lines to read
- `head` (optional): Read only the first N lines (deprecated, use `limit`)
- `tail` (optional): Read only the last N lines

**Example:**
```json
{
  "path": "/path/to/file.pas",
  "offset": 100,
  "limit": 50
}
```

**Response:**
```json
{
  "content": "line 100\nline 101\n...",
  "totalLines": 500,
  "startLine": 100,
  "endLine": 149,
  "detectedEncoding": "windows-1251",
  "encodingConfidence": 95
}
```

### write_file

Write content to file. UTF-8 writes as-is; other encodings convert from UTF-8.

**Parameters:**
- `path` (required): Path to the file
- `content` (required): Content to write
- `encoding` (optional): Target encoding (default: cp1251)

**Example:**
```json
{
  "path": "/path/to/file.pas",
  "content": "program Hello;\nbegin\n  writeln('Zdravei');\nend.",
  "encoding": "cp1251"
}
```

**Response:**
```json
{
  "message": "Successfully wrote 48 bytes to /path/to/file.pas"
}
```

### edit_file

Make line-based edits to a text file. Supports exact matching and whitespace-flexible matching. Returns a git-style unified diff showing changes.

**Parameters:**
- `path` (required): Path to the file to edit
- `edits` (required): Array of edit operations, each with `oldText` and `newText`
- `dryRun` (optional): If true, returns diff without writing changes (default: false)
- `encoding` (optional): File encoding (auto-detected if not specified)

**Features:**
- Exact text matching (first occurrence)
- Whitespace-flexible matching (ignores leading whitespace differences)
- Preserves original indentation
- CRLF line endings normalized to LF
- Atomic write (temp file + rename)

**Example:**
```json
{
  "path": "/path/to/file.go",
  "edits": [
    {
      "oldText": "func oldName()",
      "newText": "func newName()"
    }
  ],
  "dryRun": false
}
```

**Response:**
```json
{
  "diff": "--- /path/to/file.go\n+++ /path/to/file.go\n@@ -1,3 +1,3 @@\n-func oldName()\n+func newName()\n"
}
```

## Directory Operations

### list_directory

List files and directories with optional pattern filtering.

**Parameters:**
- `path` (required): Path to directory
- `pattern` (optional): Glob pattern like `*.pas` or `*.dfm` (default: `*`)

**Example:**
```json
{
  "path": "/path/to/project",
  "pattern": "*.pas"
}
```

**Response:**
```json
{
  "files": ["main.pas", "utils.pas", "forms.pas"]
}
```

### directory_tree

Get a recursive tree view of files and directories as JSON.

**Parameters:**
- `path` (required): Root directory
- `excludePatterns` (optional): Array of glob patterns to exclude

**Example:**
```json
{
  "path": "/path/to/project",
  "excludePatterns": ["node_modules", ".git"]
}
```

**Response:**
```json
{
  "tree": "{\"name\":\"project\",\"type\":\"directory\",\"children\":[...]}"
}
```

### get_file_info

Get detailed metadata about a file or directory.

**Parameters:**
- `path` (required): Path to file or directory

**Response:**
```json
{
  "size": 1234,
  "created": "2024-01-15T10:30:00Z",
  "modified": "2024-01-15T10:30:00Z",
  "accessed": "2024-01-15T10:30:00Z",
  "isDirectory": false,
  "isFile": true,
  "permissions": "rw-r--r--"
}
```

### create_directory

Create a directory recursively (like `mkdir -p`). Succeeds silently if directory already exists.

**Parameters:**
- `path` (required): Path to directory to create

**Example:**
```json
{
  "path": "/path/to/project/new/nested/dir"
}
```

**Response:**
```json
{
  "message": "Directory created: /path/to/project/new/nested/dir"
}
```

### move_file

Move or rename files and directories. Can move between directories and rename in a single operation. Fails if destination already exists.

**Parameters:**
- `source` (required): Path to file or directory to move
- `destination` (required): Destination path

**Example:**
```json
{
  "source": "/path/to/old_name.txt",
  "destination": "/path/to/new_location/new_name.txt"
}
```

**Response:**
```json
{
  "message": "Moved /path/to/old_name.txt to /path/to/new_location/new_name.txt"
}
```

### search_files

Recursively search for files and directories matching a glob pattern.

**Parameters:**
- `path` (required): Root directory to search from
- `pattern` (required): Glob pattern (`*.txt` for current dir, `**/*.txt` for recursive)
- `excludePatterns` (optional): Array of patterns to exclude

**Example:**
```json
{
  "path": "/path/to/project",
  "pattern": "**/*.go",
  "excludePatterns": ["vendor", "node_modules"]
}
```

**Response:**
```json
{
  "files": [
    "/path/to/project/main.go",
    "/path/to/project/src/utils.go"
  ]
}
```

## Encoding Tools

### detect_encoding

Detect the encoding of a file with confidence percentage.

**Parameters:**
- `path` (required): Path to the file

**Response:**
```json
{
  "encoding": "windows-1251",
  "confidence": 95,
  "has_bom": false
}
```

### list_encodings

Returns all supported encodings with metadata.

**Parameters:** None

**Response:**
```json
{
  "encodings": [
    {
      "name": "windows-1251",
      "displayName": "Windows-1251",
      "aliases": ["cp1251"],
      "description": "Windows Cyrillic"
    }
  ]
}
```

### list_allowed_directories

Returns directories the server is allowed to access.

**Parameters:** None

**Response:**
```json
{
  "directories": ["/home/user/projects", "/var/data"]
}
```

## Supported Encodings

| Name | Aliases | Description |
|------|---------|-------------|
| utf-8 | utf8, ascii | Unicode, no conversion |
| windows-1251 | cp1251 | Windows Cyrillic |
| koi8-r | koi8r | Russian Cyrillic (Unix/Linux) |
| koi8-u | koi8u | Ukrainian Cyrillic (Unix/Linux) |
| ibm866 | cp866, dos-866 | DOS Cyrillic |
| iso-8859-5 | iso88595, cyrillic | ISO Cyrillic |
| windows-1252 | cp1252 | Windows Western European |
| iso-8859-1 | iso88591, latin1 | Latin-1 Western European |
| iso-8859-15 | iso885915, latin9 | Latin-9 Western European (Euro) |
| windows-1250 | cp1250 | Windows Central European |
| iso-8859-2 | iso88592, latin2 | Latin-2 Central European |
| windows-1253 | cp1253 | Windows Greek |
| iso-8859-7 | iso88597, greek | ISO Greek |
| windows-1254 | cp1254 | Windows Turkish |
| iso-8859-9 | iso88599, latin5 | Latin-5 Turkish |
| windows-1255 | cp1255 | Windows Hebrew |
| windows-1256 | cp1256 | Windows Arabic |
| windows-1257 | cp1257 | Windows Baltic |
| windows-1258 | cp1258 | Windows Vietnamese |
| windows-874 | cp874, tis-620 | Windows Thai |

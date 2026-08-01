# Tools Reference

## Large results

The tools that can legitimately return a lot of text — `read_text_file`, `read_multiple_files`,
`grep_text_files` and `tree` — declare `anthropic/maxResultSizeChars` in their
`tools/list` `_meta`. Clients that honour it (Claude Code) raise their default result cap for
those tools instead of spilling the overflow to a file reference, which would otherwise hide
part of a tree or search result from the model. Every other tool keeps the client default.

## File Operations

### read_text_file

Read file contents with automatic encoding detection and optional partial reading. UTF-8 files pass through unchanged; other encodings convert to UTF-8.

**Parameters:**
- `path` (required): Path to the file
- `encoding` (optional): Encoding name (auto-detects if omitted)
- `offset` (optional): Start reading from this line number (1-indexed)
- `limit` (optional): Maximum number of lines to read
- `maxCharacters` (optional): Truncate content at this character count to prevent token overflow

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
  "fileSizeBytes": 15234,
  "startLine": 100,
  "endLine": 149,
  "truncated": false,
  "detectedEncoding": "windows-1251",
  "encodingConfidence": 95
}
```

`totalLines` counts newline-terminated lines: a trailing newline does not add an empty final line, and an empty file is `0`. (Matches `detect_line_endings`.)

### read_multiple_files

Read multiple files concurrently with encoding support. Individual file failures don't stop the operation.

**Parameters:**
- `paths` (required): Array of file paths to read
- `encoding` (optional): Encoding for all files (auto-detected per file if omitted)

**Example:**
```json
{
  "paths": ["/path/to/file1.pas", "/path/to/file2.pas"],
  "encoding": "cp1251"
}
```

**Response:**
```json
{
  "results": [
    {
      "path": "/path/to/file1.pas",
      "content": "program Hello;...",
      "detectedEncoding": "windows-1251",
      "encodingConfidence": 95
    },
    {
      "path": "/path/to/file2.pas",
      "content": "unit Utils;..."
    }
  ],
  "successCount": 2,
  "errorCount": 0
}
```

### write_file

Write content to file. UTF-8 writes as-is; other encodings convert from UTF-8.

**Parameters:**
- `path` (required): Path to the file
- `content` (required): Content to write
- `encoding` (optional): Target encoding. Defaults to the existing file's detected encoding; for a new file, to `MCP_DEFAULT_ENCODING` (`utf-8`)
- `bom` (optional): `auto` (default — BOM for UTF-16 targets, keeps a BOM the file already had), `always`, `never`, `preserve`
- `lineEndings` (optional): `preserve` (default), `crlf`, `lf`, `asis`

**Line endings:**

`preserve` converts the content to the existing file's dominant style before
writing. This matters because a model rewriting a CRLF file usually emits `\n`
for the lines it changes — written verbatim that leaves the file **mixed**.
A file that is already mixed is repaired to whichever style is commoner.

For a new file (or one with no line endings yet) the fallback is
`MCP_DEFAULT_LINE_ENDINGS` (`crlf`/`lf`); unset, the content is written unchanged.

Use `crlf`/`lf` to force a style, or `asis` for byte-for-byte writes.

**BOM modes:**

| Mode | Behaviour |
|---|---|
| `auto` (default) | BOM for `utf-16-*` targets; otherwise keeps one only if the file already had a BOM of the *same* encoding |
| `preserve` | Keeps the existing BOM even when the encoding changed |
| `never` | Writes no BOM (use to strip a UTF-8 BOM that breaks PHP/shell scripts) |
| `always` | Forces a BOM; fails on encodings that define none (e.g. `cp1251`) |

**Example:**
```json
{
  "path": "/path/to/file.pas",
  "content": "program Hello;\nbegin\n  writeln('Zdravei');\nend.",
  "encoding": "cp1251"
}
```

Strip a UTF-8 BOM:
```json
{
  "path": "D:\\www\\index.php",
  "content": "<?php ...",
  "bom": "never"
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
- `forceWritable` (optional): If true, clears read-only flag before editing (default: false — fails on read-only files)

**Features:**
- Exact text matching (first occurrence)
- Whitespace-flexible matching (ignores per-line leading *and* trailing whitespace; interior spacing must still match)
- Preserves original indentation
- CRLF line endings normalized to LF, so CRLF/LF differences never block a match
- Atomic write (temp file + rename)
- Fails on read-only files by default (set `forceWritable: true` only when user explicitly requests it)

Edits apply in order, and each one replaces only the **first** match remaining at that point.

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

Multiple edits in one call:
```json
{
  "path": "D:\\src\\unit1.pas",
  "edits": [
    { "oldText": "i: Integer;", "newText": "i: NativeInt;" },
    { "oldText": "for i := 0 to 10 do", "newText": "for i := 0 to 20 do" }
  ],
  "dryRun": true
}
```

**Response:**
```json
{
  "diff": "--- /path/to/file.go\n+++ /path/to/file.go\n@@ -1,3 +1,3 @@\n-func oldName()\n+func newName()\n",
  "readOnlyCleared": true
}
```

The `readOnlyCleared` field indicates if the read-only flag was removed (only present when true).

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

### tree

Compact indented tree view optimized for AI/LLM consumption.

**Parameters:**
- `path` (required): Root directory
- `maxDepth` (optional): Maximum recursion depth (0 = unlimited)
- `maxFiles` (optional): Maximum entries to return (default: 1000)
- `dirsOnly` (optional): Only show directories, not files
- `exclude` (optional): Array of patterns to exclude
- `showEncoding` (optional): Detect and display encoding per file (useful for auditing legacy codebases)

**Example:**
```json
{
  "path": "/path/to/project",
  "maxDepth": 3,
  "exclude": ["node_modules", ".git"]
}
```

**Example with encoding:**
```json
{
  "path": "/path/to/legacy-project",
  "showEncoding": true,
  "exclude": [".git"]
}
```

**Response (with showEncoding):**
```json
{
  "tree": "src/\n  main.pas  [windows-1251]\n  utils.pas  [windows-1251]\nREADME.md  [utf-8]",
  "fileCount": 3,
  "dirCount": 1,
  "truncated": false
}
```

**Response:**
```json
{
  "tree": "src/\n  handler/\n    read.go\n    write.go\n  server.go\nREADME.md",
  "fileCount": 4,
  "dirCount": 2,
  "truncated": false
}
```

### get_file_info

Get metadata about a file or directory (size, timestamps, permissions).

**Parameters:**
- `path` (required): Path to file or directory

### create_directory

Create a directory recursively (like `mkdir -p`). Succeeds if already exists.

**Parameters:**
- `path` (required): Path to directory to create

### move_file

Move or rename files and directories. Fails if destination exists.

**Parameters:**
- `source` (required): Path to move
- `destination` (required): Destination path

### copy_file

Copy a file. Fails if destination exists. Does not copy directories.

**Parameters:**
- `source` (required): Source file path
- `destination` (required): Destination path

### delete_file

Delete a file. Does not delete directories.

**Parameters:**
- `path` (required): Path to delete

### search_files

Recursively search for files and directories matching a glob pattern.

**Parameters:**
- `path` (required): Root directory to search from
- `pattern` (required): Glob pattern (`*.txt` for current dir, `**/*.txt` for recursive)
- `excludePatterns` (optional): Array of patterns to exclude
- `maxResults` (optional): Maximum number of results to return (default: 10000)

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

### grep_text_files

Search file contents using regex patterns with encoding support. Supports context lines and concurrent searching.

**Parameters:**
- `pattern` (required): Regular expression pattern to search for
- `paths` (required): Array of file or directory paths to search
- `caseSensitive` (optional): Case-sensitive matching (default: true)
- `contextBefore` (optional): Number of lines to show before each match
- `contextAfter` (optional): Number of lines to show after each match
- `maxMatches` (optional): Maximum total matches to return (default: 1000)
- `include` (optional): Glob pattern to include files (e.g., `*.go`)
- `exclude` (optional): Glob pattern to exclude files (e.g., `*_test.go`)
- `encoding` (optional): File encoding (auto-detected if omitted)

Directories in `paths` are searched recursively; individual files are searched directly.

`include`/`exclude` are single glob **strings**, not arrays, and are matched against the file
name — use `"*.pas"`, not `["*.pas"]`. Brace sets (`*.{pas,dfm}`) and directory-qualified
patterns (`src/*.pas`) are not supported by the underlying matcher and will match nothing.

**Example:**
```json
{
  "pattern": "func\\s+\\w+",
  "paths": ["/path/to/project"],
  "include": "*.go",
  "contextBefore": 1,
  "contextAfter": 2,
  "maxMatches": 100
}
```

**Response:**
```json
{
  "matches": [
    {
      "path": "/path/to/project/main.go",
      "line": 15,
      "column": 1,
      "text": "func main() {",
      "before": ["package main"],
      "after": ["    fmt.Println(\"Hello\")", "}"],
      "encoding": "utf-8"
    }
  ],
  "totalMatches": 1,
  "filesSearched": 5,
  "filesMatched": 1,
  "truncated": false
}
```

## Encoding Tools

### detect_encoding

Detect the encoding of a file with confidence percentage. Useful for diagnosing encoding issues (garbled text, � characters).

**Parameters:**
- `path` (required): Path to the file
- `mode` (optional): Detection mode
  - `sample` (default): Read begin/middle/end samples - fast, good for most files
  - `chunked`: Read all chunks with weighted averaging - thorough but slower
  - `full`: Read entire file - most accurate but uses more memory

**Example:**
```json
{
  "path": "/path/to/file.pas",
  "mode": "chunked"
}
```

**Response:**
```json
{
  "encoding": "windows-1251",
  "confidence": 95,
  "has_bom": false
}
```

### convert_encoding

Convert a file from one encoding to another. Reads in source encoding, writes in target encoding.
A source BOM is stripped before decoding; a BOM that contradicts an explicit `from` is an error.
No write (and no backup) happens if the file already holds the target bytes — `changed` reports which.

**Parameters:**
- `path` (required): Path to the file to convert
- `from` (optional): Source encoding (auto-detected if omitted)
- `to` (required): Target encoding
- `backup` (optional): Create a `.bak` backup file before converting (default: false)
- `bom` (optional): `auto` (default — BOM for UTF-16 targets, keeps a same-encoding source BOM), `always`, `never`, `preserve`

Omit `from` to auto-detect; pass it only to override detection on a file that misdetects.
A narrowing conversion (e.g. `utf-8` → `cp1251`) fails outright if the content contains
characters the target encoding lacks, rather than writing corrupted text.

**Example:**
```json
{
  "path": "/path/to/file.pas",
  "from": "cp1251",
  "to": "utf-8",
  "backup": true
}
```

Auto-detected source, with a backup:
```json
{
  "path": "D:\\legacy\\data.txt",
  "to": "utf-8",
  "backup": true
}
```

**Response:**
```json
{
  "message": "Converted /path/to/file.pas from windows-1251 to utf-8",
  "sourceEncoding": "windows-1251",
  "targetEncoding": "utf-8",
  "backupPath": "/path/to/file.pas.bak"
}
```

### detect_line_endings

Detect line ending style (CRLF/LF/mixed) and find lines with inconsistent endings. Useful for diagnosing mixed line ending issues in legacy codebases.

**Parameters:**
- `path` (required): Path to the file to analyze
- `encoding` (optional): Auto-detected by default, including most BOM-less UTF-16 text. Pass `utf-16-le` or `utf-16-be` explicitly if a very short or unusual file is misdetected.

**Example:**
```json
{
  "path": "/path/to/file.pas"
}
```

**Response:**
```json
{
  "style": "mixed",
  "totalLines": 150,
  "inconsistentLines": [45, 78, 123]
}
```

**Style values:**
- `crlf`: All lines use Windows line endings (\\r\\n)
- `lf`: All lines use Unix line endings (\\n)
- `mixed`: File has both CRLF and LF endings - `inconsistentLines` lists lines with minority style
- `none`: File has no line endings (single line or empty)

`totalLines` counts newline-terminated lines: a trailing newline does not add a phantom final line, and an empty file reports `0`. (Matches `read_text_file`.)

### change_line_endings

Convert line endings in a file to LF or CRLF. Use after `detect_line_endings` to fix mixed or wrong line endings. No-op if the file already uses the target style. UTF-16 files are converted per code unit and keep their BOM.

**Parameters:**
- `path` (required): Path to the file
- `style` (required): Target line ending style (`"lf"` or `"crlf"`)
- `encoding` (optional): Auto-detected by default, including most BOM-less UTF-16 text. Pass `utf-16-le` or `utf-16-be` explicitly if a very short or unusual file is misdetected.

**Example:**
```json
{
  "path": "/path/to/file.pas",
  "style": "lf"
}
```

**Response:**
```json
{
  "message": "Converted /path/to/file.pas from crlf to lf (3 lines changed)",
  "originalStyle": "crlf",
  "newStyle": "lf",
  "linesChanged": 3
}
```

### manage_bom

Detect, strip, or add Unicode BOM (Byte Order Mark). UTF-8 BOM breaks PHP/shell scripts. UTF-16 files need BOMs for proper detection.

**Parameters:**
- `path` (required): Path to the file
- `action` (required): `"detect"`, `"strip"`, or `"add"`
- `encoding` (required for "add"): BOM encoding — `utf-8`, `utf-16-le`, `utf-16-be`, `utf-32-le`, `utf-32-be`

**Example (detect):**
```json
{
  "path": "/path/to/file.php",
  "action": "detect"
}
```

**Response:**
```json
{
  "message": "BOM detected: utf-8 (3 bytes)",
  "hasBom": true,
  "bomType": "utf-8",
  "bomBytes": 3,
  "changed": false
}
```

**Example (strip):**
```json
{
  "path": "/path/to/file.php",
  "action": "strip"
}
```

**Response:**
```json
{
  "message": "Stripped utf-8 BOM (3 bytes) from /path/to/file.php",
  "hasBom": false,
  "bomType": "utf-8",
  "bomBytes": 3,
  "changed": true
}
```

**Example (add):**
```json
{
  "path": "/path/to/file.txt",
  "action": "add",
  "encoding": "utf-16-le"
}
```

**Response:**
```json
{
  "message": "Added utf-16-le BOM (2 bytes) to /path/to/file.txt",
  "hasBom": true,
  "bomType": "utf-16-le",
  "bomBytes": 2,
  "changed": true
}
```

### list_encodings

Returns all 24 supported encodings with name, aliases, and description.

### list_allowed_directories

Returns directories the server is allowed to access. If empty, add paths as args in config.

### check_for_updates

Checks whether a newer release is available. No parameters. Set `MCP_NO_UPDATE_CHECK=1` to
disable the check.

## Supported Encodings

| Name | Aliases | Description |
|------|---------|-------------|
| utf-8 | utf8, ascii | Unicode, no conversion |
| utf-16-le | utf16le, utf-16le | Unicode UTF-16 Little Endian |
| utf-16-be | utf16be, utf-16be | Unicode UTF-16 Big Endian |
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
| gbk | cp936, gb2312, gb-2312 | Chinese Simplified (GBK) |
| gb18030 | gb-18030 | Chinese Simplified (GB18030, full Unicode) |

UTF-32 LE/BE BOMs are detected by [`detect_encoding`](#detect_encoding) and can be added
or stripped by [`manage_bom`](#manage_bom), but UTF-32 is not a transcoding target:
`convert_encoding` and the `encoding` parameter do not accept it, and
[`change_line_endings`](#change_line_endings) refuses UTF-32 files rather than breaking
their 4-byte alignment.

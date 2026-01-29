# Tools Reference

## File Operations

### read_text_file

Read file contents with optional partial reading (head/tail). UTF-8 files pass through unchanged; other encodings convert to UTF-8.

**Parameters:**
- `path` (required): Path to the file
- `encoding` (optional): utf-8 (default), cp1251, windows-1251
- `head` (optional): Read only the first N lines
- `tail` (optional): Read only the last N lines

**Example:**
```json
{
  "path": "/path/to/file.pas",
  "encoding": "cp1251",
  "head": 50
}
```

### write_file

Write content to file. UTF-8 writes as-is; other encodings convert from UTF-8.

**Parameters:**
- `path` (required): Path to the file
- `content` (required): Content to write
- `encoding` (optional): utf-8, cp1251, windows-1251 (default: cp1251)

**Example:**
```json
{
  "path": "/path/to/file.pas",
  "content": "program Hello;\nbegin\n  writeln('Здравей');\nend.",
  "encoding": "cp1251"
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

## Encoding Tools

### detect_encoding

Detect the encoding of a file with confidence percentage. Returns encoding name, confidence, and BOM presence.

**Parameters:**
- `path` (required): Path to the file

**Example:**
```json
{
  "path": "/path/to/file.txt"
}
```

**Response:**
```json
{
  "encoding": "windows-1251",
  "confidence": 95.5,
  "hasBOM": false
}
```

### list_encodings

Returns all supported encodings.

**Parameters:** None

**Response:**
```json
{
  "encodings": ["utf-8", "cp1251", "windows-1251"]
}
```

## Supported Encodings

| Encoding | Aliases | Description |
|----------|---------|-------------|
| UTF-8 | utf8 | No conversion (passthrough) |
| CP1251 | windows-1251 | Cyrillic (Bulgarian, Russian, Serbian, Ukrainian) |

Planned: CP1250 (Central European), CP1252 (Western European), CP866 (DOS Cyrillic)

# MCP File Tools - Development Plan

## Implemented Features

### Tools (8 total)

| Tool | Status | Description |
|------|--------|-------------|
| `read_text_file` | ✅ Done | Read files with auto-detection and encoding conversion to UTF-8 |
| `write_file` | ✅ Done | Write files with encoding conversion from UTF-8 |
| `list_directory` | ✅ Done | List files with glob pattern filtering |
| `detect_encoding` | ✅ Done | Auto-detect encoding with confidence score and BOM detection |
| `list_encodings` | ✅ Done | List all supported encodings with metadata |
| `list_allowed_directories` | ✅ Done | Show accessible directories (security) |
| `get_file_info` | ✅ Done | Get file/directory metadata (size, times, permissions) |
| `directory_tree` | ✅ Done | Recursive tree view as JSON |

### Encodings Supported (20 total)

| Category | Encodings |
|----------|-----------|
| Unicode | utf-8 (utf8, ascii) |
| Cyrillic | windows-1251 (cp1251), koi8-r, koi8-u, ibm866 (cp866), iso-8859-5 |
| Western European | windows-1252 (cp1252), iso-8859-1 (latin1), iso-8859-15 (latin9) |
| Central European | windows-1250 (cp1250), iso-8859-2 (latin2) |
| Greek | windows-1253 (cp1253), iso-8859-7 |
| Turkish | windows-1254 (cp1254), iso-8859-9 (latin5) |
| Hebrew | windows-1255 (cp1255) |
| Arabic | windows-1256 (cp1256) |
| Baltic | windows-1257 (cp1257) |
| Vietnamese | windows-1258 (cp1258) |
| Thai | windows-874 (cp874, tis-620) |

### Core Features

- [x] Automatic encoding detection using chunked sampling (efficient for large files)
- [x] UTF-8 BOM detection
- [x] Confidence scoring for encoding detection
- [x] Head/tail partial file reading
- [x] Security: path validation to allowed directories only
- [x] MCP roots protocol support (dynamic directory updates)
- [x] Cross-platform support (Windows, Linux, macOS)
- [x] Encoding registry with metadata (display names, aliases, descriptions)

### Build & Distribution

- [x] GoReleaser configuration
- [x] GitHub Actions CI/CD
- [x] Pre-built binaries for Windows x64, Linux x64, macOS ARM64
- [x] Listed in MCP Registry

---

## Planned Features

### Encoding Support Expansion

| Encoding | Use Case | Priority |
|----------|----------|----------|
| GB2312 / GBK | Simplified Chinese | Future |
| Big5 | Traditional Chinese | Future |
| Shift_JIS | Japanese | Future |
| EUC-JP | Japanese (Unix) | Future |
| EUC-KR | Korean | Future |

### Possible Tool Enhancements

- [ ] `convert_encoding` - Convert file from one encoding to another
- [ ] `search_files` - Search file contents with encoding support
- [ ] `compare_files` - Compare files with different encodings
- [ ] Batch operations support

### Architecture Improvements

- [ ] Separate detection logic from registry (registry.go -> registry.go + detect.go)
- [ ] Improve detection accuracy for similar encodings (e.g., CP1251 vs KOI8-R)

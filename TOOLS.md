# Tools Reference

## Large results

The tools that can legitimately return a lot of text — `read_text_file`, `read_multiple_files`,
`grep_text_files` and `tree` — declare `anthropic/maxResultSizeChars` in their
`tools/list` `_meta`. Clients that honour it (Claude Code) raise their default result cap for
those tools instead of spilling the overflow to a file reference, which would otherwise hide
part of a tree or search result from the model. Every other tool keeps the client default.

## Gitignore

`tree`, `search_files` and `grep_text_files` honour `.gitignore` files (nested ones too,
full syntax: negation, anchoring, dir-only, `**`) and skip `.git`. Pass
`respectGitignore: false` to see everything. Trees without a `.gitignore` are unaffected.

## File Operations

### read_text_file

Read file contents with automatic encoding detection and optional partial reading. UTF-8 files pass through unchanged; other encodings convert to UTF-8.

**Parameters:**
- `path` (required): Path to the file
- `encoding` (optional): Encoding name (auto-detects if omitted)
- `offset` (optional): Start reading from this line number (1-indexed)
- `limit` (optional): Maximum number of lines to read
- `maxCharacters` (optional): Truncate content at this character count to prevent token overflow
- `lineNumbers` (optional, default false): Prefix every line with `N<tab>`. Numbering is
  absolute — a paged read starting at `offset: 100` numbers from 100 — so it lines up with
  `grep_text_files` results, `manage_line_endings` reports and encoding-error positions.
  Strip the prefix before using text as `edit_file` `oldText`; the file itself has no numbers.

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

The response may carry a `hint` field: it reports a file that already has
**mixed** line endings, and — once per file — notes that a plain utf-8 file with
no BOM is better handled by the agent's own built-in tools.

`totalLines` counts newline-terminated lines: a trailing newline does not add an empty final line, and an empty file is `0`. (Matches `manage_line_endings`.)

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

When content is normalised, the response says so and sets `lineEndings`, so the
conversion is never silent.

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

Make replacements or apply a unified diff to one text file. Returns a unified diff showing changes.

**Parameters:**
- `path` (required): Path to the file to edit
- `edits`: Array of edit operations with `oldText`, `newText`, and optional `similarity` (0.0-1.0)
- `patch`: Unified diff string with `---`, `+++`, and `@@` hunks
- `dryRun` (optional): If true, returns diff without writing changes (default: false)
- `encoding` (optional): File encoding (auto-detected if not specified)
- `forceWritable` (optional): If true, clears read-only flag before editing (default: false — fails on read-only files)

**Features:**
- Exact text matching (first occurrence)
- Whitespace-flexible matching (ignores per-line leading *and* trailing whitespace; interior spacing must still match)
- Optional fuzzy matching. The score is `1 - line edit distance / longer block length`, after trimming each line. Candidates more than `oldText`'s line count away are rejected.
- Patch hunks use the same exact then whitespace-flexible matching. Multi-file patches are rejected.
- Preserves original indentation
- CRLF line endings normalized to LF, so CRLF/LF differences never block a match
- Atomic write (temp file + rename)
- Fails on read-only files by default (set `forceWritable: true` only when user explicitly requests it)

Edits apply in order, and each one replaces only the **first** match remaining at that point.
Pass exactly one of `edits` or `patch`. A diff returned by `edit_file` can be passed back as `patch` against the original file.
If an exact edit fails, prefer copying the hint into `oldText`. Use `similarity` only to tolerate whitespace or comment drift, ideally with `dryRun: true`; it is not for different code. Below threshold, the error includes the best score.

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
- `sortBy` (optional): `name` (default), `mtime` or `size` — see [Ordering](#ordering)
- `reverse` (optional): Flip the order

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

Directories carry a `[DIR] ` prefix but sort on the bare name, so `[DIR] beta`
lands between `alpha.pas` and `gamma.pas`.

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
- `sortBy` (optional): `name` (default), `mtime` or `size` — see [Ordering](#ordering)
- `reverse` (optional): Flip the order

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

#### Ordering

`sortBy` applies to `search_files` and `list_directory`. `tree` is unsorted by
design — its output is hierarchical, and mtime order across a hierarchy means
nothing.

| `sortBy` | Order | Reverse gives |
|---|---|---|
| `name` (default) | lexical, ascending | Z → A |
| `mtime` | newest first | oldest first |
| `size` | largest first | smallest first |

This follows `ls`: `ls` is alphabetical, `ls -t` is newest first, `ls -S` is
largest first, and `-r` flips each. So "what changed recently" is plain
`sortBy: "mtime"`, no `reverse` needed. Ties break on name.

Before this, results came back in **walk order** — close to lexical but not
guaranteed, because a subdirectory's contents are emitted right after its own
entry. The default is now an explicit sort.

**`search_files` and `maxResults`.** How the cap interacts with the ordering
depends on the field, and the difference matters:

- `name` caps first, then sorts. The walk stops at `maxResults` as it always
  has, and the returned subset is sorted. Walk order is already near-lexical, so
  this stays the cheapest path — no stat calls, no full traversal.
- `mtime` and `size` rank first, then cap. The newest file may be the last one
  visited, so capping first would return "the first `maxResults` files in walk
  order, sorted by mtime" — the wrong answer to "what changed recently". These
  walk the whole tree behind a bounded heap of `maxResults` entries, so memory
  stays capped and a truncated result really is the newest/largest N.

`truncated: true` means more files matched than were returned, in both cases.

**Cost.** `name` reads nothing beyond the entry name. `mtime` and `size` call
`Info()` on each matching entry: free on Windows, where `FindNextFile` already
returned the metadata, but one `lstat` per match on Linux and macOS. An entry
whose `Info()` fails is kept with a zero mtime/size and sorts last.

### grep_text_files

Search file contents using regex patterns with encoding support. Supports context lines and concurrent searching.

**Parameters:**
- `pattern` (required): Regular expression pattern to search for
- `paths` (required): Array of file or directory paths to search
- `outputMode` (optional): `content` (default), `files_with_matches`, or `count`
- `caseSensitive` (optional): Case-sensitive matching (default: true)
- `matchesOnly` (optional): Return the matched substring instead of the whole line
- `contextBefore` (optional): Number of lines to show before each match
- `contextAfter` (optional): Number of lines to show after each match
- `maxMatches` (optional): Maximum results per page (default: 1000)
- `offset` (optional): Skip the first N results, to page past `maxMatches`
- `include` (optional): Glob pattern to include files (e.g., `*.go`)
- `exclude` (optional): Glob pattern to exclude files (e.g., `*_test.go`)
- `includes` (optional): Glob patterns; a file matching any is included
- `excludes` (optional): Glob patterns; a file matching any is excluded
- `encoding` (optional): File encoding (auto-detected if omitted)

Directories in `paths` are searched recursively; individual files are searched directly.

Patterns are matched against the file name only. Do not combine `include` with `includes`,
or `exclude` with `excludes`.

**Example:**
```json
{
  "pattern": "func\\s+\\w+",
  "paths": ["/path/to/project"],
  "includes": ["*.pas", "*.dfm"],
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

`filesSearched` counts the files actually read: a full page stops the search, so
a truncated result reports fewer than it collected.

An `encoding` the registry does not know does not fail the search — each file's
encoding is detected instead, and the response carries a `hint` saying so.
`list_encodings` prints the accepted names.

#### Output modes

`outputMode` decides what comes back, and how much of each file gets read.

| Mode | Returns | Reads |
|---|---|---|
| `content` (default) | `matches[]` with the full line, plus context | every matching line, up to the page end |
| `files_with_matches` | `files[]`, paths only | **stops at the first match in each file** |
| `count` | `counts[]` of `{path, count}` matching lines per file | the whole file, but keeps no text |

Ask `files_with_matches` when the question is *which* files contain something —
it is the cheapest answer by a wide margin, in tokens and in I/O:

```json
{"pattern": "TFormMain", "paths": ["D:\\proj\\src"], "outputMode": "files_with_matches"}
```

```json
{
  "matches": [],
  "files": ["D:\\proj\\src\\main.pas", "D:\\proj\\src\\forms.pas"],
  "totalMatches": 2,
  "filesSearched": 412,
  "filesMatched": 2
}
```

`contextBefore`/`contextAfter` are meaningless outside `content` and are ignored
silently — no context fields come back in the other two modes.

`totalMatches` reports what the page actually holds: matching lines in `content`,
returned paths in `files_with_matches` (the search stops at one hit per file, so it
is *not* a match total — use `count` for that), and the summed per-file counts in
`count`. `filesMatched` counts every file with a hit, including ones dropped by
`offset`.

#### matchesOnly

`matchesOnly: true` puts the matched substring in `text` instead of the whole
line — `ripgrep -o`. Unlike the default, which reports the first match per line,
this returns *every* occurrence on a line, each with its own `column`, so it works
for pulling values out of a file:

```json
{"pattern": "\\d+\\.\\d+\\.\\d+", "paths": ["version.inc"], "matchesOnly": true}
```

#### Paging

`maxMatches` caps one page — matches in `content`, paths in `files_with_matches`,
file entries in `count`. `offset` skips that many results first, so `truncated`
is no longer a dead end: the response carries `nextOffset`, and passing it back
returns the next page.

```json
{"pattern": "TODO", "paths": ["src"], "maxMatches": 1000, "offset": 1000}
```

The default of 1000 is deliberately unchanged; Claude Code's own `Grep` uses 250,
which is cheaper per call but pages more often. Lower `maxMatches` yourself if you
prefer that trade.

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
- `path`: Path to a single file to convert — use this **or** `paths`, never both
- `paths`: Array of files to convert as a batch
- `from` (optional): Source encoding (auto-detected per file if omitted)
- `to` (required): Target encoding
- `backup` (optional): Create a `.bak` backup file before converting (default: false)
- `dryRun` (optional): Report what would change and write nothing (default: false)
- `allowLowConfidence` (optional): Convert even when the auto-detected source is below the confidence threshold (default: false)
- `bom` (optional): `auto` (default — BOM for UTF-16 targets, keeps a same-encoding source BOM), `always`, `never`, `preserve`

Omit `from` to auto-detect; pass it only to override detection on a file that misdetects.
An untrusted detection stops the conversion rather than guessing: a bad guess writes
nonsense that then detects as valid UTF-8, so the original encoding is no longer
recoverable. The error names the best guess.
A narrowing conversion (e.g. `utf-8` → `cp1251`) fails outright if the content contains
characters the target encoding lacks, rather than writing corrupted text — and the error
names those characters with their line and column, so you can decide what to do about them.

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

#### Batch conversion

Pass `paths` instead of `path`. One bad file does not stop the rest — each gets an entry in
`results`, and `errors` summarises the failures. Combine with `dryRun` to preview a whole
migration before touching anything:

```json
{
  "paths": ["src/main.pas", "src/forms.pas", "src/legacy.pas"],
  "to": "utf-8",
  "dryRun": true
}
```

**Response:**
```json
{
  "message": "2 of 3 files would convert to utf-8; 1 failed (see results)",
  "targetEncoding": "utf-8",
  "dryRun": true,
  "successCount": 2,
  "errorCount": 1,
  "errors": ["src/legacy.pas: failed to encode to utf-8: ..."],
  "results": [
    {
      "path": "src/main.pas",
      "sourceEncoding": "windows-1251",
      "changed": true,
      "message": "Would convert src/main.pas from windows-1251 to utf-8"
    }
  ]
}
```

When a file cannot be encoded, its result carries the offending characters as data, not just
prose — `unsupportedCount` plus an `unsupported` array of `{char, code, line, column}`. That is
what makes a dry run over a whole project actionable.

**Getting the file list:** `paths` takes explicit files, not directories or globs. Use
`search_files` or `tree` to build the list first.

### manage_line_endings

Detect or convert line endings. Mirrors `manage_bom`: one tool, an `action` parameter.

**Parameters:**
- `path` (required): Path to the file
- `action` (required): `"detect"` or `"convert"`
- `style` (required for `convert`): `"lf"` or `"crlf"`
- `encoding` (optional): Auto-detected by default, including most BOM-less UTF-16 text. Pass `utf-16-le` or `utf-16-be` explicitly if a very short or unusual file is misdetected.

`convert` is a no-op if the file already uses the target style. UTF-16 files are
converted per code unit and keep their BOM.

**Detect:**
```json
{
  "path": "/path/to/file.pas",
  "action": "detect"
}
```
```json
{
  "style": "mixed",
  "totalLines": 150,
  "inconsistentLines": [45, 78, 123]
}
```

**Convert:**
```json
{
  "path": "/path/to/file.pas",
  "action": "convert",
  "style": "lf"
}
```
```json
{
  "style": "lf",
  "originalStyle": "crlf",
  "linesChanged": 3,
  "changed": true,
  "message": "Converted /path/to/file.pas from crlf to lf (3 lines changed)"
}
```

**Style values:**
- `crlf`: All lines use Windows line endings (`\r\n`)
- `lf`: All lines use Unix line endings (`\n`)
- `mixed`: File has both — `inconsistentLines` lists lines with the minority style
- `none`: File has no line endings (single line or empty)

`totalLines` counts newline-terminated lines: a trailing newline does not add a phantom final line, and an empty file reports `0`. (Matches `read_text_file`.)

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

Returns every [supported encoding](#supported-encodings) with its name, aliases, and description.

### list_allowed_directories

Returns directories the server is allowed to access. If empty, add paths as args in config.

### check_for_updates

Checks whether a newer release is available, at most one GitHub API call per 30 minutes.
Set `MCP_NO_UPDATE_CHECK=1` to disable the check.

**Parameters:**
- `force` (optional): Bypass the cached result and query GitHub now (default: false)

Returns `currentVersion`, `latestVersion`, `installMethod` (`plugin` when the Claude Code
plugin launcher started the server, otherwise `manual`), and `updateMessage` when an update
exists — with the update steps that apply to that install and client.

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
| x-mac-cyrillic | maccyrillic, mac-cyrillic | Macintosh Cyrillic |
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
[`manage_line_endings`](#manage_line_endings) refuses UTF-32 files rather than breaking
their 4-byte alignment.

## Prompts

Besides tools, the server exposes three prompts — guided workflows that clients surface as
user commands (Claude Code: `/mcp__file-tools__<name>`, opencode: commands). Each renders to
a complete task brief the model executes with the tools above.

| Prompt | Arguments | What it does |
|---|---|---|
| `audit_encodings` | `path` | Read-only survey: encoding distribution via `tree showEncoding=true`, BOMs, low-confidence detections |
| `fix_mojibake` | `path` | Diagnose garbled text, confirm the correct reading with the user, then repair with `convert_encoding backup=true` |
| `migrate_to_utf8` | `path`, `pattern` (default `*.pas`) | Batch migration: build the list, `dryRun=true` first, report lossy files, convert with backups, verify |

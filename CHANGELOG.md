# Changelog

Notable changes to `mcp-file-tools`. This file starts at 2.0.0; for earlier
versions see the [GitHub releases](https://github.com/dimitar-grigorov/mcp-file-tools/releases).

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **`read_text_file` takes `lineNumbers`** (default false): prefixes every line
  with `N<tab>`, using absolute numbers so a paged read starting at line 100
  numbers from 100. Everything else already speaks in line numbers —
  `grep_text_files` matches, `manage_line_endings` reports, and encoding errors
  ("ä at line 12, column 8") — but the reading tool gave the model nothing to
  correlate them against. Off by default so a read → write round trip cannot
  bake numbers into a file; the encoding-error message now points to it.

- **Three MCP prompts**: `audit_encodings`, `fix_mojibake` and
  `migrate_to_utf8` — guided workflows that clients surface as user commands
  (Claude Code slash commands, opencode commands). Unlike the bundled skill,
  prompts work in every MCP client. The server now advertises the `prompts`
  capability.

### Changed

- `edit_file` now tells the model not to re-read a file to verify an edit: a
  success with a diff means the edit is on disk, a failure changes nothing.
  Saves one read per edit.

## [3.1.0] - 2026-08-01

### Added

- **`search_files` and `list_directory` take `sortBy` and `reverse`.** `name`
  (the default), `mtime` (newest first) or `size` (largest first), following
  `ls`/`ls -t`/`ls -S`; `reverse` flips each. Results used to come back in walk
  order — close to lexical, but not guaranteed — and the default is now an
  explicit sort. `tree` is deliberately left alone.

  `search_files` ranks the whole tree *before* applying `maxResults` when sorting
  by `mtime` or `size`, behind a bounded heap, so a truncated result is genuinely
  the newest or largest N rather than the first N found. `name` keeps the old
  early stop, and stats nothing, so the default path costs exactly what it did.

- **`grep_text_files` can answer without the match text.** New `outputMode`:
  `content` (the default, unchanged), `files_with_matches` (paths only) and
  `count` (matching lines per file).

  "Which files mention `TFormMain`" used to cost up to 1000 match objects with
  their lines; `files_with_matches` returns the path list and stops reading each
  file at its first hit, so it is cheaper in I/O as well as tokens.

  Also new: `matchesOnly`, which puts the matched substring in `text` instead of
  the whole line — every occurrence on the line, each with its own column — for
  extracting values rather than reading context. And `offset`, which skips the
  first N results: `truncated` was previously a dead end with no way to reach
  match 1001, and the response now carries `nextOffset` to page on with.

  `maxMatches` still defaults to 1000 and `content` mode is byte-for-byte what it
  was.

- **A failed conversion now names the characters that do not fit.** Encoding to a
  narrower charset used to fail with x/text's positionless
  `rune not supported by encoding`, which left the caller nothing to act on but a
  blind retry. It now reports which characters and where:

  ```
  cp1251 cannot represent 3 characters: "ä" (U+00E4) at line 12, column 8,
  "ü" (U+00FC) at line 12, column 20, "ß" (U+00DF) at line 40, column 5.
  Convert to utf-8 instead, or remove these characters.
  ```

  Applies to `convert_encoding`, `write_file` and `edit_file` alike, since all
  three shared the same dead end. Up to 10 characters are listed; the count is
  always exact.

- **`convert_encoding` takes a batch and a dry run.** New `paths` array (mutually
  exclusive with `path`) and `dryRun`.

  A batch does not stop at the first failure — every file gets an entry in
  `results`, with `errors` summarising what went wrong. A file that cannot be
  encoded carries the offenders as data (`unsupportedCount`, plus `unsupported`
  as `{char, code, line, column}`), so `dryRun` over a whole project tells you
  which files would lose characters *before* anything is written.

  Migrating a legacy tree used to be N calls with no preview; it is now one call
  you can inspect first. `paths` takes explicit files — build the list with
  `search_files` or `tree`.

  A single `path` behaves exactly as before, including failing hard rather than
  reporting the failure in `results`.

### Changed

- **`move_file` now sets `destructiveHint: true`.** It was `false`, which the spec
  defines as "performs only additive updates" — but a move removes the source path,
  so undoing one requires knowing where the file came from. Clients use this hint to
  decide what needs confirmation, and a move was being presented as safe as a copy.
  This aligns with the reference filesystem server. `copy_file` and
  `create_directory` stay `false`; they only add.

## [3.0.0] - 2026-08-01

### Removed — BREAKING

- **`directory_tree` is gone.** It had been deprecated in favour of `tree`,
  which returns the same structure in ~85% fewer tokens. Callers still using it
  should switch to `tree`; `excludePatterns` is spelled `exclude` there, and the
  result is indented text with `fileCount`/`dirCount` rather than JSON.

  The tool count drops from 22 to 21.

- **`detect_line_endings` and `change_line_endings` merged into
  `manage_line_endings`**, which takes an `action` of `"detect"` or `"convert"`
  — the same shape `manage_bom` already used. Behaviour is unchanged; only the
  entry point moved.

  ```json
  { "path": "…", "action": "detect" }
  { "path": "…", "action": "convert", "style": "crlf" }
  ```

  Tool count 21 → 20. The merged tool is annotated as a writer, so `detect` no
  longer carries a `readOnlyHint` — the same trade-off `manage_bom` makes.

### Fixed

- **`write_file` no longer leaves CRLF files with mixed line endings.**

  It wrote `content` byte for byte, with no line-ending policy at all — while
  `read_text_file` hands back `\r\n` intact and `edit_file` carefully restores
  the original style. So the read → regenerate → `write_file` path corrupted a
  CRLF file whenever the model emitted `\n` for the lines it rewrote, which is
  what models naturally do. Intermittent by nature: `edit_file` was always safe,
  so only full rewrites were affected.

  `write_file` now converts content to the existing file's dominant style, and a
  file that is already mixed gets repaired. New `lineEndings` parameter
  (`preserve` default, `crlf`, `lf`, `asis`) and `MCP_DEFAULT_LINE_ENDINGS` for
  new files.

  **Behaviour change:** `write_file` is no longer byte-verbatim by default. Pass
  `lineEndings: "asis"` for the old behaviour.

### Added

- **The plugin now bundles a skill, `fixing-text-encodings`.** It covers what no single
  tool description can: the mojibake symptom-to-cause table, surveying a project with
  `tree showEncoding=true`, and the backup-and-verify checklist for a bulk conversion.
  Claude loads it only when a task matches, so it costs nothing otherwise.

- **`read_text_file` and `write_file` now return a `hint`** when there is
  something the agent should know: a file that already has mixed line endings
  (with counts, and a pointer to `manage_line_endings`), and — once per file per
  session — that a plain utf-8 file with no BOM is better served by the agent's
  own built-in tools, since this server adds nothing for it.

  `write_file` also reports when it normalised line endings and sets
  `lineEndings` in its result, so the conversion is never silent.

### Changed

- **Upgraded to `modelcontextprotocol/go-sdk` v1.7.0** (MCP spec `2026-07-28`).
  Older clients are unaffected: the SDK serves both the legacy `initialize`
  handshake and the new stateless model behind the same API.

- **Server capabilities are now declared explicitly** as `tools` only. Left
  unset, the SDK advertised `logging` (now deprecated) and inferred
  `listChanged: true` — a capability this server does not support, since its
  tool set is static and it never emits `list_changed`.

- `readOnlyHint` and `idempotentHint` are now always present in `tools/list`
  instead of omitted when `false`, so "false" and "unset" are distinguishable.

Roots and MCP logging are deprecated in `2026-07-28` with a 12-month window,
and are retained for older clients.

- **Usage examples in the five most ambiguous tool descriptions** —
  `read_text_file` (offset/limit paging), `write_file` and `convert_encoding`
  (BOM modes, auto-detected vs explicit `from`), `edit_file` (multi-edit arrays,
  whitespace latitude), `grep_text_files` (`include`/`exclude` are single
  basename globs, not arrays). Mirrored in `TOOLS.md`.

- **`anthropic/maxResultSizeChars` declared** on the tools that can return large
  output: `read_text_file` and `tree` (200,000), `read_multiple_files` and
  `grep_text_files` (300,000). Without it the client truncates those results to
  a file reference, hiding part of a tree or search result from the model.

## [2.0.1] - 2026-07-27

### Fixed

- **Pure-ASCII files no longer silently convert to `utf-8` on edit/write.**

  Detection reports `ascii` with full confidence for ASCII-only content, and
  `ascii` is a registered `utf-8` alias — so it was treated as a confident
  encoding match. For an existing legacy file (e.g. `cp1251`) that happened to
  contain no non-ASCII bytes yet, this silently overrode `MCP_DEFAULT_ENCODING`
  and re-saved it as `utf-8` the moment any edit added non-ASCII text.

  `ascii` detections now fall through to the configured default, same as any
  other inconclusive detection.

- **`edit_file` now honors `MCP_DEFAULT_ENCODING`.**

  It previously hardcoded `utf-8` as its fallback for inconclusive detection,
  ignoring the configured default entirely — `write_file` already respected it.

## [2.0.0] - 2026-07-26

### Changed — BREAKING

- **`write_file` now defaults new files to `utf-8` instead of `cp1251`.**

  Previously, calling `write_file` without an `encoding` on a file that did not
  yet exist fell back to `cp1251`. Because cp1251 has no umlauts, CJK or most
  Western European characters, the first write of such text failed outright:

  ```
  write_file("Bäcker Grüße Straße")
    -> failed to encode content: encoding: rune not supported by encoding.
  ```

  Failing loudly was correct — writing `B?cker` would have been worse — but a
  Cyrillic default made that the first thing a non-Cyrillic user hit.

  **Who is affected:** only callers that create **new** files and relied on the
  implicit cp1251 default.

  **Who is not affected:** everyone editing existing files. The encoding
  resolution order is unchanged —

  > explicit `encoding` > existing file's detected encoding > configured default

  so an existing cp1251 file is still detected and rewritten as cp1251. In
  testing, realistic cp1251 Delphi sources are detected at 96–99 confidence
  (threshold is 50), including files with only a single Cyrillic string literal.
  `edit_file` never consulted the configured default and is unchanged.

  **Migration —** to restore the old behaviour, set the environment variable:

  ```json
  "env": { "MCP_DEFAULT_ENCODING": "cp1251" }
  ```

  Per machine in your client config, or committed to a legacy project's
  `.mcp.json` so everyone working in that repo gets it without local setup.
  See [Legacy teams](README.md#legacy-teams-pre-200-behaviour).

### Added

- **Transitional:** the first `write_file` in a session that creates a new file
  under the built-in default appends a one-line notice about the utf-8 change to
  its result, so upgrading users hear about it without reading this file. It is
  silent when `encoding` was explicit, when an existing file's encoding was
  preserved, when `MCP_DEFAULT_ENCODING` is set, and on every later write.
  **This notice is removed in 2.3.0.**
- `make lint` now runs `staticcheck` at the same pinned version as CI, so local
  lint no longer passes on code CI rejects.

### Fixed

- Repo-wide `gofmt` drift that `make lint` would otherwise rewrite unpredictably.

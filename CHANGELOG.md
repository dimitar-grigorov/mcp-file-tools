# Changelog

Notable changes to `mcp-file-tools`. This file starts at 2.0.0; for earlier
versions see the [GitHub releases](https://github.com/dimitar-grigorov/mcp-file-tools/releases).

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [3.4.0] - 2026-08-14

### Added

- **MacCyrillic encoding** (`x-mac-cyrillic`, alias `maccyrillic`) — 25 encodings total.

### Fixed

- **CP1251 files no longer come back garbled via a MacCyrillic guess.** Detection
  keeps whichever of the two decodes better, and a read falling back to raw
  UTF-8 now says so in the response.

## [3.3.0] - 2026-08-03

### Changed

- **Update notices match how the server was installed**: plugin installs get the
  `claude plugin update` commands, manual installs get re-download steps naming
  their own client.

### Added

- **`check_for_updates` returns `installMethod`** (`plugin` or `manual`).

### Fixed

- **An allowed directory reached by an alias** (macOS `/var` → `/private/var`)
  no longer denies every path under it.
- **A Windows 8.3 short path as an allowed directory** (`C:\Users\DIMITA~1.GRI`)
  no longer denies everything under it.
- **An empty client roots list revokes that client's earlier roots** instead of
  leaving them authorized for the life of the process. CLI args are unaffected.

## [3.2.0] - 2026-08-01

### Added

- **`grep_text_files` accepts `includes` and `excludes` arrays**; the singular
  forms still work.
- **`edit_file` edits take optional `similarity`** (0.0–1.0) for bounded,
  line-based fuzzy matching.
- **`edit_file` accepts a one-file unified diff** through `patch`.
- **`read_text_file` takes `lineNumbers`** (default false): prefixes every line
  with `N<tab>`, absolute numbering. Off by default so a read → write round
  trip cannot bake numbers into a file.
- **Three MCP prompts**: `audit_encodings`, `fix_mojibake` and `migrate_to_utf8`
  — guided workflows clients surface as user commands; the server now advertises
  the `prompts` capability.
- **`tree`, `search_files` and `grep_text_files` honour `.gitignore`** and skip
  `.git`. **Behaviour change:** on by default; pass `respectGitignore: false`
  for the old behaviour.

### Changed

- `edit_file` now tells the model not to re-read a file to verify an edit.

## [3.1.0] - 2026-08-01

### Added

- **`search_files` and `list_directory` take `sortBy` and `reverse`** — `name`
  (default), `mtime`, `size`. With `mtime`/`size` the whole tree is ranked
  before `maxResults`, so a truncated result really is the newest/largest N.
- **`grep_text_files` gains `outputMode`** (`content`, `files_with_matches`,
  `count`), plus `matchesOnly` and `offset`/`nextOffset` for paging past
  `truncated`.
- **A failed conversion names the characters that do not fit** — character,
  code point, line and column, up to 10 listed — in `convert_encoding`,
  `write_file` and `edit_file`.
- **`convert_encoding` takes a batch and a dry run.** `paths` reports per-file
  `results` instead of stopping at the first failure; `dryRun` shows which files
  would lose characters before anything is written. A single `path` behaves
  exactly as before.

### Changed

- **`move_file` now sets `destructiveHint: true`** (a move removes the source);
  `copy_file` and `create_directory` stay `false`.

## [3.0.0] - 2026-08-01

### Removed — BREAKING

- **`directory_tree` is gone** — use `tree`: same structure, ~85% fewer tokens,
  `excludePatterns` is spelled `exclude`.
- **`detect_line_endings` and `change_line_endings` merged into
  `manage_line_endings`** with `action: "detect" | "convert"`. Behaviour
  unchanged; the merged tool no longer carries a `readOnlyHint`.

### Fixed

- **`write_file` no longer leaves CRLF files with mixed line endings.** Content
  converts to the existing file's dominant style; already-mixed files get
  repaired. New `lineEndings` parameter (`preserve` default, `crlf`, `lf`,
  `asis`) and `MCP_DEFAULT_LINE_ENDINGS` for new files. **Behaviour change:**
  no longer byte-verbatim by default — pass `lineEndings: "asis"` for that.

### Added

- **The plugin bundles a skill, `fixing-text-encodings`**: the mojibake
  symptom-to-cause table, project surveys, and the backup-and-verify checklist
  for bulk conversions.
- **`read_text_file` and `write_file` return a `hint`** for mixed line endings
  and (once per file per session) plain utf-8 files better served by built-in
  tools. `write_file` also reports normalised line endings in `lineEndings`.

### Changed

- **Upgraded to `modelcontextprotocol/go-sdk` v1.7.0** (MCP spec `2026-07-28`);
  older clients are unaffected.
- **Server capabilities declared explicitly** as `tools` only — no deprecated
  `logging`, no false `listChanged`.
- `readOnlyHint` and `idempotentHint` are always present in `tools/list`.
- **Usage examples in the five most ambiguous tool descriptions**, mirrored in
  `TOOLS.md`.
- **`anthropic/maxResultSizeChars` declared** on large-output tools so clients
  do not truncate their results to a file reference.

## [2.0.1] - 2026-07-27

### Fixed

- **Pure-ASCII files no longer silently convert to `utf-8` on edit/write.**
  `ascii` detections fall through to the configured default like any other
  inconclusive detection instead of overriding `MCP_DEFAULT_ENCODING`.
- **`edit_file` now honors `MCP_DEFAULT_ENCODING`** instead of hardcoding
  `utf-8` for inconclusive detection.

## [2.0.0] - 2026-07-26

### Changed — BREAKING

- **`write_file` defaults new files to `utf-8` instead of `cp1251`.** Only
  callers creating **new** files without `encoding` are affected. Existing
  files keep their detected encoding — the resolution order (explicit
  `encoding` > detected > configured default) is unchanged. To restore the old
  behaviour set `"env": { "MCP_DEFAULT_ENCODING": "cp1251" }`; see
  [Legacy teams](README.md#legacy-teams-pre-200-behaviour).

### Added

- **Transitional:** the first `write_file` creating a new file under the
  built-in default appends a one-line notice about the utf-8 change.
  **Removed in 2.3.0.**
- `make lint` runs `staticcheck` at the same pinned version as CI.

### Fixed

- Repo-wide `gofmt` drift.

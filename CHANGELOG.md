# Changelog

Notable changes to `mcp-file-tools`. This file starts at 2.0.0; for earlier
versions see the [GitHub releases](https://github.com/dimitar-grigorov/mcp-file-tools/releases).

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- **`go install` silently served v1.8.1, not the current release.** The module path
  lacked the `/v4` suffix Go requires from v2 on, so the proxy ignored every tag since
  2.0.0. The module is now `github.com/dimitar-grigorov/mcp-file-tools/v4`.

## [4.2.0] - 2026-08-15

### Added

- **`MCP_DETECTION_CANDIDATES` pins what detection may answer**, in priority order. A BOM
  still wins and a guess inside the list keeps its confidence; one outside it gives way to
  the first listed encoding that decodes the bytes cleanly, and nothing fitting means no
  answer rather than a wrong one. Fixes Spanish CP1252 read as GBK. Unset changes nothing.
- **`grep_text_files` takes `patterns`** — an array searched as one alternation, so a list
  of names is one call and one pass; `matchesOnly` says which pattern hit.

Both reimplemented from [Mario Rial](https://github.com/seguridadea1)'s fork.

## [4.1.0] - 2026-08-14

### Added

- **Ranked encoding candidates when detection cannot decide.** `detect_encoding` returns
  `candidates` (best first, each flagged `supported`) for a verdict under 80% confident or
  outside the registry; `read_text_file` names them in `hint`, `convert_encoding` in its error.
- **Progress notifications on a batch `convert_encoding`** — per file, capped at 100 per
  call, only with a `progressToken`.
- **Server title, description, website and icon** (SEP-973), the icon an inline data URI.

## [4.0.0] - 2026-08-14

### Added

- **Built-in tool parameter names are accepted as aliases.** A call shaped like
  Claude Code's Read/Write/Edit/Grep — `file_path`, flat `old_string`/`new_string`/
  `replace_all`, `-A`/`-B`/`-C`/`-i`/`-o`, `head_limit`, `output_mode`, a single
  grep `path` string — is translated where semantics match exactly, instead of
  failing schema validation. Canonical names always win; Grep's `glob`/`type` stay
  unsupported (basename-only `includes` instead).
- **`{a,b}` glob alternatives.** `search_files` patterns and excludes and grep
  `include`/`excludes` accept `*.{pas,dfm}`; grep basename globs also ignore a
  leading `**/`. Both shapes previously matched nothing, silently.

### Fixed

- **`search_files` patterns with several `**` silently matched nothing** (e.g.
  `src/**/test/**/*.go`). The matcher is now segment-based and `**` may appear
  any number of times.
- **`write_file` and `edit_file` stripped the CRLFs from UTF-16 files.** Both read
  line endings off raw bytes, where the `00` between CR and LF hides every CRLF, so
  "preserve" rewrote the file as LF. Detection now runs on decoded text, as
  `change_line_endings` already did.
- **`grep_text_files` searched with a silently wrong decode** when given an
  `encoding` it could not resolve. It now detects per file and says so in `hint`.
- **`grep_text_files` over-reported `filesSearched`** once a full page stopped the
  search early.
- **`search_files` reported `truncated` for a result set that exactly filled
  `maxResults`.**
- **`detect_encoding` with `mode="chunked"` broke a tie at random**, the winner
  coming out of a map range. Ties now break by name.

### Changed

- **`edit_file` no longer edits the first of several identical matches.** An `oldText`
  matching more than one place now fails with their line numbers and changes nothing;
  add context to pick one, or pass the new `replaceAll: true` to change them all. The
  response then reports `replacements`. Picking one silently hit the wrong copy as often
  as the right one, and the tool tells the agent not to re-read and verify.
- One `Decode` behind every tool that reads encoded bytes.

## [3.4.1] - 2026-08-14

### Fixed

- **`edit_file` could stall for minutes on a failed edit** — finding the closest
  match was cubic in `oldText`. A 50-line block against a 2,000-line file: 5.6s
  → 4ms.
- **`read_text_file` and `grep_text_files` ignored `MCP_DEFAULT_ENCODING`** on an
  inconclusive detection; writes and edits already honoured it. No change on the
  default `utf-8`.
- **`convert_encoding` converted on a detection it did not trust**, and a bad
  guess is unrecoverable — the result detects as valid UTF-8. It now stops and
  names the guess. **Behaviour change:** confirm with `from`, or pass the new
  `allowLowConfidence`.

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
  **Runs to the end of 2026.**
- `make lint` runs `staticcheck` at the same pinned version as CI.

### Fixed

- Repo-wide `gofmt` drift.

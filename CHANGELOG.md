# Changelog

Notable changes to `mcp-file-tools`. This file starts at 2.0.0; for earlier
versions see the [GitHub releases](https://github.com/dimitar-grigorov/mcp-file-tools/releases).

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [4.4.1] - 2026-09-10

- **`copy_file` dropped `idempotentHint: true`** — a repeat fails on the destination the
  first call created. Its description now says what the copy keeps.
- **Lint tooling moved into a `tools/` module** (staticcheck, govulncheck v1.8.0, actionlint)
  so Dependabot bumps the pins; CI gates on `gofmt -l`, which nothing caught before.
- **Requires Go 1.27.1**, matching the Docker builder. Bumped `x/text`, `x/sys`, `x/sync`
  and `codeql-action`.
- `read_text_file` no longer reopens a file for its BOM; `TOOLS.md` and `docs/PUBLISHING.md`
  corrected.

## [4.4.0] - 2026-09-02

- **`edit_file` collapsed a whole CRLF file to LF** if it held even one bare LF. Mixed now
  repairs to the dominant style. Thanks to [@igorzlatkov](https://github.com/igorzlatkov) (#24).
- **Sparse Cyrillic in a mostly-ASCII source read as a Latin codepage.** A Latin verdict is
  now checked against the high bytes.
- **A file read as UTF-8 only because detection gave up was advertised as plain UTF-8**, and
  the model sent to built-in tools that cannot read it.
- With `MCP_DETECTION_CANDIDATES` set, a file fitting none of them is reported as **ODD
  ENCODING** rather than read as the default in silence.

## [4.3.0] - 2026-08-19

- **Zero-config access was gone on Claude Code 2.1.232 and later** — those clients speak MCP
  2026-07-28, where SEP-2322 forbids the `roots/list` the plugin relied on. The server now
  derives its baseline from where it was started.
- **The plugin launcher swallowed `args`**, so directories set in `.mcp.json` never arrived.
- Added **`MCP_FILE_TOOLS_ALLOWED_DIRS`** and **`MCP_FILE_TOOLS_NO_CWD_FALLBACK`**.
- Resolution order is `args`, then the env var, then the working directory — which refuses a
  filesystem root or the home directory.

## [4.2.1] - 2026-08-15

- **`go install` silently served v1.8.1, not the current release.** The module path lacked
  the `/v4` suffix Go requires from v2 on, so the proxy ignored every tag since 2.0.0.
- **Releases carry build provenance** (`gh attestation verify`), plus `SECURITY.md` and
  weekly CodeQL and OpenSSF Scorecard runs.
- Every action pinned by commit SHA, both Docker images by digest.
- Go Report Card badge dropped — the service was sunset 2026-07-01.

## [4.2.0] - 2026-08-15

- **`MCP_DETECTION_CANDIDATES` pins what detection may answer**, in priority order. A BOM
  still wins; nothing fitting means no answer rather than a wrong one. Unset changes nothing.
- **`grep_text_files` takes `patterns`** — an array searched as one alternation, so a list of
  names is one call and one pass.
- Both reimplemented from [Mario Rial](https://github.com/seguridadea1)'s fork.

## [4.1.0] - 2026-08-14

- **Ranked encoding candidates when detection cannot decide** — `detect_encoding` returns
  `candidates`, and `read_text_file` and `convert_encoding` name them.
- **Progress notifications on a batch `convert_encoding`**, capped at 100 per call.
- **Server title, description, website and icon** (SEP-973).

## [4.0.0] - 2026-08-14

- **Built-in tool parameter names are accepted as aliases** — a call shaped like Claude
  Code's Read/Write/Edit/Grep is translated where semantics match exactly.
- **`{a,b}` glob alternatives** in search and grep patterns, which previously matched
  nothing, silently — as did `search_files` patterns with several `**`.
- **`write_file` and `edit_file` stripped the CRLFs from UTF-16 files**, and
  `grep_text_files` searched with a silently wrong decode on an unresolvable `encoding`.
- **`edit_file` no longer edits the first of several identical matches** — it fails with
  their line numbers; add context, or pass the new `replaceAll: true`.

## [3.4.1] - 2026-08-14

- **`edit_file` could stall for minutes on a failed edit** — finding the closest match was
  cubic in `oldText`. A 50-line block against a 2,000-line file: 5.6s → 4ms.
- **`read_text_file` and `grep_text_files` ignored `MCP_DEFAULT_ENCODING`** on an
  inconclusive detection. No change on the default `utf-8`.
- **`convert_encoding` converted on a detection it did not trust**, and a bad guess is
  unrecoverable. **Behaviour change:** confirm with `from`, or pass `allowLowConfidence`.

## [3.4.0] - 2026-08-14

- **MacCyrillic encoding** (`x-mac-cyrillic`) — 25 encodings total.
- **CP1251 files no longer come back garbled via a MacCyrillic guess**, and a read falling
  back to raw UTF-8 now says so.

## [3.3.0] - 2026-08-03

- **Update notices match how the server was installed**, and `check_for_updates` returns
  `installMethod`.
- **An allowed directory reached by an alias** (macOS `/var` → `/private/var`) or given as a
  Windows 8.3 short path no longer denies every path under it.
- **An empty client roots list revokes that client's earlier roots** instead of leaving them
  authorized for the life of the process. CLI args are unaffected.

## [3.2.0] - 2026-08-01

- **`edit_file` takes `similarity`** for bounded fuzzy matching and a one-file unified diff
  through `patch`; `grep_text_files` accepts `includes`/`excludes` arrays.
- **`read_text_file` takes `lineNumbers`**, off by default so a read → write round trip
  cannot bake numbers into a file.
- **Three MCP prompts**: `audit_encodings`, `fix_mojibake`, `migrate_to_utf8`.
- **`tree`, `search_files` and `grep_text_files` honour `.gitignore`.** **Behaviour change:**
  on by default; pass `respectGitignore: false` for the old behaviour.

## [3.1.0] - 2026-08-01

- **`search_files` and `list_directory` take `sortBy` and `reverse`** — with `mtime`/`size`
  the whole tree is ranked before `maxResults`.
- **`grep_text_files` gains `outputMode`**, plus `matchesOnly` and `offset`/`nextOffset`.
- **`convert_encoding` takes a batch and a dry run**, and a failed conversion names the
  characters that do not fit — character, code point, line and column.
- **`move_file` now sets `destructiveHint: true`** (a move removes the source).

## [3.0.0] - 2026-08-01

- **Removed — BREAKING: `directory_tree`** (use `tree`: same structure, ~85% fewer tokens,
  `excludePatterns` is spelled `exclude`), and `detect_line_endings`/`change_line_endings`
  merged into **`manage_line_endings`** with `action: "detect" | "convert"`.
- **`write_file` no longer leaves CRLF files with mixed line endings.** New `lineEndings`
  parameter. **Behaviour change:** no longer byte-verbatim by default — pass `"asis"`.
- **The plugin bundles a skill, `fixing-text-encodings`**, and reads and writes return a
  `hint` for mixed line endings and plain utf-8 files better served by built-in tools.
- **Upgraded to `modelcontextprotocol/go-sdk` v1.7.0** (MCP spec `2026-07-28`), capabilities
  declared explicitly as `tools` only; older clients are unaffected.

## [2.0.1] - 2026-07-27

- **Pure-ASCII files no longer silently convert to `utf-8` on edit/write** — `ascii`
  detections fall through to the configured default like any other inconclusive detection.
- **`edit_file` now honors `MCP_DEFAULT_ENCODING`** instead of hardcoding `utf-8`.

## [2.0.0] - 2026-07-26

- **Changed — BREAKING: `write_file` defaults new files to `utf-8` instead of `cp1251`.**
  Only callers creating **new** files without `encoding` are affected; existing files keep
  their detected encoding. Set `MCP_DEFAULT_ENCODING=cp1251` to restore — see
  [Legacy teams](README.md#legacy-teams-pre-200-behaviour).
- **Transitional:** the first `write_file` creating a new file under the built-in default
  appends a one-line notice about the change. **Runs to the end of 2026.**

# Changelog

Notable changes to `mcp-file-tools`. This file starts at 2.0.0; for earlier
versions see the [GitHub releases](https://github.com/dimitar-grigorov/mcp-file-tools/releases).

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **Upgraded to `modelcontextprotocol/go-sdk` v1.7.0**, which implements MCP
  spec revision `2026-07-28`. Older clients are unaffected — the SDK serves both
  the legacy `initialize` handshake and the new stateless model behind the same
  API, and falls back to `initialize` when `server/discover` fails.

- **Server capabilities are now declared explicitly** as `tools` only. Left
  unset, the SDK advertised `logging` (deprecated in `2026-07-28`) and inferred
  `listChanged: true` from the presence of tools — a capability this server does
  not support, since its tool set is static and it never emits `list_changed`.

- `readOnlyHint` and `idempotentHint` are now always present in `tools/list`
  output instead of being omitted when `false`. This is an SDK-side wire change;
  it makes "false" and "unset" distinguishable.

Roots, sampling and MCP logging are deprecated in `2026-07-28` with a 12-month
window. The roots integration is retained as-is for pre-`2026-07-28` clients.

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

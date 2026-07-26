# Changelog

Notable changes to `mcp-file-tools`. This file starts at 2.0.0; for earlier
versions see the [GitHub releases](https://github.com/dimitar-grigorov/mcp-file-tools/releases).

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

- `make lint` now runs `staticcheck` at the same pinned version as CI, so local
  lint no longer passes on code CI rejects.

### Fixed

- Repo-wide `gofmt` drift that `make lint` would otherwise rewrite unpredictably.

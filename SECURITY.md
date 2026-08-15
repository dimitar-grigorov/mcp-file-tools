# Security policy

## Supported versions

Only the latest 4.x release gets security fixes. Older majors are not patched —
upgrade instead.

| Version | Supported |
|---------|-----------|
| 4.x     | yes       |
| < 4.0   | no        |

## Reporting a vulnerability

Report privately through GitHub's
[Report a vulnerability](https://github.com/dimitar-grigorov/mcp-file-tools/security/advisories/new)
form. It reaches the maintainer without going public, and lets us discuss a fix
in a private fork. Please do not open a public issue for a vulnerability.

Expect an acknowledgement within 7 days and an assessment within 30. If a fix is
warranted it ships in the next release, and the advisory is published once the
release is out — coordinated disclosure within 90 days of the report.

Useful in a report: the version (`mcp-file-tools --version`), the allowed
directories the server was started with, and the exact tool call.

## What is in scope

This server hands an AI assistant real file operations, so the boundary that
matters is the allowed-directory containment in `internal/security`. In scope:

- **Reaching outside an allowed directory** — traversal, symlinks, Windows
  junctions, 8.3 short paths, path aliases, TOCTOU races between check and open.
- **Reading or writing a path the caller never named** — argument handling that
  resolves to somewhere else.
- **Crashing the server on untrusted file content** — a decoder panic on
  malformed bytes in any of the supported encodings.

Out of scope: anything a caller can already do inside the directories it was
granted. A client that asks to delete an allowed file gets the file deleted;
that is the tool working. Deciding what the assistant may touch is the job of
the allowed-directory list, not of the individual tools.

Reports against the launcher's download-and-verify path
(`plugin/bin/run.js`, `cmd/mcp-file-tools-launcher`) are in scope too — that
code fetches a binary and checks its SHA-256 before running it.

---
name: fixing-text-encodings
description: Diagnoses mojibake and audits or bulk-converts non-UTF-8 files with the file-tools MCP server. Use when text looks garbled (Cyrillic as Íàñòðîéêè, Ð/Ñ sequences, ????, �), when surveying or migrating a legacy codebase (Delphi/Pascal, VB6, legacy PHP/HTML; CP1251, CP1252, KOI8-R, GBK), when converting several files between encodings, or when a BOM or mixed CRLF/LF breaks a build or diff. Not for ordinary UTF-8 files; the built-in tools handle those.
---

# Fixing text encodings

The `file-tools` server auto-detects encoding on read and writes back in the file's
original encoding, BOM and line endings intact. Reach for this skill for the
multi-file and diagnostic work that a single tool call does not cover.

## Diagnosing garbled text

Read the symptom, then act. Do not guess encodings by trial and error.

| Symptom | Cause | Action |
|---|---|---|
| `Íàñòðîéêè` — Cyrillic as Western letters | CP1251 bytes decoded as CP1252/Latin-1 | Read it with `encoding: "cp1251"` |
| `Ð`, `Ñ`, `â€™` sequences | UTF-8 bytes decoded as a single-byte charset | The file is already UTF-8; read it as UTF-8 |
| `?` where letters should be | Characters were **already lost** on a previous save to a charset that lacks them | Not recoverable — restore from version control |
| `�` (U+FFFD) | Byte sequence invalid for the assumed encoding | Run `file-tools:detect_encoding` and use what it reports |

`file-tools:detect_encoding` returns a confidence score. Below ~50 the file is
probably not text in a supported encoding, or is too short to judge — say so rather
than converting it anyway.

## Surveying a legacy project

Before touching anything, get the shape of the problem:

```
file-tools:tree  { "path": "D:/proj/src", "showEncoding": true }
```

This reports the detected encoding per file. Use it to decide whether a codebase is
uniform (one bulk conversion) or mixed (convert per group, and expect some files to
be pure ASCII and therefore inconclusive).

## Bulk conversion

Converting is irreversible for characters the target lacks. Work through this list:

```
- [ ] 1. Survey with file-tools:tree, showEncoding=true
- [ ] 2. Convert ONE representative file, backup=true
- [ ] 3. Read it back and confirm the text is correct
- [ ] 4. Convert the rest in one call with `paths`, dryRun=true first, then backup=true
- [ ] 5. Re-run the survey to confirm nothing was missed
```

**Step 2 — prove it on one file first:**

```
file-tools:convert_encoding  { "path": "D:/proj/src/unit1.pas", "to": "utf-8", "backup": true }
```

Omit `from`; it is auto-detected. Below 50% confidence the tool refuses and names its
guess: read the file with that encoding, then confirm it with `from`. A narrowing
conversion (for example UTF-8 to CP1251) **fails rather than corrupting text** when the
target lacks a character, so a failure here is information, not a setback. Report which
files failed and why.

**Step 3 — verify before scaling up.** Read the converted file and check the
non-ASCII text actually reads correctly. Detection confidence is not proof.

Keep `backup=true` for every step. The `.bak` files are the only undo; an existing one
is never overwritten, so the tool refuses that file instead.

## Line endings and BOM

- A UTF-8 BOM breaks PHP and shell scripts: `file-tools:manage_bom` with `action: "strip"`.
- UTF-16 files need their BOM — do not strip it.
- Mixed CRLF/LF: `file-tools:manage_line_endings` with `action: "convert"`.

`file-tools:read_text_file` reports mixed line endings in its `hint` field. Surface
that to the user rather than silently normalising a file they did not ask you to touch.

## What is already handled

Do not write code or extra steps for these:

- Encoding is detected on read and preserved on write and edit.
- `file-tools:write_file` matches the file's existing line endings, so sending LF into
  a CRLF file will not leave it mixed.
- Converting a file that already holds the target bytes is a no-op.

## Legacy teams

If a project should default to a legacy encoding for **new** files, set
`MCP_DEFAULT_ENCODING` (for example `cp1251`) in the project's `.mcp.json` rather
than passing `encoding` on every call. Existing files keep their own encoding either way.

This matters most for Delphi 2007 and older, which read UTF-8 only when it carries a BOM:
a UTF-8 file without one is silently treated as ANSI. Adding a Cyrillic literal to a
previously ASCII-only unit is the one case that can change a file's encoding.

# Extra details

Longer material kept out of the [README](../README.md) so it stays short. Anything that
grows past a few lines there lands here.

- [Auto-approving tools in Claude Code](#auto-approving-tools-in-claude-code)

## Auto-approving tools in Claude Code

Skipping the permission prompt for this server's tools. Rules go in one of:

| File | Applies to | Committed |
|---|---|---|
| `.claude/settings.local.json` | you, this project | no — Claude Code gitignores it |
| `.claude/settings.json` | everyone on this project | yes |
| `~/.claude/settings.json` | all your projects | — |

`/permissions` edits them interactively, and **Always allow** on a prompt writes to the
local file.

### Allow everything

```json
{
  "permissions": {
    "allow": ["mcp__file-tools__*"]
  }
}
```

`mcp__file-tools`, without the tool part, means the same thing.

That includes `delete_file` and `move_file`. To keep those two behind a prompt — `ask`
and `deny` both win over `allow`:

```json
{
  "permissions": {
    "allow": ["mcp__file-tools__*"],
    "ask": ["mcp__file-tools__delete_file", "mcp__file-tools__move_file"]
  }
}
```

### Get the server name right

The prefix is the server name as your client registered it, and the plugin install is not
`file-tools`:

| Install | Prefix |
|---|---|
| `claude mcp add file-tools ...` | `mcp__file-tools__` |
| Claude Code plugin | `mcp__plugin_mcp-file-tools_file-tools__` |

A rule with the wrong prefix matches nothing and fails quietly — you just keep getting
prompts. The exact name is in the permission prompt itself and in `/mcp`.

### What the patterns allow

- An `allow` rule takes a glob only in the tool position, after a literal
  `mcp__<server>__`: `mcp__file-tools__*` and `mcp__file-tools__read_*` are fine,
  `mcp__file*__read_text_file` is not.
- `ask` and `deny` rules accept wildcards anywhere.
- No parentheses. The `Bash(git *)` syntax does not carry over to MCP tools.

If your Claude Code version rejects the wildcard, name the tools explicitly.

### Naming every tool

Equivalent to the wildcard plus the `ask` rules above — the two destructive tools are
left out, so Claude asks before them:

```json
{
  "permissions": {
    "allow": [
      "mcp__file-tools__read_text_file",
      "mcp__file-tools__read_multiple_files",
      "mcp__file-tools__write_file",
      "mcp__file-tools__edit_file",
      "mcp__file-tools__copy_file",
      "mcp__file-tools__list_directory",
      "mcp__file-tools__tree",
      "mcp__file-tools__search_files",
      "mcp__file-tools__grep_text_files",
      "mcp__file-tools__detect_encoding",
      "mcp__file-tools__convert_encoding",
      "mcp__file-tools__manage_line_endings",
      "mcp__file-tools__manage_bom",
      "mcp__file-tools__list_encodings",
      "mcp__file-tools__get_file_info",
      "mcp__file-tools__create_directory",
      "mcp__file-tools__list_allowed_directories",
      "mcp__file-tools__check_for_updates"
    ]
  }
}
```

### Built-in tools too

Unrelated to this server, but this is what most people add alongside it — read-only shell
commands and search, with `WebFetch` left out on purpose:

```json
{
  "permissions": {
    "allow": [
      "Bash(ls *)",
      "Bash(grep *)",
      "Bash(sort *)",
      "Bash(wc *)",
      "Bash(find *)",
      "Bash(echo *)",
      "Grep",
      "Glob",
      "WebSearch"
    ]
  }
}
```

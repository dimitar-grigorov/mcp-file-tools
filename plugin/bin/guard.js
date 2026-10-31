#!/usr/bin/env node
// PreToolUse hook: stops Claude Code's built-in Read/Edit/Write on a file that is not
// UTF-8 and points the model at this plugin's tools. Read would show U+FFFD and
// Edit/Write refuse such files, so the model otherwise falls back to shell scripts.
// Fails open: any error, a binary file or a path outside the workspace lets the call through.
// MCP_FILE_TOOLS_GUARD=off disables it.

const fs = require('fs');
const os = require('os');
const path = require('path');

const SAMPLE = 64 * 1024;
const PREFIX = 'mcp__plugin_mcp-file-tools_file-tools__';
const SKIP_EXT = new Set(['.png', '.jpg', '.jpeg', '.gif', '.webp', '.bmp', '.ico', '.pdf', '.ipynb']);
const TOOLS = { Read: 'read_text_file', Edit: 'edit_file', MultiEdit: 'edit_file', Write: 'write_file' };

function inside(file, dir) {
  const rel = path.relative(path.resolve(dir), file);
  return rel === '' || (!rel.startsWith('..') && !path.isAbsolute(rel));
}

// Mirrors internal/config/baseline.go: the env list replaces the working directory,
// which the server refuses when it is a drive root or the home folder.
function allowedDirs(cwd, env) {
  const listed = (env.MCP_FILE_TOOLS_ALLOWED_DIRS || '').split(path.delimiter).filter((d) => d.trim());
  if (listed.length) return listed;
  const noCwd = !/^(|0|false|no|off)$/i.test((env.MCP_FILE_TOOLS_NO_CWD_FALLBACK || '').trim());
  if (noCwd || !cwd) return [];
  const dir = path.resolve(cwd);
  if (dir === path.parse(dir).root || dir === path.resolve(os.homedir())) return [];
  return [dir];
}

// True when the first SAMPLE bytes are text that is not UTF-8 and has no BOM.
function isLegacyText(file) {
  const fd = fs.openSync(file, 'r');
  let buf;
  try {
    buf = Buffer.alloc(SAMPLE);
    buf = buf.subarray(0, fs.readSync(fd, buf, 0, SAMPLE, 0));
  } finally {
    fs.closeSync(fd);
  }
  if (buf.length === 0 || buf.includes(0)) return false;
  if (buf[0] === 0xef && buf[1] === 0xbb && buf[2] === 0xbf) return false;
  if ((buf[0] === 0xff && buf[1] === 0xfe) || (buf[0] === 0xfe && buf[1] === 0xff)) return false;
  try {
    // stream: a multi-byte character cut at the sample edge is not an error.
    new TextDecoder('utf-8', { fatal: true }).decode(buf, { stream: true });
    return false;
  } catch {
    return true;
  }
}

function decide(input, env = process.env) {
  if ((env.MCP_FILE_TOOLS_GUARD || '').toLowerCase() === 'off') return null;
  const tool = TOOLS[input.tool_name];
  const raw = input.tool_input && input.tool_input.file_path;
  if (!tool || typeof raw !== 'string') return null;

  const file = path.resolve(input.cwd || process.cwd(), raw);
  if (SKIP_EXT.has(path.extname(file).toLowerCase())) return null;
  // Only where the server can reach the file, or the model would be sent to a dead end.
  if (!allowedDirs(input.cwd, env).some((d) => inside(file, d))) return null;
  if (!fs.existsSync(file) || !fs.statSync(file).isFile() || !isLegacyText(file)) return null;

  return {
    hookSpecificOutput: {
      hookEventName: 'PreToolUse',
      permissionDecision: 'deny',
      permissionDecisionReason:
        `${raw} is not UTF-8 (a legacy encoding such as CP1251 or CP1252). ` +
        `Use ${PREFIX}${tool} instead: it detects the encoding and keeps it, the BOM and line endings. ` +
        'Do not convert the file or edit it with a shell command.',
    },
  };
}

if (require.main === module) {
  let data = '';
  process.stdin.setEncoding('utf8');
  process.stdin.on('data', (c) => (data += c));
  process.stdin.on('end', () => {
    try {
      const out = decide(JSON.parse(data));
      if (out) process.stdout.write(JSON.stringify(out));
    } catch {
      // Fail open.
    }
  });
}

module.exports = { decide, isLegacyText };

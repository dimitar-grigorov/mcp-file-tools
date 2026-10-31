const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');

const { decide } = require('../plugin/bin/guard');

function workspace(t, files) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'mcp-file-tools-guard-'));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  for (const [name, bytes] of Object.entries(files)) fs.writeFileSync(path.join(dir, name), bytes);
  return dir;
}

const cp1251 = Buffer.from([0xcf, 0xf0, 0xe8, 0xe2, 0xe5, 0xf2, 0x0d, 0x0a]); // "Привет\r\n"

function denied(out) {
  return out && out.hookSpecificOutput.permissionDecision === 'deny';
}

test('denies Read, Edit and Write on a CP1251 file in the workspace', (t) => {
  const cwd = workspace(t, { 'a.pas': cp1251 });
  for (const tool of ['Read', 'Edit', 'Write']) {
    const out = decide({ tool_name: tool, tool_input: { file_path: 'a.pas' }, cwd }, {});
    assert.ok(denied(out), tool);
  }
  const out = decide({ tool_name: 'Read', tool_input: { file_path: 'a.pas' }, cwd }, {});
  assert.match(out.hookSpecificOutput.permissionDecisionReason, /read_text_file/);
});

test('lets UTF-8, BOM, binary, new and skipped files through', (t) => {
  const cwd = workspace(t, {
    'u.txt': 'Привет',
    'bom.txt': Buffer.concat([Buffer.from([0xef, 0xbb, 0xbf]), cp1251]),
    'u16.txt': Buffer.from([0xff, 0xfe, 0x41, 0x00]),
    'bin.dat': Buffer.from([0xcf, 0x00, 0xf0]),
    'img.png': cp1251,
    'empty.txt': '',
  });
  for (const f of ['u.txt', 'bom.txt', 'u16.txt', 'bin.dat', 'img.png', 'empty.txt', 'new.txt']) {
    assert.equal(decide({ tool_name: 'Read', tool_input: { file_path: f }, cwd }, {}), null, f);
  }
});

test('a UTF-8 character cut at the sample edge is not legacy', (t) => {
  const big = Buffer.concat([Buffer.alloc(64 * 1024 - 1, 0x61), Buffer.from('Ж')]);
  const cwd = workspace(t, { 'big.txt': big });
  assert.equal(decide({ tool_name: 'Read', tool_input: { file_path: 'big.txt' }, cwd }, {}), null);
});

test('outside the workspace, other tools and the off switch pass', (t) => {
  const other = workspace(t, { 'a.pas': cp1251 });
  const cwd = workspace(t, { 'a.pas': cp1251 });
  const outside = path.join(other, 'a.pas');
  assert.equal(decide({ tool_name: 'Read', tool_input: { file_path: outside }, cwd }, {}), null);
  const env = { MCP_FILE_TOOLS_ALLOWED_DIRS: other };
  assert.ok(denied(decide({ tool_name: 'Read', tool_input: { file_path: outside }, cwd }, env)));
  // The env list replaces the working directory, as in the server.
  assert.equal(decide({ tool_name: 'Read', tool_input: { file_path: 'a.pas' }, cwd }, env), null);
  const noCwd = { MCP_FILE_TOOLS_NO_CWD_FALLBACK: '1' };
  assert.equal(decide({ tool_name: 'Read', tool_input: { file_path: 'a.pas' }, cwd }, noCwd), null);
  assert.equal(decide({ tool_name: 'Bash', tool_input: { command: 'cat a.pas' }, cwd }, {}), null);
  assert.equal(
    decide({ tool_name: 'Read', tool_input: { file_path: 'a.pas' }, cwd }, { MCP_FILE_TOOLS_GUARD: 'off' }),
    null,
  );
});

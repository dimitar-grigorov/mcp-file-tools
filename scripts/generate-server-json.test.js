const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');
const { spawnSync } = require('node:child_process');

const root = path.resolve(__dirname, '..');
const generator = path.join(root, 'scripts', 'generate-server-json.js');
const templatePath = path.join(root, 'server.template.json');

const binaries = [
  'mcp-file-tools_windows_amd64.exe',
  'mcp-file-tools_windows_arm64.exe',
  'mcp-file-tools_linux_amd64',
  'mcp-file-tools_linux_arm64',
  'mcp-file-tools_darwin_amd64',
  'mcp-file-tools_darwin_arm64',
];

function workspace(t) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'mcp-file-tools-registry-'));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  return dir;
}

function sha(index) {
  return (index + 1).toString(16).repeat(64).slice(0, 64);
}

// Mirrors a real GoReleaser checksums.txt: raw binaries plus archives.
function writeChecksums(file, names) {
  const lines = names.map((name, i) => `${sha(i)}  ${name}`);
  lines.push(`${'f'.repeat(64)}  mcp-file-tools_windows_amd64.zip`);
  lines.push(`${'e'.repeat(64)}  mcp-file-tools_linux_amd64.tar.gz`);
  fs.writeFileSync(file, `${lines.join('\n')}\n`, 'utf8');
}

function run(t, args, names = binaries) {
  const dir = workspace(t);
  const checksums = path.join(dir, 'checksums.txt');
  const output = path.join(dir, 'server.json');
  writeChecksums(checksums, names);
  const result = spawnSync(process.execPath, [generator, ...args, checksums, output], {
    cwd: root,
    encoding: 'utf8',
  });
  return { result, output };
}

test('generates a manifest from the template and release checksums', (t) => {
  const { result, output } = run(t, ['v2.3.4']);
  assert.equal(result.status, 0, result.stderr);

  const manifest = JSON.parse(fs.readFileSync(output, 'utf8'));
  const template = JSON.parse(fs.readFileSync(templatePath, 'utf8'));

  assert.equal(manifest.name, 'io.github.dimitar-grigorov/mcp-file-tools');
  assert.equal(manifest.version, '2.3.4');
  assert.equal(manifest.repository.url, 'https://github.com/dimitar-grigorov/mcp-file-tools');
  assert.equal(manifest.packages.length, binaries.length);
  manifest.packages.forEach((pkg, i) => {
    assert.equal(
      pkg.identifier,
      `https://github.com/dimitar-grigorov/mcp-file-tools/releases/download/v2.3.4/${binaries[i]}`,
    );
    assert.equal(pkg.fileSha256, sha(i));
  });
  // Tools are hand-maintained in the template and copied through untouched.
  assert.deepEqual(manifest.tools, template.tools);
});

test('accepts a bare version without the v prefix', (t) => {
  const { result, output } = run(t, ['2.3.4']);
  assert.equal(result.status, 0, result.stderr);
  assert.equal(JSON.parse(fs.readFileSync(output, 'utf8')).version, '2.3.4');
});

test('fails when a release binary checksum is missing', (t) => {
  const { result, output } = run(t, ['v2.3.4'], binaries.slice(0, 3));
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /missing release checksum/);
  assert.equal(fs.existsSync(output), false);
});

test('fails when the release ships a binary the template does not list', (t) => {
  const { result, output } = run(t, ['v2.3.4'], [...binaries, 'mcp-file-tools_linux_riscv64']);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /unrepresented MCP binaries: mcp-file-tools_linux_riscv64/);
  assert.equal(fs.existsSync(output), false);
});

test('rejects the placeholder version and malformed versions', (t) => {
  for (const bad of ['0.0.0', 'v0.0.0', 'latest', 'v1.2', '']) {
    const { result, output } = run(t, [bad]);
    assert.notEqual(result.status, 0, `expected ${JSON.stringify(bad)} to be rejected`);
    assert.equal(fs.existsSync(output), false);
  }
});

test('rejects a malformed checksums file', (t) => {
  const dir = workspace(t);
  const checksums = path.join(dir, 'checksums.txt');
  const output = path.join(dir, 'server.json');
  fs.writeFileSync(checksums, 'not-a-checksum  mcp-file-tools_linux_amd64\n', 'utf8');
  const result = spawnSync(process.execPath, [generator, 'v2.3.4', checksums, output], {
    cwd: root,
    encoding: 'utf8',
  });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /invalid checksums line/);
});

test('the committed template is still a placeholder', () => {
  const template = JSON.parse(fs.readFileSync(templatePath, 'utf8'));
  assert.equal(template.version, '0.0.0');
  for (const pkg of template.packages) {
    assert.equal(pkg.fileSha256, '0'.repeat(64));
    assert.match(pkg.identifier, /\/download\/v0\.0\.0\//);
  }
});

// The template's tool list is hand-maintained, so guard it against server.go.
test('the template lists exactly the registered tools', () => {
  const source = fs.readFileSync(path.join(root, 'filetoolsserver', 'server.go'), 'utf8');
  const registered = [...source.matchAll(/handler\.Wrap(?:ContentOnly)?\(logger, "([a-z_]+)"/g)]
    .map((m) => m[1])
    .sort();
  assert.ok(registered.length > 0, 'found no tool registrations in server.go');

  const template = JSON.parse(fs.readFileSync(templatePath, 'utf8'));
  assert.deepEqual([...template.tools.map((tool) => tool.name)].sort(), registered);
});

const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');

const { verifyReleaseVersion } = require('./verify-release-version');

// Throwaway repo root with just the two version files.
function fixture(t, pluginVersion, marketplaceVersion) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'mcp-file-tools-version-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));

  fs.mkdirSync(path.join(root, 'plugin', '.claude-plugin'), { recursive: true });
  fs.mkdirSync(path.join(root, '.claude-plugin'), { recursive: true });
  fs.writeFileSync(
    path.join(root, 'plugin', '.claude-plugin', 'plugin.json'),
    JSON.stringify({ name: 'mcp-file-tools', version: pluginVersion }),
  );
  fs.writeFileSync(
    path.join(root, '.claude-plugin', 'marketplace.json'),
    JSON.stringify({ plugins: [{ name: 'mcp-file-tools', version: marketplaceVersion }] }),
  );
  return root;
}

test('this repository is internally consistent', () => {
  const root = path.resolve(__dirname, '..');
  const plugin = JSON.parse(
    fs.readFileSync(path.join(root, 'plugin', '.claude-plugin', 'plugin.json'), 'utf8'),
  );
  assert.deepEqual(verifyReleaseVersion(`v${plugin.version}`, root), {
    version: plugin.version,
    pluginVersion: plugin.version,
    marketplaceVersion: plugin.version,
  });
});

test('accepts a v-prefixed tag matching both files', (t) => {
  const root = fixture(t, '1.8.0', '1.8.0');
  assert.deepEqual(verifyReleaseVersion('v1.8.0', root), {
    version: '1.8.0',
    pluginVersion: '1.8.0',
    marketplaceVersion: '1.8.0',
  });
});

test('rejects a tag that does not match plugin.json', (t) => {
  const root = fixture(t, '1.8.0', '1.8.0');
  assert.throws(
    () => verifyReleaseVersion('v1.8.1', root),
    /tag version 1\.8\.1 does not match plugin version 1\.8\.0/,
  );
});

test('rejects plugin/marketplace drift', (t) => {
  const root = fixture(t, '1.8.0', '1.7.3');
  assert.throws(
    () => verifyReleaseVersion('v1.8.0', root),
    /marketplace version 1\.7\.3 does not match plugin version 1\.8\.0/,
  );
});

test('rejects a malformed tag', (t) => {
  const root = fixture(t, '1.8.0', '1.8.0');
  assert.throws(() => verifyReleaseVersion('release-1.8.0', root), /release tag/);
  assert.throws(() => verifyReleaseVersion('v1.8', root), /release tag/);
  assert.throws(() => verifyReleaseVersion(undefined, root), /release tag/);
});

test('rejects a non-semver version in plugin.json', (t) => {
  const root = fixture(t, '1.8', '1.8');
  assert.throws(() => verifyReleaseVersion('v1.8.0', root), /plugin version must be/);
});

test('rejects a marketplace file without our plugin entry', (t) => {
  const root = fixture(t, '1.8.0', '1.8.0');
  fs.writeFileSync(
    path.join(root, '.claude-plugin', 'marketplace.json'),
    JSON.stringify({ plugins: [{ name: 'something-else', version: '1.8.0' }] }),
  );
  assert.throws(() => verifyReleaseVersion('v1.8.0', root), /no "mcp-file-tools" plugin entry/);
});

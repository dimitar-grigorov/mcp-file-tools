const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const { extractPins, compareVersions, problems } = require('./check-pinned-tools');

const byName = (pins, name) => pins.find((t) => t.name === name);

test('this repository pins every tool, and the same version everywhere', () => {
  const dir = path.resolve(__dirname, '..', '.github', 'workflows');
  const files = {};
  for (const f of fs.readdirSync(dir)) files[f] = fs.readFileSync(path.join(dir, f), 'utf8');
  for (const tool of extractPins(files)) {
    const highest = tool.pins.map((p) => p.version).sort(compareVersions).at(-1) ?? '0';
    assert.deepEqual(problems(tool, highest), []);
  }
});

test('versions compare numerically', () => {
  assert.equal(compareVersions('2.18.10', '2.18.9'), 1);
  assert.equal(compareVersions('1.8', '1.8.0'), 0);
  assert.equal(compareVersions('1.7.9', '1.8.0'), -1);
});

test('a stale pin, a mismatch and a missing pin are reported', () => {
  const pins = extractPins({
    'a.yml': "distribution: goreleaser\n          version: 'v2.18.2'\n",
    'b.yml': "distribution: goreleaser\n          version: 'v2.18.3'\n",
  });
  assert.deepEqual(problems(byName(pins, 'GoReleaser'), '2.18.3'), [
    'GoReleaser: pinned differently in a.yml (2.18.2), b.yml (2.18.3)',
    'GoReleaser: 2.18.2 pinned, 2.18.3 released',
  ]);
  assert.match(problems(byName(pins, 'mcp-publisher'), '1.8.1')[0], /no pin found/);
});

#!/usr/bin/env node
// Bump the release version from one command:
//   node scripts/bump-version.js 1.7.4
// plugin.json is the source of truth. run.js reads it at runtime, and server.json
// is generated in CI from server.template.json, so neither is touched here --
// only plugin.json and the marketplace catalog entry.

const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..');

// Each target: file + a regex whose first capture group is the version to replace.
const targets = [
  { file: 'plugin/.claude-plugin/plugin.json',      re: /("version":\s*")([^"]+)(")/ },
  { file: '.claude-plugin/marketplace.json',        re: /("version":\s*")([^"]+)(")/ },
];

function fail(msg) {
  console.error(`error: ${msg}`);
  process.exit(1);
}

const version = (process.argv[2] || '').replace(/^v/, '');
if (!/^\d+\.\d+\.\d+$/.test(version)) {
  fail(`usage: node scripts/bump-version.js <major.minor.patch>  (got "${process.argv[2] || ''}")`);
}

let changed = 0;
for (const { file, re } of targets) {
  const full = path.join(root, file);
  const before = fs.readFileSync(full, 'utf8');
  const m = before.match(re);
  if (!m) fail(`could not find a version to replace in ${file}`);
  const after = before.replace(re, `$1${version}$3`);
  if (after !== before) {
    fs.writeFileSync(full, after);
    console.log(`  ${file}: ${m[2]} -> ${version}`);
    changed++;
  } else {
    console.log(`  ${file}: already ${version}`);
  }
}

console.log(`\nbumped ${changed} file(s) to ${version}. Next:`);
console.log(`  git commit -am "chore: release ${version} (bump plugin.json, marketplace.json)"`);
console.log(`  node scripts/verify-release-version.js v${version}`);
console.log(`  git tag v${version} && git push origin main v${version}`);

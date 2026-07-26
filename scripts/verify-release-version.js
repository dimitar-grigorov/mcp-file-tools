#!/usr/bin/env node
// Assert a release tag matches plugin.json (source of truth) and the marketplace entry:
//   node scripts/verify-release-version.js v1.7.4

const fs = require('fs');
const path = require('path');

const semver = /^\d+\.\d+\.\d+$/;
const releaseTag = /^v(\d+\.\d+\.\d+)$/;
const pluginName = 'mcp-file-tools';

function readJSON(file) {
  let raw;
  try {
    raw = fs.readFileSync(file, 'utf8');
  } catch (err) {
    throw new Error(`could not read ${file}: ${err.message}`);
  }
  try {
    return JSON.parse(raw);
  } catch (err) {
    throw new Error(`invalid JSON in ${file}: ${err.message}`);
  }
}

function requireSemver(value, label) {
  if (typeof value !== 'string' || !semver.test(value)) {
    throw new Error(`${label} must be major.minor.patch, got ${JSON.stringify(value ?? null)}`);
  }
  return value;
}

function verifyReleaseVersion(tag, root = path.resolve(__dirname, '..')) {
  const match = releaseTag.exec(tag || '');
  if (!match) {
    throw new Error(`expected a v<major.minor.patch> release tag, got ${JSON.stringify(tag || '')}`);
  }
  const version = match[1];

  const plugin = readJSON(path.join(root, 'plugin', '.claude-plugin', 'plugin.json'));
  const marketplace = readJSON(path.join(root, '.claude-plugin', 'marketplace.json'));

  const pluginVersion = requireSemver(plugin.version, 'plugin version');
  const entry = Array.isArray(marketplace.plugins)
    ? marketplace.plugins.find((p) => p && p.name === pluginName)
    : undefined;
  if (!entry) {
    throw new Error(`marketplace.json has no "${pluginName}" plugin entry`);
  }
  const marketplaceVersion = requireSemver(entry.version, 'marketplace version');

  if (marketplaceVersion !== pluginVersion) {
    throw new Error(
      `marketplace version ${marketplaceVersion} does not match plugin version ${pluginVersion}`,
    );
  }
  if (version !== pluginVersion) {
    throw new Error(`tag version ${version} does not match plugin version ${pluginVersion}`);
  }

  return { version, pluginVersion, marketplaceVersion };
}

if (require.main === module) {
  try {
    console.log(`release version verified: v${verifyReleaseVersion(process.argv[2]).version}`);
  } catch (err) {
    console.error(`error: ${err.message}`);
    process.exit(1);
  }
}

module.exports = { verifyReleaseVersion };

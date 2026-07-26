#!/usr/bin/env node
// Build the registry manifest at publish time, so it can never go stale:
//   node scripts/generate-server-json.js v1.7.4 checksums.txt server.json

const fs = require('fs');
const path = require('path');

const repository = 'https://github.com/dimitar-grigorov/mcp-file-tools';
const registryName = 'io.github.dimitar-grigorov/mcp-file-tools';
const zeroSha256 = '0'.repeat(64);
const templateVersion = '0.0.0';

// Raw binaries only -- the .tar.gz/.zip archives fail the anchored end.
// Deliberately os/arch-agnostic, so a newly built platform is caught, not skipped.
const rawBinary = /^mcp-file-tools_[a-z0-9]+_[a-z0-9]+(?:\.exe)?$/;

function fail(msg) {
  console.error(`error: ${msg}`);
  process.exit(1);
}

const versionArg = process.argv[2] || '';
const version = versionArg.replace(/^v/, '');
if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(version)) {
  fail(`invalid release version "${versionArg}"`);
}
if (version === templateVersion) {
  fail(`refusing to generate a manifest for the placeholder version ${templateVersion}`);
}

const root = path.resolve(__dirname, '..');
const templatePath = path.join(root, 'server.template.json');
const checksumsPath = path.resolve(process.cwd(), process.argv[3] || 'checksums.txt');
const outputPath = path.resolve(process.cwd(), process.argv[4] || 'server.json');

const manifest = JSON.parse(fs.readFileSync(templatePath, 'utf8'));

// The template must stay a template: our ids, placeholder version, no real hashes.
if (manifest.name !== registryName) {
  fail(`template name must be ${registryName}`);
}
if (manifest.repository?.url !== repository || manifest.homepage !== repository) {
  fail(`template repository metadata must target ${repository}`);
}
if (manifest.version !== templateVersion) {
  fail(`template version must be the ${templateVersion} placeholder, got "${manifest.version}"`);
}
if (!Array.isArray(manifest.packages) || manifest.packages.length === 0) {
  fail('template must contain at least one package');
}
if (!Array.isArray(manifest.tools) || manifest.tools.length === 0) {
  fail('template must list the tools inline');
}

const toolNames = new Set();
for (const tool of manifest.tools) {
  if (typeof tool.name !== 'string' || tool.name.trim() === '') {
    fail('template contains a tool with no name');
  }
  if (typeof tool.description !== 'string' || tool.description.trim() === '') {
    fail(`template tool ${tool.name} has no description`);
  }
  if (toolNames.has(tool.name)) {
    fail(`template lists ${tool.name} twice`);
  }
  toolNames.add(tool.name);
}

const checksums = new Map();
for (const rawLine of fs.readFileSync(checksumsPath, 'utf8').split(/\r?\n/)) {
  const line = rawLine.trim();
  if (!line) continue;
  const match = line.match(/^([0-9a-fA-F]{64})\s+\*?(.+)$/);
  if (!match) fail(`invalid checksums line: ${rawLine}`);
  const filename = path.posix.basename(match[2].trim().replace(/\\/g, '/'));
  if (checksums.has(filename)) fail(`duplicate checksum entry for ${filename}`);
  checksums.set(filename, match[1].toLowerCase());
}

manifest.version = version;
const seen = new Set();
for (const pkg of manifest.packages) {
  let filename;
  try {
    filename = path.posix.basename(new URL(pkg.identifier).pathname);
  } catch {
    fail(`invalid package identifier in template: ${pkg.identifier}`);
  }
  if (seen.has(filename)) fail(`duplicate package entry for ${filename}`);
  seen.add(filename);

  if (pkg.fileSha256 !== zeroSha256) {
    fail(`template checksum for ${filename} must be the all-zero placeholder`);
  }
  const checksum = checksums.get(filename);
  if (!checksum || checksum === zeroSha256) {
    fail(`missing release checksum for ${filename}`);
  }
  pkg.identifier = `${repository}/releases/download/v${version}/${filename}`;
  pkg.fileSha256 = checksum;
}

const unrepresented = [...checksums.keys()].filter((f) => rawBinary.test(f) && !seen.has(f));
if (unrepresented.length > 0) {
  fail(`release contains unrepresented MCP binaries: ${unrepresented.join(', ')}`);
}

const tempPath = `${outputPath}.${process.pid}.tmp`;
try {
  fs.writeFileSync(tempPath, `${JSON.stringify(manifest, null, 2)}\n`, 'utf8');
  fs.renameSync(tempPath, outputPath);
} finally {
  if (fs.existsSync(tempPath)) fs.unlinkSync(tempPath);
}

console.log(`generated ${outputPath} for ${registryName} v${version}`);

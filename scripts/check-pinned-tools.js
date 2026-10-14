#!/usr/bin/env node
// Report tool versions pinned in workflows that Dependabot cannot see, and fail when one is behind
// its latest GitHub release or two workflows pin it differently:
//   node scripts/check-pinned-tools.js

const fs = require('fs');
const path = require('path');

const tools = [
  { name: 'GoReleaser', repo: 'goreleaser/goreleaser', pattern: /distribution: goreleaser\s+version: '?v?([\d.]+)'?/g },
  { name: 'mcp-publisher', repo: 'modelcontextprotocol/registry', pattern: /MCP_PUBLISHER_VERSION: '?v?([\d.]+)'?/g },
];

// Every pinned version per tool, keyed by the workflow file it appears in.
function extractPins(files) {
  return tools.map((tool) => {
    const pins = [];
    for (const [file, text] of Object.entries(files)) {
      for (const m of text.matchAll(tool.pattern)) pins.push({ file, version: m[1] });
    }
    return { ...tool, pins };
  });
}

function compareVersions(a, b) {
  const pa = a.split('.').map(Number);
  const pb = b.split('.').map(Number);
  for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
    const d = (pa[i] ?? 0) - (pb[i] ?? 0);
    if (d !== 0) return Math.sign(d);
  }
  return 0;
}

// Problems for one tool, given its latest released version.
function problems(tool, latest) {
  const out = [];
  if (tool.pins.length === 0) return [`${tool.name}: no pin found, the pattern is out of date`];
  const versions = [...new Set(tool.pins.map((p) => p.version))];
  if (versions.length > 1) {
    out.push(`${tool.name}: pinned differently in ${tool.pins.map((p) => `${p.file} (${p.version})`).join(', ')}`);
  }
  for (const v of versions) {
    if (compareVersions(v, latest) < 0) out.push(`${tool.name}: ${v} pinned, ${latest} released`);
  }
  return out;
}

async function latestRelease(repo) {
  const headers = { Accept: 'application/vnd.github+json' };
  if (process.env.GITHUB_TOKEN) headers.Authorization = `Bearer ${process.env.GITHUB_TOKEN}`;
  const res = await fetch(`https://api.github.com/repos/${repo}/releases/latest`, { headers });
  if (!res.ok) throw new Error(`${repo}: GitHub API ${res.status}`);
  return (await res.json()).tag_name.replace(/^v/, '');
}

async function main() {
  const dir = path.resolve(__dirname, '..', '.github', 'workflows');
  const files = {};
  for (const f of fs.readdirSync(dir).filter((f) => /\.ya?ml$/.test(f))) {
    files[f] = fs.readFileSync(path.join(dir, f), 'utf8');
  }
  let failed = false;
  for (const tool of extractPins(files)) {
    const latest = await latestRelease(tool.repo);
    const found = problems(tool, latest);
    if (found.length === 0) console.log(`${tool.name}: ${latest}, up to date`);
    for (const p of found) {
      console.log(`::error::${p}`);
      failed = true;
    }
  }
  if (failed) process.exit(1);
}

if (require.main === module) {
  main().catch((err) => {
    console.error(`error: ${err.message}`);
    process.exit(1);
  });
}

module.exports = { extractPins, compareVersions, problems };

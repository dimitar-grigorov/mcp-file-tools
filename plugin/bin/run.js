#!/usr/bin/env node
// Launcher for the mcp-file-tools MCP server. Downloads the pinned release binary
// on first run, verifies its SHA-256, caches it, then hands stdio to it.
// Written in Node so it runs the same on Windows/macOS/Linux and avoids the
// "bash resolves to WSL" problem when Claude Code spawns the server without a shell.
//
// Logs go to stderr only; stdout is the MCP channel and must stay clean.

const fs = require('fs');
const os = require('os');
const path = require('path');
const crypto = require('crypto');
const { spawn } = require('child_process');

const REPO = 'dimitar-grigorov/mcp-file-tools';
// Single source of truth: plugin.json (shipped alongside this launcher).
const VERSION = 'v' + require('../.claude-plugin/plugin.json').version;

const OS = { win32: 'windows', darwin: 'darwin', linux: 'linux' }[process.platform];
const ARCH = { x64: 'amd64', arm64: 'arm64' }[process.arch];

async function download(url) {
  const res = await fetch(url); // fetch follows GitHub's redirect to the asset
  if (!res.ok) throw new Error(`GET ${url} -> ${res.status}`);
  return Buffer.from(await res.arrayBuffer());
}

async function main() {
  if (!OS || !ARCH) {
    throw new Error(`unsupported platform ${process.platform}/${process.arch}`);
  }

  const ext = OS === 'windows' ? '.exe' : '';
  const dataDir = process.env.CLAUDE_PLUGIN_DATA || path.join(os.homedir(), '.cache', 'mcp-file-tools');
  const binDir = path.join(dataDir, 'bin');
  const bin = path.join(binDir, `mcp-file-tools-${VERSION}-${OS}-${ARCH}${ext}`);

  if (!fs.existsSync(bin)) {
    fs.mkdirSync(binDir, { recursive: true });
    const asset = `mcp-file-tools_${OS}_${ARCH}${ext}`;
    const base = `https://github.com/${REPO}/releases/download/${VERSION}`;
    process.stderr.write(`mcp-file-tools: downloading ${VERSION} (${OS}/${ARCH})...\n`);

    const [data, sums] = await Promise.all([
      download(`${base}/${asset}`),
      download(`${base}/checksums.txt`),
    ]);

    const want = sums.toString('utf8').split('\n').map(l => l.trim())
      .find(l => l.endsWith(' ' + asset))?.split(/\s+/)[0];
    const got = crypto.createHash('sha256').update(data).digest('hex');
    if (!want || want !== got) {
      throw new Error(`checksum mismatch for ${asset} (want=${want} got=${got})`);
    }

    const tmp = `${bin}.${process.pid}.tmp`;
    fs.writeFileSync(tmp, data, { mode: 0o755 });
    fs.renameSync(tmp, bin); // atomic
  }

  // Hand our stdio straight to the server so it speaks MCP on this process's pipes.
  // Directories come from the client via the MCP roots protocol, so no args needed.
  // The marker makes the server print plugin update steps, not binary ones.
  const child = spawn(bin, [], {
    stdio: 'inherit',
    env: { ...process.env, MCP_FILE_TOOLS_VIA_LAUNCHER: '1' },
  });
  child.on('exit', (code, signal) => {
    if (signal) process.kill(process.pid, signal);
    else process.exit(code ?? 0);
  });
}

main().catch(err => {
  process.stderr.write(`mcp-file-tools: ${err.message}\n`);
  process.exit(1);
});

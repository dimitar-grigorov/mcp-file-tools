// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// pinnedVersion reads plugin.json, so Claude Code and the launcher share one version.
func pinnedVersion(launcherDir string) (string, error) {
	manifest := filepath.Join(launcherDir, "..", ".claude-plugin", "plugin.json")

	raw, err := os.ReadFile(manifest) // the PathError already names the file
	if err != nil {
		return "", err
	}

	var plugin struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &plugin); err != nil {
		return "", fmt.Errorf("parsing %s: %w", manifest, err)
	}
	if plugin.Version == "" {
		return "", fmt.Errorf("no version in %s", manifest)
	}

	// The version goes into a URL, so keep it to version characters.
	if strings.ContainsFunc(plugin.Version, func(r rune) bool {
		return !(r == '.' || r == '-' || r == '+' ||
			r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	}) {
		return "", fmt.Errorf("implausible version %q in %s", plugin.Version, manifest)
	}

	return plugin.Version, nil
}

// cachedServerPath prefers CLAUDE_PLUGIN_DATA, which Claude Code removes on uninstall.
// The fallback matches bin/run.js so both launchers share one cache.
func cachedServerPath(version, arch string) (string, error) {
	dir := os.Getenv("CLAUDE_PLUGIN_DATA")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".cache", "mcp-file-tools")
	}

	name := fmt.Sprintf("mcp-file-tools-v%s-windows-%s.exe", version, arch)
	return filepath.Join(dir, "bin", name), nil
}

// install puts the server in place only once it matches the release checksums.
func install(version, arch, dest string) error {
	asset := fmt.Sprintf("mcp-file-tools_windows_%s.exe", arch)
	base := fmt.Sprintf("/%s/releases/download/v%s/", repo, version)

	server, err := httpGet(releaseHost, base+asset)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", asset, err)
	}
	checksums, err := httpGet(releaseHost, base+checksumsFile)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", checksumsFile, err)
	}

	want, err := checksumFor(checksums, asset)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(server)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf("checksum mismatch for %s: release says %s, download is %s",
			asset, want, got)
	}

	return writeAtomic(dest, server)
}

// checksumFor picks one entry out of a "<sha256>  <asset>" listing.
func checksumFor(checksums []byte, asset string) (string, error) {
	for line := range strings.Lines(string(checksums)) {
		if fields := strings.Fields(line); len(fields) == 2 && fields[1] == asset {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("%s is not listed in %s", asset, checksumsFile)
}

// writeAtomic stages beside the destination and renames, so a killed launcher cannot
// leave a half-written server. The pid and the tolerated rename cover concurrent sessions.
func writeAtomic(dest string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	staged := fmt.Sprintf("%s.%d.tmp", dest, os.Getpid())
	if err := os.WriteFile(staged, content, 0o755); err != nil {
		return err
	}

	if err := os.Rename(staged, dest); err != nil {
		os.Remove(staged)
		if _, statErr := os.Stat(dest); statErr == nil {
			return nil // another launcher won the race, which is fine
		}
		return err
	}
	return nil
}

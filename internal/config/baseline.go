// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/security"
)

const (
	// EnvAllowedDirs is an OS path list (";" on Windows, ":" elsewhere) of allowed directories.
	EnvAllowedDirs = "MCP_FILE_TOOLS_ALLOWED_DIRS"

	// EnvNoCWDFallback turns off the working-directory fallback. On by default.
	EnvNoCWDFallback = "MCP_FILE_TOOLS_NO_CWD_FALLBACK"
)

// DirSource names where the baseline allowed directories came from.
type DirSource string

const (
	SourceNone DirSource = "none"
	SourceCLI  DirSource = "cli-args"
	SourceEnv  DirSource = "env"
	SourceCWD  DirSource = "cwd"
)

// Baseline is the set the server starts with; roots merge on top and revoking them falls back here.
type Baseline struct {
	Dirs   []string
	Source DirSource

	// Explicit means the user configured Dirs rather than the server deriving them.
	Explicit bool

	// Reason is the log line's why, including why a working directory was refused.
	Reason string
}

// ResolveBaseline picks the startup dirs: CLI args, then EnvAllowedDirs, then the working
// directory — the last is all a 2026-07-28 client leaves us, roots being refused (SEP-2322).
func ResolveBaseline(cliArgs []string) (Baseline, error) {
	if len(cliArgs) > 0 {
		dirs, err := security.NormalizeAllowedDirs(cliArgs)
		if err != nil {
			return Baseline{}, err
		}
		return Baseline{Dirs: dirs, Source: SourceCLI, Explicit: true, Reason: "command-line arguments"}, nil
	}

	if dirs := envAllowedDirs(); len(dirs) > 0 {
		return Baseline{Dirs: dirs, Source: SourceEnv, Explicit: true, Reason: EnvAllowedDirs}, nil
	}

	if envEnabled(EnvNoCWDFallback) {
		return Baseline{Source: SourceNone, Reason: EnvNoCWDFallback + " is set"}, nil
	}

	dir, err := workingDirBaseline()
	if err != nil {
		return Baseline{Source: SourceNone, Reason: err.Error()}, nil
	}
	return Baseline{Dirs: []string{dir}, Source: SourceCWD, Reason: "working directory"}, nil
}

// workingDirBaseline grants the working directory, refusing a filesystem root or home
// and above — those need an explicit channel.
func workingDirBaseline() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine the working directory: %w", err)
	}

	dir, err := security.NormalizeAllowedDir(cwd)
	if err != nil {
		return "", fmt.Errorf("cannot use working directory %s: %w", cwd, err)
	}

	if filepath.Dir(dir) == dir {
		return "", fmt.Errorf("refusing working directory %s: a filesystem root is too broad to grant by default", dir)
	}

	if home, err := os.UserHomeDir(); err == nil {
		// Succeeds exactly when dir contains home, covering dir == home and C:\Users above it.
		if _, err := security.ValidatePath(home, []string{dir}); err == nil {
			return "", fmt.Errorf("refusing working directory %s: it is the home directory or an ancestor of it", dir)
		}
	}

	return dir, nil
}

// envAllowedDirs drops unusable entries so one stale path does not cost the rest.
func envAllowedDirs() []string {
	raw := os.Getenv(EnvAllowedDirs)
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var dirs []string
	for _, entry := range filepath.SplitList(raw) {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		dir, err := security.NormalizeAllowedDir(entry)
		if err != nil {
			slog.Warn("ignoring unusable directory in "+EnvAllowedDirs, "dir", entry, "error", err)
			continue
		}
		dirs = append(dirs, dir)
	}
	return dirs
}

// envEnabled reports a variable set to anything but an explicit off value.
func envEnabled(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

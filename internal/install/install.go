// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

// Package install reports which MCP client is connected and whether a launcher
// or a manual install started this server, so update steps can match.
package install

import (
	"os"
	"strings"
)

// EnvLauncher is set by the plugin launcher on the server it spawns.
const EnvLauncher = "MCP_FILE_TOOLS_VIA_LAUNCHER"

// Method is how the binary got here.
type Method string

const (
	Plugin Method = "plugin"
	Manual Method = "manual"
)

// Client is the MCP client, coarsened to the ones with distinct update steps.
type Client string

const (
	ClaudeCode Client = "claude-code"
	Codex      Client = "codex"
	OtherHost  Client = "other"
)

// Env is the pair the update message is written for.
type Env struct {
	Client Client
	Method Method
	// RootsOnly means no CLI directories, so the workspace-scoped plugin would
	// grant the same access.
	RootsOnly bool
}

// DetectMethod checks the launcher marker, falling back to the plugin env vars
// for launchers older than it.
func DetectMethod() Method {
	for _, key := range []string{EnvLauncher, "CLAUDE_PLUGIN_ROOT", "CLAUDE_PLUGIN_DATA"} {
		if os.Getenv(key) != "" {
			return Plugin
		}
	}
	return Manual
}

// DetectClient maps an MCP clientInfo name onto a Client. Anything unrecognized,
// Claude Desktop and editors included, gets client-neutral instructions.
func DetectClient(name string) Client {
	n := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(n, "codex"):
		return Codex
	case strings.Contains(n, "claude-code"), strings.Contains(n, "claude code"):
		return ClaudeCode
	default:
		return OtherHost
	}
}

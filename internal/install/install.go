// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

// Package install reports how this server was started — which MCP client, and
// whether the plugin launcher or a manual install spawned it. Update
// instructions differ per setup; nothing here changes tool behaviour.
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
	// Plugin means the Claude Code plugin launcher downloaded and started it.
	Plugin Method = "plugin"
	// Manual means a downloaded or go-installed binary registered by hand.
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
	// RootsOnly means no directories came from CLI args, so the workspace-scoped
	// plugin would grant the same access this install has.
	RootsOnly bool
}

// DetectMethod reports whether a launcher started this process. The launcher sets
// EnvLauncher; CLAUDE_PLUGIN_ROOT and CLAUDE_PLUGIN_DATA cover launchers older
// than that variable.
func DetectMethod() Method {
	for _, key := range []string{EnvLauncher, "CLAUDE_PLUGIN_ROOT", "CLAUDE_PLUGIN_DATA"} {
		if os.Getenv(key) != "" {
			return Plugin
		}
	}
	return Manual
}

// DetectClient maps an MCP clientInfo name onto a Client. Anything unrecognized —
// including Claude Desktop and editor extensions — is OtherHost, which gets
// client-neutral instructions.
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

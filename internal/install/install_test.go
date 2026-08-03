// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package install

import "testing"

func TestDetectClient(t *testing.T) {
	tests := []struct {
		name string
		want Client
	}{
		{"claude-code", ClaudeCode},
		{"Claude Code", ClaudeCode},
		{"  CLAUDE-CODE  ", ClaudeCode},
		{"codex", Codex},
		{"codex-cli", Codex},
		{"claude-ai", OtherHost}, // Claude Desktop: no plugin support
		{"cursor-vscode", OtherHost},
		{"", OtherHost},
	}

	for _, tt := range tests {
		if got := DetectClient(tt.name); got != tt.want {
			t.Errorf("DetectClient(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestDetectMethod(t *testing.T) {
	for _, key := range []string{EnvLauncher, "CLAUDE_PLUGIN_ROOT", "CLAUDE_PLUGIN_DATA"} {
		t.Run(key, func(t *testing.T) {
			clearPluginEnv(t)
			t.Setenv(key, "x")
			if got := DetectMethod(); got != Plugin {
				t.Errorf("with %s set, DetectMethod() = %q, want %q", key, got, Plugin)
			}
		})
	}

	t.Run("manual", func(t *testing.T) {
		clearPluginEnv(t)
		if got := DetectMethod(); got != Manual {
			t.Errorf("DetectMethod() = %q, want %q", got, Manual)
		}
	})
}

func clearPluginEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{EnvLauncher, "CLAUDE_PLUGIN_ROOT", "CLAUDE_PLUGIN_DATA"} {
		t.Setenv(key, "")
	}
}

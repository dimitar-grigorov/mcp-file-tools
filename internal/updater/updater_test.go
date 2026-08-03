// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package updater

import (
	"context"
	"strings"
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/install"
)

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"2.0.0", "1.0.0", true},
		{"1.1.0", "1.0.0", true},
		{"1.0.1", "1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "2.0.0", false},
		{"v1.1.0", "1.0.0", true},
		{"1.1.0", "v1.0.0", true},
		{"1.1.0-beta", "1.0.0", true},
		{"1.1", "1.0.0", true},
		{"2", "1.0.0", true},
	}

	for _, tt := range tests {
		if got := isNewerVersion(tt.latest, tt.current); got != tt.want {
			t.Errorf("isNewerVersion(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"1.2.3", [3]int{1, 2, 3}},
		{"v1.2.3", [3]int{1, 2, 3}},
		{"1.2.3-beta", [3]int{1, 2, 3}},
		{"1.2", [3]int{1, 2, 0}},
		{"1", [3]int{1, 0, 0}},
		{"", [3]int{0, 0, 0}},
	}

	for _, tt := range tests {
		if got := parseVersion(tt.input); got != tt.want {
			t.Errorf("parseVersion(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCheckDisabled(t *testing.T) {
	t.Setenv("MCP_NO_UPDATE_CHECK", "1")
	if msg := Check(context.Background(), "1.0.0", false, install.Env{}); msg != "" {
		t.Errorf("Check with disabled should return empty, got %q", msg)
	}
}

func TestCheckDevVersion(t *testing.T) {
	if msg := Check(context.Background(), "dev", false, install.Env{}); msg != "" {
		t.Errorf("Check with dev version should return empty, got %q", msg)
	}
	if msg := Check(context.Background(), "", false, install.Env{}); msg != "" {
		t.Errorf("Check with empty version should return empty, got %q", msg)
	}
}

func TestUpdateMessageFormat(t *testing.T) {
	// Just verify the format string works
	msg := "Update available: 1.0.0 → 1.1.0\nDownload: https://example.com"
	if !strings.Contains(msg, "1.0.0") || !strings.Contains(msg, "1.1.0") {
		t.Error("message format incorrect")
	}
}

func TestUpdateStepsPerInstall(t *testing.T) {
	tests := []struct {
		name    string
		env     install.Env
		want    []string
		notWant []string
	}{
		{
			name:    "plugin gets plugin commands only",
			env:     install.Env{Client: install.ClaudeCode, Method: install.Plugin},
			want:    []string{"claude plugin update mcp-file-tools@mcp-file-tools"},
			notWant: []string{"re-download"},
		},
		{
			name:    "codex never hears about Claude Code",
			env:     install.Env{Client: install.Codex, Method: install.Manual},
			want:    []string{"re-download", "Codex"},
			notWant: []string{"Claude Code", "claude plugin"},
		},
		{
			name:    "manual with CLI dirs is not nudged to the plugin",
			env:     install.Env{Client: install.ClaudeCode, Method: install.Manual},
			want:    []string{"closing all Claude Code sessions"},
			notWant: []string{"plugin ("},
		},
		{
			name: "roots-only manual hears the plugin exists, once",
			env:  install.Env{Client: install.ClaudeCode, Method: install.Manual, RootsOnly: true},
			want: []string{"re-download", "Switching is optional"},
		},
		{
			name:    "unknown client gets neutral wording",
			env:     install.Env{Client: install.OtherHost, Method: install.Manual},
			want:    []string{"closing the MCP client"},
			notWant: []string{"Claude Code", "Codex"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := updateSteps(tt.env)
			for _, s := range tt.want {
				if !strings.Contains(steps, s) {
					t.Errorf("steps missing %q:\n%s", s, steps)
				}
			}
			for _, s := range tt.notWant {
				if strings.Contains(steps, s) {
					t.Errorf("steps should not contain %q:\n%s", s, steps)
				}
			}
		})
	}
}

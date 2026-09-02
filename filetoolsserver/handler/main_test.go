// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"os"
	"strings"
	"testing"
)

// NewHandler goes through config.Load, so a developer who exports what the server
// documents would otherwise get different results than CI. Tests that want a
// variable set it with t.Setenv.
func TestMain(m *testing.M) {
	for _, name := range mcpEnvNames() {
		_ = os.Unsetenv(name)
	}
	os.Exit(m.Run())
}

func TestNoAmbientMCPEnv(t *testing.T) {
	if got := mcpEnvNames(); len(got) != 0 {
		t.Errorf("MCP_ variables reached the tests: %v", got)
	}
}

func mcpEnvNames() []string {
	var names []string
	for _, kv := range os.Environ() {
		if name, _, ok := strings.Cut(kv, "="); ok && strings.HasPrefix(name, "MCP_") {
			names = append(names, name)
		}
	}
	return names
}

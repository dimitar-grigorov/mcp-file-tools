// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	installmode "github.com/dimitar-grigorov/mcp-file-tools/internal/install"
)

const (
	repo          = "dimitar-grigorov/mcp-file-tools"
	releaseHost   = "github.com"
	checksumsFile = "checksums.txt"
	userAgent     = "mcp-file-tools-launcher"
)

func main() {
	code, err := run()
	if err != nil {
		// stdout is the MCP channel, so diagnostics go to stderr
		fmt.Fprintln(os.Stderr, "mcp-file-tools:", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func run() (int, error) {
	self, err := os.Executable()
	if err != nil {
		return 0, err
	}

	version, err := pinnedVersion(filepath.Dir(self))
	if err != nil {
		return 0, err
	}

	arch := nativeArch()
	server, err := cachedServerPath(version, arch)
	if err != nil {
		return 0, err
	}

	if _, err := os.Stat(server); err != nil {
		fmt.Fprintf(os.Stderr, "mcp-file-tools: downloading v%s (windows/%s)...\n",
			version, arch)
		if err := install(version, arch, server); err != nil {
			return 0, err
		}
	}

	return serve(server)
}

// Claude Code terminates the whole process tree, so no signal forwarding is needed.
func serve(server string) (int, error) {
	cmd := exec.Command(server, os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	// The marker lets the server print plugin update steps instead of binary ones.
	cmd.Env = append(os.Environ(), installmode.EnvLauncher+"=1")

	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}

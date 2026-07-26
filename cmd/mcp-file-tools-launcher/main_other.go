//go:build !windows

package main

import "os"

// Stub so the package still builds and vets off Windows.
func main() {
	os.Stderr.WriteString("mcp-file-tools-launcher: Windows only; " +
		"macOS and Linux use plugin/launcher/mcp-file-tools\n")
	os.Exit(1)
}

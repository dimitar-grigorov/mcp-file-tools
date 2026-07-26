// Command mcp-file-tools-launcher starts the Windows build of the MCP server,
// downloading and verifying it on first run. macOS and Linux use the shell script
// beside it in plugin/launcher/. See docs/node-free-launcher.md.
//
// Standard library only and cgo free, so the build stays reproducible.
package main

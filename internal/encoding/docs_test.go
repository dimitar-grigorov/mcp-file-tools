// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TOOLS.md's encoding table is hand-maintained; MacCyrillic shipped missing from it in 3.4.0.
func TestToolsDocListsEveryEncoding(t *testing.T) {
	path := filepath.Join("..", "..", "TOOLS.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	doc := string(data)

	for _, item := range ListEncodings() {
		if !strings.Contains(doc, "| "+item.Name+" |") {
			t.Errorf("TOOLS.md has no Supported Encodings row for %q", item.Name)
		}
		for _, alias := range item.Aliases {
			if !strings.Contains(doc, alias) {
				t.Errorf("TOOLS.md does not mention alias %q of %q", alias, item.Name)
			}
		}
	}
}

// Files that state the count outright, read before anyone installs the server.
var encodingCountDocs = []string{
	"README.md",
	filepath.Join(".claude-plugin", "marketplace.json"),
	filepath.Join("plugin", ".claude-plugin", "plugin.json"),
}

var encodingCountClaim = regexp.MustCompile(`(\d+) encodings`)

func TestDocsStateTheRealEncodingCount(t *testing.T) {
	for _, name := range encodingCountDocs {
		path := filepath.Join("..", "..", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		claims := encodingCountClaim.FindAllStringSubmatch(string(data), -1)
		if len(claims) == 0 {
			t.Errorf("%s no longer states an encoding count; drop it from encodingCountDocs", name)
			continue
		}
		for _, claim := range claims {
			stated, err := strconv.Atoi(claim[1])
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if stated != Count() {
				t.Errorf("%s says %d encodings, the registry has %d", name, stated, Count())
			}
		}
	}
}

// Two hand-edited copies of one listing; only a match makes them one text.
func TestPluginDescriptionsMatch(t *testing.T) {
	var plugin struct {
		Description string `json:"description"`
	}
	var marketplace struct {
		Plugins []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"plugins"`
	}
	readJSON(t, filepath.Join("plugin", ".claude-plugin", "plugin.json"), &plugin)
	readJSON(t, filepath.Join(".claude-plugin", "marketplace.json"), &marketplace)

	for _, entry := range marketplace.Plugins {
		if entry.Name != "mcp-file-tools" {
			continue
		}
		if entry.Description != plugin.Description {
			t.Errorf("marketplace.json and plugin.json describe the plugin differently:\n  marketplace: %s\n  plugin:      %s",
				entry.Description, plugin.Description)
		}
		return
	}
	t.Error(`marketplace.json has no "mcp-file-tools" plugin entry`)
}

func readJSON(t *testing.T, name string, into any) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if err := json.Unmarshal(data, into); err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
}

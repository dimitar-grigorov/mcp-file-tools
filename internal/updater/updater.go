// Package updater checks for new releases on GitHub and notifies users.
package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// UpdateCheckURL is the GitHub API endpoint for latest release
	UpdateCheckURL = "https://api.github.com/repos/dimitar-grigorov/mcp-file-tools/releases/latest"

	// UpdateDownloadURL is the releases page for users to download binaries
	UpdateDownloadURL = "https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest"

	// CheckInterval is the minimum time between API calls (respects GitHub rate limits)
	CheckInterval = 2 * time.Hour
)

// cache stores the last check result to avoid excessive API calls
type cache struct {
	LastCheck     time.Time `json:"lastCheck"`
	LatestVersion string    `json:"latestVersion"`
}

// Check checks for updates and returns a notification message if available.
// Returns empty string if: no update, disabled via MCP_NO_UPDATE_CHECK=1, dev version, or error.
func Check(ctx context.Context, currentVersion string) string {
	// Skip if disabled or running dev build
	if os.Getenv("MCP_NO_UPDATE_CHECK") == "1" || currentVersion == "dev" || currentVersion == "" {
		return ""
	}

	cacheFile := getCacheFile()
	latestVersion := ""

	// Use cached result if within check interval
	if c := readCache(cacheFile); c != nil && time.Since(c.LastCheck) < CheckInterval {
		latestVersion = c.LatestVersion
	} else {
		// Fetch fresh version from GitHub
		var err error
		latestVersion, err = fetchLatestVersion(ctx)
		if err != nil {
			return "" // Fail silently - don't block on network issues
		}
		writeCache(cacheFile, latestVersion)
	}

	if isNewerVersion(latestVersion, currentVersion) {
		return fmt.Sprintf("Update available: %s → %s\nDownload: %s",
			currentVersion, latestVersion, UpdateDownloadURL)
	}
	return ""
}

// fetchLatestVersion queries GitHub API for the latest release tag
func fetchLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, UpdateCheckURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "mcp-file-tools-update-checker")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return strings.TrimPrefix(release.TagName, "v"), nil
}

// getCacheFile returns the path to the cache file in user's cache directory
func getCacheFile() string {
	if dir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(dir, "mcp-file-tools", "update-check.json")
	}
	return ""
}

func readCache(path string) *cache {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var c cache
	if json.Unmarshal(data, &c) != nil {
		return nil
	}
	return &c
}

func writeCache(path, version string) {
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.Marshal(cache{LastCheck: time.Now(), LatestVersion: version})
	_ = os.WriteFile(path, data, 0644)
}

// isNewerVersion compares semver versions (major.minor.patch)
func isNewerVersion(latest, current string) bool {
	l, c := parseVersion(latest), parseVersion(current)
	for i := 0; i < 3; i++ {
		if l[i] > c[i] {
			return true
		}
		if l[i] < c[i] {
			return false
		}
	}
	return false
}

// parseVersion extracts [major, minor, patch] from version string
// Handles: "1.2.3", "v1.2.3", "1.2.3-beta", "1.2", "1"
func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	var r [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		r[i], _ = strconv.Atoi(strings.Split(parts[i], "-")[0])
	}
	return r
}

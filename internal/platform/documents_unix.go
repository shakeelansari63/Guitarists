//go:build !windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DocumentsDir returns the user's Documents directory on Unix-like systems.
// It honours XDG_DOCUMENTS_DIR, then the XDG user-dirs configuration, and
// finally falls back to $HOME/Documents.
func DocumentsDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("XDG_DOCUMENTS_DIR")); dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}

	configHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}

	if dir, err := readUserDirs(filepath.Join(configHome, "user-dirs.dirs"), "XDG_DOCUMENTS_DIR", home); err == nil && dir != "" {
		return dir, nil
	}

	return filepath.Join(home, "Documents"), nil
}

// readUserDirs extracts a directory from an XDG user-dirs.dirs file.
func readUserDirs(path, key, home string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return parseUserDirs(string(data), key, home)
}

// parseUserDirs parses the subset of the XDG user-dirs format needed here.
// Values are shell-like quoted strings where $HOME expands to the user's home.
func parseUserDirs(contents, key, home string) (string, error) {
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(name) != key {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"`)
		if value == "" {
			continue
		}
		value = strings.ReplaceAll(value, "$HOME", home)
		value = strings.ReplaceAll(value, "${HOME}", home)
		if !filepath.IsAbs(value) {
			value = filepath.Join(home, value)
		}
		return filepath.Clean(value), nil
	}
	return "", errors.New("documents directory not configured")
}

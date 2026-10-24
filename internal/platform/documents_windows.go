//go:build windows

package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// DocumentsDir returns the user's Documents directory on Windows.
func DocumentsDir() (string, error) {
	if dir := os.Getenv("USERPROFILE"); dir != "" {
		return filepath.Join(dir, "Documents"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, "Documents"), nil
}

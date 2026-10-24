// Package platform resolves OS-specific user directories.
package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// TabsDirName is the directory under the user's Documents directory where tabs are stored.
const TabsDirName = "guitarists-tabs"

// TabsDir returns the directory used to store tablature files, creating it if needed.
func TabsDir() (string, error) {
	docs, err := DocumentsDir()
	if err != nil {
		return "", fmt.Errorf("resolving documents directory: %w", err)
	}
	dir := filepath.Join(docs, TabsDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating tabs directory %q: %w", dir, err)
	}
	return dir, nil
}

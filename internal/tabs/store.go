package tabs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	maxFileSize = 10 << 20
	maxNameLen  = 100
)

// Summary is a lightweight description of a stored tab file.
type Summary struct {
	Name     string    `json:"name"`
	Title    string    `json:"title"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// Store persists tabs as XML files inside a single directory.
type Store struct {
	dir string
}

// NewStore creates a store rooted at dir, creating it when necessary.
func NewStore(dir string) (*Store, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("tabs directory is empty")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving tabs directory: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("creating tabs directory %q: %w", abs, err)
	}
	return &Store{dir: abs}, nil
}

// Dir returns the absolute directory backing the store.
func (s *Store) Dir() string {
	return s.dir
}

// List returns stored tabs sorted by most recently modified.
func (s *Store) List() ([]Summary, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Summary{}, nil
		}
		return nil, fmt.Errorf("reading tabs directory: %w", err)
	}

	summaries := make([]Summary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".xml") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		summary := Summary{Name: entry.Name(), Size: info.Size(), Modified: info.ModTime()}
		if title, err := titleOfFile(filepath.Join(s.dir, entry.Name())); err == nil {
			summary.Title = title
		}
		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Modified.Equal(summaries[j].Modified) {
			return summaries[i].Name < summaries[j].Name
		}
		return summaries[i].Modified.After(summaries[j].Modified)
	})
	return summaries, nil
}

func titleOfFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	tab, err := Decode(io.LimitReader(f, maxFileSize))
	if err != nil {
		return "", err
	}
	return tab.Title, nil
}

// Load reads and validates a tab by file name.
func (s *Store) Load(name string) (*Tab, error) {
	path, err := s.resolve(name)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening tab %q: %w", name, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspecting tab %q: %w", name, err)
	}
	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("tab %q is too large", name)
	}

	tab, err := Decode(f)
	if err != nil {
		return nil, fmt.Errorf("loading tab %q: %w", name, err)
	}
	return tab, nil
}

// Save validates and atomically writes a tab under name.
func (s *Store) Save(name string, tab *Tab) (Summary, error) {
	if tab == nil {
		return Summary{}, errors.New("tab is nil")
	}
	path, err := s.resolve(name)
	if err != nil {
		return Summary{}, err
	}

	var buf bytes.Buffer
	if err := Encode(&buf, tab); err != nil {
		return Summary{}, err
	}

	tmp, err := os.CreateTemp(s.dir, ".guitarists-*.tmp")
	if err != nil {
		return Summary{}, fmt.Errorf("creating temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		return Summary{}, fmt.Errorf("writing tab: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return Summary{}, fmt.Errorf("flushing tab: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return Summary{}, fmt.Errorf("closing tab: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return Summary{}, fmt.Errorf("setting tab permissions: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return Summary{}, fmt.Errorf("saving tab %q: %w", name, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return Summary{}, fmt.Errorf("inspecting saved tab: %w", err)
	}
	return Summary{
		Name:     filepath.Base(path),
		Title:    tab.Title,
		Size:     info.Size(),
		Modified: info.ModTime(),
	}, nil
}

// Delete removes a stored tab by file name.
func (s *Store) Delete(name string) error {
	path, err := s.resolve(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleting tab %q: %w", name, err)
	}
	return nil
}

// NormalizeName validates a user-supplied file name and appends the XML extension.
func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("file name is required")
	}
	if strings.HasPrefix(name, ".") {
		return "", errors.New("file name cannot start with a dot")
	}
	if strings.ContainsAny(name, `<>:"/\|?*`) {
		return "", errors.New("file name contains an invalid character")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", errors.New("file name contains a control character")
		}
	}
	if strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return "", errors.New("file name cannot end with a dot or space")
	}
	if !strings.EqualFold(filepath.Ext(name), ".xml") {
		name += ".xml"
	}
	if len(name) > maxNameLen {
		return "", fmt.Errorf("file name is too long (maximum %d characters)", maxNameLen)
	}
	if filepath.Base(name) != name {
		return "", errors.New("file name must not contain a path")
	}
	return name, nil
}

func (s *Store) resolve(name string) (string, error) {
	clean, err := NormalizeName(name)
	if err != nil {
		return "", err
	}
	path := filepath.Join(s.dir, clean)
	rel, err := filepath.Rel(s.dir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("file name escapes the tabs directory")
	}
	return path, nil
}

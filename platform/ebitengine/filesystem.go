package ebitengine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FileSystem implements core.FileSystem on top of the OS filesystem.
// It is identical in behavior to the SDL implementation but pure Go.
type FileSystem struct{}

// ReadFile reads the entire contents of a file.
func (s *FileSystem) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return data, nil
}

// WriteFile writes data to a file.
func (s *FileSystem) WriteFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}
	return nil
}

// FileExists checks if a file exists.
func (s *FileSystem) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ListFiles lists all files in a directory (non-recursive).
func (s *FileSystem) ListFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

// ListFilesRecursive lists all files in a directory recursively.
func (s *FileSystem) ListFilesRecursive(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			rel, err := filepath.Rel(dir, path)
			if err == nil {
				files = append(files, rel)
			}
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to walk directory %s: %w", dir, err)
	}
	return files, nil
}

// CreateDirectory creates a new directory.
func (s *FileSystem) CreateDirectory(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

// Delete deletes a file or empty directory.
func (s *FileSystem) Delete(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete %s: %w", path, err)
	}
	return nil
}

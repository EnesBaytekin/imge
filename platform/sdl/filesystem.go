package sdl

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

// SDLFileSystem implements core.FileSystem interface using OS filesystem.
type SDLFileSystem struct{}

// ReadFile reads the entire contents of a file.
func (fs *SDLFileSystem) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", path, err)
	}

	log.Printf("[SDLFileSystem] ReadFile: %s (%d bytes)", path, len(data))
	return data, nil
}

// WriteFile writes data to a file.
func (fs *SDLFileSystem) WriteFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %v", path, err)
	}

	log.Printf("[SDLFileSystem] WriteFile: %s (%d bytes)", path, len(data))
	return nil
}

// FileExists checks if a file exists.
func (fs *SDLFileSystem) FileExists(path string) bool {
	_, err := os.Stat(path)
	exists := err == nil

	log.Printf("[SDLFileSystem] FileExists: %s -> %v", path, exists)
	return exists
}

// ListFiles lists all files in a directory (non-recursive).
func (fs *SDLFileSystem) ListFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %v", dir, err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	log.Printf("[SDLFileSystem] ListFiles: %s -> %d files", dir, len(files))
	return files, nil
}

// ListFilesRecursive lists all files in a directory recursively.
func (fs *SDLFileSystem) ListFilesRecursive(dir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip files/directories we can't access
			return nil
		}

		if !d.IsDir() {
			// Make path relative to starting directory
			rel, err := filepath.Rel(dir, path)
			if err == nil {
				files = append(files, rel)
			}
		}
		return nil
	})

	if err != nil {
		// If directory doesn't exist, return empty slice
		if os.IsNotExist(err) {
			log.Printf("[SDLFileSystem] ListFilesRecursive: %s (directory not found)", dir)
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to walk directory %s: %v", dir, err)
	}

	log.Printf("[SDLFileSystem] ListFilesRecursive: %s -> %d files", dir, len(files))
	return files, nil
}

// CreateDirectory creates a new directory.
func (fs *SDLFileSystem) CreateDirectory(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", path, err)
	}

	log.Printf("[SDLFileSystem] CreateDirectory: %s", path)
	return nil
}

// Delete deletes a file or empty directory.
func (fs *SDLFileSystem) Delete(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete %s: %v", path, err)
	}

	log.Printf("[SDLFileSystem] Delete: %s", path)
	return nil
}
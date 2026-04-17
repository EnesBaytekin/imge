package mock

import (
	"fmt"
	"os"
)

// MockFileSystem implements core.FileSystem interface with debug prints.
type MockFileSystem struct {
	files map[string][]byte
	dirs  map[string]bool
}

// ReadFile reads the entire contents of a file.
func (fs *MockFileSystem) ReadFile(path string) ([]byte, error) {
	fmt.Printf("[MockFileSystem] ReadFile(path=%s)\n", path)

	// Try to read actual file first
	if data, err := os.ReadFile(path); err == nil {
		return data, nil
	}

	// If file doesn't exist, check mock storage
	if data, exists := fs.files[path]; exists {
		return data, nil
	}

	return nil, fmt.Errorf("file not found: %s", path)
}

// WriteFile writes data to a file.
func (fs *MockFileSystem) WriteFile(path string, data []byte) error {
	fmt.Printf("[MockFileSystem] WriteFile(path=%s, data_size=%d)\n", path, len(data))

	// Initialize maps if nil
	if fs.files == nil {
		fs.files = make(map[string][]byte)
	}

	fs.files[path] = data
	return nil
}

// FileExists checks if a file exists.
func (fs *MockFileSystem) FileExists(path string) bool {
	fmt.Printf("[MockFileSystem] FileExists(path=%s)\n", path)

	// Check actual file system
	if _, err := os.Stat(path); err == nil {
		return true
	}

	// Check mock storage
	_, exists := fs.files[path]
	return exists
}

// ListFiles lists all files in a directory (non-recursive).
func (fs *MockFileSystem) ListFiles(dir string) ([]string, error) {
	fmt.Printf("[MockFileSystem] ListFiles(dir=%s)\n", dir)

	// Try actual filesystem first
	if entries, err := os.ReadDir(dir); err == nil {
		var files []string
		for _, entry := range entries {
			if !entry.IsDir() {
				files = append(files, entry.Name())
			}
		}
		return files, nil
	}

	// Return empty list for mock
	return []string{}, nil
}

// ListFilesRecursive lists all files in a directory recursively.
func (fs *MockFileSystem) ListFilesRecursive(dir string) ([]string, error) {
	fmt.Printf("[MockFileSystem] ListFilesRecursive(dir=%s)\n", dir)

	// Simple implementation - just return empty for mock
	return []string{}, nil
}

// CreateDirectory creates a new directory.
func (fs *MockFileSystem) CreateDirectory(path string) error {
	fmt.Printf("[MockFileSystem] CreateDirectory(path=%s)\n", path)

	// Initialize dirs map if nil
	if fs.dirs == nil {
		fs.dirs = make(map[string]bool)
	}

	fs.dirs[path] = true
	return os.MkdirAll(path, 0755)
}

// Delete deletes a file or empty directory.
func (fs *MockFileSystem) Delete(path string) error {
	fmt.Printf("[MockFileSystem] Delete(path=%s)\n", path)

	// Remove from mock storage
	delete(fs.files, path)
	delete(fs.dirs, path)

	// Try actual delete
	return os.Remove(path)
}
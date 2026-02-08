package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ning0612/Syncrules/internal/domain"
)

// Adapter implements the adapter.Adapter interface for local filesystem
type Adapter struct {
	root string
}

// New creates a new local filesystem adapter
// root must be an absolute path to an existing directory
func New(root string) (*Adapter, error) {
	// Convert to absolute path
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	// Verify root exists and is a directory
	info, err := os.Stat(absRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, domain.ErrNotDirectory
	}

	return &Adapter{root: absRoot}, nil
}

// resolvePath safely resolves a relative path to absolute path within root
// Returns error if path attempts to escape root directory
func (a *Adapter) resolvePath(relPath string) (string, error) {
	// Handle empty path as root
	if relPath == "" || relPath == "." {
		return a.root, nil
	}

	// Normalize path separators
	relPath = filepath.FromSlash(relPath)

	// Clean the path to remove . and ..
	relPath = filepath.Clean(relPath)

	// Reject absolute paths
	if filepath.IsAbs(relPath) {
		return "", domain.ErrPermissionDenied
	}

	// Join with root
	fullPath := filepath.Join(a.root, relPath)

	// Use filepath.Rel to safely verify the path is within root
	// This handles edge cases like root="C:\root" and fullPath="C:\root2"
	rel, err := filepath.Rel(a.root, fullPath)
	if err != nil {
		return "", domain.ErrPermissionDenied
	}

	// If rel starts with "..", it's outside root
	if strings.HasPrefix(rel, "..") {
		return "", domain.ErrPermissionDenied
	}

	return fullPath, nil
}

// List returns all files and directories under the given path
func (a *Adapter) List(ctx context.Context, path string) ([]domain.FileInfo, error) {
	fullPath, err := a.resolvePath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, a.mapError(err)
	}

	result := make([]domain.FileInfo, 0, len(entries))
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't read
		}

		entryPath := filepath.Join(path, entry.Name())
		fileInfo := a.fileInfoFromOS(entryPath, info)

		// Phase 3: Calculate checksum for regular files
		// Only compute checksum for files <= 100MB to avoid performance issues
		if fileInfo.IsFile() && fileInfo.Size <= 100*1024*1024 {
			checksum, err := a.computeChecksum(ctx, entryPath)
			if err == nil {
				fileInfo.Checksum = checksum
			}
			// If checksum calculation fails, continue with empty checksum
			// This allows the system to fall back to time-based comparison
		}

		result = append(result, fileInfo)
	}

	return result, nil
}

// Read opens a file for reading
func (a *Adapter) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath, err := a.resolvePath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, a.mapError(err)
	}
	if info.IsDir() {
		return nil, domain.ErrNotFile
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, a.mapError(err)
	}

	return file, nil
}

// Write creates or overwrites a file
func (a *Adapter) Write(ctx context.Context, path string, r io.Reader) error {
	fullPath, err := a.resolvePath(path)
	if err != nil {
		return err
	}

	// Create parent directories
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return a.mapError(err)
	}

	// Write to temp file first for atomic operation
	tempPath := fullPath + ".syncrules.tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return a.mapError(err)
	}

	_, copyErr := io.Copy(file, r)
	closeErr := file.Close()

	if copyErr != nil {
		os.Remove(tempPath)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tempPath)
		return closeErr
	}

	// Atomic rename
	if err := os.Rename(tempPath, fullPath); err != nil {
		os.Remove(tempPath)
		return a.mapError(err)
	}

	return nil
}

// Delete removes a file or empty directory
func (a *Adapter) Delete(ctx context.Context, path string) error {
	fullPath, err := a.resolvePath(path)
	if err != nil {
		return err
	}

	err = os.Remove(fullPath)
	return a.mapError(err)
}

// Stat returns metadata for a single path
func (a *Adapter) Stat(ctx context.Context, path string) (domain.FileInfo, error) {
	fullPath, err := a.resolvePath(path)
	if err != nil {
		return domain.FileInfo{}, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return domain.FileInfo{}, a.mapError(err)
	}

	return a.fileInfoFromOS(path, info), nil
}

// StatWithChecksum returns metadata including checksum for a file
func (a *Adapter) StatWithChecksum(ctx context.Context, path string) (domain.FileInfo, error) {
	info, err := a.Stat(ctx, path)
	if err != nil {
		return info, err
	}

	if info.IsFile() {
		checksum, err := a.computeChecksum(ctx, path)
		if err != nil {
			return info, err
		}
		info.Checksum = checksum
	}

	return info, nil
}

// Mkdir creates a directory and any necessary parents
func (a *Adapter) Mkdir(ctx context.Context, path string) error {
	fullPath, err := a.resolvePath(path)
	if err != nil {
		return err
	}

	return os.MkdirAll(fullPath, 0755)
}

// Exists checks if a path exists
func (a *Adapter) Exists(ctx context.Context, path string) (bool, error) {
	fullPath, err := a.resolvePath(path)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Close releases any resources (no-op for local adapter)
func (a *Adapter) Close() error {
	return nil
}

// Root returns the root path of this adapter
func (a *Adapter) Root() string {
	return a.root
}

// fileInfoFromOS converts os.FileInfo to domain.FileInfo
func (a *Adapter) fileInfoFromOS(path string, info os.FileInfo) domain.FileInfo {
	fileType := domain.FileTypeRegular
	if info.IsDir() {
		fileType = domain.FileTypeDirectory
	} else if info.Mode()&os.ModeSymlink != 0 {
		fileType = domain.FileTypeSymlink
	}

	return domain.FileInfo{
		Path:    filepath.ToSlash(path), // Normalize to forward slashes
		Type:    fileType,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}
}

// computeChecksum calculates SHA256 checksum of a file
func (a *Adapter) computeChecksum(ctx context.Context, path string) (string, error) {
	reader, err := a.Read(ctx, path)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// mapError converts OS errors to domain errors
func (a *Adapter) mapError(err error) error {
	if err == nil {
		return nil
	}

	if os.IsNotExist(err) {
		return domain.ErrNotFound
	}
	if os.IsPermission(err) {
		return domain.ErrPermissionDenied
	}
	if os.IsExist(err) {
		return domain.ErrAlreadyExists
	}

	// Check for directory not empty (platform specific)
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		if strings.Contains(pathErr.Err.Error(), "not empty") ||
			strings.Contains(pathErr.Err.Error(), "directory not empty") {
			return domain.ErrNotDirectory
		}
	}

	return err
}

package domain

import "time"

// FileType represents the type of a filesystem entry
type FileType int

const (
	FileTypeRegular FileType = iota
	FileTypeDirectory
	FileTypeSymlink
)

// FileInfo represents metadata about a file or directory
type FileInfo struct {
	// Path is the relative path from the endpoint root
	Path string

	// Type indicates if this is a file, directory, or symlink
	Type FileType

	// Size in bytes (0 for directories)
	Size int64

	// ModTime is the last modification time
	ModTime time.Time

	// Checksum is the content hash (empty for directories)
	// Algorithm should be consistent within a sync session
	Checksum string

	// ETag is the remote version identifier (for cloud adapters)
	ETag string

	// IsDeleted marks tombstones for tracking deletions
	IsDeleted bool
}

// IsDir returns true if this is a directory
func (f FileInfo) IsDir() bool {
	return f.Type == FileTypeDirectory
}

// IsFile returns true if this is a regular file
func (f FileInfo) IsFile() bool {
	return f.Type == FileTypeRegular
}

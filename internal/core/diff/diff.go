package diff

import "github.com/Ning0612/Syncrules/internal/domain"

// DiffResult represents the comparison result between two files
type DiffResult int

const (
	// FilesIdentical indicates files are the same
	FilesIdentical DiffResult = iota
	// FileModified indicates file exists in both but differs
	FileModified
	// FileOnlyInSource indicates file only exists in source
	FileOnlyInSource
	// FileOnlyInTarget indicates file only exists in target
	FileOnlyInTarget
)

// Comparer compares two files and determines if sync is needed
type Comparer interface {
	// Compare compares source and target file info
	// Returns the diff result indicating whether sync is needed
	Compare(src, tgt *domain.FileInfo) DiffResult
}

// DefaultComparer uses mtime + size comparison
// This is the default strategy as specified in CLAUDE.md line 139
//
// Future enhancement: Support optional checksum/hash comparison
// for detecting content changes when size is unchanged
type DefaultComparer struct{}

// NewDefaultComparer creates a new DefaultComparer
func NewDefaultComparer() *DefaultComparer {
	return &DefaultComparer{}
}

// Compare implements the Comparer interface
// Migrated from service/sync.go line 215-216, 316
func (c *DefaultComparer) Compare(src, tgt *domain.FileInfo) DiffResult {
	// Both nil - shouldn't happen but handle gracefully
	if src == nil && tgt == nil {
		return FilesIdentical
	}

	// Only in source
	if src != nil && tgt == nil {
		return FileOnlyInSource
	}

	// Only in target
	if src == nil && tgt != nil {
		return FileOnlyInTarget
	}

	// Both exist - compare size and mtime
	// Note: We don't support directory comparison here
	// Directories are handled separately in planner
	if src.IsFile() && tgt.IsFile() {
		// Size differs - definitely modified
		if src.Size != tgt.Size {
			return FileModified
		}

		// Size same, check mtime
		// Use ModTime.Equal() to handle platform-specific precision
		if !src.ModTime.Equal(tgt.ModTime) {
			// Phase 2: If both have checksums, use content comparison
			if src.Checksum != "" && tgt.Checksum != "" {
				if src.Checksum == tgt.Checksum {
					// Content is identical despite different mtime
					return FilesIdentical
				}
				// Checksums differ - content is different
				return FileModified
			}

			// No checksums available (e.g., large files exceeding MaxSize)
			// For large files, if size matches, assume content is identical
			// Rationale: Large files rarely have same size but different content
			// This prevents unnecessary copying of large files
			if src.Checksum == "" && tgt.Checksum == "" {
				// Both lack checksums - use size as heuristic
				// Size already matched (we're in the "size same" branch)
				return FilesIdentical
			}

			// Only one has checksum - fall back to conservative strategy
			// If source is newer, it's modified
			if src.ModTime.After(tgt.ModTime) {
				return FileModified
			}
			// Target is newer - for conservative strategy, consider identical
			// For two-way sync, this should be handled as conflict by planner
			// Here we just report the fact
			return FileModified
		}

		// Size and mtime identical
		return FilesIdentical
	}

	// For directories or mixed types, consider as different
	// This shouldn't normally happen as directories are filtered by planner
	return FileModified
}

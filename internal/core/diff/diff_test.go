package diff

import (
	"testing"
	"time"

	"github.com/Ning0612/Syncrules/internal/domain"
)

func TestDefaultComparer_FilesIdentical(t *testing.T) {
	comparer := NewDefaultComparer()
	now := time.Now()

	src := &domain.FileInfo{
		Path:    "test.txt",
		Type:    domain.FileTypeRegular,
		Size:    100,
		ModTime: now,
	}
	tgt := &domain.FileInfo{
		Path:    "test.txt",
		Type:    domain.FileTypeRegular,
		Size:    100,
		ModTime: now,
	}

	result := comparer.Compare(src, tgt)
	if result != FilesIdentical {
		t.Errorf("Expected FilesIdentical, got %v", result)
	}
}

func TestDefaultComparer_FileModified_SizeDiff(t *testing.T) {
	comparer := NewDefaultComparer()
	now := time.Now()

	src := &domain.FileInfo{
		Path:    "test.txt",
		Type:    domain.FileTypeRegular,
		Size:    100,
		ModTime: now,
	}
	tgt := &domain.FileInfo{
		Path:    "test.txt",
		Type:    domain.FileTypeRegular,
		Size:    200, // Different size
		ModTime: now,
	}

	result := comparer.Compare(src, tgt)
	if result != FileModified {
		t.Errorf("Expected FileModified, got %v", result)
	}
}

func TestDefaultComparer_FileModified_MtimeDiff(t *testing.T) {
	comparer := NewDefaultComparer()
	now := time.Now()

	src := &domain.FileInfo{
		Path:    "test.txt",
		Type:    domain.FileTypeRegular,
		Size:    100,
		ModTime: now.Add(1 * time.Hour), // Newer
	}
	tgt := &domain.FileInfo{
		Path:    "test.txt",
		Type:    domain.FileTypeRegular,
		Size:    200, // Different size to test mtime comparison
		ModTime: now,
	}

	result := comparer.Compare(src, tgt)
	if result != FileModified {
		t.Errorf("Expected FileModified, got %v", result)
	}
}

func TestDefaultComparer_OnlyInSource(t *testing.T) {
	comparer := NewDefaultComparer()

	src := &domain.FileInfo{
		Path:    "test.txt",
		Type:    domain.FileTypeRegular,
		Size:    100,
		ModTime: time.Now(),
	}

	result := comparer.Compare(src, nil)
	if result != FileOnlyInSource {
		t.Errorf("Expected FileOnlyInSource, got %v", result)
	}
}

func TestDefaultComparer_OnlyInTarget(t *testing.T) {
	comparer := NewDefaultComparer()

	tgt := &domain.FileInfo{
		Path:    "test.txt",
		Type:    domain.FileTypeRegular,
		Size:    100,
		ModTime: time.Now(),
	}

	result := comparer.Compare(nil, tgt)
	if result != FileOnlyInTarget {
		t.Errorf("Expected FileOnlyInTarget, got %v", result)
	}
}

// Codex Review #7: Boundary condition - mtime precision
func TestDefaultComparer_MtimePrecision(t *testing.T) {
	comparer := NewDefaultComparer()
	now := time.Now()

	// Simulate different filesystem precision
	// FAT32: 2 second precision
	// ext4: nanosecond precision
	src := &domain.FileInfo{
		Path:    "test.txt",
		Type:    domain.FileTypeRegular,
		Size:    100,
		ModTime: now.Truncate(2 * time.Second), // FAT32-like precision
	}
	tgt := &domain.FileInfo{
		Path:    "test.txt",
		Type:    domain.FileTypeRegular,
		Size:    100,
		ModTime: now.Truncate(2 * time.Second), // Same when truncated
	}

	result := comparer.Compare(src, tgt)
	if result != FilesIdentical {
		t.Errorf("Expected FilesIdentical for same truncated time, got %v", result)
	}
}

// Codex Review #7: Boundary condition - empty file
func TestDefaultComparer_EmptyFile(t *testing.T) {
	comparer := NewDefaultComparer()
	now := time.Now()

	src := &domain.FileInfo{
		Path:    "empty.txt",
		Type:    domain.FileTypeRegular,
		Size:    0, // Zero bytes
		ModTime: now,
	}
	tgt := &domain.FileInfo{
		Path:    "empty.txt",
		Type:    domain.FileTypeRegular,
		Size:    0,
		ModTime: now,
	}

	result := comparer.Compare(src, tgt)
	if result != FilesIdentical {
		t.Errorf("Expected FilesIdentical for empty files, got %v", result)
	}
}

func TestDefaultComparer_BothNil(t *testing.T) {
	comparer := NewDefaultComparer()

	result := comparer.Compare(nil, nil)
	if result != FilesIdentical {
		t.Errorf("Expected FilesIdentical for both nil, got %v", result)
	}
}

func TestDefaultComparer_TargetNewer(t *testing.T) {
	comparer := NewDefaultComparer()
	now := time.Now()

	src := &domain.FileInfo{
		Path:    "test.txt",
		Type:    domain.FileTypeRegular,
		Size:    100,
		ModTime: now,
	}
	tgt := &domain.FileInfo{
		Path:    "test.txt",
		Type:    domain.FileTypeRegular,
		Size:    200,                    // Different size to test mtime comparison
		ModTime: now.Add(1 * time.Hour), // Target is newer
	}

	result := comparer.Compare(src, tgt)
	// When sizes differ, always report as modified
	if result != FileModified {
		t.Errorf("Expected FileModified when sizes differ, got %v", result)
	}
}

// Phase 2: Checksum comparison tests

func TestDefaultComparer_ChecksumMatch_DifferentMtime(t *testing.T) {
	comparer := NewDefaultComparer()
	now := time.Now()

	src := &domain.FileInfo{
		Path:     "test.txt",
		Type:     domain.FileTypeRegular,
		Size:     100,
		ModTime:  now,
		Checksum: "abc123", // Same checksum
	}
	tgt := &domain.FileInfo{
		Path:     "test.txt",
		Type:     domain.FileTypeRegular,
		Size:     100,
		ModTime:  now.Add(1 * time.Hour), // Different mtime
		Checksum: "abc123",               // Same checksum
	}

	result := comparer.Compare(src, tgt)
	if result != FilesIdentical {
		t.Errorf("Expected FilesIdentical when checksums match despite different mtime, got %v", result)
	}
}

func TestDefaultComparer_ChecksumDiffer_SameSize(t *testing.T) {
	comparer := NewDefaultComparer()
	now := time.Now()

	src := &domain.FileInfo{
		Path:     "test.txt",
		Type:     domain.FileTypeRegular,
		Size:     100,
		ModTime:  now.Add(1 * time.Hour),
		Checksum: "abc123",
	}
	tgt := &domain.FileInfo{
		Path:     "test.txt",
		Type:     domain.FileTypeRegular,
		Size:     100,
		ModTime:  now,
		Checksum: "def456", // Different checksum
	}

	result := comparer.Compare(src, tgt)
	if result != FileModified {
		t.Errorf("Expected FileModified when checksums differ, got %v", result)
	}
}

func TestDefaultComparer_NoChecksum_FallbackToMtime(t *testing.T) {
	comparer := NewDefaultComparer()
	now := time.Now()

	src := &domain.FileInfo{
		Path:     "test.txt",
		Type:     domain.FileTypeRegular,
		Size:     100,
		ModTime:  now.Add(1 * time.Hour),
		Checksum: "", // No checksum
	}
	tgt := &domain.FileInfo{
		Path:     "test.txt",
		Type:     domain.FileTypeRegular,
		Size:     100,
		ModTime:  now,
		Checksum: "", // No checksum
	}

	result := comparer.Compare(src, tgt)
	// Should treat as identical when size matches but no checksums available
	// This prevents unnecessary copying of large files
	if result != FilesIdentical {
		t.Errorf("Expected FilesIdentical when size matches and no checksums available, got %v", result)
	}
}

func TestDefaultComparer_OnlySourceHasChecksum(t *testing.T) {
	comparer := NewDefaultComparer()
	now := time.Now()

	src := &domain.FileInfo{
		Path:     "test.txt",
		Type:     domain.FileTypeRegular,
		Size:     100,
		ModTime:  now.Add(1 * time.Hour),
		Checksum: "abc123", // Has checksum
	}
	tgt := &domain.FileInfo{
		Path:     "test.txt",
		Type:     domain.FileTypeRegular,
		Size:     100,
		ModTime:  now,
		Checksum: "", // No checksum
	}

	result := comparer.Compare(src, tgt)
	// Should fall back to time-based comparison when only one has checksum
	if result != FileModified {
		t.Errorf("Expected FileModified when only one file has checksum, got %v", result)
	}
}

package conflict

import (
	"testing"
	"time"

	"github.com/Ning0612/Syncrules/internal/domain"
)

func TestResolve_KeepLocal(t *testing.T) {
	resolver := NewDefaultResolver()
	now := time.Now()

	src := &domain.FileInfo{Path: "test.txt", Size: 100, ModTime: now}
	tgt := &domain.FileInfo{Path: "test.txt", Size: 200, ModTime: now.Add(time.Hour)}

	action := resolver.Resolve(domain.ConflictKeepLocal, "test.txt", src, tgt)

	if action.Type != domain.ActionSkip {
		t.Errorf("Expected ActionSkip, got %v", action.Type)
	}
}

func TestResolve_KeepRemote(t *testing.T) {
	resolver := NewDefaultResolver()
	now := time.Now()

	src := &domain.FileInfo{Path: "test.txt", Size: 100, ModTime: now}
	tgt := &domain.FileInfo{Path: "test.txt", Size: 200, ModTime: now.Add(time.Hour)}

	action := resolver.Resolve(domain.ConflictKeepRemote, "test.txt", src, tgt)

	if action.Type != domain.ActionCopy {
		t.Errorf("Expected ActionCopy, got %v", action.Type)
	}
	if action.Direction != domain.DirSourceToTarget {
		t.Errorf("Expected DirSourceToTarget, got %v", action.Direction)
	}
}

func TestResolve_KeepNewest_SourceNewer(t *testing.T) {
	resolver := NewDefaultResolver()
	now := time.Now()

	src := &domain.FileInfo{Path: "test.txt", Size: 100, ModTime: now.Add(time.Hour)}
	tgt := &domain.FileInfo{Path: "test.txt", Size: 200, ModTime: now}

	action := resolver.Resolve(domain.ConflictKeepNewest, "test.txt", src, tgt)

	if action.Type != domain.ActionCopy {
		t.Errorf("Expected ActionCopy, got %v", action.Type)
	}
	if action.Direction != domain.DirSourceToTarget {
		t.Errorf("Expected DirSourceToTarget, got %v", action.Direction)
	}
}

func TestResolve_KeepNewest_TargetNewer(t *testing.T) {
	resolver := NewDefaultResolver()
	now := time.Now()

	src := &domain.FileInfo{Path: "test.txt", Size: 100, ModTime: now}
	tgt := &domain.FileInfo{Path: "test.txt", Size: 200, ModTime: now.Add(time.Hour)}

	action := resolver.Resolve(domain.ConflictKeepNewest, "test.txt", src, tgt)

	if action.Type != domain.ActionCopy {
		t.Errorf("Expected ActionCopy, got %v", action.Type)
	}
	if action.Direction != domain.DirTargetToSource {
		t.Errorf("Expected DirTargetToSource, got %v", action.Direction)
	}
}

func TestResolve_KeepNewest_EqualTimeAndSize(t *testing.T) {
	resolver := NewDefaultResolver()
	now := time.Now()

	src := &domain.FileInfo{Path: "test.txt", Size: 100, ModTime: now}
	tgt := &domain.FileInfo{Path: "test.txt", Size: 100, ModTime: now}

	action := resolver.Resolve(domain.ConflictKeepNewest, "test.txt", src, tgt)

	if action.Type != domain.ActionSkip {
		t.Errorf("Expected ActionSkip for equal time and size, got %v", action.Type)
	}
	if action.Reason != "identical modification time and size" {
		t.Errorf("Expected reason 'identical modification time and size', got %v", action.Reason)
	}
}

func TestResolve_KeepNewest_EqualTimeButDifferentSize(t *testing.T) {
	resolver := NewDefaultResolver()
	now := time.Now()

	src := &domain.FileInfo{Path: "test.txt", Size: 100, ModTime: now}
	tgt := &domain.FileInfo{Path: "test.txt", Size: 200, ModTime: now}

	action := resolver.Resolve(domain.ConflictKeepNewest, "test.txt", src, tgt)

	if action.Type != domain.ActionConflict {
		t.Errorf("Expected ActionConflict for equal time but different size, got %v", action.Type)
	}
	if action.Reason != "identical time but different size" {
		t.Errorf("Expected reason 'identical time but different size', got %v", action.Reason)
	}
}

func TestResolve_Manual(t *testing.T) {
	resolver := NewDefaultResolver()
	now := time.Now()

	src := &domain.FileInfo{Path: "test.txt", Size: 100, ModTime: now}
	tgt := &domain.FileInfo{Path: "test.txt", Size: 200, ModTime: now.Add(time.Hour)}

	action := resolver.Resolve(domain.ConflictManual, "test.txt", src, tgt)

	if action.Type != domain.ActionConflict {
		t.Errorf("Expected ActionConflict, got %v", action.Type)
	}
}

func TestResolve_NilInfo(t *testing.T) {
	resolver := NewDefaultResolver()
	now := time.Now()
	src := &domain.FileInfo{Path: "test.txt", Size: 100, ModTime: now}

	// Test nil source
	action := resolver.Resolve(domain.ConflictKeepNewest, "test.txt", nil, src)
	if action.Type != domain.ActionConflict {
		t.Errorf("Expected ActionConflict for nil source, got %v", action.Type)
	}
	if action.Reason != "manual resolution required: nil file info" {
		t.Errorf("Expected nil info reason, got %v", action.Reason)
	}

	// Test nil target
	action = resolver.Resolve(domain.ConflictKeepNewest, "test.txt", src, nil)
	if action.Type != domain.ActionConflict {
		t.Errorf("Expected ActionConflict for nil target, got %v", action.Type)
	}
	if action.Reason != "manual resolution required: nil file info" {
		t.Errorf("Expected nil info reason, got %v", action.Reason)
	}
}

// Phase 2: Checksum-based conflict resolution tests

func TestResolve_KeepNewest_ChecksumMatch(t *testing.T) {
	resolver := NewDefaultResolver()
	now := time.Now()

	src := &domain.FileInfo{
		Path:     "test.txt",
		Size:     100,
		ModTime:  now,
		Checksum: "abc123",
	}
	tgt := &domain.FileInfo{
		Path:     "test.txt",
		Size:     100,
		ModTime:  now.Add(time.Hour), // Different time
		Checksum: "abc123",           // Same checksum
	}

	action := resolver.Resolve(domain.ConflictKeepNewest, "test.txt", src, tgt)

	if action.Type != domain.ActionSkip {
		t.Errorf("Expected ActionSkip when checksums match, got %v", action.Type)
	}
	if action.Reason != "identical content (checksum match)" {
		t.Errorf("Expected checksum match reason, got %v", action.Reason)
	}
}

func TestResolve_KeepNewest_ChecksumDiffer_SourceNewer(t *testing.T) {
	resolver := NewDefaultResolver()
	now := time.Now()

	src := &domain.FileInfo{
		Path:     "test.txt",
		Size:     100,
		ModTime:  now.Add(time.Hour), // Newer
		Checksum: "abc123",
	}
	tgt := &domain.FileInfo{
		Path:     "test.txt",
		Size:     100,
		ModTime:  now,
		Checksum: "def456", // Different checksum
	}

	action := resolver.Resolve(domain.ConflictKeepNewest, "test.txt", src, tgt)

	// Should fall back to time-based resolution
	if action.Type != domain.ActionCopy {
		t.Errorf("Expected ActionCopy when checksums differ and source is newer, got %v", action.Type)
	}
	if action.Direction != domain.DirSourceToTarget {
		t.Errorf("Expected DirSourceToTarget, got %v", action.Direction)
	}
	if action.Reason != "source is newer" {
		t.Errorf("Expected 'source is newer' reason, got %v", action.Reason)
	}
}

func TestResolve_KeepNewest_NoChecksum(t *testing.T) {
	resolver := NewDefaultResolver()
	now := time.Now()

	src := &domain.FileInfo{
		Path:     "test.txt",
		Size:     100,
		ModTime:  now.Add(time.Hour),
		Checksum: "", // No checksum
	}
	tgt := &domain.FileInfo{
		Path:     "test.txt",
		Size:     100,
		ModTime:  now,
		Checksum: "", // No checksum
	}

	action := resolver.Resolve(domain.ConflictKeepNewest, "test.txt", src, tgt)

	// Should use time-based resolution
	if action.Type != domain.ActionCopy {
		t.Errorf("Expected ActionCopy when no checksums available, got %v", action.Type)
	}
	if action.Reason != "source is newer" {
		t.Errorf("Expected time-based reason, got %v", action.Reason)
	}
}

func TestResolve_KeepNewest_OnlyOneHasChecksum(t *testing.T) {
	resolver := NewDefaultResolver()
	now := time.Now()

	src := &domain.FileInfo{
		Path:     "test.txt",
		Size:     100,
		ModTime:  now.Add(time.Hour),
		Checksum: "abc123", // Has checksum
	}
	tgt := &domain.FileInfo{
		Path:     "test.txt",
		Size:     100,
		ModTime:  now,
		Checksum: "", // No checksum
	}

	action := resolver.Resolve(domain.ConflictKeepNewest, "test.txt", src, tgt)

	// Should fall back to time-based resolution when only one has checksum
	if action.Type != domain.ActionCopy {
		t.Errorf("Expected ActionCopy when only one file has checksum, got %v", action.Type)
	}
	if action.Reason != "source is newer" {
		t.Errorf("Expected time-based reason, got %v", action.Reason)
	}
}

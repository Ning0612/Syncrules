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

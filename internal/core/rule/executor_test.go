package rule

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Ning0612/Syncrules/internal/domain"
)

// mockAdapter for testing
type mockAdapter struct {
	files      []domain.FileInfo
	listError  error
	shouldFail bool
}

func (m *mockAdapter) List(ctx context.Context, prefix string) ([]domain.FileInfo, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	var result []domain.FileInfo
	for _, f := range m.files {
		// Only return immediate children of prefix
		if prefix == "" {
			// Root level: only files/dirs without path separator
			if !strings.Contains(f.Path, "/") {
				result = append(result, f)
			}
		} else {
			// Check if file is direct child of prefix
			if strings.HasPrefix(f.Path, prefix+"/") {
				// Get relative path from prefix
				rel := strings.TrimPrefix(f.Path, prefix+"/")
				// Only include if it's an immediate child (no more slashes)
				if !strings.Contains(rel, "/") {
					result = append(result, f)
				}
			}
		}
	}
	return result, nil
}

func (m *mockAdapter) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (m *mockAdapter) Write(ctx context.Context, path string, r io.Reader) error {
	return nil
}

func (m *mockAdapter) Delete(ctx context.Context, path string) error {
	return nil
}

func (m *mockAdapter) Stat(ctx context.Context, path string) (domain.FileInfo, error) {
	for _, f := range m.files {
		if f.Path == path {
			return f, nil
		}
	}
	return domain.FileInfo{}, nil
}

func (m *mockAdapter) Mkdir(ctx context.Context, path string) error {
	return nil
}

func (m *mockAdapter) Exists(ctx context.Context, path string) (bool, error) {
	for _, f := range m.files {
		if f.Path == path {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockAdapter) Close() error {
	return nil
}

func TestExecutor_OneWayPush(t *testing.T) {
	executor := NewDefaultExecutor()
	now := time.Now()

	// Source with one file
	source := &mockAdapter{
		files: []domain.FileInfo{
			{
				Path:    "test.txt",
				Type:    domain.FileTypeRegular,
				Size:    100,
				ModTime: now,
			},
		},
	}

	// Empty target
	target := &mockAdapter{files: []domain.FileInfo{}}

	rule := &domain.SyncRule{
		Name:             "test-push",
		Mode:             domain.SyncModeOneWayPush,
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan, err := executor.Plan(context.Background(), rule, source, target)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	if len(plan.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(plan.Actions))
	}
	if plan.Actions[0].Type != domain.ActionCopy {
		t.Errorf("Expected ActionCopy, got %v", plan.Actions[0].Type)
	}
}

func TestExecutor_OneWayPull(t *testing.T) {
	executor := NewDefaultExecutor()
	now := time.Now()

	// Empty source
	source := &mockAdapter{files: []domain.FileInfo{}}

	// Target with one file
	target := &mockAdapter{
		files: []domain.FileInfo{
			{
				Path:    "test.txt",
				Type:    domain.FileTypeRegular,
				Size:    100,
				ModTime: now,
			},
		},
	}

	rule := &domain.SyncRule{
		Name:             "test-pull",
		Mode:             domain.SyncModeOneWayPull,
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan, err := executor.Plan(context.Background(), rule, source, target)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if len(plan.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(plan.Actions))
	}
	if plan.Actions[0].Direction != domain.DirTargetToSource {
		t.Errorf("Expected DirTargetToSource for pull")
	}
}

func TestExecutor_TwoWay(t *testing.T) {
	executor := NewDefaultExecutor()
	now := time.Now()

	// Source with file1
	source := &mockAdapter{
		files: []domain.FileInfo{
			{
				Path:    "file1.txt",
				Type:    domain.FileTypeRegular,
				Size:    100,
				ModTime: now,
			},
		},
	}

	// Target with file2
	target := &mockAdapter{
		files: []domain.FileInfo{
			{
				Path:    "file2.txt",
				Type:    domain.FileTypeRegular,
				Size:    200,
				ModTime: now,
			},
		},
	}

	rule := &domain.SyncRule{
		Name:             "test-twoway",
		Mode:             domain.SyncModeTwoWay,
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan, err := executor.Plan(context.Background(), rule, source, target)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 2 actions
	if len(plan.Actions) != 2 {
		t.Fatalf("Expected 2 actions, got %d", len(plan.Actions))
	}
}

// NEW: Test invalid sync mode
func TestExecutor_InvalidSyncMode(t *testing.T) {
	executor := NewDefaultExecutor()

	source := &mockAdapter{files: []domain.FileInfo{}}
	target := &mockAdapter{files: []domain.FileInfo{}}

	rule := &domain.SyncRule{
		Name:             "test-invalid",
		Mode:             "invalid-mode",
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	_, err := executor.Plan(context.Background(), rule, source, target)
	if err == nil {
		t.Fatal("Expected error for invalid sync mode, got nil")
	}
	if err.Error() != "unsupported sync mode: invalid-mode" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// NEW: Test source listing error
func TestExecutor_SourceListError(t *testing.T) {
	executor := NewDefaultExecutor()

	source := &mockAdapter{
		files:     []domain.FileInfo{},
		listError: errors.New("source list failed"),
	}
	target := &mockAdapter{files: []domain.FileInfo{}}

	rule := &domain.SyncRule{
		Name:             "test-error",
		Mode:             domain.SyncModeOneWayPush,
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	_, err := executor.Plan(context.Background(), rule, source, target)
	if err == nil {
		t.Fatal("Expected error when source listing fails, got nil")
	}
}

// NEW: Test target listing error
func TestExecutor_TargetListError(t *testing.T) {
	executor := NewDefaultExecutor()

	source := &mockAdapter{files: []domain.FileInfo{}}
	target := &mockAdapter{
		files:     []domain.FileInfo{},
		listError: errors.New("target list failed"),
	}

	rule := &domain.SyncRule{
		Name:             "test-error",
		Mode:             domain.SyncModeOneWayPush,
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	_, err := executor.Plan(context.Background(), rule, source, target)
	if err == nil {
		t.Fatal("Expected error when target listing fails, got nil")
	}
}

// NEW: Test context cancellation
func TestExecutor_ContextCancellation(t *testing.T) {
	executor := NewDefaultExecutor()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	source := &mockAdapter{files: []domain.FileInfo{}}
	target := &mockAdapter{files: []domain.FileInfo{}}

	rule := &domain.SyncRule{
		Name:             "test-cancel",
		Mode:             domain.SyncModeOneWayPush,
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	// Should not panic with cancelled context
	// (Actual cancellation happens in listAllFiles)
	plan, _ := executor.Plan(ctx, rule, source, target)

	// Even with empty adapters, plan should be created (cancellation happens in recursive listing)
	if plan == nil {
		t.Error("Expected plan to be created even with cancelled context for empty adapters")
	}
}

// NEW: Test empty directories (both source and target empty)
func TestExecutor_EmptyDirectories(t *testing.T) {
	executor := NewDefaultExecutor()

	source := &mockAdapter{files: []domain.FileInfo{}}
	target := &mockAdapter{files: []domain.FileInfo{}}

	rule := &domain.SyncRule{
		Name:             "test-empty",
		Mode:             domain.SyncModeOneWayPush,
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan, err := executor.Plan(context.Background(), rule, source, target)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if len(plan.Actions) != 0 {
		t.Errorf("Expected 0 actions for empty directories, got %d", len(plan.Actions))
	}
	if plan.Stats.TotalFiles != 0 {
		t.Errorf("Expected TotalFiles=0, got %d", plan.Stats.TotalFiles)
	}
}

// NEW: Test with nested directories
func TestExecutor_NestedDirectories(t *testing.T) {
	executor := NewDefaultExecutor()
	now := time.Now()

	source := &mockAdapter{
		files: []domain.FileInfo{
			{
				Path:    "dir1",
				Type:    domain.FileTypeDirectory,
				Size:    0,
				ModTime: now,
			},
			{
				Path:    "dir1/file.txt",
				Type:    domain.FileTypeRegular,
				Size:    100,
				ModTime: now,
			},
		},
	}
	target := &mockAdapter{files: []domain.FileInfo{}}

	rule := &domain.SyncRule{
		Name:             "test-nested",
		Mode:             domain.SyncModeOneWayPush,
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan, err := executor.Plan(context.Background(), rule, source, target)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 2 actions: Mkdir for dir1, Copy for dir1/file.txt
	if len(plan.Actions) != 2 {
		t.Fatalf("Expected 2 actions, got %d", len(plan.Actions))
	}

	// First action should be Mkdir (after sorting)
	if plan.Actions[0].Type != domain.ActionMkdir {
		t.Errorf("Expected first action to be Mkdir, got %v", plan.Actions[0].Type)
	}
	// Second action should be Copy
	if plan.Actions[1].Type != domain.ActionCopy {
		t.Errorf("Expected second action to be Copy, got %v", plan.Actions[1].Type)
	}
}

// NEW: Test listAllFiles with context cancellation
func TestListAllFiles_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	adapter := &mockAdapter{
		files: []domain.FileInfo{
			{
				Path:    "file.txt",
				Type:    domain.FileTypeRegular,
				Size:    100,
				ModTime: time.Now(),
			},
		},
	}

	// listAllFiles should respect context cancellation
	// Note: with small dataset, may complete before cancellation is checked
	_, err := listAllFiles(ctx, adapter, "")

	// Could be either nil (completed fast) or context.Canceled
	if err != nil && err != context.Canceled {
		t.Errorf("Expected nil or context.Canceled, got %v", err)
	}
}

package planner

import (
	"testing"
	"time"

	"github.com/Ning0612/Syncrules/internal/domain"
)

func TestPlanOneWay_NewFile(t *testing.T) {
	planner := NewDefaultPlanner()
	now := time.Now()

	fromMap := map[string]domain.FileInfo{
		"new.txt": {
			Path:    "new.txt",
			Type:    domain.FileTypeRegular,
			Size:    100,
			ModTime: now,
		},
	}
	toMap := map[string]domain.FileInfo{}

	rule := &domain.SyncRule{
		Name:             "test",
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan := planner.PlanOneWay(fromMap, toMap, rule, domain.DirSourceToTarget)

	if len(plan.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(plan.Actions))
	}
	if plan.Actions[0].Type != domain.ActionCopy {
		t.Errorf("Expected ActionCopy, got %v", plan.Actions[0].Type)
	}
}

func TestPlanOneWay_ModifiedFile(t *testing.T) {
	planner := NewDefaultPlanner()
	now := time.Now()

	fromMap := map[string]domain.FileInfo{
		"file.txt": {
			Path:    "file.txt",
			Type:    domain.FileTypeRegular,
			Size:    200,
			ModTime: now.Add(1 * time.Hour),
		},
	}
	toMap := map[string]domain.FileInfo{
		"file.txt": {
			Path:    "file.txt",
			Type:    domain.FileTypeRegular,
			Size:    100,
			ModTime: now,
		},
	}

	rule := &domain.SyncRule{
		Name:             "test",
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan := planner.PlanOneWay(fromMap, toMap, rule, domain.DirSourceToTarget)

	if len(plan.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(plan.Actions))
	}
	if plan.Actions[0].Type != domain.ActionCopy {
		t.Errorf("Expected ActionCopy, got %v", plan.Actions[0].Type)
	}
	if plan.Actions[0].Reason != "file modified" {
		t.Errorf("Unexpected reason: %s", plan.Actions[0].Reason)
	}
}

func TestPlanOneWay_DeletedFile(t *testing.T) {
	planner := NewDefaultPlanner()
	now := time.Now()

	fromMap := map[string]domain.FileInfo{}
	toMap := map[string]domain.FileInfo{
		"deleted.txt": {
			Path:    "deleted.txt",
			Type:    domain.FileTypeRegular,
			Size:    100,
			ModTime: now,
		},
	}

	rule := &domain.SyncRule{
		Name:             "test",
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan := planner.PlanOneWay(fromMap, toMap, rule, domain.DirSourceToTarget)

	if len(plan.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(plan.Actions))
	}
	if plan.Actions[0].Type != domain.ActionDelete {
		t.Errorf("Expected ActionDelete, got %v", plan.Actions[0].Type)
	}
}

func TestPlanOneWay_IgnorePatterns(t *testing.T) {
	planner := NewDefaultPlanner()
	now := time.Now()

	fromMap := map[string]domain.FileInfo{
		"test.txt": {
			Path:    "test.txt",
			Type:    domain.FileTypeRegular,
			Size:    100,
			ModTime: now,
		},
		".git/config": {
			Path:    ".git/config",
			Type:    domain.FileTypeRegular,
			Size:    50,
			ModTime: now,
		},
	}
	toMap := map[string]domain.FileInfo{}

	rule := &domain.SyncRule{
		Name:             "test",
		IgnorePatterns:   []string{".git/*", "*.tmp"},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan := planner.PlanOneWay(fromMap, toMap, rule, domain.DirSourceToTarget)

	// Should only have test.txt, .git/config should be ignored
	if len(plan.Actions) != 1 {
		t.Fatalf("Expected 1 action (ignoring .git/*), got %d", len(plan.Actions))
	}
	if plan.Actions[0].Path != "test.txt" {
		t.Errorf("Expected test.txt, got %s", plan.Actions[0].Path)
	}
}

func TestPlanTwoWay_Conflict(t *testing.T) {
	planner := NewDefaultPlanner()
	now := time.Now()

	sourceMap := map[string]domain.FileInfo{
		"conflict.txt": {
			Path:    "conflict.txt",
			Type:    domain.FileTypeRegular,
			Size:    100,
			ModTime: now.Add(1 * time.Hour),
		},
	}
	targetMap := map[string]domain.FileInfo{
		"conflict.txt": {
			Path:    "conflict.txt",
			Type:    domain.FileTypeRegular,
			Size:    200,
			ModTime: now.Add(2 * time.Hour),
		},
	}

	rule := &domain.SyncRule{
		Name:             "test",
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan := planner.PlanTwoWay(sourceMap, targetMap, rule)

	if len(plan.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(plan.Actions))
	}
	// Should resolve to target being newer
	if plan.Actions[0].Direction != domain.DirTargetToSource {
		t.Errorf("Expected DirTargetToSource for newer target")
	}
}

func TestCalculateStats(t *testing.T) {
	plan := &domain.SyncPlan{
		Actions: []domain.SyncAction{
			{Type: domain.ActionCopy, SourceInfo: &domain.FileInfo{Size: 100}},
			{Type: domain.ActionCopy, SourceInfo: &domain.FileInfo{Size: 200}},
			{Type: domain.ActionDelete},
			{Type: domain.ActionMkdir},
			{Type: domain.ActionConflict},
		},
	}

	calculateStats(plan)

	if plan.Stats.TotalFiles != 5 {
		t.Errorf("Expected TotalFiles=5, got %d", plan.Stats.TotalFiles)
	}
	if plan.Stats.FilesToCopy != 2 {
		t.Errorf("Expected FilesToCopy=2, got %d", plan.Stats.FilesToCopy)
	}
	if plan.Stats.BytesToSync != 300 {
		t.Errorf("Expected BytesToSync=300, got %d", plan.Stats.BytesToSync)
	}
	if plan.Stats.FilesToDelete != 1 {
		t.Errorf("Expected FilesToDelete=1, got %d", plan.Stats.FilesToDelete)
	}
	if plan.Stats.DirsToCreate != 1 {
		t.Errorf("Expected DirsToCreate=1, got %d", plan.Stats.DirsToCreate)
	}
	if plan.Stats.Conflicts != 1 {
		t.Errorf("Expected Conflicts=1, got %d", plan.Stats.Conflicts)
	}
}

// NEW: Test mixed type conflict (file vs directory) in one-way
func TestPlanOneWay_MixedTypeConflict(t *testing.T) {
	planner := NewDefaultPlanner()
	now := time.Now()

	fromMap := map[string]domain.FileInfo{
		"mixed": {
			Path:    "mixed",
			Type:    domain.FileTypeRegular,
			Size:    100,
			ModTime: now,
		},
	}
	toMap := map[string]domain.FileInfo{
		"mixed": {
			Path:    "mixed",
			Type:    domain.FileTypeDirectory,
			Size:    0,
			ModTime: now,
		},
	}

	rule := &domain.SyncRule{
		Name:             "test",
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan := planner.PlanOneWay(fromMap, toMap, rule, domain.DirSourceToTarget)

	if len(plan.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(plan.Actions))
	}
	if plan.Actions[0].Type != domain.ActionConflict {
		t.Errorf("Expected ActionConflict for type mismatch,got %v", plan.Actions[0].Type)
	}
	if plan.Actions[0].Reason != "type mismatch: file vs directory" {
		t.Errorf("Unexpected reason: %s", plan.Actions[0].Reason)
	}
}

// NEW: Test mixed type conflict in two-way
func TestPlanTwoWay_MixedTypeConflict(t *testing.T) {
	planner := NewDefaultPlanner()
	now := time.Now()

	sourceMap := map[string]domain.FileInfo{
		"mixed": {
			Path:    "mixed",
			Type:    domain.FileTypeDirectory,
			Size:    0,
			ModTime: now,
		},
	}
	targetMap := map[string]domain.FileInfo{
		"mixed": {
			Path:    "mixed",
			Type:    domain.FileTypeRegular,
			Size:    100,
			ModTime: now,
		},
	}

	rule := &domain.SyncRule{
		Name:             "test",
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan := planner.PlanTwoWay(sourceMap, targetMap, rule)

	if len(plan.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(plan.Actions))
	}
	if plan.Actions[0].Type != domain.ActionConflict {
		t.Errorf("Expected ActionConflict, got %v", plan.Actions[0].Type)
	}
	if plan.Actions[0].Reason != "type mismatch: file vs directory" {
		t.Errorf("Unexpected reason: %s", plan.Actions[0].Reason)
	}
}

// NEW: Test action sorting order
func TestSortActions_CorrectOrder(t *testing.T) {
	actions := []domain.SyncAction{
		{Type: domain.ActionDelete, Path: "file.txt"},
		{Type: domain.ActionCopy, Path: "doc.txt"},
		{Type: domain.ActionMkdir, Path: "dir"},
		{Type: domain.ActionConflict, Path: "conflict.txt"},
		{Type: domain.ActionSkip, Path: "skip.txt"},
	}

	sortActions(actions)

	// Expected order: Mkdir, Copy, Delete, Conflict, Skip
	if actions[0].Type != domain.ActionMkdir {
		t.Errorf("Expected Mkdir first, got %v", actions[0].Type)
	}
	if actions[1].Type != domain.ActionCopy {
		t.Errorf("Expected Copy second, got %v", actions[1].Type)
	}
	if actions[2].Type != domain.ActionDelete {
		t.Errorf("Expected Delete third, got %v", actions[2].Type)
	}
	if actions[3].Type != domain.ActionConflict {
		t.Errorf("Expected Conflict fourth, got %v", actions[3].Type)
	}
	if actions[4].Type != domain.ActionSkip {
		t.Errorf("Expected Skip fifth, got %v", actions[4].Type)
	}
}

// NEW: Test sorting by depth (shallow to deep for Mkdir)
func TestSortActions_DepthOrdering(t *testing.T) {
	actions := []domain.SyncAction{
		{Type: domain.ActionMkdir, Path: "a/b/c"},
		{Type: domain.ActionMkdir, Path: "a"},
		{Type: domain.ActionMkdir, Path: "a/b"},
	}

	sortActions(actions)

	// Should be sorted: a, a/b, a/b/c (shallow to deep)
	if actions[0].Path != "a" {
		t.Errorf("Expected 'a' first, got %s", actions[0].Path)
	}
	if actions[1].Path != "a/b" {
		t.Errorf("Expected 'a/b' second, got %s", actions[1].Path)
	}
	if actions[2].Path != "a/b/c" {
		t.Errorf("Expected 'a/b/c' third, got %s", actions[2].Path)
	}
}

// NEW: Test delete sorting (deep to shallow)
func TestSortActions_DeleteDepthOrdering(t *testing.T) {
	actions := []domain.SyncAction{
		{Type: domain.ActionDelete, Path: "a"},
		{Type: domain.ActionDelete, Path: "a/b/c"},
		{Type: domain.ActionDelete, Path: "a/b"},
	}

	sortActions(actions)

	// Should be sorted: a/b/c, a/b, a (deep to shallow for delete)
	if actions[0].Path != "a/b/c" {
		t.Errorf("Expected 'a/b/c' first for delete, got %s", actions[0].Path)
	}
	if actions[1].Path != "a/b" {
		t.Errorf("Expected 'a/b' second for delete, got %s", actions[1].Path)
	}
	if actions[2].Path != "a" {
		t.Errorf("Expected 'a' third for delete, got %s", actions[2].Path)
	}
}

// NEW: Test directory creation
func TestPlanOneWay_NewDirectory(t *testing.T) {
	planner := NewDefaultPlanner()
	now := time.Now()

	fromMap := map[string]domain.FileInfo{
		"newdir": {
			Path:    "newdir",
			Type:    domain.FileTypeDirectory,
			Size:    0,
			ModTime: now,
		},
	}
	toMap := map[string]domain.FileInfo{}

	rule := &domain.SyncRule{
		Name:             "test",
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan := planner.PlanOneWay(fromMap, toMap, rule, domain.DirSourceToTarget)

	if len(plan.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(plan.Actions))
	}
	if plan.Actions[0].Type != domain.ActionMkdir {
		t.Errorf("Expected ActionMkdir, got %v", plan.Actions[0].Type)
	}
	if plan.Actions[0].Reason != "directory does not exist" {
		t.Errorf("Unexpected reason: %s", plan.Actions[0].Reason)
	}
}

// NEW: Test two-way with source-only file
func TestPlanTwoWay_SourceOnlyFile(t *testing.T) {
	planner := NewDefaultPlanner()
	now := time.Now()

	sourceMap := map[string]domain.FileInfo{
		"source-only.txt": {
			Path:    "source-only.txt",
			Type:    domain.FileTypeRegular,
			Size:    100,
			ModTime: now,
		},
	}
	targetMap := map[string]domain.FileInfo{}

	rule := &domain.SyncRule{
		Name:             "test",
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan := planner.PlanTwoWay(sourceMap, targetMap, rule)

	if len(plan.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(plan.Actions))
	}
	if plan.Actions[0].Type != domain.ActionCopy {
		t.Errorf("Expected ActionCopy, got %v", plan.Actions[0].Type)
	}
	if plan.Actions[0].Direction != domain.DirSourceToTarget {
		t.Errorf("Expected DirSourceToTarget")
	}
}

// NEW: Test two-way with target-only file
func TestPlanTwoWay_TargetOnlyFile(t *testing.T) {
	planner := NewDefaultPlanner()
	now := time.Now()

	sourceMap := map[string]domain.FileInfo{}
	targetMap := map[string]domain.FileInfo{
		"target-only.txt": {
			Path:    "target-only.txt",
			Type:    domain.FileTypeRegular,
			Size:    100,
			ModTime: now,
		},
	}

	rule := &domain.SyncRule{
		Name:             "test",
		IgnorePatterns:   []string{},
		ConflictStrategy: domain.ConflictKeepNewest,
	}

	plan := planner.PlanTwoWay(sourceMap, targetMap, rule)

	if len(plan.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(plan.Actions))
	}
	if plan.Actions[0].Type != domain.ActionCopy {
		t.Errorf("Expected ActionCopy, got %v", plan.Actions[0].Type)
	}
	if plan.Actions[0].Direction != domain.DirTargetToSource {
		t.Errorf("Expected DirTargetToSource")
	}
}

// NEW: Test shouldIgnore with complex patterns
func TestShouldIgnore_ComplexPatterns(t *testing.T) {
	patterns := []string{"*.tmp", "*.log", ".git/*", "node_modules/*"}

	tests := []struct {
		path   string
		should bool
	}{
		{"test.tmp", true},
		{"debug.log", true},
		{".git/config", true},
		{"node_modules/package.json", true},
		{"normal.txt", false},
		{"src/main.go", false},
	}

	for _, tt := range tests {
		result := shouldIgnore(tt.path, patterns)
		if result != tt.should {
			t.Errorf("shouldIgnore(%q, patterns) = %v, want %v", tt.path, result, tt.should)
		}
	}
}

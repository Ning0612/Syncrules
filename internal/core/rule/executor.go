package rule

import (
	"context"
	"fmt"

	"github.com/Ning0612/Syncrules/internal/adapter"
	"github.com/Ning0612/Syncrules/internal/core/planner"
	"github.com/Ning0612/Syncrules/internal/domain"
)

// Executor orchestrates the sync planning process
type Executor interface {
	Plan(ctx context.Context, rule *domain.SyncRule, sourceAdapter, targetAdapter adapter.Adapter) (*domain.SyncPlan, error)
}

// DefaultExecutor implements the planning orchestration
// Migrated from service/sync.go line 124-179, 499-527
type DefaultExecutor struct {
	Planner planner.Planner
}

// NewDefaultExecutor creates a new rule executor
func NewDefaultExecutor() *DefaultExecutor {
	return &DefaultExecutor{
		Planner: planner.NewDefaultPlanner(),
	}
}

// Plan creates a sync plan for a rule
// This is the core orchestration logic migrated from service.PlanSync
func (e *DefaultExecutor) Plan(ctx context.Context, rule *domain.SyncRule, sourceAdapter, targetAdapter adapter.Adapter) (*domain.SyncPlan, error) {
	// List source files
	sourceFiles, err := listAllFiles(ctx, sourceAdapter, "", rule.IgnorePatterns)
	if err != nil {
		return nil, fmt.Errorf("listing source files: %w", err)
	}

	// List target files
	targetFiles, err := listAllFiles(ctx, targetAdapter, "", rule.IgnorePatterns)
	if err != nil {
		return nil, fmt.Errorf("listing target files: %w", err)
	}

	// Convert to maps for efficient lookup
	sourceMap := make(map[string]domain.FileInfo)
	for _, f := range sourceFiles {
		sourceMap[f.Path] = f
	}

	targetMap := make(map[string]domain.FileInfo)
	for _, f := range targetFiles {
		targetMap[f.Path] = f
	}

	// Generate plan based on sync mode
	var plan *domain.SyncPlan
	switch rule.Mode {
	case domain.SyncModeOneWayPush:
		// Source → Target only
		plan = e.Planner.PlanOneWay(sourceMap, targetMap, rule, domain.DirSourceToTarget)
	case domain.SyncModeOneWayPull:
		// Target → Source only (reversed direction)
		plan = e.Planner.PlanOneWay(targetMap, sourceMap, rule, domain.DirTargetToSource)
	case domain.SyncModeTwoWay:
		// Bidirectional sync
		plan = e.Planner.PlanTwoWay(sourceMap, targetMap, rule)
	default:
		return nil, fmt.Errorf("unsupported sync mode: %s", rule.Mode)
	}

	return plan, nil
}

// listAllFiles recursively lists all files from an adapter
// Migrated from service/sync.go line 499-527
func listAllFiles(ctx context.Context, adp adapter.Adapter, prefix string, ignorePatterns []string) ([]domain.FileInfo, error) {
	var allFiles []domain.FileInfo

	items, err := adp.List(ctx, prefix)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Skip ignored files and directories
		if planner.ShouldIgnore(item.Path, ignorePatterns) {
			continue
		}

		if item.IsDir() {
			allFiles = append(allFiles, item)
			subFiles, err := listAllFiles(ctx, adp, item.Path, ignorePatterns)
			if err != nil {
				return nil, err
			}
			allFiles = append(allFiles, subFiles...)
		} else {
			allFiles = append(allFiles, item)
		}
	}

	return allFiles, nil
}

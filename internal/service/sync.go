package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Ning0612/Syncrules/internal/adapter"
	"github.com/Ning0612/Syncrules/internal/adapter/local"
	"github.com/Ning0612/Syncrules/internal/config"
	"github.com/Ning0612/Syncrules/internal/domain"
)

// SyncService orchestrates sync operations
type SyncService struct {
	config   *config.Config
	adapters map[string]adapter.Adapter
}

// NewSyncService creates a new sync service
func NewSyncService(cfg *config.Config) (*SyncService, error) {
	return &SyncService{
		config:   cfg,
		adapters: make(map[string]adapter.Adapter),
	}, nil
}

// getAdapter returns or creates an adapter for the given endpoint
func (s *SyncService) getAdapter(endpointName string) (adapter.Adapter, error) {
	// Check cache
	if a, ok := s.adapters[endpointName]; ok {
		return a, nil
	}

	// Get endpoint config
	endpoint, err := s.config.GetEndpoint(endpointName)
	if err != nil {
		return nil, err
	}

	// Get transport config
	transport, err := s.config.GetTransport(endpoint.Transport)
	if err != nil {
		return nil, err
	}

	// Create adapter based on transport type
	var a adapter.Adapter
	switch transport.Type {
	case domain.TransportLocal:
		a, err = local.New(endpoint.Root)
		if err != nil {
			return nil, fmt.Errorf("failed to create local adapter for %s: %w", endpointName, err)
		}
	case domain.TransportGDrive:
		return nil, fmt.Errorf("gdrive transport not yet implemented")
	default:
		return nil, fmt.Errorf("unknown transport type: %s", transport.Type)
	}

	// Cache the adapter
	s.adapters[endpointName] = a
	return a, nil
}

// PlanSync creates a sync plan for a rule without executing it
func (s *SyncService) PlanSync(ctx context.Context, ruleName string) (*domain.SyncPlan, error) {
	rule, err := s.config.GetRule(ruleName)
	if err != nil {
		return nil, err
	}

	sourceAdapter, err := s.getAdapter(rule.SourceEndpoint)
	if err != nil {
		return nil, fmt.Errorf("source endpoint: %w", err)
	}

	targetAdapter, err := s.getAdapter(rule.TargetEndpoint)
	if err != nil {
		return nil, fmt.Errorf("target endpoint: %w", err)
	}

	// Build file lists
	sourceFiles, err := s.listAllFiles(ctx, sourceAdapter, "")
	if err != nil {
		return nil, fmt.Errorf("listing source files: %w", err)
	}

	targetFiles, err := s.listAllFiles(ctx, targetAdapter, "")
	if err != nil {
		return nil, fmt.Errorf("listing target files: %w", err)
	}

	// Create lookup maps
	sourceMap := make(map[string]domain.FileInfo)
	for _, f := range sourceFiles {
		sourceMap[f.Path] = f
	}

	targetMap := make(map[string]domain.FileInfo)
	for _, f := range targetFiles {
		targetMap[f.Path] = f
	}

	// Generate sync plan based on mode
	plan := &domain.SyncPlan{
		RuleName: ruleName,
		Actions:  make([]domain.SyncAction, 0),
	}

	switch rule.Mode {
	case domain.SyncModeOneWayPush:
		s.planOneWaySync(ctx, sourceMap, targetMap, rule, plan, sourceAdapter)
	case domain.SyncModeOneWayPull:
		s.planOneWaySync(ctx, targetMap, sourceMap, rule, plan, targetAdapter)
	case domain.SyncModeTwoWay:
		s.planTwoWaySync(ctx, sourceMap, targetMap, rule, plan, sourceAdapter, targetAdapter)
	}

	// Calculate stats
	s.calculateStats(plan)

	return plan, nil
}

// planOneWaySync generates actions for one-way sync (source -> target)
func (s *SyncService) planOneWaySync(
	ctx context.Context,
	sourceMap, targetMap map[string]domain.FileInfo,
	rule *domain.SyncRule,
	plan *domain.SyncPlan,
	sourceAdapter adapter.Adapter,
) {
	// Files to copy: in source but not in target, or different
	for path, srcInfo := range sourceMap {
		if s.shouldIgnore(path, rule.IgnorePatterns) {
			continue
		}

		tgtInfo, exists := targetMap[path]
		if !exists {
			// New file/dir
			if srcInfo.IsDir() {
				plan.Actions = append(plan.Actions, domain.SyncAction{
					Type:       domain.ActionMkdir,
					SourcePath: path,
					TargetPath: path,
					SourceInfo: &srcInfo,
					Reason:     "directory does not exist on target",
				})
			} else {
				srcCopy := srcInfo
				plan.Actions = append(plan.Actions, domain.SyncAction{
					Type:       domain.ActionCopy,
					SourcePath: path,
					TargetPath: path,
					SourceInfo: &srcCopy,
					Reason:     "file does not exist on target",
				})
			}
		} else if srcInfo.IsFile() && tgtInfo.IsFile() {
			// Check if different (by mtime and size)
			if srcInfo.Size != tgtInfo.Size || srcInfo.ModTime.After(tgtInfo.ModTime) {
				srcCopy := srcInfo
				tgtCopy := tgtInfo
				plan.Actions = append(plan.Actions, domain.SyncAction{
					Type:       domain.ActionCopy,
					SourcePath: path,
					TargetPath: path,
					SourceInfo: &srcCopy,
					TargetInfo: &tgtCopy,
					Reason:     "file modified on source",
				})
			}
		}
	}

	// Files to delete: in target but not in source (for one-way push)
	for path, tgtInfo := range targetMap {
		if s.shouldIgnore(path, rule.IgnorePatterns) {
			continue
		}

		if _, exists := sourceMap[path]; !exists {
			tgtCopy := tgtInfo
			plan.Actions = append(plan.Actions, domain.SyncAction{
				Type:       domain.ActionDelete,
				TargetPath: path,
				TargetInfo: &tgtCopy,
				Reason:     "file does not exist on source",
			})
		}
	}
}

// planTwoWaySync generates actions for bidirectional sync
func (s *SyncService) planTwoWaySync(
	ctx context.Context,
	sourceMap, targetMap map[string]domain.FileInfo,
	rule *domain.SyncRule,
	plan *domain.SyncPlan,
	sourceAdapter, targetAdapter adapter.Adapter,
) {
	allPaths := make(map[string]bool)
	for path := range sourceMap {
		allPaths[path] = true
	}
	for path := range targetMap {
		allPaths[path] = true
	}

	for path := range allPaths {
		if s.shouldIgnore(path, rule.IgnorePatterns) {
			continue
		}

		srcInfo, srcExists := sourceMap[path]
		tgtInfo, tgtExists := targetMap[path]

		switch {
		case srcExists && !tgtExists:
			// Only on source -> copy to target
			if srcInfo.IsDir() {
				plan.Actions = append(plan.Actions, domain.SyncAction{
					Type:       domain.ActionMkdir,
					SourcePath: path,
					TargetPath: path,
					SourceInfo: &srcInfo,
					Reason:     "directory only exists on source",
				})
			} else {
				srcCopy := srcInfo
				plan.Actions = append(plan.Actions, domain.SyncAction{
					Type:       domain.ActionCopy,
					SourcePath: path,
					TargetPath: path,
					SourceInfo: &srcCopy,
					Reason:     "file only exists on source",
				})
			}

		case !srcExists && tgtExists:
			// Only on target -> copy to source (reverse)
			if tgtInfo.IsDir() {
				plan.Actions = append(plan.Actions, domain.SyncAction{
					Type:       domain.ActionMkdir,
					SourcePath: path,
					TargetPath: path,
					TargetInfo: &tgtInfo,
					Reason:     "directory only exists on target",
				})
			} else {
				tgtCopy := tgtInfo
				plan.Actions = append(plan.Actions, domain.SyncAction{
					Type:       domain.ActionCopy,
					SourcePath: path,
					TargetPath: path,
					TargetInfo: &tgtCopy,
					Reason:     "file only exists on target",
				})
			}

		case srcExists && tgtExists && srcInfo.IsFile() && tgtInfo.IsFile():
			// Both exist - check for conflict
			if srcInfo.Size != tgtInfo.Size || !srcInfo.ModTime.Equal(tgtInfo.ModTime) {
				srcCopy := srcInfo
				tgtCopy := tgtInfo
				action := s.resolveConflict(rule.ConflictStrategy, &srcCopy, &tgtCopy)
				action.SourcePath = path
				action.TargetPath = path
				plan.Actions = append(plan.Actions, action)
			}
		}
	}
}

// resolveConflict determines the action for a conflict
func (s *SyncService) resolveConflict(strategy domain.ConflictStrategy, srcInfo, tgtInfo *domain.FileInfo) domain.SyncAction {
	switch strategy {
	case domain.ConflictKeepLocal:
		return domain.SyncAction{
			Type:       domain.ActionSkip,
			SourceInfo: srcInfo,
			TargetInfo: tgtInfo,
			Reason:     "keeping local version (conflict strategy)",
		}
	case domain.ConflictKeepRemote:
		return domain.SyncAction{
			Type:       domain.ActionCopy,
			SourceInfo: srcInfo,
			TargetInfo: tgtInfo,
			Reason:     "using remote version (conflict strategy)",
		}
	case domain.ConflictKeepNewest:
		if srcInfo.ModTime.After(tgtInfo.ModTime) {
			return domain.SyncAction{
				Type:       domain.ActionCopy,
				SourceInfo: srcInfo,
				TargetInfo: tgtInfo,
				Reason:     "source is newer",
			}
		}
		return domain.SyncAction{
			Type:       domain.ActionSkip,
			SourceInfo: srcInfo,
			TargetInfo: tgtInfo,
			Reason:     "target is newer or same",
		}
	default: // ConflictManual
		return domain.SyncAction{
			Type:       domain.ActionConflict,
			SourceInfo: srcInfo,
			TargetInfo: tgtInfo,
			Reason:     "manual resolution required",
		}
	}
}

// ExecuteSync executes a sync plan
func (s *SyncService) ExecuteSync(ctx context.Context, plan *domain.SyncPlan) error {
	rule, err := s.config.GetRule(plan.RuleName)
	if err != nil {
		return err
	}

	sourceAdapter, err := s.getAdapter(rule.SourceEndpoint)
	if err != nil {
		return err
	}

	targetAdapter, err := s.getAdapter(rule.TargetEndpoint)
	if err != nil {
		return err
	}

	for _, action := range plan.Actions {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := s.executeAction(ctx, action, sourceAdapter, targetAdapter); err != nil {
			return fmt.Errorf("action %s on %s: %w", action.Type, action.TargetPath, err)
		}
	}

	return nil
}

// executeAction performs a single sync action
func (s *SyncService) executeAction(
	ctx context.Context,
	action domain.SyncAction,
	sourceAdapter, targetAdapter adapter.Adapter,
) error {
	switch action.Type {
	case domain.ActionCopy:
		reader, err := sourceAdapter.Read(ctx, action.SourcePath)
		if err != nil {
			return err
		}
		defer reader.Close()
		return targetAdapter.Write(ctx, action.TargetPath, reader)

	case domain.ActionMkdir:
		return targetAdapter.Mkdir(ctx, action.TargetPath)

	case domain.ActionDelete:
		return targetAdapter.Delete(ctx, action.TargetPath)

	case domain.ActionSkip, domain.ActionConflict:
		// No action needed
		return nil

	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// listAllFiles recursively lists all files under a path
func (s *SyncService) listAllFiles(ctx context.Context, a adapter.Adapter, basePath string) ([]domain.FileInfo, error) {
	var result []domain.FileInfo

	entries, err := a.List(ctx, basePath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result = append(result, entry)

		if entry.IsDir() {
			subEntries, err := s.listAllFiles(ctx, a, entry.Path)
			if err != nil {
				return nil, err
			}
			result = append(result, subEntries...)
		}
	}

	return result, nil
}

// shouldIgnore checks if a path matches any ignore pattern
func (s *SyncService) shouldIgnore(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
		// Also try matching the full path
		matched, err = filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// calculateStats computes summary statistics for a plan
func (s *SyncService) calculateStats(plan *domain.SyncPlan) {
	for _, action := range plan.Actions {
		plan.Stats.TotalFiles++
		switch action.Type {
		case domain.ActionCopy:
			plan.Stats.FilesToCopy++
			if action.SourceInfo != nil {
				plan.Stats.BytesToSync += action.SourceInfo.Size
			}
		case domain.ActionDelete:
			plan.Stats.FilesToDelete++
		case domain.ActionMkdir:
			plan.Stats.DirsToCreate++
		case domain.ActionConflict:
			plan.Stats.Conflicts++
			plan.Conflicts = append(plan.Conflicts, action)
		}
	}
}

// Close releases all adapters
func (s *SyncService) Close() error {
	var lastErr error
	for _, a := range s.adapters {
		if err := a.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Ensure SyncService doesn't use io directly but needs it for interface
var _ io.Closer = (*SyncService)(nil)

package planner

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/Ning0612/Syncrules/internal/core/conflict"
	"github.com/Ning0612/Syncrules/internal/core/diff"
	"github.com/Ning0612/Syncrules/internal/domain"
)

// Planner generates sync plans
type Planner interface {
	PlanOneWay(fromMap, toMap map[string]domain.FileInfo, rule *domain.SyncRule, direction domain.SyncDirection) *domain.SyncPlan
	PlanTwoWay(sourceMap, targetMap map[string]domain.FileInfo, rule *domain.SyncRule) *domain.SyncPlan
}

// DefaultPlanner uses diff and conflict modules
// Migrated from service/sync.go line 182-324, 530-565
type DefaultPlanner struct {
	Differ   diff.Comparer
	Resolver conflict.Resolver
}

// NewDefaultPlanner creates a new planner with default components
func NewDefaultPlanner() *DefaultPlanner {
	return &DefaultPlanner{
		Differ:   diff.NewDefaultComparer(),
		Resolver: conflict.NewDefaultResolver(),
	}
}

// PlanOneWay generates actions for one-way sync
// Migrated from service/sync.go line 182-247
func (p *DefaultPlanner) PlanOneWay(fromMap, toMap map[string]domain.FileInfo, rule *domain.SyncRule, direction domain.SyncDirection) *domain.SyncPlan {
	plan := &domain.SyncPlan{
		RuleName: rule.Name,
		Actions:  make([]domain.SyncAction, 0),
	}

	// Files to copy: in "from" but not in "to", or different
	for path, fromInfo := range fromMap {
		if shouldIgnore(path, rule.IgnorePatterns) {
			continue
		}

		fromCopy := fromInfo
		toInfo, exists := toMap[path]
		if !exists {
			if fromInfo.IsDir() {
				plan.Actions = append(plan.Actions, domain.SyncAction{
					Type:       domain.ActionMkdir,
					Direction:  direction,
					Path:       path,
					SourceInfo: &fromCopy,
					Reason:     "directory does not exist",
				})
			} else {
				plan.Actions = append(plan.Actions, domain.SyncAction{
					Type:       domain.ActionCopy,
					Direction:  direction,
					Path:       path,
					SourceInfo: &fromCopy,
					Reason:     "file does not exist",
				})
			}
		} else if fromInfo.Type != toInfo.Type {
			// Mixed type conflict (file vs directory)
			toCopy := toInfo
			plan.Actions = append(plan.Actions, domain.SyncAction{
				Type:       domain.ActionConflict,
				Direction:  direction,
				Path:       path,
				SourceInfo: &fromCopy,
				TargetInfo: &toCopy,
				Reason:     "type mismatch: file vs directory",
			})
		} else if fromInfo.IsFile() && toInfo.IsFile() {
			// Use differ to check if files differ
			result := p.Differ.Compare(&fromInfo, &toInfo)
			if result == diff.FileModified {
				toCopy := toInfo
				plan.Actions = append(plan.Actions, domain.SyncAction{
					Type:       domain.ActionCopy,
					Direction:  direction,
					Path:       path,
					SourceInfo: &fromCopy,
					TargetInfo: &toCopy,
					Reason:     "file modified",
				})
			}
		}
	}

	// Files to delete: in "to" but not in "from"
	for path, toInfo := range toMap {
		if shouldIgnore(path, rule.IgnorePatterns) {
			continue
		}

		if _, exists := fromMap[path]; !exists {
			toCopy := toInfo
			plan.Actions = append(plan.Actions, domain.SyncAction{
				Type:       domain.ActionDelete,
				Direction:  direction,
				Path:       path,
				TargetInfo: &toCopy,
				Reason:     "file does not exist on source",
			})
		}
	}

	// Sort actions: Mkdir first, then Copy, then Delete
	sortActions(plan.Actions)

	calculateStats(plan)
	return plan
}

// PlanTwoWay generates actions for bidirectional sync
// Migrated from service/sync.go line 249-324
func (p *DefaultPlanner) PlanTwoWay(sourceMap, targetMap map[string]domain.FileInfo, rule *domain.SyncRule) *domain.SyncPlan {
	plan := &domain.SyncPlan{
		RuleName: rule.Name,
		Actions:  make([]domain.SyncAction, 0),
	}

	allPaths := make(map[string]bool)
	for path := range sourceMap {
		allPaths[path] = true
	}
	for path := range targetMap {
		allPaths[path] = true
	}

	for path := range allPaths {
		if shouldIgnore(path, rule.IgnorePatterns) {
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
					Direction:  domain.DirSourceToTarget,
					Path:       path,
					SourceInfo: &srcInfo,
					Reason:     "directory only exists on source",
				})
			} else {
				srcCopy := srcInfo
				plan.Actions = append(plan.Actions, domain.SyncAction{
					Type:       domain.ActionCopy,
					Direction:  domain.DirSourceToTarget,
					Path:       path,
					SourceInfo: &srcCopy,
					Reason:     "file only exists on source",
				})
			}

		case !srcExists && tgtExists:
			// Only on target -> copy to source (reverse direction)
			if tgtInfo.IsDir() {
				plan.Actions = append(plan.Actions, domain.SyncAction{
					Type:       domain.ActionMkdir,
					Direction:  domain.DirTargetToSource,
					Path:       path,
					TargetInfo: &tgtInfo,
					Reason:     "directory only exists on target",
				})
			} else {
				tgtCopy := tgtInfo
				plan.Actions = append(plan.Actions, domain.SyncAction{
					Type:       domain.ActionCopy,
					Direction:  domain.DirTargetToSource,
					Path:       path,
					TargetInfo: &tgtCopy,
					Reason:     "file only exists on target",
				})
			}

		case srcExists && tgtExists && srcInfo.Type != tgtInfo.Type:
			// Mixed type conflict in two-way sync
			srcCopy := srcInfo
			tgtCopy := tgtInfo
			plan.Actions = append(plan.Actions, domain.SyncAction{
				Type:       domain.ActionConflict,
				Direction:  domain.DirSourceToTarget,
				Path:       path,
				SourceInfo: &srcCopy,
				TargetInfo: &tgtCopy,
				Reason:     "type mismatch: file vs directory",
			})

		case srcExists && tgtExists && srcInfo.IsFile() && tgtInfo.IsFile():
			// Both exist - check for conflict using differ
			result := p.Differ.Compare(&srcInfo, &tgtInfo)
			if result == diff.FileModified {
				srcCopy := srcInfo
				tgtCopy := tgtInfo
				action := p.Resolver.Resolve(rule.ConflictStrategy, path, &srcCopy, &tgtCopy)
				plan.Actions = append(plan.Actions, action)
			}
		}
	}

	// Sort actions for two-way sync as well
	sortActions(plan.Actions)

	calculateStats(plan)
	return plan
}

// sortActions sorts actions to ensure correct execution order
// 1. Mkdir (create directories first, sorted by depth shallow->deep)
// 2. Copy (copy files)
// 3. Delete (delete last)
// 4. Conflicts (flagged for manual resolution)
func sortActions(actions []domain.SyncAction) {
	sort.Slice(actions, func(i, j int) bool {
		typeOrderI := actionTypeOrder(actions[i].Type)
		typeOrderJ := actionTypeOrder(actions[j].Type)

		// Sort by type first
		if typeOrderI != typeOrderJ {
			return typeOrderI < typeOrderJ
		}

		// Within same type, sort by path depth (shallow first)
		depthI := strings.Count(actions[i].Path, "/") + strings.Count(actions[i].Path, "\\")
		depthJ := strings.Count(actions[j].Path, "/") + strings.Count(actions[j].Path, "\\")

		if depthI != depthJ {
			// For Delete, reverse order (deep first)
			if actions[i].Type == domain.ActionDelete {
				return depthI > depthJ
			}
			return depthI < depthJ
		}

		// Finally sort by path name for determinism
		return actions[i].Path < actions[j].Path
	})
}

// actionTypeOrder returns the sort priority for action types
func actionTypeOrder(t domain.ActionType) int {
	switch t {
	case domain.ActionMkdir:
		return 1
	case domain.ActionCopy:
		return 2
	case domain.ActionDelete:
		return 3
	case domain.ActionConflict:
		return 4
	case domain.ActionSkip:
		return 5
	default:
		return 99
	}
}

// shouldIgnore checks if a path matches any ignore pattern
// Migrated from service/sync.go line 530-541
func shouldIgnore(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
		matched, err = filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// calculateStats computes summary statistics for a plan
// Migrated from service/sync.go line 545-565
func calculateStats(plan *domain.SyncPlan) {
	for _, action := range plan.Actions {
		plan.Stats.TotalFiles++
		switch action.Type {
		case domain.ActionCopy:
			plan.Stats.FilesToCopy++
			if action.SourceInfo != nil {
				plan.Stats.BytesToSync += action.SourceInfo.Size
			} else if action.TargetInfo != nil {
				plan.Stats.BytesToSync += action.TargetInfo.Size
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

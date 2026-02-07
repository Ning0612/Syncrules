package domain

// SyncMode defines how synchronization should occur
type SyncMode string

const (
	// SyncModeOneWayPush syncs from source to target only
	SyncModeOneWayPush SyncMode = "one-way-push"

	// SyncModeOneWayPull syncs from target to source only
	SyncModeOneWayPull SyncMode = "one-way-pull"

	// SyncModeTwoWay performs bidirectional sync
	SyncModeTwoWay SyncMode = "two-way"
)

// IsValid checks if the sync mode is a known value
func (m SyncMode) IsValid() bool {
	switch m {
	case SyncModeOneWayPush, SyncModeOneWayPull, SyncModeTwoWay:
		return true
	}
	return false
}

// ConflictStrategy defines how to resolve sync conflicts
type ConflictStrategy string

const (
	// ConflictKeepLocal always keeps the local version
	ConflictKeepLocal ConflictStrategy = "keep_local"

	// ConflictKeepRemote always keeps the remote version
	ConflictKeepRemote ConflictStrategy = "keep_remote"

	// ConflictKeepNewest keeps the version with newer mtime
	ConflictKeepNewest ConflictStrategy = "keep_newest"

	// ConflictManual requires user intervention
	ConflictManual ConflictStrategy = "manual"
)

// IsValid checks if the conflict strategy is a known value
func (s ConflictStrategy) IsValid() bool {
	switch s {
	case ConflictKeepLocal, ConflictKeepRemote, ConflictKeepNewest, ConflictManual:
		return true
	}
	return false
}

// SyncAction represents a single operation in a sync plan
type SyncAction struct {
	// Type of action to perform
	Type ActionType

	// Direction indicates the sync direction for this action
	Direction SyncDirection

	// Path is the relative path being operated on
	Path string

	// SourceInfo file metadata from source (nil for delete)
	SourceInfo *FileInfo

	// TargetInfo file metadata from target (nil for create)
	TargetInfo *FileInfo

	// Reason explains why this action was chosen
	Reason string
}

// ActionType represents the type of sync action
type ActionType string

const (
	ActionCopy     ActionType = "copy"
	ActionDelete   ActionType = "delete"
	ActionMkdir    ActionType = "mkdir"
	ActionConflict ActionType = "conflict"
	ActionSkip     ActionType = "skip"
)

// SyncDirection indicates the direction of a sync action
type SyncDirection int

const (
	// DirSourceToTarget copies from source endpoint to target endpoint
	DirSourceToTarget SyncDirection = iota
	// DirTargetToSource copies from target endpoint to source endpoint (reverse)
	DirTargetToSource
)

// SyncPlan represents a complete plan for synchronization
type SyncPlan struct {
	// RuleName identifies which rule generated this plan
	RuleName string

	// Actions to execute in order
	Actions []SyncAction

	// Conflicts that require manual resolution
	Conflicts []SyncAction

	// Stats summary
	Stats SyncPlanStats
}

// SyncPlanStats provides summary statistics for a sync plan
type SyncPlanStats struct {
	TotalFiles   int
	FilesToCopy  int
	FilesToDelete int
	DirsToCreate int
	Conflicts    int
	BytesToSync  int64
}

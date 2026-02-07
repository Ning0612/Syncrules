package conflict

import "github.com/Ning0612/Syncrules/internal/domain"

// Resolver resolves conflicts according to a strategy
type Resolver interface {
	// Resolve determines the appropriate action for a conflict
	// Returns a SyncAction with the resolution decision
	Resolve(strategy domain.ConflictStrategy, path string, src, tgt *domain.FileInfo) domain.SyncAction
}

// DefaultResolver implements standard conflict resolution strategies
// Migrated from service/sync.go line 327-379
type DefaultResolver struct{}

// NewDefaultResolver creates a new DefaultResolver
func NewDefaultResolver() *DefaultResolver {
	return &DefaultResolver{}
}

// Resolve implements the Resolver interface
func (r *DefaultResolver) Resolve(strategy domain.ConflictStrategy, path string, src, tgt *domain.FileInfo) domain.SyncAction {
	// Defensive nil checks to prevent panics
	if src == nil || tgt == nil {
		return domain.SyncAction{
			Type:       domain.ActionConflict,
			Direction:  domain.DirSourceToTarget,
			Path:       path,
			SourceInfo: src,
			TargetInfo: tgt,
			Reason:     "manual resolution required: nil file info",
		}
	}

	switch strategy {
	case domain.ConflictKeepLocal:
		// Local = target (where user is), skip copying from remote
		return domain.SyncAction{
			Type:       domain.ActionSkip,
			Direction:  domain.DirSourceToTarget,
			Path:       path,
			SourceInfo: src,
			TargetInfo: tgt,
			Reason:     "keeping local version (conflict strategy)",
		}

	case domain.ConflictKeepRemote:
		// Remote = source, copy from source to target
		return domain.SyncAction{
			Type:       domain.ActionCopy,
			Direction:  domain.DirSourceToTarget,
			Path:       path,
			SourceInfo: src,
			TargetInfo: tgt,
			Reason:     "using remote version (conflict strategy)",
		}

	case domain.ConflictKeepNewest:
		// Compare modification times
		if src.ModTime.After(tgt.ModTime) {
			// Source is newer
			return domain.SyncAction{
				Type:       domain.ActionCopy,
				Direction:  domain.DirSourceToTarget,
				Path:       path,
				SourceInfo: src,
				TargetInfo: tgt,
				Reason:     "source is newer",
			}
		} else if tgt.ModTime.After(src.ModTime) {
			// Target is newer, copy from target to source
			return domain.SyncAction{
				Type:       domain.ActionCopy,
				Direction:  domain.DirTargetToSource,
				Path:       path,
				SourceInfo: src,
				TargetInfo: tgt,
				Reason:     "target is newer",
			}
		} else if src.Size == tgt.Size {
			// Timestamps AND sizes are equal - truly identical
			return domain.SyncAction{
				Type:       domain.ActionSkip,
				Direction:  domain.DirSourceToTarget,
				Path:       path,
				SourceInfo: src,
				TargetInfo: tgt,
				Reason:     "identical modification time and size",
			}
		} else {
			// Timestamps equal but sizes differ - potential conflict
			return domain.SyncAction{
				Type:       domain.ActionConflict,
				Direction:  domain.DirSourceToTarget,
				Path:       path,
				SourceInfo: src,
				TargetInfo: tgt,
				Reason:     "identical time but different size",
			}
		}

	default: // ConflictManual or unknown
		// Require manual resolution
		return domain.SyncAction{
			Type:       domain.ActionConflict,
			Direction:  domain.DirSourceToTarget,
			Path:       path,
			SourceInfo: src,
			TargetInfo: tgt,
			Reason:     "manual resolution required",
		}
	}
}

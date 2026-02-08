package service

import (
	"context"
	"fmt"
	"io"

	"github.com/Ning0612/Syncrules/internal/adapter"
	"github.com/Ning0612/Syncrules/internal/adapter/gdrive"
	"github.com/Ning0612/Syncrules/internal/adapter/local"
	"github.com/Ning0612/Syncrules/internal/config"
	"github.com/Ning0612/Syncrules/internal/core/rule"
	"github.com/Ning0612/Syncrules/internal/domain"
	"github.com/Ning0612/Syncrules/internal/lock"
	"github.com/Ning0612/Syncrules/internal/logger"
	"github.com/Ning0612/Syncrules/internal/progress"
)

// SyncService orchestrates sync operations
type SyncService struct {
	config   *config.Config
	adapters map[string]adapter.Adapter
	lock     *lock.FileLock
	reporter progress.Reporter
	executor rule.Executor
}

// NewSyncService creates a new sync service
func NewSyncService(cfg *config.Config) (*SyncService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	lockPath := cfg.GetLockPath()
	fileLock, err := lock.NewFileLock(lockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file lock: %w", err)
	}

	return &SyncService{
		config:   cfg,
		adapters: make(map[string]adapter.Adapter),
		lock:     fileLock,
		executor: rule.NewDefaultExecutor(),
	}, nil
}

// AcquireLock acquires the sync lock for a specific rule
func (s *SyncService) AcquireLock(ruleName string) error {
	return s.lock.Acquire(ruleName)
}

// ReleaseLock releases the sync lock
func (s *SyncService) ReleaseLock() error {
	return s.lock.Release()
}

// IsLocked checks if another sync operation is in progress
func (s *SyncService) IsLocked() bool {
	return s.lock.IsLocked()
}

// GetLockHolder returns information about the current lock holder
func (s *SyncService) GetLockHolder() (*lock.LockInfo, error) {
	return s.lock.GetHolder()
}

// ForceUnlock forcibly releases the lock (use with caution)
func (s *SyncService) ForceUnlock() error {
	return s.lock.ForceRelease()
}

// SetProgressReporter sets the progress reporter for sync operations
func (s *SyncService) SetProgressReporter(reporter progress.Reporter) {
	s.reporter = reporter
}

// getReporter returns the current progress reporter or a null reporter
func (s *SyncService) getReporter() progress.Reporter {
	if s.reporter != nil {
		return s.reporter
	}
	return progress.NullReporter{}
}

// getAdapter returns or creates an adapter for the given endpoint
func (s *SyncService) getAdapter(endpointName string) (adapter.Adapter, error) {
	if a, ok := s.adapters[endpointName]; ok {
		return a, nil
	}

	endpoint, err := s.config.GetEndpoint(endpointName)
	if err != nil {
		return nil, err
	}

	transport, err := s.config.GetTransport(endpoint.Transport)
	if err != nil {
		return nil, err
	}

	var a adapter.Adapter
	switch transport.Type {
	case domain.TransportLocal:
		a, err = local.New(endpoint.Root)
		if err != nil {
			return nil, fmt.Errorf("failed to create local adapter for %s: %w", endpointName, err)
		}
	case domain.TransportGDrive:
		// Get OAuth credentials from transport config
		clientID := transport.Config["client_id"]
		clientSecret := transport.Config["client_secret"]
		tokenPath := transport.Config["token_path"]

		if clientID == "" || clientSecret == "" {
			return nil, fmt.Errorf("gdrive transport requires client_id and client_secret in config")
		}

		ctx := context.Background()
		a, err = gdrive.New(ctx, clientID, clientSecret, tokenPath, endpoint.Root)
		if err != nil {
			return nil, fmt.Errorf("failed to create gdrive adapter for %s: %w", endpointName, err)
		}
	default:
		return nil, fmt.Errorf("unknown transport type: %s", transport.Type)
	}

	s.adapters[endpointName] = a
	return a, nil
}

// PlanSync creates a sync plan for a rule without executing it
func (s *SyncService) PlanSync(ctx context.Context, ruleName string) (*domain.SyncPlan, error) {
	logger.Get().Debug("planning sync", "rule", ruleName)

	rule, err := s.config.GetRule(ruleName)
	if err != nil {
		logger.Get().Error("failed to get rule", "rule", ruleName, "error", err)
		return nil, err
	}

	sourceAdapter, err := s.getAdapter(rule.SourceEndpoint)
	if err != nil {
		logger.Get().Error("failed to get source adapter",
			"rule", ruleName,
			"endpoint", rule.SourceEndpoint,
			"error", err,
		)
		return nil, fmt.Errorf("source endpoint: %w", err)
	}

	targetAdapter, err := s.getAdapter(rule.TargetEndpoint)
	if err != nil {
		logger.Get().Error("failed to get target adapter",
			"rule", ruleName,
			"endpoint", rule.TargetEndpoint,
			"error", err,
		)
		return nil, fmt.Errorf("target endpoint: %w", err)
	}

	logger.Get().Debug("delegating to executor", "rule", ruleName)

	// Delegate to core/rule executor
	plan, err := s.executor.Plan(ctx, rule, sourceAdapter, targetAdapter)
	if err != nil {
		logger.Get().Error("executor plan failed", "rule", ruleName, "error", err)
		return nil, err
	}

	logger.Get().Info("sync plan created",
		"rule", ruleName,
		"files_to_copy", plan.Stats.FilesToCopy,
		"files_to_delete", plan.Stats.FilesToDelete,
		"bytes_to_sync", plan.Stats.BytesToSync,
	)

	return plan, nil
}

// ExecuteSync executes a sync plan
func (s *SyncService) ExecuteSync(ctx context.Context, plan *domain.SyncPlan) error {
	logger.Get().Debug("executing sync", "rule", plan.RuleName)

	// Acquire lock before executing sync
	logger.Get().Info("acquiring lock", "rule", plan.RuleName)
	if err := s.lock.Acquire(plan.RuleName); err != nil {
		logger.Get().Error("failed to acquire sync lock", "rule", plan.RuleName, "error", err)
		return fmt.Errorf("failed to acquire sync lock: %w", err)
	}
	defer func() {
		if err := s.lock.Release(); err != nil {
			logger.Get().Error("failed to release sync lock", "rule", plan.RuleName, "error", err)
		}
	}()

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

	// Set total progress
	reporter := s.getReporter()
	reporter.SetTotal(plan.Stats.FilesToCopy, plan.Stats.BytesToSync)

	filesCompleted := 0
	bytesCompleted := int64(0)

	for _, action := range plan.Actions {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := s.executeAction(ctx, action, sourceAdapter, targetAdapter, reporter); err != nil {
			reporter.Error(err)
			return fmt.Errorf("action %s on %s: %w", action.Type, action.Path, err)
		}

		// Update overall progress
		if action.Type == domain.ActionCopy {
			filesCompleted++
			if action.SourceInfo != nil {
				bytesCompleted += action.SourceInfo.Size
			} else if action.TargetInfo != nil {
				bytesCompleted += action.TargetInfo.Size
			}
			reporter.OverallProgress(filesCompleted, bytesCompleted)

			logger.Get().Debug("action executed",
				"rule", plan.RuleName,
				"action", action.Type,
				"path", action.Path,
			)
		}
	}

	logger.Get().Info("sync execution completed",
		"rule", plan.RuleName,
		"files_synced", filesCompleted,
		"bytes_synced", bytesCompleted,
	)

	return nil
}

// executeAction performs a single sync action with correct direction
func (s *SyncService) executeAction(
	ctx context.Context,
	action domain.SyncAction,
	sourceAdapter, targetAdapter adapter.Adapter,
	reporter progress.Reporter,
) error {
	// Determine actual from/to adapters based on direction
	var fromAdapter, toAdapter adapter.Adapter
	if action.Direction == domain.DirSourceToTarget {
		fromAdapter = sourceAdapter
		toAdapter = targetAdapter
	} else {
		fromAdapter = targetAdapter
		toAdapter = sourceAdapter
	}

	switch action.Type {
	case domain.ActionCopy:
		// Get file size for progress reporting
		var fileSize int64
		if action.SourceInfo != nil {
			fileSize = action.SourceInfo.Size
		} else if action.TargetInfo != nil {
			fileSize = action.TargetInfo.Size
		}

		reporter.Start(action.Path, fileSize)

		reader, err := fromAdapter.Read(ctx, action.Path)
		if err != nil {
			reporter.Error(err)
			return err
		}
		defer reader.Close()

		// Wrap reader with progress tracking
		progressReader := progress.NewProgressReader(reader, reporter)

		if err := toAdapter.Write(ctx, action.Path, progressReader); err != nil {
			reporter.Error(err)
			return err
		}

		reporter.Complete()
		return nil

	case domain.ActionMkdir:
		return toAdapter.Mkdir(ctx, action.Path)

	case domain.ActionDelete:
		return toAdapter.Delete(ctx, action.Path)

	case domain.ActionSkip, domain.ActionConflict:
		return nil

	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
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

var _ io.Closer = (*SyncService)(nil)

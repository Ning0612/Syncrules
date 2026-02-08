package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Ning0612/Syncrules/internal/config"
	"github.com/Ning0612/Syncrules/internal/domain"
	"github.com/Ning0612/Syncrules/internal/scheduler"
	"github.com/Ning0612/Syncrules/internal/state"
)

// DaemonService manages the scheduled sync daemon
type DaemonService struct {
	mu        sync.RWMutex
	config    *config.Config
	scheduler scheduler.Scheduler
	syncSvc   *SyncService
	stateMgr  *state.Manager
}

// DaemonStatus represents the current daemon status
type DaemonStatus struct {
	Running        bool
	SchedulerStats *scheduler.Status
	LastExecution  *state.ExecutionRecord
}

// NewDaemonService creates a new daemon service
func NewDaemonService(cfg *config.Config) (*DaemonService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Create sync service
	syncSvc, err := NewSyncService(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create sync service: %w", err)
	}

	// Create state manager
	stateMgr, err := state.NewManager(cfg.GetLockPath())
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	daemon := &DaemonService{
		config:   cfg,
		syncSvc:  syncSvc,
		stateMgr: stateMgr,
	}

	// Create scheduler (if configured)
	// For now, we'll create it on-demand in Start() to allow for
	// configuration to be loaded
	return daemon, nil
}

// Start starts the daemon in the background
func (d *DaemonService) Start(ctx context.Context, interval time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.scheduler != nil {
		return fmt.Errorf("daemon is already running")
	}

	// Create sync runner that integrates with state manager
	runner := &syncRunner{
		config:   d.config,
		syncSvc:  d.syncSvc,
		stateMgr: d.stateMgr,
	}

	// Create interval scheduler
	schedConfig := scheduler.Config{
		Mode:     "interval",
		Interval: interval,
		Rules:    []string{}, // Empty = all enabled rules
	}

	sched, err := scheduler.NewIntervalScheduler(schedConfig, runner)
	if err != nil {
		return fmt.Errorf("failed to create scheduler: %w", err)
	}

	d.scheduler = sched

	// Start scheduler
	if err := sched.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	return nil
}

// Stop stops the daemon
func (d *DaemonService) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.scheduler == nil {
		return fmt.Errorf("daemon is not running")
	}

	if err := d.scheduler.Stop(); err != nil {
		return fmt.Errorf("failed to stop scheduler: %w", err)
	}

	d.scheduler = nil
	return nil
}

// Status returns the current daemon status
func (d *DaemonService) Status() *DaemonStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	status := &DaemonStatus{
		Running: d.scheduler != nil,
	}

	if d.scheduler != nil {
		status.SchedulerStats = d.scheduler.Status()
	}

	// Get last execution from state manager (for any rule)
	if d.stateMgr != nil {
		history, err := d.stateMgr.GetAllHistory(1)
		if err == nil && len(history) > 0 {
			status.LastExecution = &history[0]
		}
	}

	return status
}

// Close releases all resources
func (d *DaemonService) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var lastErr error

	if d.scheduler != nil {
		if err := d.scheduler.Stop(); err != nil {
			lastErr = err
		}
		d.scheduler = nil
	}

	if d.syncSvc != nil {
		if err := d.syncSvc.Close(); err != nil {
			lastErr = err
		}
	}

	if d.stateMgr != nil {
		if err := d.stateMgr.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// syncRunner implements scheduler.SyncRunner
type syncRunner struct {
	config   *config.Config
	syncSvc  *SyncService
	stateMgr *state.Manager
}

// RunSync executes a sync operation and records it in state
// It attempts to run all rules and aggregates errors instead of failing fast
func (r *syncRunner) RunSync(ctx context.Context, ruleName string) error {
	// Determine which rules to run
	var rules []domain.SyncRule
	if ruleName != "" {
		rule, err := r.config.GetRule(ruleName)
		if err != nil {
			return fmt.Errorf("rule not found: %s", ruleName)
		}
		rules = []domain.SyncRule{*rule}
	} else {
		rules = r.config.GetEnabledRules()
	}

	// Execute each rule and collect errors
	var errors []error
	for _, rule := range rules {
		record := state.ExecutionRecord{
			RuleName:  rule.Name,
			StartTime: time.Now(),
			Status:    "success",
		}

		// Generate plan
		plan, err := r.syncSvc.PlanSync(ctx, rule.Name)
		if err != nil {
			record.EndTime = time.Now()
			record.Status = "failed"
			record.Error = err.Error()
			r.stateMgr.SaveExecution(record)
			errors = append(errors, fmt.Errorf("rule %s: plan failed: %w", rule.Name, err))
			continue // Continue to next rule instead of returning
		}

		// Execute sync
		if err := r.syncSvc.ExecuteSync(ctx, plan); err != nil {
			record.EndTime = time.Now()
			record.Status = "failed"
			record.Error = err.Error()
			r.stateMgr.SaveExecution(record)
			errors = append(errors, fmt.Errorf("rule %s: execution failed: %w", rule.Name, err))
			continue // Continue to next rule
		}

		// Record successful execution
		record.EndTime = time.Now()
		record.FilesSynced = plan.Stats.FilesToCopy
		record.BytesSynced = plan.Stats.BytesToSync
		r.stateMgr.SaveExecution(record)
	}

	// Return aggregated errors if any
	if len(errors) > 0 {
		return fmt.Errorf("sync completed with %d error(s): %v", len(errors), errors)
	}
	return nil
}

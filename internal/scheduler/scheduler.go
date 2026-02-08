package scheduler

import (
	"context"
	"time"
)

// Scheduler defines the interface for sync schedulers
type Scheduler interface {
	// Start begins the scheduling loop
	Start(ctx context.Context) error

	// Stop gracefully stops the scheduler
	Stop() error

	// Status returns the current scheduler status
	Status() *Status
}

// Status represents the current state of a scheduler
type Status struct {
	Running        bool
	LastRunTime    time.Time
	NextRunTime    time.Time
	TotalRuns      int
	SuccessfulRuns int
	FailedRuns     int
	LastError      string
}

// Config contains scheduler configuration
type Config struct {
	// Mode specifies the scheduling mode ("interval" or "watch")
	Mode string

	// Interval specifies the duration between sync runs (for interval mode)
	Interval time.Duration

	// Rules specifies which rules to run (empty = all enabled rules)
	Rules []string
}

// SyncRunner is the interface that schedulers use to execute sync operations
type SyncRunner interface {
	// RunSync executes a sync operation for the specified rule
	RunSync(ctx context.Context, ruleName string) error
}

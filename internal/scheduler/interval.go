package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// IntervalScheduler implements periodic scheduling using time.Ticker
type IntervalScheduler struct {
	config Config
	runner SyncRunner

	// Runtime state
	mu          sync.RWMutex
	running     bool
	stopped     bool      // Track if stopped to prevent restart
	stopOnce    sync.Once // Ensure Stop() is idempotent
	closeOnce   sync.Once // Ensure stoppedChan is closed exactly once
	stopChan    chan struct{}
	stoppedChan chan struct{}

	// Statistics
	stats struct {
		lastRunTime    time.Time
		nextRunTime    time.Time
		totalRuns      int
		successfulRuns int
		failedRuns     int
		lastError      string
	}
}

// NewIntervalScheduler creates a new interval-based scheduler
func NewIntervalScheduler(config Config, runner SyncRunner) (*IntervalScheduler, error) {
	if config.Interval <= 0 {
		return nil, fmt.Errorf("interval must be positive, got %v", config.Interval)
	}

	if runner == nil {
		return nil, fmt.Errorf("sync runner cannot be nil")
	}

	return &IntervalScheduler{
		config:      config,
		runner:      runner,
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}, nil
}

// Start begins the scheduling loop
func (s *IntervalScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	if s.stopped {
		return fmt.Errorf("scheduler cannot be restarted after stop")
	}

	s.running = true
	s.stats.nextRunTime = time.Now().Add(s.config.Interval)

	// Start the scheduling loop in a goroutine
	go s.run(ctx)

	return nil
}

// run is the main scheduling loop
func (s *IntervalScheduler) run(ctx context.Context) {
	// Ensure stoppedChan is closed exactly once and stopped flag is set
	defer s.closeOnce.Do(func() {
		s.mu.Lock()
		s.stopped = true
		s.running = false
		s.mu.Unlock()
		close(s.stoppedChan)
	})

	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context cancelled - return gracefully
			return
		case <-s.stopChan:
			// Stop requested - return gracefully
			return
		case <-ticker.C:
			// Execute sync on schedule
			s.executeSync(ctx)
		}
	}
}

// executeSync runs sync for all configured rules
func (s *IntervalScheduler) executeSync(ctx context.Context) {
	s.mu.Lock()
	s.stats.lastRunTime = time.Now()
	s.stats.totalRuns++
	s.stats.nextRunTime = time.Now().Add(s.config.Interval)
	s.mu.Unlock()

	// Determine which rules to run
	rules := s.config.Rules
	if len(rules) == 0 {
		// No specific rules configured - runner should handle this
		// by running all enabled rules
		rules = []string{""}
	}

	// Execute sync for each rule
	hasError := false
	var lastErr error

	for _, ruleName := range rules {
		if err := s.runner.RunSync(ctx, ruleName); err != nil {
			hasError = true
			lastErr = err
		}
	}

	// Update statistics
	s.mu.Lock()
	if hasError {
		s.stats.failedRuns++
		if lastErr != nil {
			s.stats.lastError = lastErr.Error()
		}
	} else {
		s.stats.successfulRuns++
		s.stats.lastError = ""
	}
	s.mu.Unlock()
}

// Stop gracefully stops the scheduler
func (s *IntervalScheduler) Stop() error {
	s.mu.RLock()
	if !s.running {
		s.mu.RUnlock()
		return fmt.Errorf("scheduler is not running")
	}
	s.mu.RUnlock()

	// Use sync.Once to ensure stop channel is closed only once
	s.stopOnce.Do(func() {
		close(s.stopChan)
	})

	// Wait for scheduler to stop
	<-s.stoppedChan

	// Mark as stopped to prevent restart
	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()

	return nil
}

// Status returns the current scheduler status
func (s *IntervalScheduler) Status() *Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &Status{
		Running:        s.running,
		LastRunTime:    s.stats.lastRunTime,
		NextRunTime:    s.stats.nextRunTime,
		TotalRuns:      s.stats.totalRuns,
		SuccessfulRuns: s.stats.successfulRuns,
		FailedRuns:     s.stats.failedRuns,
		LastError:      s.stats.lastError,
	}
}

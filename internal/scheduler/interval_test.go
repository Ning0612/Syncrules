package scheduler

import (
	"context"
	"testing"
	"time"
)

// mockSyncRunner is a mock implementation of SyncRunner for testing
type mockSyncRunner struct {
	calls     []string
	shouldErr bool
	delay     time.Duration
}

func (m *mockSyncRunner) RunSync(ctx context.Context, ruleName string) error {
	m.calls = append(m.calls, ruleName)
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.shouldErr {
		return context.DeadlineExceeded
	}
	return nil
}

func TestNewIntervalScheduler(t *testing.T) {
	runner := &mockSyncRunner{}

	// Valid configuration
	config := Config{
		Mode:     "interval",
		Interval: 1 * time.Second,
	}

	scheduler, err := NewIntervalScheduler(config, runner)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	if scheduler == nil {
		t.Fatal("Scheduler is nil")
	}
}

func TestNewIntervalScheduler_InvalidInterval(t *testing.T) {
	runner := &mockSyncRunner{}

	config := Config{
		Mode:     "interval",
		Interval: 0, // Invalid
	}

	_, err := NewIntervalScheduler(config, runner)
	if err == nil {
		t.Error("Expected error for zero interval, got nil")
	}
}

func TestNewIntervalScheduler_NilRunner(t *testing.T) {
	config := Config{
		Mode:     "interval",
		Interval: 1 * time.Second,
	}

	_, err := NewIntervalScheduler(config, nil)
	if err == nil {
		t.Error("Expected error for nil runner, got nil")
	}
}

func TestIntervalScheduler_Start(t *testing.T) {
	runner := &mockSyncRunner{}
	config := Config{
		Mode:     "interval",
		Interval: 100 * time.Millisecond,
	}

	scheduler, err := NewIntervalScheduler(config, runner)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	// Check status
	status := scheduler.Status()
	if !status.Running {
		t.Error("Scheduler should be running")
	}

	// Wait for at least 2 runs
	time.Sleep(250 * time.Millisecond)

	status = scheduler.Status()
	if status.TotalRuns < 2 {
		t.Errorf("Expected at least 2 runs, got %d", status.TotalRuns)
	}

	// Stop scheduler
	cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestIntervalScheduler_Stop(t *testing.T) {
	runner := &mockSyncRunner{}
	config := Config{
		Mode:     "interval",
		Interval: 100 * time.Millisecond,
	}

	scheduler, err := NewIntervalScheduler(config, runner)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	ctx := context.Background()

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	// Wait a bit
	time.Sleep(150 * time.Millisecond)

	// Stop scheduler
	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("Failed to stop scheduler: %v", err)
	}

	// Check status
	status := scheduler.Status()
	if status.Running {
		t.Error("Scheduler should not be running after stop")
	}
}

func TestIntervalScheduler_DoubleStart(t *testing.T) {
	runner := &mockSyncRunner{}
	config := Config{
		Mode:     "interval",
		Interval: 1 * time.Second,
	}

	scheduler, err := NewIntervalScheduler(config, runner)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	ctx := context.Background()

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()

	// Try to start again
	err = scheduler.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running scheduler")
	}
}

func TestIntervalScheduler_StopNotRunning(t *testing.T) {
	runner := &mockSyncRunner{}
	config := Config{
		Mode:     "interval",
		Interval: 1 * time.Second,
	}

	scheduler, err := NewIntervalScheduler(config, runner)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	// Try to stop without starting
	err = scheduler.Stop()
	if err == nil {
		t.Error("Expected error when stopping non-running scheduler")
	}
}

func TestIntervalScheduler_ContextCancellation(t *testing.T) {
	runner := &mockSyncRunner{}
	config := Config{
		Mode:     "interval",
		Interval: 100 * time.Millisecond,
	}

	scheduler, err := NewIntervalScheduler(config, runner)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	// Wait a bit
	time.Sleep(150 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for scheduler to stop
	time.Sleep(100 * time.Millisecond)

	// Check status
	status := scheduler.Status()
	if status.Running {
		t.Error("Scheduler should stop when context is cancelled")
	}
}

func TestIntervalScheduler_ErrorHandling(t *testing.T) {
	runner := &mockSyncRunner{shouldErr: true}
	config := Config{
		Mode:     "interval",
		Interval: 100 * time.Millisecond,
	}

	scheduler, err := NewIntervalScheduler(config, runner)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	// Wait for at least 2 runs
	time.Sleep(250 * time.Millisecond)

	status := scheduler.Status()
	if status.FailedRuns == 0 {
		t.Error("Expected failed runs when runner returns error")
	}

	if status.LastError == "" {
		t.Error("Expected last error to be set")
	}

	cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestIntervalScheduler_Statistics(t *testing.T) {
	runner := &mockSyncRunner{}
	config := Config{
		Mode:     "interval",
		Interval: 50 * time.Millisecond,
	}

	scheduler, err := NewIntervalScheduler(config, runner)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	// Wait for multiple runs
	time.Sleep(150 * time.Millisecond)

	status := scheduler.Status()

	if status.TotalRuns == 0 {
		t.Error("Expected total runs > 0")
	}

	if status.SuccessfulRuns == 0 {
		t.Error("Expected successful runs > 0")
	}

	if !status.LastRunTime.IsZero() {
		// Last run time should be recent
		if time.Since(status.LastRunTime) > 200*time.Millisecond {
			t.Error("Last run time seems too old")
		}
	}

	if status.NextRunTime.IsZero() {
		t.Error("Next run time should be set")
	}

	cancel()
	time.Sleep(50 * time.Millisecond)
}

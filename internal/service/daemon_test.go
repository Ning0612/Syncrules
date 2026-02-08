package service

import (
	"context"
	"testing"
	"time"

	"github.com/Ning0612/Syncrules/internal/config"
	"github.com/Ning0612/Syncrules/internal/domain"
)

// mockConfig creates a minimal config for testing
func mockDaemonConfig() *config.Config {
	return &config.Config{
		Transports: []domain.Transport{
			{
				Name: "local",
				Type: domain.TransportLocal,
			},
		},
		Endpoints: []domain.Endpoint{
			{
				Name:      "source",
				Transport: "local",
				Root:      "/tmp/test-source",
			},
			{
				Name:      "target",
				Transport: "local",
				Root:      "/tmp/test-target",
			},
		},
		Rules: []domain.SyncRule{
			{
				Name:             "test-rule",
				Mode:             domain.SyncModeTwoWay,
				SourceEndpoint:   "source",
				TargetEndpoint:   "target",
				ConflictStrategy: domain.ConflictKeepNewest,
				Enabled:          true,
			},
		},
		Settings: config.Settings{
			LockPath: "",
		},
	}
}

func TestNewDaemonService(t *testing.T) {
	cfg := mockDaemonConfig()

	daemon, err := NewDaemonService(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon service: %v", err)
	}
	defer daemon.Close()

	if daemon == nil {
		t.Fatal("Daemon service is nil")
	}

	if daemon.config == nil {
		t.Error("Daemon config is nil")
	}

	if daemon.syncSvc == nil {
		t.Error("Sync service is nil")
	}

	if daemon.stateMgr == nil {
		t.Error("State manager is nil")
	}
}

func TestNewDaemonService_NilConfig(t *testing.T) {
	_, err := NewDaemonService(nil)
	if err == nil {
		t.Error("Expected error for nil config, got nil")
	}
}

func TestDaemonService_StartStop(t *testing.T) {
	cfg := mockDaemonConfig()

	daemon, err := NewDaemonService(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon service: %v", err)
	}
	defer daemon.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start daemon with short interval
	err = daemon.Start(ctx, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}

	// Check status
	status := daemon.Status()
	if !status.Running {
		t.Error("Daemon should be running")
	}

	// Wait a bit for at least one execution
	time.Sleep(200 * time.Millisecond)

	// Stop daemon
	err = daemon.Stop()
	if err != nil {
		t.Fatalf("Failed to stop daemon: %v", err)
	}

	// Check status again
	status = daemon.Status()
	if status.Running {
		t.Error("Daemon should not be running after stop")
	}
}

func TestDaemonService_DoubleStart(t *testing.T) {
	cfg := mockDaemonConfig()

	daemon, err := NewDaemonService(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon service: %v", err)
	}
	defer daemon.Close()

	ctx := context.Background()

	err = daemon.Start(ctx, 1*time.Second)
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}
	defer daemon.Stop()

	// Try to start again
	err = daemon.Start(ctx, 1*time.Second)
	if err == nil {
		t.Error("Expected error when starting already running daemon")
	}
}

func TestDaemonService_StopNotRunning(t *testing.T) {
	cfg := mockDaemonConfig()

	daemon, err := NewDaemonService(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon service: %v", err)
	}
	defer daemon.Close()

	// Try to stop without starting
	err = daemon.Stop()
	if err == nil {
		t.Error("Expected error when stopping non-running daemon")
	}
}

func TestDaemonService_Status(t *testing.T) {
	cfg := mockDaemonConfig()

	daemon, err := NewDaemonService(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon service: %v", err)
	}
	defer daemon.Close()

	// Status before starting
	status := daemon.Status()
	if status == nil {
		t.Fatal("Status should not be nil")
	}

	if status.Running {
		t.Error("Daemon should not be running initially")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start daemon
	err = daemon.Start(ctx, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}
	defer daemon.Stop()

	// Status while running
	status = daemon.Status()
	if !status.Running {
		t.Error("Daemon should be running")
	}

	if status.SchedulerStats == nil {
		t.Error("Scheduler stats should not be nil when running")
	}

	// Wait for some executions
	time.Sleep(250 * time.Millisecond)

	status = daemon.Status()
	if status.SchedulerStats.TotalRuns == 0 {
		t.Error("Expected totalruns > 0")
	}

	cancel()
	time.Sleep(50 * time.Millisecond)
}

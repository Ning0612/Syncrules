package daemon_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ning0612/Syncrules/internal/daemon"
)

func TestPIDFile_WriteAndRead(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "test.pid")

	pidFile := daemon.NewPIDFile(pidPath)

	// Write PID file
	err := pidFile.Write()
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	// Read PID file
	pid, err := pidFile.Read()
	if err != nil {
		t.Fatalf("Failed to read PID file: %v", err)
	}

	// Verify PID matches current process
	if pid != os.Getpid() {
		t.Errorf("Expected PID %d, got %d", os.Getpid(), pid)
	}

	// Clean up
	pidFile.Remove()
}

func TestPIDFile_IsRunning(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "test.pid")

	pidFile := daemon.NewPIDFile(pidPath)

	// Write PID for current process
	err := pidFile.Write()
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}
	defer pidFile.Remove()

	// Check if running (should be true for current process)
	running, err := pidFile.IsRunning()
	if err != nil {
		t.Fatalf("Failed to check if running: %v", err)
	}

	if !running {
		t.Error("Expected process to be running")
	}
}

func TestPIDFile_WriteExisting(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "test.pid")

	pidFile := daemon.NewPIDFile(pidPath)

	// Write first time
	err := pidFile.Write()
	if err != nil {
		t.Fatalf("Failed to write PID file first time: %v", err)
	}
	defer pidFile.Remove()

	// Try to write again (should fail since process is still running)
	err = pidFile.Write()
	if err == nil {
		t.Error("Expected error when writing PID file for running process")
	}
}

func TestPIDFile_StalePIDCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "test.pid")

	pidFile := daemon.NewPIDFile(pidPath)

	// Write a fake PID for a non-existent process
	fakePID := 99999
	err := os.WriteFile(pidPath, []byte(string(rune(fakePID))+"\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write fake PID: %v", err)
	}

	// Write should succeed and remove stale PID file
	err = pidFile.Write()
	if err != nil {
		t.Fatalf("Failed to write after stale PID: %v", err)
	}
	defer pidFile.Remove()

	// Verify current PID is written
	pid, err := pidFile.Read()
	if err != nil {
		t.Fatalf("Failed to read PID: %v", err)
	}

	if pid != os.Getpid() {
		t.Errorf("Expected current PID %d, got %d", os.Getpid(), pid)
	}
}

func TestDefaultPIDPath(t *testing.T) {
	path, err := daemon.DefaultPIDPath()
	if err != nil {
		t.Fatalf("Failed to get default PID path: %v", err)
	}

	// Verify path is not empty
	if path == "" {
		t.Error("Expected non-empty PID path")
	}

	// Verify directory is created
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("PID directory was not created: %s", dir)
	}
}

func TestPIDFile_Remove(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "test.pid")

	pidFile := daemon.NewPIDFile(pidPath)

	// Write PID file
	err := pidFile.Write()
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	// Remove PID file
	err = pidFile.Remove()
	if err != nil {
		t.Fatalf("Failed to remove PID file: %v", err)
	}

	// Verify file is removed
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("PID file still exists after removal")
	}

	// Removing again should not error
	err = pidFile.Remove()
	if err != nil {
		t.Errorf("Expected no error when removing non-existent PID file, got: %v", err)
	}
}

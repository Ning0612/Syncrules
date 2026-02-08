package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// PIDFile manages the daemon process ID file
type PIDFile struct {
	path string
}

// NewPIDFile creates a new PID file manager
func NewPIDFile(path string) *PIDFile {
	return &PIDFile{path: path}
}

// DefaultPIDPath returns the default PID file path
func DefaultPIDPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	pidDir := filepath.Join(homeDir, ".config", "syncrules")
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create PID directory: %w", err)
	}

	return filepath.Join(pidDir, "daemon.pid"), nil
}

// Write writes the current process ID to the PID file
func (p *PIDFile) Write() error {
	// Check if PID file already exists
	if _, err := os.Stat(p.path); err == nil {
		// PID file exists, check if process is running
		if running, _ := p.IsRunning(); running {
			return fmt.Errorf("daemon is already running (PID file exists: %s)", p.path)
		}
		// Stale PID file, remove it
		os.Remove(p.path)
	}

	pid := os.Getpid()
	content := fmt.Sprintf("%d\n", pid)

	if err := os.WriteFile(p.path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

// Read reads the PID from the PID file
func (p *PIDFile) Read() (int, error) {
	content, err := os.ReadFile(p.path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("PID file does not exist: %s", p.path)
		}
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	pidStr := strings.TrimSpace(string(content))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %s", pidStr)
	}

	return pid, nil
}

// Remove removes the PID file
func (p *PIDFile) Remove() error {
	if err := os.Remove(p.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}
	return nil
}

// IsRunning checks if the process in the PID file is running
func (p *PIDFile) IsRunning() (bool, error) {
	pid, err := p.Read()
	if err != nil {
		return false, err
	}

	return isProcessRunning(pid), nil
}

// Kill sends a termination signal to the process in the PID file
func (p *PIDFile) Kill() error {
	pid, err := p.Read()
	if err != nil {
		return err
	}

	return killProcess(pid)
}

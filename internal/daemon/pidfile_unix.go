//go:build !windows
// +build !windows

package daemon

import (
	"fmt"
	"os"
	"syscall"
)

// isProcessRunning checks if a process is running on Unix systems
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists without actually sending a signal
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// killProcess sends a termination signal to a process on Unix systems
func killProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to signal process %d: %w", pid, err)
	}

	return nil
}

//go:build !windows

package lock

import (
	"errors"
	"os"
	"syscall"
)

// processExists checks if a process with the given PID exists
func processExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds, so we need to send signal 0
	// to check if the process exists
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	// EPERM means process exists but we don't have permission to signal it
	// This is still a valid running process
	if errors.Is(err, syscall.EPERM) {
		return true
	}
	// ESRCH means no such process
	return false
}

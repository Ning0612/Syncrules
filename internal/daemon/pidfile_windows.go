//go:build windows
// +build windows

package daemon

import (
	"fmt"
	"os"
	"syscall"
)

// isProcessRunning checks if a process is running on Windows
func isProcessRunning(pid int) bool {
	// Try to open the process with PROCESS_QUERY_INFORMATION
	h, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)

	// Check if process has exited
	var exitCode uint32
	err = syscall.GetExitCodeProcess(h, &exitCode)
	if err != nil {
		return false
	}

	// STILL_ACTIVE = 259
	return exitCode == 259
}

// killProcess sends a termination signal to a process on Windows
func killProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	// On Windows, Kill sends SIGKILL which terminates immediately
	if err := process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process %d: %w", pid, err)
	}

	return nil
}

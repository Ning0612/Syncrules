//go:build windows

package lock

import (
	"errors"

	"golang.org/x/sys/windows"
)

// processExists checks if a process with the given PID exists on Windows
func processExists(pid int) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		// ERROR_ACCESS_DENIED (5) means process exists but we lack permission
		// This is still a valid running process, so return true
		// Use errors.Is for robust error type checking
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return true
		}
		// Other errors (e.g., ERROR_INVALID_PARAMETER) mean process doesn't exist
		return false
	}
	windows.CloseHandle(handle)
	return true
}

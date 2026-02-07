package testutil

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TempDir creates a temporary directory for testing
// It returns the directory path and a cleanup function
func TempDir(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "syncrules-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return dir, cleanup
}

// CreateTestFile creates a test file with the given content
func CreateTestFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	return path
}

// CreateTestFileWithSize creates a test file with random content of the given size
func CreateTestFileWithSize(t *testing.T, dir, name string, size int64) string {
	t.Helper()

	path := filepath.Join(dir, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer file.Close()

	// Write random data in chunks
	const chunkSize = 1024 * 1024 // 1MB chunks
	buf := make([]byte, chunkSize)
	remaining := size

	for remaining > 0 {
		writeSize := chunkSize
		if remaining < int64(chunkSize) {
			writeSize = int(remaining)
		}

		rand.Read(buf[:writeSize])
		if _, err := file.Write(buf[:writeSize]); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		remaining -= int64(writeSize)
	}

	return path
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(timeout time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return true
		}

		if time.Now().After(deadline) {
			return false
		}

		<-ticker.C
	}
}

// AssertEventually asserts that a condition becomes true within timeout
func AssertEventually(t *testing.T, timeout time.Duration, condition func() bool, msgAndArgs ...interface{}) {
	t.Helper()

	if !WaitForCondition(timeout, condition) {
		if len(msgAndArgs) > 0 {
			t.Fatalf("condition not met within %v: %v", timeout, msgAndArgs[0])
		} else {
			t.Fatalf("condition not met within %v", timeout)
		}
	}
}

// RandomString generates a random string of the given length
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

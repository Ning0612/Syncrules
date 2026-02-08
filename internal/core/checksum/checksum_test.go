package checksum

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestMD5Calculation tests MD5 checksum computation
func TestMD5Calculation(t *testing.T) {
	calc := NewDefaultCalculator()
	ctx := context.Background()

	// Test vector: "hello world"
	input := strings.NewReader("hello world")
	expected := "5eb63bbbe01eeed093cb22bb8f5acdc3" // Known MD5 of "hello world"

	result, err := calc.Calculate(ctx, input, MD5)
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}

	if result != expected {
		t.Errorf("MD5 mismatch: got %s, want %s", result, expected)
	}
}

// TestSHA256Calculation tests SHA256 checksum computation
func TestSHA256Calculation(t *testing.T) {
	calc := NewDefaultCalculator()
	ctx := context.Background()

	// Test vector: "hello world"
	input := strings.NewReader("hello world")
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9" // Known SHA256

	result, err := calc.Calculate(ctx, input, SHA256)
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}

	if result != expected {
		t.Errorf("SHA256 mismatch: got %s, want %s", result, expected)
	}
}

// TestEmptyFile tests checksum of empty content
func TestEmptyFile(t *testing.T) {
	calc := NewDefaultCalculator()
	ctx := context.Background()

	input := strings.NewReader("")
	expectedMD5 := "d41d8cd98f00b204e9800998ecf8427e"                                    // MD5 of empty string
	expectedSHA256 := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // SHA256 of empty string

	// Test MD5
	result, err := calc.Calculate(ctx, input, MD5)
	if err != nil {
		t.Fatalf("MD5 Calculate failed: %v", err)
	}
	if result != expectedMD5 {
		t.Errorf("MD5 empty file mismatch: got %s, want %s", result, expectedMD5)
	}

	// Test SHA256
	input = strings.NewReader("")
	result, err = calc.Calculate(ctx, input, SHA256)
	if err != nil {
		t.Fatalf("SHA256 Calculate failed: %v", err)
	}
	if result != expectedSHA256 {
		t.Errorf("SHA256 empty file mismatch: got %s, want %s", result, expectedSHA256)
	}
}

// TestMaxSizeLimit tests that files exceeding MaxSize return an error
func TestMaxSizeLimit(t *testing.T) {
	opts := Options{
		MaxSize:    10, // Only allow 10 bytes
		BufferSize: 4096,
	}
	calc := NewCalculator(opts)
	ctx := context.Background()

	// Create input larger than MaxSize
	input := strings.NewReader("this is a long string that exceeds 10 bytes")

	_, err := calc.Calculate(ctx, input, SHA256)
	if err == nil {
		t.Fatal("Expected error for file exceeding MaxSize, got nil")
	}

	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("Expected 'exceeds maximum' error, got: %v", err)
	}
}

// TestContextCancellation tests that calculation respects context cancellation
func TestContextCancellation(t *testing.T) {
	calc := NewDefaultCalculator()

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	input := strings.NewReader("some data")

	_, err := calc.Calculate(ctx, input, SHA256)
	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
}

// TestContextTimeout tests that calculation respects context timeout
func TestContextTimeout(t *testing.T) {
	calc := NewCalculator(Options{
		MaxSize:    0,
		BufferSize: 1, // Very small buffer to slow down processing
	})

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Create a large input
	input := strings.NewReader(strings.Repeat("a", 10000))

	time.Sleep(2 * time.Millisecond) // Ensure timeout has passed

	_, err := calc.Calculate(ctx, input, SHA256)
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	if err != context.DeadlineExceeded {
		t.Logf("Note: Got %v instead of DeadlineExceeded (may pass quickly)", err)
	}
}

// TestUnsupportedAlgorithm tests error handling for unsupported algorithms
func TestUnsupportedAlgorithm(t *testing.T) {
	calc := NewDefaultCalculator()
	ctx := context.Background()

	input := strings.NewReader("test")

	_, err := calc.Calculate(ctx, input, Algorithm("invalid"))
	if err == nil {
		t.Fatal("Expected error for unsupported algorithm, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported algorithm") {
		t.Errorf("Expected 'unsupported algorithm' error, got: %v", err)
	}
}

// TestIsSupported tests the IsSupported function
func TestIsSupported(t *testing.T) {
	tests := []struct {
		algo     Algorithm
		expected bool
	}{
		{MD5, true},
		{SHA256, true},
		{Algorithm("sha1"), false},
		{Algorithm(""), false},
	}

	for _, tt := range tests {
		result := IsSupported(tt.algo)
		if result != tt.expected {
			t.Errorf("IsSupported(%s) = %v, want %v", tt.algo, result, tt.expected)
		}
	}
}

// TestLargeFileStreaming tests that large files are handled via streaming
func TestLargeFileStreaming(t *testing.T) {
	calc := NewDefaultCalculator()
	ctx := context.Background()

	// Create a 1MB string
	largeContent := strings.Repeat("a", 1024*1024)
	input := strings.NewReader(largeContent)

	result, err := calc.Calculate(ctx, input, SHA256)
	if err != nil {
		t.Fatalf("Calculate failed for large file: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty checksum for large file")
	}

	// Verify determinism - same input should give same output
	input2 := strings.NewReader(largeContent)
	result2, err := calc.Calculate(ctx, input2, SHA256)
	if err != nil {
		t.Fatalf("Second calculate failed: %v", err)
	}

	if result != result2 {
		t.Errorf("Checksums should be identical: %s != %s", result, result2)
	}
}

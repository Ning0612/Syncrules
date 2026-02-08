package checksum

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
)

// Algorithm represents the hashing algorithm to use
type Algorithm string

const (
	// MD5 algorithm (faster but less secure, suitable for content comparison)
	MD5 Algorithm = "md5"
	// SHA256 algorithm (more secure, recommended default)
	SHA256 Algorithm = "sha256"
)

// Options configures the checksum calculator
type Options struct {
	// MaxSize: files larger than this will not be checksummed (0 = unlimited)
	// Default: 100MB to avoid performance issues
	MaxSize int64

	// BufferSize: size of buffer for streaming reads
	// Default: 32KB
	BufferSize int
}

// DefaultOptions returns the recommended default options
func DefaultOptions() Options {
	return Options{
		MaxSize:    100 * 1024 * 1024, // 100MB
		BufferSize: 32 * 1024,         // 32KB
	}
}

// Calculator computes file checksums
type Calculator interface {
	// Calculate computes checksum from an io.Reader
	// Returns empty string if size exceeds MaxSize or if context is cancelled
	Calculate(ctx context.Context, reader io.Reader, algo Algorithm) (string, error)
}

// DefaultCalculator implements Calculator with streaming support
type DefaultCalculator struct {
	opts Options
}

// NewCalculator creates a new calculator with the given options
func NewCalculator(opts Options) *DefaultCalculator {
	return &DefaultCalculator{opts: opts}
}

// NewDefaultCalculator creates a calculator with default options
func NewDefaultCalculator() *DefaultCalculator {
	return NewCalculator(DefaultOptions())
}

// Calculate implements the Calculator interface
func (c *DefaultCalculator) Calculate(ctx context.Context, reader io.Reader, algo Algorithm) (string, error) {
	// Create hasher based on algorithm
	var h hash.Hash
	switch algo {
	case MD5:
		h = md5.New()
	case SHA256:
		h = sha256.New()
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", algo)
	}

	// Create a limited reader if MaxSize is set
	var limitedReader io.Reader = reader
	if c.opts.MaxSize > 0 {
		limitedReader = io.LimitReader(reader, c.opts.MaxSize+1)
	}

	// Stream the data through the hasher
	buffer := make([]byte, c.opts.BufferSize)
	totalBytes := int64(0)

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// Read next chunk
		n, err := limitedReader.Read(buffer)
		if n > 0 {
			totalBytes += int64(n)

			// Check if we exceeded MaxSize
			if c.opts.MaxSize > 0 && totalBytes > c.opts.MaxSize {
				return "", fmt.Errorf("file size exceeds maximum (%d bytes)", c.opts.MaxSize)
			}

			// Write to hasher
			if _, hashErr := h.Write(buffer[:n]); hashErr != nil {
				return "", fmt.Errorf("hash write error: %w", hashErr)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read error: %w", err)
		}
	}

	// Return hex-encoded hash
	return hex.EncodeToString(h.Sum(nil)), nil
}

// IsSupported checks if the given algorithm is supported
func IsSupported(algo Algorithm) bool {
	switch algo {
	case MD5, SHA256:
		return true
	default:
		return false
	}
}

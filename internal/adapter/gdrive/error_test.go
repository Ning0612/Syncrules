package gdrive

import (
	"errors"
	"testing"

	"google.golang.org/api/googleapi"

	"github.com/Ning0612/Syncrules/internal/domain"
)

// TestMapError tests error mapping from Google API errors to domain errors
func TestMapError(t *testing.T) {
	adapter := &Adapter{}

	tests := []struct {
		name  string
		input error
		want  error
	}{
		{
			name:  "nil error",
			input: nil,
			want:  nil,
		},
		{
			name:  "404 not found",
			input: &googleapi.Error{Code: 404},
			want:  domain.ErrNotFound,
		},
		{
			name:  "403 permission denied",
			input: &googleapi.Error{Code: 403},
			want:  domain.ErrPermissionDenied,
		},
		{
			name:  "409 already exists",
			input: &googleapi.Error{Code: 409},
			want:  domain.ErrAlreadyExists,
		},
		{
			name:  "429 rate limit (wrapped)",
			input: &googleapi.Error{Code: 429},
			want:  nil, // Should contain "rate limit exceeded" string
		},
		{
			name:  "500 internal server error (passthrough)",
			input: &googleapi.Error{Code: 500, Message: "server error"},
			want:  nil, // Should return original error
		},
		{
			name:  "non-googleapi error with notFound string",
			input: errors.New("file notFound in drive"),
			want:  domain.ErrNotFound,
		},
		{
			name:  "generic error",
			input: errors.New("generic error"),
			want:  nil, // Should return original error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.mapError(tt.input)

			if tt.want == nil {
				// Special cases: nil input, rate limit, and passthrough
				if tt.name == "nil error" {
					if got != nil {
						t.Errorf("mapError(nil) = %v, want nil", got)
					}
				} else if tt.name == "429 rate limit (wrapped)" {
					if got == nil {
						t.Error("mapError(429) should return error")
					} else if !contains(got.Error(), "rate limit exceeded") {
						t.Errorf("mapError(429) should contain 'rate limit exceeded', got %v", got)
					}
					// Verify error wrapping preserves original error
					if !errors.Is(got, tt.input) {
						t.Errorf("mapError(429) should wrap original error, but errors.Is failed")
					}
				} else if tt.name == "500 internal server error (passthrough)" || tt.name == "generic error" {
					// Should return original error
					if got != tt.input {
						t.Errorf("mapError() should return original error, got %v, want %v", got, tt.input)
					}
				}
			} else {
				// Check if error matches expected domain error
				if !errors.Is(got, tt.want) {
					t.Errorf("mapError() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

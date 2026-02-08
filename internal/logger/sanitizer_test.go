package logger

import (
	"testing"
)

func TestSanitizer_Sanitize(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "password",
			input:    "login with password=secret123",
			expected: "login with password=***",
		},
		{
			name:     "token",
			input:    "auth token=abc123xyz",
			expected: "auth token=***",
		},
		{
			name:     "bearer token",
			input:    "Authorization: Bearer eyJhbGc...",
			expected: "Authorization: bearer ***",
		},
		{
			name:     "windows user path",
			input:    "file at C:\\Users\\john\\Documents\\file.txt",
			expected: "file at ***:\\Users\\***\\Documents\\file.txt",
		},
		{
			name:     "unix home path",
			input:    "config in /home/john/.config/app",
			expected: "config in /home/***/.config/app",
		},
		{
			name:     "email partial mask",
			input:    "user email: john.doe@example.com",
			expected: "user email: joh***@example.com",
		},
		{
			name:     "no sensitive data",
			input:    "normal log message",
			expected: "normal log message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.Sanitize(tt.input)
			if result != tt.expected {
				t.Errorf("Sanitize() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSanitizer_SanitizeArgs(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    []any
		validate func([]any) bool
	}{
		{
			name:  "password key-value",
			input: []any{"user", "john", "password", "secret123"},
			validate: func(result []any) bool {
				// password 值應該被遮蔽
				return len(result) == 4 && result[3] != "secret123"
			},
		},
		{
			name:  "token in string",
			input: []any{"msg", "token=abc123"},
			validate: func(result []any) bool {
				// "msg" 不是敏感鍵，值不會被遮罩
				return len(result) == 2
			},
		},
		{
			name:  "no sensitive data",
			input: []any{"file", "test.txt", "size", 1024},
			validate: func(result []any) bool {
				return len(result) == 4
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeArgs(tt.input)
			if !tt.validate(result) {
				t.Errorf("SanitizeArgs() validation failed for %v", result)
			}
		})
	}
}

func TestSanitizer_AddRule(t *testing.T) {
	s := NewSanitizer()

	// Add custom rule
	err := s.AddRule(`SSN=\d{3}-\d{2}-\d{4}`, "SSN=***")
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	input := "User SSN=123-45-6789 registered"
	expected := "User SSN=*** registered"
	result := s.Sanitize(input)

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestSanitizer_MaskValue(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		input    string
		expected string
	}{
		{"ab", "***"},
		{"abc", "a***"},
		{"abcdefgh", "a***"},
		{"abcdefghi", "a***i"},
		{"verylongpassword", "v***d"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := s.maskValue(tt.input)
			if result != tt.expected {
				t.Errorf("maskValue(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizer_IsSensitiveKey(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		input    string
		expected bool
	}{
		{"password", true},
		{"user_password", true},
		{"PASSWORD", true},
		{"token", true},
		{"api_key", true},
		{"username", false},
		{"file", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := s.isSensitiveKey(tt.input)
			if result != tt.expected {
				t.Errorf("isSensitiveKey(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

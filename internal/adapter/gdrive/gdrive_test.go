package gdrive

import (
	"strings"
	"sync"
	"testing"
)

// TestNormalizeRoot tests path normalization
func TestNormalizeRoot(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},  // Empty/root becomes empty
		{"/", ""}, // Root becomes empty
		{"folder", "/folder"},
		{"/folder", "/folder"},
		{"/folder/", "/folder"},
	}

	for _, tt := range tests {
		got := normalizeRoot(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeRoot(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// TestEscapeQueryString tests query string escaping for injection prevention
func TestEscapeQueryString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"file'name", "file\\'name"},                 // Single quote escaped
		{"file's name", "file\\'s name"},             // Single quote escaped
		{"file''name", "file\\'\\'name"},             // Multiple quotes
		{"no'special\"chars", "no\\'special\"chars"}, // Only single quotes escaped
	}

	for _, tt := range tests {
		got := escapeQueryString(tt.input)
		if got != tt.expected {
			t.Errorf("escapeQueryString(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// TestSecurity_QueryInjection tests protection against query injection
func TestSecurity_QueryInjection(t *testing.T) {
	// Test malicious filenames that could break Drive API queries
	maliciousNames := []string{
		"file' or '1'='1",
		"'; DROP TABLE files; --",
		"file' AND trashed=false AND '1'='1",
	}

	for _, name := range maliciousNames {
		escaped := escapeQueryString(name)

		// Verify single quotes are escaped
		if !strings.Contains(escaped, "\\'") && strings.Contains(name, "'") {
			t.Errorf("Query injection vulnerability: %q not properly escaped", name)
		}

		// Verify no unescaped single quotes remain
		unescaped := strings.ReplaceAll(escaped, "\\'", "")
		if strings.Contains(unescaped, "'") {
			t.Errorf("Unescaped single quote found in %q", escaped)
		}
	}
}

// TestJoinPath tests path joining
func TestJoinPath(t *testing.T) {
	adapter := &Adapter{
		root: "/test-root",
	}

	tests := []struct {
		relPath     string
		expectError bool
		expected    string
	}{
		{"file.txt", false, "/test-root/file.txt"},
		{"folder/file.txt", false, "/test-root/folder/file.txt"},
		{"/file.txt", true, ""}, // Absolute paths rejected
		{"", false, "/test-root"},
	}

	for _, tt := range tests {
		got, err := adapter.joinPath(tt.relPath)
		if tt.expectError && err == nil {
			t.Errorf("joinPath(%q) expected error, got none", tt.relPath)
		}
		if !tt.expectError && err != nil {
			t.Errorf("joinPath(%q) unexpected error: %v", tt.relPath, err)
		}
		if !tt.expectError && got != tt.expected {
			t.Errorf("joinPath(%q) = %q, want %q", tt.relPath, got, tt.expected)
		}
	}
}

// TestSecurity_PathTraversal tests protection against path traversal attacks
func TestSecurity_PathTraversal(t *testing.T) {
	adapter := &Adapter{
		root: "/safe-root",
	}

	maliciousPaths := []string{
		"../../../etc/passwd",
		"folder/../../outside",
		"./../../escape",
		"..\\..\\windows\\system32", // Windows-style
	}

	for _, path := range maliciousPaths {
		result, err := adapter.joinPath(path)
		if err != nil {
			// Error is acceptable - path validation rejected it
			continue
		}

		// If no error, verify result is still within root
		if !strings.HasPrefix(result, adapter.root) {
			t.Errorf("Path traversal vulnerability: %q resulted in %q (outside root %q)",
				path, result, adapter.root)
		}
	}
}

// TestIDCache_Get tests cache retrieval
func TestIDCache_Get(t *testing.T) {
	cache := newIDCache()

	// Empty cache
	_, ok := cache.get("/test")
	if ok {
		t.Error("expected cache miss for empty cache")
	}

	// Add item
	cache.set("/test", "id-123")
	id, ok := cache.get("/test")
	if !ok {
		t.Error("expected cache hit")
	}
	if id != "id-123" {
		t.Errorf("expected id 'id-123', got %q", id)
	}
}

// TestIDCache_Set tests cache storage
func TestIDCache_Set(t *testing.T) {
	cache := newIDCache()

	cache.set("/path1", "id1")
	cache.set("/path2", "id2")

	id1, _ := cache.get("/path1")
	id2, _ := cache.get("/path2")

	if id1 != "id1" || id2 != "id2" {
		t.Error("cache set/get mismatch")
	}

	// Overwrite
	cache.set("/path1", "id1-new")
	id1New, _ := cache.get("/path1")
	if id1New != "id1-new" {
		t.Error("cache overwrite failed")
	}
}

// TestIDCache_Delete tests cache deletion
func TestIDCache_Delete(t *testing.T) {
	cache := newIDCache()

	cache.set("/test", "id-123")
	cache.delete("/test")

	_, ok := cache.get("/test")
	if ok {
		t.Error("expected cache miss after delete")
	}
}

// TestSecurity_CacheConcurrency tests cache thread safety
func TestSecurity_CacheConcurrency(t *testing.T) {
	cache := newIDCache()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			path := strings.Repeat("x", idx%10)
			cache.set(path, "id")
			cache.get(path)
			if idx%2 == 0 {
				cache.delete(path)
			}
		}(i)
	}

	wg.Wait()

	// Test should not panic or race
	t.Log("Cache concurrency test passed")
}

// Benchmark tests
func BenchmarkEscapeQueryString(b *testing.B) {
	testStr := "file'with'many'quotes'in'it"
	for i := 0; i < b.N; i++ {
		_ = escapeQueryString(testStr)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := newIDCache()
	cache.set("/test", "id-123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.get("/test")
	}
}

func BenchmarkCacheConcurrent(b *testing.B) {
	cache := newIDCache()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			path := "/path"
			if i%2 == 0 {
				cache.set(path, "id")
			} else {
				cache.get(path)
			}
			i++
		}
	})
}

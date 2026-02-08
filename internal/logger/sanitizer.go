package logger

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Sanitizer 負責過濾日誌中的敏感資訊
//
// 限制說明：
//   - SanitizeArgs() 僅對「敏感 key 的 value」進行遮罩（如 password、token 等）
//   - 若敏感資料藏在非敏感 key 的 value 中，不會被遮罩
//   - 範例：logger.Info("msg", "url", "http://example.com?password=secret")
//     其中 password=secret 不會被遮罩，因為 key 是 "url"
//   - 建議：避免將敏感資料放在非敏感 key 的 value 中
type Sanitizer struct {
	mu       sync.RWMutex
	patterns []SanitizeRule
}

// SanitizeRule 單一過濾規則
type SanitizeRule struct {
	Pattern     *regexp.Regexp
	Replacement string
}

// NewSanitizer 建立預設 sanitizer
func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		patterns: defaultSanitizeRules(),
	}
}

// defaultSanitizeRules 回傳預設過濾規則
func defaultSanitizeRules() []SanitizeRule {
	return []SanitizeRule{
		// 密碼相關
		{regexp.MustCompile(`(?i)password=\S+`), "password=***"},
		{regexp.MustCompile(`(?i)passwd=\S+`), "passwd=***"},
		{regexp.MustCompile(`(?i)pwd=\S+`), "pwd=***"},

		// Token 相關
		{regexp.MustCompile(`(?i)token=\S+`), "token=***"},
		{regexp.MustCompile(`(?i)bearer\s+\S+`), "bearer ***"},
		{regexp.MustCompile(`(?i)api[_-]?key=\S+`), "api_key=***"},

		// Windows 使用者路徑 (支援所有磁碟機與 UNC，不區分大小寫)
		{regexp.MustCompile(`(?i)[A-Z]:\\Users\\[^\\]+`), "***:\\Users\\***"},
		{regexp.MustCompile(`(?i)\\\\[^\\]+\\[^\\]+\\Users\\[^\\]+`), "\\\\***\\***\\Users\\***"},

		// Unix 家目錄
		{regexp.MustCompile(`/home/[^/]+`), "/home/***"},
		{regexp.MustCompile(`/Users/[^/]+`), "/Users/***"},

		// Email 部分遮蔽
		{regexp.MustCompile(`([a-zA-Z0-9._%+-]{1,3})[a-zA-Z0-9._%+-]*@`), "$1***@"},
	}
}

// Sanitize sanitizes a string by applying all patterns
func (s *Sanitizer) Sanitize(input string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := input
	for _, rule := range s.patterns {
		result = rule.Pattern.ReplaceAllString(result, rule.Replacement)
	}
	return result
}

// SanitizeArgs sanitizes logging arguments
func (s *Sanitizer) SanitizeArgs(args []any) []any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(args) == 0 {
		return args
	}

	result := make([]any, len(args))
	copy(result, args)

	// Process key-value pairs
	for i := 0; i < len(result)-1; i += 2 {
		key, ok := result[i].(string)
		if !ok {
			continue
		}

		// Check if key is sensitive
		if s.isSensitiveKey(key) {
			// Mask the value
			switch v := result[i+1].(type) {
			case string:
				result[i+1] = s.maskValue(v)
			case error:
				result[i+1] = s.maskValue(v.Error())
			default:
				// For other types, leave as-is (documented limitation)
				continue
			}
		}
	}

	return result
}

// isSensitiveKey 判斷鍵名是否為敏感鍵
func (s *Sanitizer) isSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)
	sensitiveKeys := []string{
		"password", "passwd", "pwd",
		"token", "secret", "api_key", "apikey",
		"credential", "auth",
	}

	for _, sk := range sensitiveKeys {
		if strings.Contains(lowerKey, sk) {
			return true
		}
	}
	return false
}

// maskValue 遮蔽值（保留前後各1字元）
func (s *Sanitizer) maskValue(value string) string {
	if len(value) <= 2 {
		return "***"
	}
	if len(value) <= 8 {
		return fmt.Sprintf("%s***", string(value[0]))
	}
	return fmt.Sprintf("%s***%s", string(value[0]), string(value[len(value)-1]))
}

// AddRule 新增自訂過濾規則
func (s *Sanitizer) AddRule(pattern string, replacement string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	s.patterns = append(s.patterns, SanitizeRule{
		Pattern:     re,
		Replacement: replacement,
	})
	return nil
}

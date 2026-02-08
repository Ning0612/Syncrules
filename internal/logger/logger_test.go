package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogger_InitAndGet(t *testing.T) {
	buf := &bytes.Buffer{}
	config := Config{
		Level:  LevelInfo,
		Format: FormatText,
		Outputs: []OutputConfig{
			{Type: OutputStdout, Writer: buf},
		},
	}

	err := Init(config)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer Shutdown()

	logger := Get()
	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("log output missing message: %s", output)
	}
}

func TestLogger_NullLogger(t *testing.T) {
	// 測試未初始化時的行為
	Shutdown() // 確保未初始化

	logger := Get()
	// 不應該 panic
	logger.Info("should not crash")
	logger.Debug("should not crash")
	logger.Warn("should not crash")
	logger.Error("should not crash")
}

func TestLogger_With(t *testing.T) {
	buf := &bytes.Buffer{}
	config := Config{
		Level:  LevelInfo,
		Format: FormatText,
		Outputs: []OutputConfig{
			{Type: OutputStdout, Writer: buf},
		},
	}

	Init(config)
	defer Shutdown()

	childLogger := With("component", "test")
	childLogger.Info("message")

	output := buf.String()
	if !strings.Contains(output, "component=test") {
		t.Errorf("output missing context: %s", output)
	}
}

func TestLogger_Sync(t *testing.T) {
	buf := &bytes.Buffer{}
	config := Config{
		Level:  LevelInfo,
		Format: FormatText,
		Outputs: []OutputConfig{
			{Type: OutputStdout, Writer: buf},
		},
	}

	Init(config)
	defer Shutdown()

	Get().Info("test")
	err := Sync()
	if err != nil {
		t.Errorf("Sync() error = %v", err)
	}
}

func TestLogger_Shutdown(t *testing.T) {
	buf := &bytes.Buffer{}
	config := Config{
		Level:  LevelInfo,
		Format: FormatText,
		Outputs: []OutputConfig{
			{Type: OutputStdout, Writer: buf},
		},
	}

	Init(config)
	Get().Info("before shutdown")

	err := Shutdown()
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	// 再次呼叫應該不會 panic
	err = Shutdown()
	if err != nil {
		t.Errorf("second Shutdown() error = %v", err)
	}
}

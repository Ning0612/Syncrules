package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	manager, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	if manager.db == nil {
		t.Error("Database connection is nil")
	}

	// Verify database file was created
	dbPath := filepath.Join(tmpDir, "syncrules.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestNewManager_EmptyDir(t *testing.T) {
	_, err := NewManager("")
	if err == nil {
		t.Error("Expected error for empty directory, got nil")
	}
}

func TestSaveAndGetExecution(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Save an execution record
	record := ExecutionRecord{
		RuleName:    "test-rule",
		StartTime:   time.Now().Add(-10 * time.Minute),
		EndTime:     time.Now(),
		Status:      "success",
		FilesSynced: 10,
		BytesSynced: 1024,
		Error:       "",
	}

	err = manager.SaveExecution(record)
	if err != nil {
		t.Fatalf("Failed to save execution: %v", err)
	}

	// Retrieve history
	history, err := manager.GetHistory("test-rule", 10)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(history))
	}

	retrieved := history[0]
	if retrieved.RuleName != record.RuleName {
		t.Errorf("Expected rule name %s, got %s", record.RuleName, retrieved.RuleName)
	}

	if retrieved.Status != record.Status {
		t.Errorf("Expected status %s, got %s", record.Status, retrieved.Status)
	}

	if retrieved.FilesSynced != record.FilesSynced {
		t.Errorf("Expected files synced %d, got %d", record.FilesSynced, retrieved.FilesSynced)
	}

	if retrieved.BytesSynced != record.BytesSynced {
		t.Errorf("Expected bytes synced %d, got %d", record.BytesSynced, retrieved.BytesSynced)
	}
}

func TestGetLastSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Save multiple records with different statuses
	records := []ExecutionRecord{
		{
			RuleName:    "test-rule",
			StartTime:   time.Now().Add(-30 * time.Minute),
			EndTime:     time.Now().Add(-29 * time.Minute),
			Status:      "success",
			FilesSynced: 5,
			BytesSynced: 512,
		},
		{
			RuleName:    "test-rule",
			StartTime:   time.Now().Add(-20 * time.Minute),
			EndTime:     time.Now().Add(-19 * time.Minute),
			Status:      "failed",
			FilesSynced: 0,
			BytesSynced: 0,
			Error:       "network error",
		},
		{
			RuleName:    "test-rule",
			StartTime:   time.Now().Add(-10 * time.Minute),
			EndTime:     time.Now().Add(-9 * time.Minute),
			Status:      "success",
			FilesSynced: 10,
			BytesSynced: 1024,
		},
	}

	for _, record := range records {
		if err := manager.SaveExecution(record); err != nil {
			t.Fatalf("Failed to save execution: %v", err)
		}
	}

	// Retrieve last success
	lastSuccess, err := manager.GetLastSuccess("test-rule")
	if err != nil {
		t.Fatalf("Failed to get last success: %v", err)
	}

	if lastSuccess == nil {
		t.Fatal("Expected last success, got nil")
	}

	if lastSuccess.FilesSynced != 10 {
		t.Errorf("Expected last success to have 10 files, got %d", lastSuccess.FilesSynced)
	}

	if lastSuccess.Status != "success" {
		t.Errorf("Expected status 'success', got %s", lastSuccess.Status)
	}
}

func TestGetLastSuccess_NoSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Save only failed records
	record := ExecutionRecord{
		RuleName:    "test-rule",
		StartTime:   time.Now().Add(-10 * time.Minute),
		EndTime:     time.Now(),
		Status:      "failed",
		FilesSynced: 0,
		BytesSynced: 0,
		Error:       "test error",
	}

	if err := manager.SaveExecution(record); err != nil {
		t.Fatalf("Failed to save execution: %v", err)
	}

	// Retrieve last success (should be nil)
	lastSuccess, err := manager.GetLastSuccess("test-rule")
	if err != nil {
		t.Fatalf("Failed to get last success: %v", err)
	}

	if lastSuccess != nil {
		t.Error("Expected nil for last success, got a record")
	}
}

func TestGetAllHistory(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Save records for multiple rules
	records := []ExecutionRecord{
		{RuleName: "rule-1", StartTime: time.Now().Add(-30 * time.Minute), EndTime: time.Now().Add(-29 * time.Minute), Status: "success", FilesSynced: 5, BytesSynced: 512},
		{RuleName: "rule-2", StartTime: time.Now().Add(-20 * time.Minute), EndTime: time.Now().Add(-19 * time.Minute), Status: "success", FilesSynced: 10, BytesSynced: 1024},
		{RuleName: "rule-1", StartTime: time.Now().Add(-10 * time.Minute), EndTime: time.Now().Add(-9 * time.Minute), Status: "failed", FilesSynced: 0, BytesSynced: 0, Error: "error"},
	}

	for _, record := range records {
		if err := manager.SaveExecution(record); err != nil {
			t.Fatalf("Failed to save execution: %v", err)
		}
	}

	// Get all history
	allHistory, err := manager.GetAllHistory(100)
	if err != nil {
		t.Fatalf("Failed to get all history: %v", err)
	}

	if len(allHistory) != 3 {
		t.Fatalf("Expected 3 records, got %d", len(allHistory))
	}

	// Verify ordering (should be DESC by start_time)
	if allHistory[0].RuleName != "rule-1" || allHistory[0].Status != "failed" {
		t.Error("Expected most recent record to be rule-1 failed execution")
	}
}

func TestGetHistory_Limit(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Save 5 records
	for i := 0; i < 5; i++ {
		record := ExecutionRecord{
			RuleName:    "test-rule",
			StartTime:   time.Now().Add(time.Duration(-i*10) * time.Minute),
			EndTime:     time.Now().Add(time.Duration(-i*10+1) * time.Minute),
			Status:      "success",
			FilesSynced: i,
			BytesSynced: int64(i * 100),
		}
		if err := manager.SaveExecution(record); err != nil {
			t.Fatalf("Failed to save execution: %v", err)
		}
	}

	// Get only 3 most recent
	history, err := manager.GetHistory("test-rule", 3)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if len(history) != 3 {
		t.Fatalf("Expected 3 records, got %d", len(history))
	}

	// Verify we got the most recent ones
	if history[0].FilesSynced != 0 {
		t.Errorf("Expected most recent record to have 0 files synced, got %d", history[0].FilesSynced)
	}
}

// Test validation: invalid status
func TestSaveExecution_InvalidStatus(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	record := ExecutionRecord{
		RuleName:  "test-rule",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Status:    "invalid_status", // Invalid status
	}

	err = manager.SaveExecution(record)
	if err == nil {
		t.Error("Expected error for invalid status, got nil")
	}
}

// Test validation: invalid limit in GetHistory
func TestGetHistory_InvalidLimit(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	_, err = manager.GetHistory("test-rule", 0)
	if err == nil {
		t.Error("Expected error for limit=0, got nil")
	}

	_, err = manager.GetHistory("test-rule", -1)
	if err == nil {
		t.Error("Expected error for limit=-1, got nil")
	}
}

// Test validation: invalid limit in GetAllHistory
func TestGetAllHistory_InvalidLimit(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	_, err = manager.GetAllHistory(0)
	if err == nil {
		t.Error("Expected error for limit=0, got nil")
	}

	_, err = manager.GetAllHistory(-1)
	if err == nil {
		t.Error("Expected error for limit=-1, got nil")
	}
}

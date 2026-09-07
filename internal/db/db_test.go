package db

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestMutationCallbackAndCheckpoint(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dbtest-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origWd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	d, err := open("test")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer d.Close()

	var mutationFired atomic.Int32
	d.OnMutation(func() {
		mutationFired.Add(1)
	})

	// Create an account
	err = d.DB.Create(&Account{
		BaleUserID:  123456,
		Role:        "SERVER",
		DisplayName: "Test Account",
		Phone:       "09123456789",
	}).Error
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Wait up to 500ms for goroutine callback
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if mutationFired.Load() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if mutationFired.Load() == 0 {
		t.Fatalf("expected mutation callback to fire on Create")
	}

	// Test CheckpointWAL
	if err := d.CheckpointWAL(); err != nil {
		t.Fatalf("CheckpointWAL failed: %v", err)
	}

	// Verify file exists
	dbFile := filepath.Join(tmpDir, "data", "test.db")
	if _, err := os.Stat(dbFile); err != nil {
		t.Fatalf("expected db file at %s: %v", dbFile, err)
	}
}

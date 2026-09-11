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

func TestResetDataPreservesSettingsAndAdmin(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dbtest-reset-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origWd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	d, err := open("test_reset")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer d.Close()

	// 1. Seed accounts
	acct1 := Account{BaleUserID: 111, Role: "CLIENT", DisplayName: "Client 1", Phone: "09111111111"}
	acct2 := Account{BaleUserID: 222, Role: "SERVER", DisplayName: "Server 1", Phone: "09222222222"}
	if err := d.DB.Create(&acct1).Error; err != nil {
		t.Fatal(err)
	}
	if err := d.DB.Create(&acct2).Error; err != nil {
		t.Fatal(err)
	}

	// 2. Seed pairing
	pairing := Pairing{ClientAccountID: acct1.ID, ServerAccountID: acct2.ID, Active: true}
	if err := d.DB.Create(&pairing).Error; err != nil {
		t.Fatal(err)
	}

	// 3. Seed connection log
	cLog := ConnectionLog{ClientAcctID: acct1.ID, ServerAcctID: acct2.ID, PairingID: pairing.ID, StartTime: time.Now()}
	if err := d.DB.Create(&cLog).Error; err != nil {
		t.Fatal(err)
	}

	// 4. Seed setting (MUST BE PRESERVED)
	if err := d.SetSetting("dns.primary", "1.1.1.1"); err != nil {
		t.Fatal(err)
	}
	if err := d.SetSetting("bale.app_version", "169491"); err != nil {
		t.Fatal(err)
	}

	// 5. Seed admin user (MUST BE PRESERVED)
	if err := d.SeedAdminUser("salman", "Salman136517"); err != nil {
		t.Fatal(err)
	}

	// 6. Perform ResetData
	stats, err := d.ResetData()
	if err != nil {
		t.Fatalf("ResetData failed: %v", err)
	}

	if stats.AccountsDeleted != 2 {
		t.Fatalf("expected 2 accounts deleted, got %d", stats.AccountsDeleted)
	}
	if stats.PairingsDeleted != 1 {
		t.Fatalf("expected 1 pairing deleted, got %d", stats.PairingsDeleted)
	}
	if stats.LogsDeleted != 1 {
		t.Fatalf("expected 1 log deleted, got %d", stats.LogsDeleted)
	}

	// Verify accounts and pairings are 0
	cnt, _ := d.CountAccounts("", "")
	if cnt != 0 {
		t.Fatalf("expected 0 accounts, got %d", cnt)
	}
	pairings, _ := d.ListPairings()
	if len(pairings) != 0 {
		t.Fatalf("expected 0 pairings, got %d", len(pairings))
	}

	// Verify settings ARE PRESERVED!
	dnsVal, err := d.GetSetting("dns.primary")
	if err != nil || dnsVal != "1.1.1.1" {
		t.Fatalf("expected dns.primary=1.1.1.1 to be preserved, got err=%v val=%s", err, dnsVal)
	}
	baleVal, err := d.GetSetting("bale.app_version")
	if err != nil || baleVal != "169491" {
		t.Fatalf("expected bale.app_version=169491 to be preserved, got err=%v val=%s", err, baleVal)
	}

	// Verify admin user IS PRESERVED!
	adminUser, err := d.AuthenticateAdmin("salman", "Salman136517")
	if err != nil || adminUser == nil {
		t.Fatalf("expected admin user to be preserved, got err=%v", err)
	}
}


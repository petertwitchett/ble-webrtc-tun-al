package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/salman/ble-webrtc-tun/internal/accounts"
	"github.com/salman/ble-webrtc-tun/internal/db"
	"github.com/salman/ble-webrtc-tun/internal/router"
)

func TestHandleDBResetPreservesSettingsAndAdmin(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "api-reset-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origWd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	testDB, err := db.Init("test_api_reset")
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer testDB.Close()

	mgr := accounts.NewManager(testDB)
	rt := router.NewRouter(testDB)
	defer rt.Close()

	srv := NewServer(testDB, mgr, rt, Config{})

	// 1. Seed accounts & pairings
	acct1 := db.Account{BaleUserID: 101, Role: "CLIENT", DisplayName: "Client A"}
	acct2 := db.Account{BaleUserID: 202, Role: "SERVER", DisplayName: "Server B"}
	if err := testDB.DB.Create(&acct1).Error; err != nil {
		t.Fatal(err)
	}
	if err := testDB.DB.Create(&acct2).Error; err != nil {
		t.Fatal(err)
	}

	pairing := db.Pairing{ClientAccountID: acct1.ID, ServerAccountID: acct2.ID, Active: true}
	if err := testDB.DB.Create(&pairing).Error; err != nil {
		t.Fatal(err)
	}

	cLog := db.ConnectionLog{ClientAcctID: acct1.ID, ServerAcctID: acct2.ID, PairingID: pairing.ID, StartTime: time.Now()}
	if err := testDB.DB.Create(&cLog).Error; err != nil {
		t.Fatal(err)
	}

	// 2. Seed settings (MUST BE PRESERVED)
	if err := testDB.SetSetting("bale.app_version", "169491"); err != nil {
		t.Fatal(err)
	}
	if err := testDB.SetSetting("dns.primary", "1.1.1.1"); err != nil {
		t.Fatal(err)
	}

	// 3. Seed admin user (MUST BE PRESERVED)
	if err := testDB.SeedAdminUser("salman", "Salman136517"); err != nil {
		t.Fatal(err)
	}

	// 4. Send POST /api/db/reset request
	req := httptest.NewRequest(http.MethodPost, "/api/db/reset", nil)
	w := httptest.NewRecorder()

	srv.handleDBReset(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["success"] != true {
		t.Fatalf("expected success=true, got %v", result["success"])
	}

	// 5. Verify records in DB
	cnt, _ := testDB.CountAccounts("", "")
	if cnt != 0 {
		t.Fatalf("expected 0 accounts, got %d", cnt)
	}

	pairings, _ := testDB.ListPairings()
	if len(pairings) != 0 {
		t.Fatalf("expected 0 pairings, got %d", len(pairings))
	}

	// 6. Verify settings are preserved
	val, err := testDB.GetSetting("bale.app_version")
	if err != nil || val != "169491" {
		t.Fatalf("expected bale.app_version to be preserved, got err=%v val=%s", err, val)
	}

	val, err = testDB.GetSetting("dns.primary")
	if err != nil || val != "1.1.1.1" {
		t.Fatalf("expected dns.primary to be preserved, got err=%v val=%s", err, val)
	}

	// 7. Verify admin user is preserved
	admin, err := testDB.AuthenticateAdmin("salman", "Salman136517")
	if err != nil || admin == nil {
		t.Fatalf("expected admin user to be preserved, got err=%v", err)
	}
}

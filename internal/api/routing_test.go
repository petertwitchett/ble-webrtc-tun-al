package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleRoutingSettingsServer(t *testing.T) {
	s := &Server{}

	var reloadedPrimary, reloadedSecondary, reloadedBypass string
	s.OnReloadRouting = func(primary, secondary, bypassDomains string) error {
		reloadedPrimary = primary
		reloadedSecondary = secondary
		reloadedBypass = bypassDomains
		return nil
	}
	s.GetRoutingSettings = func() (map[string]string, error) {
		return map[string]string{
			"dns_primary":    reloadedPrimary,
			"dns_secondary":  reloadedSecondary,
			"bypass_domains": reloadedBypass,
		}, nil
	}

	// 1. POST update DNS
	payload := map[string]string{
		"dns_primary":    "185.161.112.33",
		"dns_secondary":  "185.161.112.34",
		"bypass_domains": "",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/routing/settings", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.handleRoutingSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if reloadedPrimary != "185.161.112.33" || reloadedSecondary != "185.161.112.34" {
		t.Fatalf("unexpected reloaded DNS: %s, %s", reloadedPrimary, reloadedSecondary)
	}

	// 2. GET settings
	reqGet := httptest.NewRequest("GET", "/api/routing/settings", nil)
	recGet := httptest.NewRecorder()
	s.handleRoutingSettings(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recGet.Code)
	}
	var res map[string]string
	_ = json.Unmarshal(recGet.Body.Bytes(), &res)
	if res["dns_primary"] != "185.161.112.33" {
		t.Fatalf("expected 185.161.112.33, got %s", res["dns_primary"])
	}

	// 3. Reset DNS to empty (revert to native host DNS)
	resetPayload := map[string]string{
		"dns_primary":    "",
		"dns_secondary":  "",
		"bypass_domains": "",
	}
	resetBody, _ := json.Marshal(resetPayload)
	reqReset := httptest.NewRequest("POST", "/api/routing/settings", bytes.NewReader(resetBody))
	recReset := httptest.NewRecorder()
	s.handleRoutingSettings(recReset, reqReset)
	if recReset.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recReset.Code)
	}
	if reloadedPrimary != "" || reloadedSecondary != "" {
		t.Fatalf("expected empty DNS after reset, got %s, %s", reloadedPrimary, reloadedSecondary)
	}
}

package bale

import (
	"strings"
	"testing"
)

func TestClientConstants(t *testing.T) {
	if AppVersion() != "169491" {
		t.Fatalf("expected app_version 169491, got %s", AppVersion())
	}
	if BrowserVersion() != "151.0.0.0" {
		t.Fatalf("expected browser_version 151.0.0.0, got %s", BrowserVersion())
	}
}

func TestBuildMetadata(t *testing.T) {
	meta := buildMetadata()
	if len(meta) == 0 {
		t.Fatal("expected non-empty metadata")
	}

	requiredKeys := []string{
		"app_version",
		"browser_type",
		"browser_version",
		"os_type",
		"session_id",
		"mt_app_version",
		"mt_browser_type",
		"mt_browser_version",
		"mt_os_type",
		"mt_session_id",
		"language",
		"mt_language",
	}

	metaStr := string(meta)
	for _, key := range requiredKeys {
		if !strings.Contains(metaStr, key) {
			t.Errorf("missing key %s in metadata", key)
		}
	}
	if !strings.Contains(metaStr, "169491") {
		t.Errorf("expected 169491 in metadata")
	}
	if !strings.Contains(metaStr, "151.0.0.0") {
		t.Errorf("expected 151.0.0.0 in metadata")
	}
	if !strings.Contains(metaStr, "fa") {
		t.Errorf("expected fa in metadata")
	}
}

func TestBuildDiscardCallMsg(t *testing.T) {
	callID := int64(1234567890123)
	seq := uint32(42)
	msg := buildDiscardCallMsg(callID, seq)
	if len(msg) == 0 {
		t.Fatal("expected non-empty DiscardCall message")
	}

	msgStr := string(msg)
	if !strings.Contains(msgStr, "bale.meet.v1.Meet") {
		t.Errorf("expected service bale.meet.v1.Meet")
	}
	if !strings.Contains(msgStr, "DiscardCall") {
		t.Errorf("expected method DiscardCall")
	}
	if !strings.Contains(msgStr, "169491") {
		t.Errorf("expected app_version 169491 in DiscardCall metadata")
	}
}

func TestSentryReleaseExtraction(t *testing.T) {
	sampleJS := `try{self.SENTRY_RELEASE={id:"web@5.5.1+169491"}}catch(e){}`
	found := false
	for _, pat := range reAppVersion {
		if m := pat.FindStringSubmatch(sampleJS); len(m) >= 2 {
			if m[1] == "169491" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("failed to extract 169491 from SENTRY_RELEASE string")
	}
}

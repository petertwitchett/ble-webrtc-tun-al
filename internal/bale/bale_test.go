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

	// Test that invalid CDN asset URLs are rejected by SetBaleGRPCBase
	origGRPC := BaleGRPCBase()
	defer SetBaleGRPCBase(origGRPC)

	SetBaleGRPCBase("https://assets.bale.ai/configs.json")
	if BaleGRPCBase() == "https://assets.bale.ai/configs.json" {
		t.Fatalf("expected assets URL to be rejected by SetBaleGRPCBase")
	}

	SetBaleGRPCBase("https://next-ws.bale.ai")
	if BaleGRPCBase() != "https://next-ws.bale.ai" {
		t.Fatalf("expected https://next-ws.bale.ai to be accepted")
	}
}

func TestExtractInfraURLs(t *testing.T) {
	sample := `var config = { ws: "wss://next-ws.bale.ai/ws/", grpc: "https://next-ws.bale.ai", url: "https://assets.bale.ai/configs.json", flag: "https://flags.ble.ir/api/frontend", meet: "https://web.ble.ir" };`
	ws, grpc, lk, origin := extractInfraURLs(sample)
	if ws != "wss://next-ws.bale.ai/ws/" {
		t.Errorf("expected ws wss://next-ws.bale.ai/ws/, got %s", ws)
	}
	if grpc != "https://next-ws.bale.ai" {
		t.Errorf("expected grpc https://next-ws.bale.ai, got %s", grpc)
	}
	if lk != "https://web.ble.ir" {
		t.Errorf("expected livekit origin https://web.ble.ir, got %s", lk)
	}
	if origin != "https://web.bale.ai" {
		t.Errorf("expected origin https://web.bale.ai, got %s", origin)
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

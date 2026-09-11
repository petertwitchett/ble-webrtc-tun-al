package s3sync

import (
	"context"
	"testing"
	"time"
)

func TestSigV4Signing(t *testing.T) {
	cfg := Config{
		Host:      "cellar-c2.services.clever-cloud.com",
		KeyID:     "TESTKEYID",
		KeySecret: "TESTSECRETKEY",
		Bucket:    "test-bucket",
		Region:    "us-east-1",
		UseSSL:    true,
	}
	client := NewClient(cfg)
	if !client.Config().IsConfigured() {
		t.Fatalf("expected client to be configured")
	}

	// Verify signature key derivation is non-empty
	key := getSignatureKey(cfg.KeySecret, "20260907", "us-east-1", "s3")
	if len(key) != 32 {
		t.Fatalf("expected 32-byte HMAC key, got %d", len(key))
	}
}

func TestLiveCellarRoundtrip(t *testing.T) {
	cfg := LoadConfig()
	if !cfg.IsConfigured() {
		t.Skip("Cellar S3 not configured")
	}

	client := NewClient(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Ensure bucket
	if err := client.EnsureBucket(ctx); err != nil {
		t.Fatalf("EnsureBucket failed: %v", err)
	}

	// 2. Put test object
	testKey := "test_verify.txt"
	testContent := []byte("Cellar SigV4 integration test " + time.Now().String())
	if err := client.PutObject(ctx, testKey, testContent); err != nil {
		t.Fatalf("PutObject failed: %v", err)
	}

	// 3. Head test object
	exists, size, err := client.HeadObject(ctx, testKey)
	if err != nil {
		t.Fatalf("HeadObject failed: %v", err)
	}
	if !exists || size != int64(len(testContent)) {
		t.Fatalf("HeadObject expected size %d, got exists=%v size=%d", len(testContent), exists, size)
	}

	// 4. Get test object
	retrieved, found, err := client.GetObject(ctx, testKey)
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	}
	if !found {
		t.Fatalf("GetObject expected found=true")
	}
	if string(retrieved) != string(testContent) {
		t.Fatalf("GetObject expected %s, got %s", string(testContent), string(retrieved))
	}

	// 5. Clean up
	_, _, _ = client.request(ctx, "DELETE", "/"+cfg.Bucket+"/"+testKey, nil)
}

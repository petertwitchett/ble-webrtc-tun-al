package s3sync

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/salman/ble-webrtc-tun/internal/logger"
)

var s3Log = logger.New("s3")

// Config holds S3 / Cellar connection parameters.
type Config struct {
	Host       string // e.g. "cellar-c2.services.clever-cloud.com"
	KeyID      string // CELLAR_ADDON_KEY_ID
	KeySecret  string // CELLAR_ADDON_KEY_SECRET
	Bucket     string // e.g. "ble-tunnel-server-db"
	Region     string // default "us-east-1"
	UseSSL     bool   // default true
	PathPrefix string // optional prefix, default ""
}

// LoadConfig loads S3 configuration with Clever Cloud Cellar conventions,
// environment variables, and fallback defaults.
func LoadConfig() Config {
	host := os.Getenv("CELLAR_ADDON_HOST")
	if host == "" {
		host = os.Getenv("S3_ENDPOINT")
	}
	if host == "" {
		host = "cellar-c2.services.clever-cloud.com"
	}
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimRight(host, "/")

	keyID := os.Getenv("CELLAR_ADDON_KEY_ID")
	if keyID == "" {
		keyID = os.Getenv("AWS_ACCESS_KEY_ID")
	}
	if keyID == "" {
		keyID = "J1B95ZC0ADI3PRASHSYS" // Fallback to Clever Cloud Cellar Add-on key
	}

	keySecret := os.Getenv("CELLAR_ADDON_KEY_SECRET")
	if keySecret == "" {
		keySecret = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}
	if keySecret == "" {
		keySecret = "WbvNnIRc90mWquFXkHvXkJvN9WTXpnXwnHn9Dmx1" // Fallback to Clever Cloud Cellar Add-on secret
	}

	bucket := os.Getenv("CELLAR_ADDON_BUCKET_NAME")
	if bucket == "" {
		bucket = os.Getenv("S3_BUCKET")
	}
	if bucket == "" {
		bucket = "ble-tunnel-server-db"
	}

	region := os.Getenv("S3_REGION")
	if region == "" {
		region = "us-east-1"
	}

	return Config{
		Host:      host,
		KeyID:     keyID,
		KeySecret: keySecret,
		Bucket:    bucket,
		Region:    region,
		UseSSL:    true,
	}
}

// IsConfigured returns true if credentials and host are present.
func (c Config) IsConfigured() bool {
	return c.Host != "" && c.KeyID != "" && c.KeySecret != "" && c.Bucket != ""
}

// Client provides pure-Go AWS SigV4 S3 operations.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient creates a new S3 client.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Config returns the current configuration.
func (c *Client) Config() Config {
	return c.cfg
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func getSignatureKey(secretKey, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	kSigning := hmacSHA256(kService, "aws4_request")
	return kSigning
}

// request executes a signed S3 request using AWS SigV4.
func (c *Client) request(ctx context.Context, method, path string, payload []byte) (*http.Response, []byte, error) {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	payloadHash := sha256Hex(payload)

	// Canonical headers (must be lowercase and sorted alphabetically)
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		c.cfg.Host, payloadHash, amzDate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalURI := path
	canonicalQueryString := ""

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method, canonicalURI, canonicalQueryString, canonicalHeaders, signedHeaders, payloadHash)

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, c.cfg.Region)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate, credentialScope, sha256Hex([]byte(canonicalRequest)))

	signingKey := getSignatureKey(c.cfg.KeySecret, dateStamp, c.cfg.Region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.cfg.KeyID, credentialScope, signedHeaders, signature)

	scheme := "https"
	if !c.cfg.UseSSL {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s%s", scheme, c.cfg.Host, path)

	var reqBody io.Reader
	if len(payload) > 0 {
		reqBody = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Host", c.cfg.Host)
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.Header.Set("Authorization", authHeader)
	if len(payload) > 0 {
		req.Header.Set("Content-Length", fmt.Sprintf("%d", len(payload)))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("s3 request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("read response body: %w", err)
	}

	return resp, body, nil
}

// EnsureBucket checks if the configured bucket exists; if not, attempts to create it.
func (c *Client) EnsureBucket(ctx context.Context) error {
	path := "/" + c.cfg.Bucket

	// First test with HEAD
	resp, _, err := c.request(ctx, "HEAD", path, nil)
	if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusForbidden) {
		// 200 = exists and accessible; 403 = exists in account
		return nil
	}

	// Try creating with PUT
	resp, body, err := c.request(ctx, "PUT", path, nil)
	if err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusCreated {
		return nil
	}

	// Some S3 providers return 409 BucketAlreadyOwnedByYou which is fine
	if resp.StatusCode == 409 || strings.Contains(string(body), "BucketAlreadyOwnedByYou") || strings.Contains(string(body), "BucketAlreadyExists") {
		return nil
	}

	return fmt.Errorf("ensure bucket failed (status %d): %s", resp.StatusCode, string(body))
}

// PutObject uploads data to the specified key in the configured bucket.
func (c *Client) PutObject(ctx context.Context, key string, data []byte) error {
	key = strings.TrimPrefix(key, "/")
	path := fmt.Sprintf("/%s/%s", c.cfg.Bucket, key)

	resp, body, err := c.request(ctx, "PUT", path, data)
	if err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("put object %s failed (status %d): %s", key, resp.StatusCode, string(body))
	}

	return nil
}

// GetObject retrieves data for the specified key from the bucket.
// Returns (data, found, error). If 404 Not Found, returns (nil, false, nil).
func (c *Client) GetObject(ctx context.Context, key string) ([]byte, bool, error) {
	key = strings.TrimPrefix(key, "/")
	path := fmt.Sprintf("/%s/%s", c.cfg.Bucket, key)

	resp, body, err := c.request(ctx, "GET", path, nil)
	if err != nil {
		return nil, false, fmt.Errorf("get object %s: %w", key, err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("get object %s failed (status %d): %s", key, resp.StatusCode, string(body))
	}

	return body, true, nil
}

// HeadObject checks if an object exists and returns its size.
func (c *Client) HeadObject(ctx context.Context, key string) (bool, int64, error) {
	key = strings.TrimPrefix(key, "/")
	path := fmt.Sprintf("/%s/%s", c.cfg.Bucket, key)

	resp, _, err := c.request(ctx, "HEAD", path, nil)
	if err != nil {
		return false, 0, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, 0, nil
	}

	if resp.StatusCode == http.StatusOK {
		return true, resp.ContentLength, nil
	}

	return false, 0, fmt.Errorf("head object status: %d", resp.StatusCode)
}

package dns

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestAppResolverCacheAndHotSwap(t *testing.T) {
	r := NewAppResolver("1.1.1.1", "1.0.0.1")

	p, s := r.Servers()
	if p != "1.1.1.1" || s != "1.0.0.1" {
		t.Fatalf("unexpected servers: %s, %s", p, s)
	}

	// Inject an entry into cache
	r.cacheMu.Lock()
	r.cache["fake.ble.ir"] = &cacheEntry{
		ips:       []net.IP{net.ParseIP("185.161.112.33")},
		expiresAt: time.Now().Add(10 * time.Minute),
	}
	r.cacheMu.Unlock()

	if r.CacheStats() != 1 {
		t.Fatalf("expected cache size 1, got %d", r.CacheStats())
	}

	ctx := context.Background()
	ips, err := r.LookupIP(ctx, "fake.ble.ir")
	if err != nil || len(ips) == 0 || ips[0].String() != "185.161.112.33" {
		t.Fatalf("expected cache hit for fake.ble.ir, got %v, err=%v", ips, err)
	}

	// Hot-swap servers
	r.SetServers("8.8.8.8", "8.8.4.4")
	p2, s2 := r.Servers()
	if p2 != "8.8.8.8" || s2 != "8.8.4.4" {
		t.Fatalf("unexpected hot-swapped servers: %s, %s", p2, s2)
	}

	// Verify cache was cleared on SetServers
	if r.CacheStats() != 0 {
		t.Fatalf("expected cache to be cleared after SetServers, got size %d", r.CacheStats())
	}
}

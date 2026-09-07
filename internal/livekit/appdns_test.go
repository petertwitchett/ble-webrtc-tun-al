package livekit

import (
	"context"
	"net"
	"testing"
)

func TestResolveICEURLs(t *testing.T) {
	// Set mock lookup
	SetAppLookupIP(func(ctx context.Context, host string) ([]net.IP, error) {
		if host == "meet-turn.ble.ir" {
			return []net.IP{net.ParseIP("185.143.232.10")}, nil
		}
		return nil, nil
	})
	defer SetAppLookupIP(nil)

	urls := []string{
		"turn:meet-turn.ble.ir:443?transport=tcp",
		"stun:1.2.3.4:3478",
	}

	resolved := ResolveICEURLs(urls)
	if len(resolved) < 3 {
		t.Fatalf("expected at least 3 URLs (resolved + original + existing), got %d: %v", len(resolved), resolved)
	}

	expectedPrefix := "turn:185.143.232.10:443?transport=tcp"
	if resolved[0] != expectedPrefix {
		t.Fatalf("expected first URL to be %s, got %s", expectedPrefix, resolved[0])
	}
}

func TestPionNetResolution(t *testing.T) {
	lookup := func(ctx context.Context, host string) ([]net.IP, error) {
		if host == "sfu.ble.ir" {
			return []net.IP{net.ParseIP("185.51.200.1")}, nil
		}
		return nil, nil
	}

	pNet, err := NewPionNet(lookup, nil)
	if err != nil {
		t.Fatalf("NewPionNet failed: %v", err)
	}

	tcpAddr, err := pNet.ResolveTCPAddr("tcp", "sfu.ble.ir:7880")
	if err != nil {
		t.Fatalf("ResolveTCPAddr failed: %v", err)
	}
	if tcpAddr.IP.String() != "185.51.200.1" || tcpAddr.Port != 7880 {
		t.Fatalf("unexpected tcpAddr: %v", tcpAddr)
	}

	udpAddr, err := pNet.ResolveUDPAddr("udp", "sfu.ble.ir:7881")
	if err != nil {
		t.Fatalf("ResolveUDPAddr failed: %v", err)
	}
	if udpAddr.IP.String() != "185.51.200.1" || udpAddr.Port != 7881 {
		t.Fatalf("unexpected udpAddr: %v", udpAddr)
	}
}

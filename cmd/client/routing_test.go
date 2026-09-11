package main

import "testing"

func TestShouldDropLocally(t *testing.T) {
	tests := []struct {
		target string
		want   bool
	}{
		// Upstream loopback pseudo-domains (sing-box / v2ray)
		{"sp.v2.udp-over-tcp.arpa:0", true},
		{"custom.udp-over-tcp.arpa:443", true},
		{"1.0.0.127.in-addr.arpa:53", true},
		{"sub.domain.arpa", true},

		// Local / private mDNS and test domains
		{"device.local:80", true},
		{"printer.local", true},
		{"service.internal:8080", true},
		{"app.localhost:3000", true},

		// Loopback and unroutable addresses
		{"localhost:8080", true},
		{"localhost", true},
		{"127.0.0.1:10909", true},
		{"127.0.0.1", true},
		{"[::1]:8080", true},
		{"::1", true},
		{"0.0.0.0:80", true},

		// Legitimate external internet targets that MUST NOT be dropped
		{"google.com:443", false},
		{"142.250.110.188:5228", false},
		{"meet.bale.ai:443", false},
		{"apple.com:80", false},
		{"1.1.1.1:53", false},
	}

	for _, tt := range tests {
		got := ShouldDropLocally(tt.target)
		if got != tt.want {
			t.Errorf("ShouldDropLocally(%q) = %v; want %v", tt.target, got, tt.want)
		}
	}
}

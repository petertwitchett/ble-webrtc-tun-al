package livekit

// appdns.go — Application-level DNS injection for the LiveKit SFU package.
//
// The LiveKit signaling and SFU transports make outbound WebSocket connections
// and WebRTC media streams to Bale's LiveKit infrastructure (e.g. meet-gwbm*.ble.ir,
// meet-turn.ble.ir). These connections must resolve their target hosts through
// the admin-configured application DNS roots rather than the host OS resolver.
//
// SetAppDialContext installs a context-aware dial function that the package
// uses inside websocket.Dialer.NetDialContext.
//
// SetAppNet installs a custom transport.Net for Pion WebRTC's SettingEngine
// so all ICE candidate checks and STUN/TURN host lookups use the custom DNS.

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pion/transport/v2"
	"github.com/pion/transport/v2/stdnet"
)

// DialContextFunc resolves the host portion of addr through the application
// DNS engine before establishing the TCP connection.
type DialContextFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// LookupIPFunc resolves a domain name through the application DNS engine.
type LookupIPFunc func(ctx context.Context, host string) ([]net.IP, error)

var appDialContext atomic.Pointer[DialContextFunc]
var appLookupIP atomic.Pointer[LookupIPFunc]
var appNet atomic.Pointer[transport.Net]

// SetAppDialContext installs (or clears, with nil) the application-level DNS
// dial function used by all LiveKit SFU outbound WebSocket connections.
func SetAppDialContext(fn DialContextFunc) {
	if fn == nil {
		appDialContext.Store(nil)
		return
	}
	appDialContext.Store(&fn)
}

// SetAppLookupIP installs (or clears, with nil) the IP resolution function.
func SetAppLookupIP(fn LookupIPFunc) {
	if fn == nil {
		appLookupIP.Store(nil)
		return
	}
	appLookupIP.Store(&fn)
}

// SetAppNet installs (or clears, with nil) the custom Pion transport.Net.
func SetAppNet(n transport.Net) {
	if n == nil {
		appNet.Store(nil)
		return
	}
	appNet.Store(&n)
}

// appDial returns the currently-installed application dial function, or nil
// if none is configured.
func appDial() DialContextFunc {
	fnp := appDialContext.Load()
	if fnp == nil {
		return nil
	}
	return *fnp
}

// getAppLookupIP returns the currently-installed IP lookup function, or nil.
func getAppLookupIP() LookupIPFunc {
	fnp := appLookupIP.Load()
	if fnp == nil {
		return nil
	}
	return *fnp
}

// getAppNet returns the currently-installed Pion transport.Net, or nil.
func getAppNet() transport.Net {
	np := appNet.Load()
	if np == nil {
		return nil
	}
	return *np
}

// pionAppNet intercepts DNS resolution for Pion WebRTC.
type pionAppNet struct {
	*stdnet.Net
	lookupIP LookupIPFunc
	dialFn   DialContextFunc
}

// NewPionNet creates a custom transport.Net wrapping stdnet.Net with custom DNS.
func NewPionNet(lookupIP LookupIPFunc, dialFn DialContextFunc) (transport.Net, error) {
	base, err := stdnet.NewNet()
	if err != nil {
		return nil, err
	}
	if lookupIP == nil && dialFn == nil {
		return base, nil
	}
	return &pionAppNet{
		Net:      base,
		lookupIP: lookupIP,
		dialFn:   dialFn,
	}, nil
}

func (p *pionAppNet) ResolveTCPAddr(network, address string) (*net.TCPAddr, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return p.Net.ResolveTCPAddr(network, address)
	}
	if ip := net.ParseIP(host); ip != nil {
		return p.Net.ResolveTCPAddr(network, address)
	}
	if p.lookupIP != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		ips, err := p.lookupIP(ctx, host)
		cancel()
		if err == nil && len(ips) > 0 {
			port, _ := strconv.Atoi(portStr)
			return &net.TCPAddr{IP: ips[0], Port: port}, nil
		}
	}
	return p.Net.ResolveTCPAddr(network, address)
}

func (p *pionAppNet) ResolveUDPAddr(network, address string) (*net.UDPAddr, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return p.Net.ResolveUDPAddr(network, address)
	}
	if ip := net.ParseIP(host); ip != nil {
		return p.Net.ResolveUDPAddr(network, address)
	}
	if p.lookupIP != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		ips, err := p.lookupIP(ctx, host)
		cancel()
		if err == nil && len(ips) > 0 {
			port, _ := strconv.Atoi(portStr)
			return &net.UDPAddr{IP: ips[0], Port: port}, nil
		}
	}
	return p.Net.ResolveUDPAddr(network, address)
}

func (p *pionAppNet) ResolveIPAddr(network, address string) (*net.IPAddr, error) {
	if ip := net.ParseIP(address); ip != nil {
		return p.Net.ResolveIPAddr(network, address)
	}
	if p.lookupIP != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		ips, err := p.lookupIP(ctx, address)
		cancel()
		if err == nil && len(ips) > 0 {
			return &net.IPAddr{IP: ips[0]}, nil
		}
	}
	return p.Net.ResolveIPAddr(network, address)
}

func (p *pionAppNet) Dial(network, address string) (net.Conn, error) {
	if p.dialFn != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return p.dialFn(ctx, network, address)
	}
	return p.Net.Dial(network, address)
}

// ResolveICEURLs pre-resolves any hostnames in STUN/TURN URLs through custom DNS.
func ResolveICEURLs(urls []string) []string {
	lookupIP := getAppLookupIP()
	if lookupIP == nil {
		return urls
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result := make([]string, 0, len(urls)*2)
	for _, rawURL := range urls {
		parts := strings.SplitN(rawURL, ":", 2)
		if len(parts) != 2 {
			result = append(result, rawURL)
			continue
		}
		scheme := parts[0]
		rest := parts[1]

		var query string
		if qIdx := strings.Index(rest, "?"); qIdx != -1 {
			query = rest[qIdx:]
			rest = rest[:qIdx]
		}

		host, port, err := net.SplitHostPort(rest)
		if err != nil {
			result = append(result, rawURL)
			continue
		}

		if net.ParseIP(host) != nil {
			result = append(result, rawURL)
			continue
		}

		ips, err := lookupIP(ctx, host)
		if err != nil || len(ips) == 0 {
			result = append(result, rawURL)
			continue
		}

		// Inject pre-resolved IP URL first, then fallback to original hostname URL
		resolvedURL := fmt.Sprintf("%s:%s:%s%s", scheme, ips[0].String(), port, query)
		result = append(result, resolvedURL, rawURL)
	}
	return result
}


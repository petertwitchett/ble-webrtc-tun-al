package dns

import (
	"context"
	"fmt"
	"math"
	"net"
	"sort"
	"sync"
	"time"
)

// DefaultTestDomains are the benchmark target domains: google.com plus all 6 Bale Meet gateways.
var DefaultTestDomains = []string{
	"google.com",
	"meet-gwbm1.ble.ir",
	"meet-gwbm2.ble.ir",
	"meet-gwbm3.ble.ir",
	"meet-gwbm4.ble.ir",
	"meet-gwbm5.ble.ir",
	"meet-gwbm6.ble.ir",
}

// DefaultBenchmarkDNSServers contains the curated list of Iranian and international DNS servers.
var DefaultBenchmarkDNSServers = []string{
	"185.161.112.33", "185.161.112.34", "185.51.200.10", "185.231.182.126",
	"46.224.1.42", "194.225.62.80", "213.176.123.5", "91.99.101.12",
	"185.187.84.15", "37.156.145.229", "185.97.117.187", "185.113.59.253",
	"80.191.40.41", "194.225.73.141", "91.245.229.1", "185.51.200.50",
	"37.156.145.21", "217.218.155.155", "2.189.44.44", "2.188.21.131",
	"2.188.21.132", "81.91.144.116", "2.188.21.130", "92.119.56.162",
	"109.69.8.51", "208.67.220.200", "208.67.222.222", "91.239.100.100",
	"89.233.43.71", "194.36.174.161", "178.22.122.100", "185.51.200.2",
	"78.157.42.100", "78.157.42.101", "10.202.10.10", "10.202.10.11",
	"1.1.1.1", "1.0.0.1", "8.8.8.8", "8.2.2.8",
	"217.218.127.127", "5.202.100.101", "87.107.110.109", "194.225.152.10",
	"85.15.1.14", "85.15.1.15", "185.164.72.97", "5.200.200.200",
	"194.225.152.14", "172.30.130.30", "172.30.131.30", "185.212.51.144",
	"46.100.40.59", "5.202.171.124", "46.100.164.165", "93.114.104.185",
	"2.188.162.78", "87.107.82.213", "91.92.130.49", "81.29.248.38",
	"188.136.162.218", "185.83.114.233", "94.182.154.105", "178.252.143.133",
	"85.185.157.2", "79.127.125.126", "78.157.56.133", "80.191.68.247",
	"87.107.109.107", "87.107.184.123", "31.214.251.47", "31.214.251.58",
	"2.144.22.176", "2.144.20.37", "194.225.101.22", "94.183.149.147",
	"2.189.1.1", "2.177.129.246", "178.252.189.82", "93.118.123.32",
	"93.118.108.232", "2.177.228.177", "212.16.76.19", "78.38.182.201",
	"2.144.21.157", "2.177.161.64", "194.225.62.66", "80.210.52.165",
	"80.191.88.90", "78.39.8.27", "80.210.41.221", "5.160.121.70",
	"77.237.82.49", "2.144.5.164", "2.144.21.202", "95.38.102.86",
	"91.92.208.51", "109.201.11.75", "217.170.251.102", "37.202.225.135",
	"87.107.146.4", "5.159.51.205", "37.202.225.137", "37.202.225.156",
	"2.189.86.98", "5.202.100.100", "5.202.133.117", "80.210.48.24",
	"91.240.60.60", "185.11.70.217", "185.192.113.26", "194.150.68.29",
	"87.248.130.22", "178.22.122.246", "46.209.157.19", "5.160.233.150",
	"2.144.6.138", "82.99.195.82", "91.92.208.88", "95.80.160.58",
	"188.213.65.54", "5.202.170.49", "62.60.136.158", "93.114.106.187",
	"80.210.22.217", "185.129.197.235", "79.175.190.166", "185.173.171.252",
	"212.23.216.12", "2.177.236.183", "89.46.219.199", "194.53.122.91",
	"151.233.49.80", "212.80.20.132", "94.232.173.145", "2.180.43.13",
	"78.157.56.101", "80.210.44.184", "78.38.24.122", "93.118.101.153",
	"185.224.179.27", "5.159.48.40", "93.118.137.221", "217.144.107.239",
	"217.144.106.113", "93.118.138.109", "5.145.114.83", "45.81.18.141",
	"93.118.140.224", "2.144.23.164", "109.238.188.148", "109.230.91.226",
	"2.188.21.230", "85.185.157.181", "185.208.76.105", "5.239.245.240",
	"2.188.21.100", "109.230.90.86", "2.188.21.240", "2.188.21.190",
	"2.188.21.120", "2.188.21.90", "2.188.21.20", "2.144.22.69",
	"2.188.21.200", "87.10.19.84", "45.147.75.243", "93.126.22.206",
	"80.191.92.188", "95.38.132.6", "2.181.0.54", "185.235.197.2",
	"62.60.206.68", "45.159.197.99", "37.32.4.61", "78.39.139.149",
	"80.210.54.182", "45.92.94.207", "81.91.145.7", "81.91.145.2",
	"5.202.248.74", "93.126.24.8", "185.23.128.161", "93.126.2.252",
	"93.126.35.228", "93.114.111.108", "89.46.219.16", "89.46.219.197",
	"45.135.243.61", "2.190.233.153", "93.115.126.157", "94.183.163.248",
	"94.182.17.202", "94.182.17.206", "89.46.219.198", "91.222.196.8",
	"185.208.76.106", "37.191.79.105", "5.202.78.2", "109.109.32.10",
	"80.210.44.187", "78.157.52.10", "80.210.53.97", "109.109.32.125",
	"109.109.32.102", "109.109.32.110", "109.109.32.21", "94.182.17.205",
	"89.46.219.84", "93.115.122.89", "93.118.131.12", "217.219.120.82",
	"217.144.107.162", "37.202.186.29", "46.209.48.5", "31.47.32.34",
	"46.100.90.168", "194.53.122.168", "93.118.115.240", "194.53.122.139",
	"81.12.47.235", "195.211.47.199", "37.32.121.130", "188.213.209.146",
	"212.33.203.10", "185.147.40.88", "31.7.78.133", "5.160.13.83",
	"188.121.103.250", "2.188.167.236", "5.160.139.74", "2.188.174.222",
	"46.100.63.131", "2.186.229.200", "5.106.18.134", "5.160.119.225",
	"80.210.41.48", "62.220.116.100", "78.38.50.218", "81.12.121.19",
	"31.214.174.163", "37.156.12.65", "5.106.18.218", "85.185.159.76",
	"2.188.26.10", "2.188.20.5", "109.109.32.124", "185.208.149.226",
	"176.65.242.54", "185.129.216.60", "109.109.32.18", "185.174.250.131",
	"109.230.83.155", "109.109.32.152", "188.121.97.36", "185.235.197.34",
	"185.208.76.104", "109.109.34.118", "109.109.32.11", "185.206.229.34",
	"185.235.197.59", "109.109.32.155", "185.81.99.44", "2.180.31.171",
	"2.189.91.202", "2.190.0.142", "45.92.94.208", "37.148.33.206",
	"194.225.101.8", "80.210.40.54", "185.255.89.57", "91.92.190.84",
	"2.182.253.245", "109.125.136.149", "109.125.169.20", "109.230.72.243",
	"109.230.78.13", "109.230.79.12", "109.95.61.243", "176.65.240.86",
	"178.131.180.73", "178.173.144.224", "178.252.147.146", "178.252.147.84",
	"178.252.170.222", "178.252.178.205", "185.112.36.134", "185.112.36.64",
	"185.125.244.3", "185.128.138.2", "185.159.153.254", "185.172.215.219",
	"185.181.183.56", "185.206.92.250", "185.208.148.211", "185.231.181.206",
	"185.49.86.202", "185.24.255.148", "185.24.255.80", "185.51.201.243",
	"185.8.174.140", "185.83.91.3", "188.121.96.94", "185.112.149.118",
	"185.208.76.103", "185.51.200.1", "185.206.238.3", "2.144.20.150",
	"2.144.7.110",
}

// DNSBenchmarkResult holds the benchmark test metrics for a single DNS server.
type DNSBenchmarkResult struct {
	IP           string             `json:"ip"`
	Online       bool               `json:"online"`
	GoogleAvgMs  float64            `json:"google_avg_ms"` // -1 if offline or failed
	BaleAvgMs    float64            `json:"bale_avg_ms"`   // average latency across all Bale Meet gateways
	BaleGateways map[string]float64 `json:"bale_gateways"` // "B1": 18.2, "B2": 19.5, etc.
	GlobalAvgMs  float64            `json:"global_avg_ms"` // overall average across all queries
	Reliability  float64            `json:"reliability"`   // 0.0 to 100.0%
	Score        float64            `json:"score"`         // weighted composite penalty score (lower is better)
	Rank         int                `json:"rank,omitempty"`
}

// BenchmarkStatus reports live progress and collected results for the frontend.
type BenchmarkStatus struct {
	Running     bool                  `json:"running"`
	Total       int                   `json:"total"`
	Completed   int                   `json:"completed"`
	Percent     int                   `json:"percent"`
	OnlineCount int                   `json:"online_count"`
	Results     []*DNSBenchmarkResult `json:"results"`
	StartedAt   time.Time             `json:"started_at,omitempty"`
	FinishedAt  time.Time             `json:"finished_at,omitempty"`
	Error       string                `json:"error,omitempty"`
}

// BenchmarkRunner manages asynchronous DNS benchmarks.
type BenchmarkRunner struct {
	mu        sync.RWMutex
	running   bool
	cancel    context.CancelFunc
	status    BenchmarkStatus
	domains   []string
	dnsList   []string
}

// Global default benchmark runner instance.
var defaultRunner = NewBenchmarkRunner()

// DefaultBenchmarkRunner returns the global benchmark runner singleton.
func DefaultBenchmarkRunner() *BenchmarkRunner {
	return defaultRunner
}

// NewBenchmarkRunner creates an independent benchmark runner.
func NewBenchmarkRunner() *BenchmarkRunner {
	return &BenchmarkRunner{
		domains: DefaultTestDomains,
		dnsList: DefaultBenchmarkDNSServers,
	}
}

// Start launches the benchmark in the background.
// If servers is empty, DefaultBenchmarkDNSServers is used.
func (br *BenchmarkRunner) Start(customServers []string) error {
	br.mu.Lock()
	if br.running {
		br.mu.Unlock()
		return fmt.Errorf("benchmark is already running")
	}

	targets := customServers
	if len(targets) == 0 {
		targets = DefaultBenchmarkDNSServers
	}

	// Deduplicate servers while preserving order
	seen := make(map[string]bool)
	var deduped []string
	for _, s := range targets {
		if s != "" && !seen[s] {
			seen[s] = true
			deduped = append(deduped, s)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	br.running = true
	br.cancel = cancel
	br.dnsList = deduped
	br.status = BenchmarkStatus{
		Running:   true,
		Total:     len(deduped),
		Completed: 0,
		Percent:   0,
		Results:   make([]*DNSBenchmarkResult, 0, len(deduped)),
		StartedAt: time.Now(),
	}
	br.mu.Unlock()

	go br.execute(ctx, deduped)
	return nil
}

// Stop terminates any active benchmark run.
func (br *BenchmarkRunner) Stop() {
	br.mu.Lock()
	defer br.mu.Unlock()
	if br.running && br.cancel != nil {
		br.cancel()
		br.running = false
		br.status.Running = false
		br.status.FinishedAt = time.Now()
	}
}

// GetStatus returns a snapshot of current progress and sorted results.
func (br *BenchmarkRunner) GetStatus() BenchmarkStatus {
	br.mu.RLock()
	defer br.mu.RUnlock()

	// Return a clean copy of the results
	copyStatus := br.status
	resCopy := make([]*DNSBenchmarkResult, len(br.status.Results))
	copy(resCopy, br.status.Results)
	copyStatus.Results = resCopy
	return copyStatus
}

// execute runs the benchmark across all targets with bounded concurrency.
func (br *BenchmarkRunner) execute(ctx context.Context, servers []string) {
	defer func() {
		br.mu.Lock()
		br.running = false
		br.status.Running = false
		br.status.FinishedAt = time.Now()
		br.mu.Unlock()
	}()

	// Concurrency limiter: max 50 concurrent DNS servers tested at once
	concurrency := 50
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var resultsMu sync.Mutex

	total := len(servers)

	for _, ip := range servers {
		select {
		case <-ctx.Done():
			return
		default:
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(targetIP string) {
			defer func() {
				<-sem
				wg.Done()
			}()

			res := br.testSingleDNS(ctx, targetIP)

			resultsMu.Lock()
			br.mu.Lock()

			br.status.Results = append(br.status.Results, res)
			br.status.Completed++
			br.status.Percent = (br.status.Completed * 100) / total
			if res.Online {
				br.status.OnlineCount++
			}

			// Sort in-place by Score ascending (online first, lowest penalty score first)
			sort.Slice(br.status.Results, func(i, j int) bool {
				if br.status.Results[i].Online != br.status.Results[j].Online {
					return br.status.Results[i].Online // online servers before offline
				}
				return br.status.Results[i].Score < br.status.Results[j].Score
			})

			// Assign ranks to online results
			rank := 1
			for _, r := range br.status.Results {
				if r.Online {
					r.Rank = rank
					rank++
				} else {
					r.Rank = 0
				}
			}

			br.mu.Unlock()
			resultsMu.Unlock()
		}(ip)
	}

	wg.Wait()
}

// testSingleDNS executes the pre-check and multi-domain benchmark for one server.
func (br *BenchmarkRunner) testSingleDNS(ctx context.Context, ip string) *DNSBenchmarkResult {
	// Pre-check online status with 1.2s timeout
	if !br.checkOnline(ctx, ip) {
		return &DNSBenchmarkResult{
			IP:          ip,
			Online:      false,
			GoogleAvgMs: -1,
			BaleAvgMs:   -1,
			BaleGateways: map[string]float64{
				"B1": -1, "B2": -1, "B3": -1, "B4": -1, "B5": -1, "B6": -1,
			},
			GlobalAvgMs: -1,
			Reliability: 0,
			Score:       math.MaxFloat64,
		}
	}

	iterations := 3
	timeout := 2 * time.Second

	type domainResult struct {
		domain      string
		avgMs       float64
		successRate float64
	}

	// Test all 7 domains (google.com + 6 Bale Meet gateways)
	domResults := make([]domainResult, len(br.domains))
	var domWg sync.WaitGroup

	for idx, domain := range br.domains {
		domWg.Add(1)
		go func(i int, d string) {
			defer domWg.Done()
			avg, sr := br.testDomain(ctx, ip, d, iterations, timeout)
			domResults[i] = domainResult{
				domain:      d,
				avgMs:       avg,
				successRate: sr,
			}
		}(idx, domain)
	}
	domWg.Wait()

	// Parse Google stats (index 0)
	googleAvg := domResults[0].avgMs

	// Parse Bale stats (indices 1 to 6)
	baleResults := domResults[1:]
	var baleTimes []float64
	var baleSRs []float64
	baleGateways := make(map[string]float64)

	for idx, r := range baleResults {
		gatewayKey := fmt.Sprintf("B%d", idx+1)
		if r.avgMs >= 0 {
			baleTimes = append(baleTimes, r.avgMs)
			baleGateways[gatewayKey] = math.Round(r.avgMs*10) / 10
		} else {
			baleGateways[gatewayKey] = -1
		}
		baleSRs = append(baleSRs, r.successRate)
	}

	baleAvg := -1.0
	if len(baleTimes) > 0 {
		sum := 0.0
		for _, t := range baleTimes {
			sum += t
		}
		baleAvg = math.Round((sum/float64(len(baleTimes)))*10) / 10
	}

	baleSRSum := 0.0
	for _, sr := range baleSRs {
		baleSRSum += sr
	}
	baleSR := baleSRSum / float64(len(baleSRs))

	// Global statistics across all domains
	var allTimes []float64
	allSRSum := 0.0
	for _, r := range domResults {
		if r.avgMs >= 0 {
			allTimes = append(allTimes, r.avgMs)
		}
		allSRSum += r.successRate
	}
	globalSR := allSRSum / float64(len(domResults))

	globalAvg := -1.0
	if len(allTimes) > 0 {
		sum := 0.0
		for _, t := range allTimes {
			sum += t
		}
		globalAvg = math.Round((sum/float64(len(allTimes)))*10) / 10
	}

	// Composite Scoring Algorithm:
	// Prioritizes high reliability and low latency specifically for Bale Meet gateways.
	var score float64
	if globalSR == 0 || globalAvg < 0 {
		score = math.MaxFloat64
	} else {
		// Penalty for overall packet loss (+100ms per 1% packet loss)
		packetLoss := 100.0 - globalSR
		lossPenalty := packetLoss * 100.0

		// Special penalty for unreliability on domestic Bale Meet gateways (+50ms per 1% loss)
		baleLoss := 100.0 - baleSR
		balePenalty := baleLoss * 50.0

		// Heavy penalty for completely failed domains (+500ms per completely failed domain)
		completeFailures := 0
		for _, r := range domResults {
			if r.successRate == 0 {
				completeFailures++
			}
		}
		failurePenalty := float64(completeFailures) * 500.0

		score = math.Round((globalAvg+lossPenalty+balePenalty+failurePenalty)*10) / 10
	}

	return &DNSBenchmarkResult{
		IP:           ip,
		Online:       true,
		GoogleAvgMs:  math.Round(googleAvg*10) / 10,
		BaleAvgMs:    baleAvg,
		BaleGateways: baleGateways,
		GlobalAvgMs:  globalAvg,
		Reliability:  math.Round(globalSR*10) / 10,
		Score:        score,
	}
}

// checkOnline performs a rapid pre-check to verify if the DNS server is reachable.
func (br *BenchmarkRunner) checkOnline(ctx context.Context, ip string) bool {
	probeDomains := []string{"meet-gwbm1.ble.ir", "google.com"}
	timeout := 1200 * time.Millisecond

	for _, d := range probeDomains {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		r := &net.Resolver{
			PreferGo: true,
			Dial: func(dialCtx context.Context, network, address string) (net.Conn, error) {
				netDialer := net.Dialer{Timeout: timeout}
				return netDialer.DialContext(dialCtx, "udp", net.JoinHostPort(ip, "53"))
			},
		}

		qCtx, cancel := context.WithTimeout(ctx, timeout)
		ips, err := r.LookupIP(qCtx, "ip4", d)
		cancel()

		if err == nil && len(ips) > 0 {
			return true
		}
	}
	return false
}

// testDomain queries a domain multiple times to compute average latency and reliability.
func (br *BenchmarkRunner) testDomain(ctx context.Context, ip, domain string, iterations int, timeout time.Duration) (float64, float64) {
	var times []float64
	failures := 0

	for i := 0; i < iterations; i++ {
		select {
		case <-ctx.Done():
			return -1, 0
		default:
		}

		r := &net.Resolver{
			PreferGo: true,
			Dial: func(dialCtx context.Context, network, address string) (net.Conn, error) {
				netDialer := net.Dialer{Timeout: timeout}
				return netDialer.DialContext(dialCtx, "udp", net.JoinHostPort(ip, "53"))
			},
		}

		qCtx, cancel := context.WithTimeout(ctx, timeout)
		start := time.Now()
		ips, err := r.LookupIP(qCtx, "ip4", domain)
		dur := time.Since(start)
		cancel()

		if err != nil || len(ips) == 0 {
			failures++
		} else {
			times = append(times, dur.Seconds()*1000.0) // convert to milliseconds
		}
	}

	// Outlier Mitigation: Drop worst lag spike if >= 3 successful samples
	if len(times) >= 3 {
		maxIdx := 0
		maxVal := times[0]
		for i, v := range times {
			if v > maxVal {
				maxVal = v
				maxIdx = i
			}
		}
		times = append(times[:maxIdx], times[maxIdx+1:]...)
	}

	avgMs := -1.0
	if len(times) > 0 {
		sum := 0.0
		for _, t := range times {
			sum += t
		}
		avgMs = sum / float64(len(times))
	}

	successRate := (float64(iterations-failures) / float64(iterations)) * 100.0
	return avgMs, successRate
}

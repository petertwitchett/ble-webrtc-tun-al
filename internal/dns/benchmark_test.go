package dns

import (
	"context"
	"testing"
	"time"
)

func TestDNSBenchmarkSingleServer(t *testing.T) {
	runner := NewBenchmarkRunner()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test a well-known public DNS server
	res := runner.testSingleDNS(ctx, "1.1.1.1")
	if !res.Online {
		t.Logf("1.1.1.1 was detected as offline (network might be restricted)")
		return
	}

	t.Logf("1.1.1.1 Benchmark Result: Google=%.1fms BaleAvg=%.1fms GlobalAvg=%.1fms Reliability=%.1f%% Score=%.1f",
		res.GoogleAvgMs, res.BaleAvgMs, res.GlobalAvgMs, res.Reliability, res.Score)

	if res.Reliability <= 0 {
		t.Errorf("Expected reliability > 0, got %.1f", res.Reliability)
	}
}

func TestDNSBenchmarkRunnerStartStop(t *testing.T) {
	runner := NewBenchmarkRunner()
	servers := []string{"1.1.1.1", "8.8.8.8"}

	err := runner.Start(servers)
	if err != nil {
		t.Fatalf("Failed to start benchmark: %v", err)
	}

	status := runner.GetStatus()
	if !status.Running {
		t.Errorf("Expected running to be true")
	}

	runner.Stop()
	status = runner.GetStatus()
	if status.Running {
		t.Errorf("Expected running to be false after stop")
	}
}

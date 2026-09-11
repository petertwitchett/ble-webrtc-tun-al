package artery

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestOrchestratorDormancy(t *testing.T) {
	pool := NewArteryPool()
	orch := NewOrchestrator(pool, nil)

	var demoteCalled atomic.Bool
	var wakeupCalled atomic.Bool

	orch.SetDormancyHandlers(
		func(ctx context.Context) {
			demoteCalled.Store(true)
		},
		func(ctx context.Context) {
			wakeupCalled.Store(true)
		},
	)

	if orch.IsDormant() {
		t.Fatal("expected initial state not dormant")
	}

	// 1. Simulate opening a stream
	orch.OnStreamOpened()
	if orch.ActiveStreams() != 1 {
		t.Fatalf("expected 1 active stream, got %d", orch.ActiveStreams())
	}

	// 2. Simulate opening a second stream
	orch.OnStreamOpened()
	if orch.ActiveStreams() != 2 {
		t.Fatalf("expected 2 active streams, got %d", orch.ActiveStreams())
	}

	// 3. Close one stream
	orch.OnStreamClosed()
	if orch.ActiveStreams() != 1 {
		t.Fatalf("expected 1 active stream, got %d", orch.ActiveStreams())
	}
	if orch.IsDormant() {
		t.Fatal("expected not dormant while 1 stream remains")
	}

	// 4. Close last stream
	orch.OnStreamClosed()
	if orch.ActiveStreams() != 0 {
		t.Fatalf("expected 0 active streams, got %d", orch.ActiveStreams())
	}

	// Manually trigger dormancy to test wakeup behavior
	orch.dormantMu.Lock()
	orch.isDormant = true
	orch.dormantMu.Unlock()

	if !orch.IsDormant() {
		t.Fatal("expected dormant state true")
	}

	// 5. New stream arrives -> should immediately wake up
	orch.OnStreamOpened()
	if orch.IsDormant() {
		t.Fatal("expected isDormant to become false on stream open")
	}

	// Wait briefly for goroutine
	time.Sleep(50 * time.Millisecond)
	if !wakeupCalled.Load() {
		t.Fatal("expected wakeup handler to be called")
	}
}

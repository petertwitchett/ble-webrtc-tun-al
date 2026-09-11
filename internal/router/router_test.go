package router

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/salman/ble-webrtc-tun/internal/db"
)

func TestForceEndCallTriggersCancelAndDiscard(t *testing.T) {
	database, err := db.Init(db.RoleServer)
	if err != nil {
		t.Fatalf("failed to initialize db: %v", err)
	}

	r := NewRouter(database)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cancelCalled int32
	var discardCalled int32

	sess := &Session{
		ID:              1,
		ServerAccountID: 10,
		CallID:          99999,
		StartTime:       time.Now(),
	}
	sess.SetCancelFunc(func() {
		atomic.StoreInt32(&cancelCalled, 1)
		cancel()
	})
	sess.SetDiscardFunc(func() {
		atomic.StoreInt32(&discardCalled, 1)
	})

	r.mu.Lock()
	r.sessions[10] = sess
	r.mu.Unlock()

	r.ForceEndCall(10)

	if atomic.LoadInt32(&cancelCalled) != 1 {
		t.Errorf("expected cancelFn to be called")
	}
	if atomic.LoadInt32(&discardCalled) != 1 {
		t.Errorf("expected onDiscard to be called")
	}
	if selectSession := r.GetSession(10); selectSession != nil {
		t.Errorf("expected session 10 to be deleted from active sessions")
	}
	_ = ctx
}

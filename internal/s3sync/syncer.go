package s3sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SyncStatus holds real-time S3 sync metrics for UI reporting.
type SyncStatus struct {
	Configured   bool      `json:"configured"`
	Bucket       string    `json:"bucket"`
	Host         string    `json:"host"`
	LastSync     time.Time `json:"last_sync"`
	LastSyncAgo  string    `json:"last_sync_ago"`
	LastError    string    `json:"last_error,omitempty"`
	SyncCount    int       `json:"sync_count"`
	DBSizeBytes  int64     `json:"db_size_bytes"`
	IsSyncing    bool      `json:"is_syncing"`
	RestoredInit bool      `json:"restored_init"`
}

// Syncer manages continuous, debounced synchronization of the server database to S3.
type Syncer struct {
	client       *Client
	dbPath       string
	remoteKey    string
	checkpointFn func() error

	mu           sync.RWMutex
	lastSync     time.Time
	lastError    error
	syncCount    int
	isSyncing    bool
	restoredInit bool

	notifyCh  chan struct{}
	stopCh    chan struct{}
	closeOnce sync.Once
}

// NewSyncer initializes a new Syncer.
func NewSyncer(client *Client, dbPath string, checkpointFn func() error) *Syncer {
	return &Syncer{
		client:       client,
		dbPath:       dbPath,
		remoteKey:    "server.db",
		checkpointFn: checkpointFn,
		notifyCh:     make(chan struct{}, 10),
		stopCh:       make(chan struct{}),
	}
}

// Restore attempts to download the database from S3 before opening it.
// Returns (true, nil) if a backup was found and restored.
func (s *Syncer) Restore(ctx context.Context) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil || !s.client.Config().IsConfigured() {
		return false, fmt.Errorf("s3 client not configured")
	}

	s3Log.Info("Checking S3 bucket '%s' for existing database '%s'...", s.client.Config().Bucket, s.remoteKey)

	// Ensure bucket exists
	if err := s.client.EnsureBucket(ctx); err != nil {
		s3Log.Warn("Failed to ensure S3 bucket: %v", err)
	}

	data, found, err := s.client.GetObject(ctx, s.remoteKey)
	if err != nil {
		s3Log.Error("Failed to fetch database from S3: %v", err)
		return false, fmt.Errorf("fetch database from s3: %w", err)
	}

	if !found || len(data) == 0 {
		s3Log.Info("No existing database found in S3 bucket '%s'. Starting with fresh database.", s.client.Config().Bucket)
		return false, nil
	}

	// Ensure destination directory exists
	dir := filepath.Dir(s.dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, fmt.Errorf("create db dir %s: %w", dir, err)
	}

	// Backup any existing local file before overwriting
	if _, err := os.Stat(s.dbPath); err == nil {
		_ = os.Rename(s.dbPath, s.dbPath+".pre-restore")
	}

	if err := os.WriteFile(s.dbPath, data, 0644); err != nil {
		return false, fmt.Errorf("write restored database to %s: %w", s.dbPath, err)
	}

	s.restoredInit = true
	s.lastSync = time.Now().UTC()
	s3Log.Info("✅ Successfully restored database from S3 (%d bytes) to %s", len(data), s.dbPath)
	return true, nil
}

// Start begins the background debounce loop.
func (s *Syncer) Start(ctx context.Context) {
	go s.runLoop(ctx)
}

func (s *Syncer) runLoop(ctx context.Context) {
	debounceDuration := 2 * time.Second
	maxInterval := 15 * time.Second
	periodicTicker := time.NewTicker(60 * time.Second)
	defer periodicTicker.Stop()

	var (
		timer       *time.Timer
		timerCh     <-chan time.Time
		dirty       bool
		firstDirty  time.Time
	)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return

		case <-s.notifyCh:
			now := time.Now()
			if !dirty {
				dirty = true
				firstDirty = now
			}

			// If dirty for longer than maxInterval, trigger immediately
			if now.Sub(firstDirty) >= maxInterval {
				if timer != nil {
					timer.Stop()
					timer = nil
					timerCh = nil
				}
				dirty = false
				s.doSync(context.Background())
				continue
			}

			// Reset debounce timer
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(debounceDuration)
			timerCh = timer.C

		case <-timerCh:
			timer = nil
			timerCh = nil
			dirty = false
			s.doSync(context.Background())

		case <-periodicTicker.C:
			// Periodic safety sync if any changes were pending
			if dirty {
				if timer != nil {
					timer.Stop()
					timer = nil
					timerCh = nil
				}
				dirty = false
				s.doSync(context.Background())
			}
		}
	}
}

// NotifyChange signals that the database has been modified.
func (s *Syncer) NotifyChange() {
	select {
	case s.notifyCh <- struct{}{}:
	default:
		// Queue full, sync will happen shortly
	}
}

// SyncNow performs an immediate synchronous upload to S3.
func (s *Syncer) SyncNow(ctx context.Context) error {
	return s.doSync(ctx)
}

// FlushSync is called during graceful shutdown to ensure all changes are written.
func (s *Syncer) FlushSync(ctx context.Context) error {
	s.closeOnce.Do(func() {
		close(s.stopCh)
	})
	return s.doSync(ctx)
}

func (s *Syncer) doSync(ctx context.Context) error {
	s.mu.Lock()
	if s.isSyncing {
		s.mu.Unlock()
		return nil
	}
	s.isSyncing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isSyncing = false
		s.mu.Unlock()
	}()

	if s.client == nil || !s.client.Config().IsConfigured() {
		return fmt.Errorf("s3 client not configured")
	}

	// 1. Safely checkpoint WAL if function provided
	if s.checkpointFn != nil {
		if err := s.checkpointFn(); err != nil {
			s3Log.Warn("WAL checkpoint failed before S3 sync: %v", err)
		}
	}

	// 2. Read database file
	data, err := os.ReadFile(s.dbPath)
	if err != nil {
		s.mu.Lock()
		s.lastError = err
		s.mu.Unlock()
		s3Log.Error("Failed to read database file %s for S3 sync: %v", s.dbPath, err)
		return fmt.Errorf("read db file: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("database file %s is empty", s.dbPath)
	}

	// 3. Upload to S3
	if err := s.client.PutObject(ctx, s.remoteKey, data); err != nil {
		s.mu.Lock()
		s.lastError = err
		s.mu.Unlock()
		s3Log.Error("Failed to upload database to S3: %v", err)
		return fmt.Errorf("upload to s3: %w", err)
	}

	s.mu.Lock()
	s.lastSync = time.Now().UTC()
	s.lastError = nil
	s.syncCount++
	s.mu.Unlock()

	s3Log.Info("☁️ Database synchronized to S3 (%s/%s: %d bytes, sync #%d)",
		s.client.Config().Bucket, s.remoteKey, len(data), s.syncCount)
	return nil
}

// Status returns the current sync status for API and UI reporting.
func (s *Syncer) Status() SyncStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg := s.client.Config()
	var size int64
	if fi, err := os.Stat(s.dbPath); err == nil {
		size = fi.Size()
	}

	var lastErrStr string
	if s.lastError != nil {
		lastErrStr = s.lastError.Error()
	}

	var lastAgo string
	if !s.lastSync.IsZero() {
		diff := time.Since(s.lastSync)
		if diff < time.Minute {
			lastAgo = "Just now"
		} else if diff < time.Hour {
			lastAgo = fmt.Sprintf("%dm ago", int(diff.Minutes()))
		} else {
			lastAgo = fmt.Sprintf("%dh ago", int(diff.Hours()))
		}
	} else {
		lastAgo = "Never"
	}

	return SyncStatus{
		Configured:   cfg.IsConfigured(),
		Bucket:       cfg.Bucket,
		Host:         cfg.Host,
		LastSync:     s.lastSync,
		LastSyncAgo:  lastAgo,
		LastError:    lastErrStr,
		SyncCount:    s.syncCount,
		DBSizeBytes:  size,
		IsSyncing:    s.isSyncing,
		RestoredInit: s.restoredInit,
	}
}

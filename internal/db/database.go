package db

import (
	"fmt"
	gormlog "log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/salman/ble-webrtc-tun/internal/logger"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var (
	instance *Database
	once     sync.Once
	dbLog    = logger.New("db")
)

// Database wraps the GORM DB connection with helper methods.
type Database struct {
	DB                *gorm.DB
	role              string // "client" or "server"
	path              string
	mutationMu        sync.RWMutex
	mutationCallbacks []func()
}

// Init initializes the database singleton for the given role.
// The DB file is created at data/{role}.db relative to the working directory.
func Init(role string) (*Database, error) {
	var initErr error
	once.Do(func() {
		db, err := open(role)
		if err != nil {
			initErr = err
			return
		}
		instance = db
	})
	if initErr != nil {
		return nil, initErr
	}
	return instance, nil
}

// Get returns the database singleton. Panics if Init() hasn't been called.
func Get() *Database {
	if instance == nil {
		dbLog.Fatal("Database not initialized — call db.Init() first")
	}
	return instance
}

// open creates the SQLite database file and runs auto-migrations.
func open(role string) (*Database, error) {
	// Ensure data directory exists
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, role+".db")
	dbLog.Info("Opening database at %s", dbPath)

	// SQLite pragmas for performance + safety
	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)"

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.New(
			gormlog.New(dbLog.ErrorWriter(), "", 0),
			gormlogger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  gormlogger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
		// Disable default transaction wrapping for single queries (performance)
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Auto-migrate all models
	if err := db.AutoMigrate(
		&Account{},
		&Pairing{},
		&ConnectionLog{},
		&Event{},
		&Setting{},
		&AdminUser{},
	); err != nil {
		return nil, fmt.Errorf("auto-migrate: %w", err)
	}

	d := &Database{
		DB:   db,
		role: role,
		path: dbPath,
	}

	// Register mutation hooks on database modifications (used for S3 sync)
	notifyFn := func(tx *gorm.DB) {
		if tx.Error == nil {
			d.triggerMutation()
		}
	}
	_ = db.Callback().Create().After("gorm:create").Register("db:after_create_mutation", notifyFn)
	_ = db.Callback().Update().After("gorm:update").Register("db:after_update_mutation", notifyFn)
	_ = db.Callback().Delete().After("gorm:delete").Register("db:after_delete_mutation", notifyFn)

	dbLog.Info("✅ Database ready (role=%s)", role)

	return d, nil
}

// OnMutation registers a callback executed whenever records are created, updated, or deleted.
func (d *Database) OnMutation(cb func()) {
	d.mutationMu.Lock()
	defer d.mutationMu.Unlock()
	d.mutationCallbacks = append(d.mutationCallbacks, cb)
}

func (d *Database) triggerMutation() {
	d.mutationMu.RLock()
	if len(d.mutationCallbacks) == 0 {
		d.mutationMu.RUnlock()
		return
	}
	cbs := make([]func(), len(d.mutationCallbacks))
	copy(cbs, d.mutationCallbacks)
	d.mutationMu.RUnlock()

	for _, cb := range cbs {
		go cb()
	}
}

// CheckpointWAL instructs SQLite to checkpoint all WAL changes into the main database file.
func (d *Database) CheckpointWAL() error {
	if d.DB == nil {
		return fmt.Errorf("db not initialized")
	}
	return d.DB.Exec("PRAGMA wal_checkpoint(FULL)").Error
}

// ResetStats holds the count of records purged during a database reset.
type ResetStats struct {
	AccountsDeleted int64 `json:"accounts_deleted"`
	PairingsDeleted int64 `json:"pairings_deleted"`
	LogsDeleted     int64 `json:"logs_deleted"`
	EventsDeleted   int64 `json:"events_deleted"`
}

// ResetData completely purges accounts, pairings, connection logs, and sync events,
// while strictly preserving settings and admin users.
func (d *Database) ResetData() (*ResetStats, error) {
	stats := &ResetStats{}

	err := d.DB.Transaction(func(tx *gorm.DB) error {
		_ = tx.Exec("PRAGMA foreign_keys = OFF")
		defer tx.Exec("PRAGMA foreign_keys = ON")

		// 1. Delete connection logs first (references Accounts and Pairings)
		resL := tx.Unscoped().Where("1 = 1").Delete(&ConnectionLog{})
		if resL.Error != nil {
			return resL.Error
		}
		stats.LogsDeleted = resL.RowsAffected

		// 2. Delete pairings (references Accounts)
		resP := tx.Unscoped().Where("1 = 1").Delete(&Pairing{})
		if resP.Error != nil {
			return resP.Error
		}
		stats.PairingsDeleted = resP.RowsAffected

		// 3. Delete sync events
		resE := tx.Unscoped().Where("1 = 1").Delete(&Event{})
		if resE.Error != nil {
			return resE.Error
		}
		stats.EventsDeleted = resE.RowsAffected

		// 4. Delete accounts
		resA := tx.Unscoped().Where("1 = 1").Delete(&Account{})
		if resA.Error != nil {
			return resA.Error
		}
		stats.AccountsDeleted = resA.RowsAffected

		return nil
	})

	if err != nil {
		return nil, err
	}

	d.triggerMutation()
	_ = d.CheckpointWAL()

	dbLog.Info("🗑️ Database reset completed: %d accounts, %d pairings, %d logs, %d events deleted (settings & admin preserved)",
		stats.AccountsDeleted, stats.PairingsDeleted, stats.LogsDeleted, stats.EventsDeleted)

	return stats, nil
}

// Role returns the database role (client or server).
func (d *Database) Role() string {
	return d.role
}

// Path returns the absolute path to the database file.
func (d *Database) Path() string {
	return d.path
}

// Close closes the underlying database connection.
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}



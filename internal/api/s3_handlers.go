package api

import (
	"net/http"

	"github.com/salman/ble-webrtc-tun/internal/s3sync"
)

// handleS3Status returns the current S3 / Cellar cloud persistence status.
func (s *Server) handleS3Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.S3Syncer == nil {
		writeJSON(w, http.StatusOK, s3sync.SyncStatus{
			Configured: false,
		})
		return
	}

	status := s.S3Syncer.Status()
	writeJSON(w, http.StatusOK, status)
}

// handleS3Backup triggers an immediate database backup to S3.
func (s *Server) handleS3Backup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.S3Syncer == nil {
		writeError(w, http.StatusBadRequest, "S3 persistence is not enabled on this instance")
		return
	}

	if err := s.S3Syncer.SyncNow(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "S3 backup failed: "+err.Error())
		return
	}

	status := s.S3Syncer.Status()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Database successfully backed up to S3",
		"status":  status,
	})
}

// handleS3Restore triggers a manual database restore from S3.
func (s *Server) handleS3Restore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.S3Syncer == nil {
		writeError(w, http.StatusBadRequest, "S3 persistence is not enabled on this instance")
		return
	}

	restored, err := s.S3Syncer.Restore(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "S3 restore failed: "+err.Error())
		return
	}

	if !restored {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":  false,
			"restored": false,
			"message":  "No existing backup found in S3 bucket",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"restored": true,
		"message":  "Database successfully restored from S3",
		"status":   s.S3Syncer.Status(),
	})
}

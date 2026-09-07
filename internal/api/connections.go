package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/salman/ble-webrtc-tun/internal/db"
)

// handleActiveConnections handles GET /api/connections/active.
func (s *Server) handleActiveConnections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Get sessions from router (in-memory, real-time)
	sessions := s.router.GetAllSessions()

	// Also get from DB (for sessions that might have been missed)
	dbActive, err := s.database.GetActiveConnections()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := map[string]interface{}{
		"sessions":   sessions,
		"db_records": dbActive,
		"count":      len(sessions),
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleConnectionHistory handles GET /api/connections/history?limit=50.
func (s *Server) handleConnectionHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := s.database.GetConnectionHistory(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, history)
}

// handleForceEndCall handles POST /api/connections/end/{server_account_id}.
// Forces termination of an active call by server account ID.
func (s *Server) handleForceEndCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract server_account_id from URL: /api/connections/end/123
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) == 0 {
		writeError(w, http.StatusBadRequest, "missing server_account_id")
		return
	}
	idStr := parts[len(parts)-1]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid server_account_id")
		return
	}

	session := s.router.GetSession(uint(id))
	var callID int64
	if session != nil {
		callID = session.CallID
		s.router.ForceEndCall(uint(id))
		apiLog.Info("Admin force-ended call for server account %d (callID=%d)", id, session.CallID)
	} else {
		// Even if no active memory session exists, reset stuck account in DB
		s.database.SetAccountStatus(uint(id), db.StatusIdle)
		apiLog.Info("Admin reset server account %d status to idle (no active in-memory session)", id)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":           "call ended",
		"server_account_id": id,
		"call_id":           callID,
	})
}

// handleForceEndAllCalls handles POST /api/connections/end-all.
// Forces termination of ALL active calls.
func (s *Server) handleForceEndAllCalls(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sessions := s.router.GetAllSessions()
	count := len(sessions)

	for _, session := range sessions {
		s.router.ForceEndCall(session.ServerAccountID)
	}

	// Also reset any accounts stuck in IN_CALL or RESERVED state in the DB
	accounts, _ := s.database.ListAccounts("")
	resetCount := 0
	for _, a := range accounts {
		if a.Status == db.StatusInCall || a.Status == db.StatusReserved {
			s.database.SetAccountStatus(a.ID, db.StatusIdle)
			resetCount++
		}
	}

	apiLog.Info("Admin force-ended ALL calls (%d active sessions, %d DB accounts reset)", count, resetCount)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":        fmt.Sprintf("%d calls ended, %d accounts reset", count, resetCount),
		"count":          count,
		"accounts_reset": resetCount,
	})
}

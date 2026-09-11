package api

import (
	"encoding/json"
	"net/http"

	"github.com/salman/ble-webrtc-tun/internal/dns"
)

type dnsBenchmarkStartRequest struct {
	Servers []string `json:"servers,omitempty"`
}

// handleDNSBenchmarkStart begins an asynchronous benchmark across DNS servers.
func (s *Server) handleDNSBenchmarkStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}

	var req dnsBenchmarkStartRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	runner := dns.DefaultBenchmarkRunner()
	if err := runner.Start(req.Servers); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	status := runner.GetStatus()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "started",
		"total":  status.Total,
	})
}

// handleDNSBenchmarkStatus returns the current progress and sorted ranking results.
func (s *Server) handleDNSBenchmarkStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}

	runner := dns.DefaultBenchmarkRunner()
	status := runner.GetStatus()
	writeJSON(w, http.StatusOK, status)
}

// handleDNSBenchmarkStop halts an ongoing benchmark run.
func (s *Server) handleDNSBenchmarkStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}

	runner := dns.DefaultBenchmarkRunner()
	runner.Stop()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "stopped",
	})
}

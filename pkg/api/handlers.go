package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"freenode/pkg/logger"
	"freenode/pkg/mhr"
)

// handleMHRConfig handles GET (read) and POST (save) for MHR settings
func (s *Server) handleMHRConfig(w http.ResponseWriter, r *http.Request) {
	if s.mhrMgr == nil {
		writeErr(w, http.StatusServiceUnavailable, "MHR manager not initialized")
		return
	}

	switch r.Method {
	case http.MethodGet:
		cfg := s.mhrMgr.GetConfig()
		writeJSON(w, http.StatusOK, cfg)

	case http.MethodPost:
		var req mhr.Config
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}

		if err := s.mhrMgr.SaveConfig(req); err != nil {
			logger.Error("api", "Failed saving MHR config: %v", err)
			writeErr(w, http.StatusInternalServerError, fmt.Sprintf("failed saving MHR config: %v", err))
			return
		}

		logger.Info("api", "MHR configuration updated (Variant: %s, HTTP: %d, SOCKS5: %d)",
			req.Variant, req.HTTPPort, req.Socks5Port)
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"config":  s.mhrMgr.GetConfig(),
			"message": "MHR configuration saved successfully",
		})

	default:
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleMHRStart launches the MHR relay process
func (s *Server) handleMHRStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.mhrMgr == nil {
		writeErr(w, http.StatusServiceUnavailable, "MHR manager not initialized")
		return
	}

	logger.Info("api", "MHR relay start requested")
	if err := s.mhrMgr.Start(); err != nil {
		logger.Error("api", "Failed starting MHR relay: %v", err)
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("Failed to start MHR: %v", err))
		return
	}

	status := s.mhrMgr.GetStatus()
	logger.Info("api", "MHR relay started successfully (PID: %d)", status.PID)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"status":  status,
		"message": fmt.Sprintf("MHR Relay started on HTTP :%d and SOCKS5 :%d", status.HTTPPort, status.Socks5Port),
	})
}

// handleMHRStop cleanly stops the MHR relay process
func (s *Server) handleMHRStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.mhrMgr == nil {
		writeErr(w, http.StatusServiceUnavailable, "MHR manager not initialized")
		return
	}

	logger.Info("api", "MHR relay stop requested")
	if err := s.mhrMgr.Stop(); err != nil {
		logger.Error("api", "Error stopping MHR relay: %v", err)
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("Error stopping MHR: %v", err))
		return
	}

	status := s.mhrMgr.GetStatus()
	logger.Info("api", "MHR relay stopped successfully")
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"status":  status,
		"message": "MHR Relay stopped and system proxy restored",
	})
}

// handleMHRStatus returns current live status and metrics
func (s *Server) handleMHRStatus(w http.ResponseWriter, r *http.Request) {
	if s.mhrMgr == nil {
		writeErr(w, http.StatusServiceUnavailable, "MHR manager not initialized")
		return
	}
	writeJSON(w, http.StatusOK, s.mhrMgr.GetStatus())
}

// handleMHRTest verifies connectivity through MHR proxy
func (s *Server) handleMHRTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.mhrMgr == nil {
		writeErr(w, http.StatusServiceUnavailable, "MHR manager not initialized")
		return
	}

	if !s.mhrMgr.IsRunning() {
		writeErr(w, http.StatusBadRequest, "MHR Relay is not running")
		return
	}

	lat, exitIP, err := s.mhrMgr.VerifyTunnel(10 * time.Second)
	if err != nil {
		writeErr(w, http.StatusBadGateway, fmt.Sprintf("MHR connectivity check failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"latency": lat,
		"exit_ip": exitIP,
		"message": fmt.Sprintf("MHR tunnel verified (%d ms, exit IP: %s)", lat, exitIP),
	})
}

// handleMHRScanGoogleIP runs TLS SNI probes on candidate Google IPs
func (s *Server) handleMHRScanGoogleIP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.mhrMgr == nil {
		writeErr(w, http.StatusServiceUnavailable, "MHR manager not initialized")
		return
	}

	var req struct {
		FrontDomain string `json:"front_domain"`
		TimeoutMs   int    `json:"timeout_ms"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	timeout := 2500 * time.Millisecond
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}

	results, err := s.mhrMgr.ScanGoogleIPs(req.FrontDomain, timeout)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("scan failed: %v", err))
		return
	}

	workingCount := 0
	var fastestIP string
	for _, res := range results {
		if res.Reachable {
			workingCount++
			if fastestIP == "" {
				fastestIP = res.IP
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"total":         len(results),
		"working_count": workingCount,
		"fastest_ip":    fastestIP,
		"results":       results,
	})
}

// handleMHRSystemProxy toggles Windows system proxy to MHR HTTP port
func (s *Server) handleMHRSystemProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.mhrMgr == nil {
		writeErr(w, http.StatusServiceUnavailable, "MHR manager not initialized")
		return
	}

	var req struct {
		Enable bool `json:"enable"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.mhrMgr.SetSystemProxy(req.Enable); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":      true,
		"system_proxy": req.Enable,
		"message":      fmt.Sprintf("System proxy updated: %v", req.Enable),
	})
}

// handleMHRScripts returns deployment script code for Google Apps Script & Cloudflare Worker
func (s *Server) handleMHRScripts(w http.ResponseWriter, r *http.Request) {
	if s.mhrMgr == nil {
		writeErr(w, http.StatusServiceUnavailable, "MHR manager not initialized")
		return
	}

	scripts, err := s.mhrMgr.GetDeploymentScripts()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("failed retrieving scripts: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, scripts)
}

// handleMHRInstallCA installs the MITM CA certificate into the Windows Root Certificate store
func (s *Server) handleMHRInstallCA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.mhrMgr == nil {
		writeErr(w, http.StatusServiceUnavailable, "MHR manager not initialized")
		return
	}

	output, err := s.mhrMgr.InstallCA()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("Failed installing CA: %v (output: %s)", err, strings.TrimSpace(output)))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"output":  strings.TrimSpace(output),
		"message": "MHR Root CA certificate installed successfully",
	})
}

// handleMHRUninstallCA removes the MITM CA certificate from the Windows Root Certificate store
func (s *Server) handleMHRUninstallCA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.mhrMgr == nil {
		writeErr(w, http.StatusServiceUnavailable, "MHR manager not initialized")
		return
	}

	output, err := s.mhrMgr.UninstallCA()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("Failed removing CA: %v (output: %s)", err, strings.TrimSpace(output)))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"output":  strings.TrimSpace(output),
		"message": "MHR Root CA certificate removed successfully",
	})
}

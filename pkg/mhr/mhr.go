package mhr

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"freenode/pkg/database"
	"freenode/pkg/proxy"
)

const (
	VariantMHR    = "mhr"
	VariantMHRCFW = "mhr-cfw"

	DefaultHTTPPort    = 8085
	DefaultSocks5Port  = 1080
	DefaultGoogleIP    = "216.239.38.120"
	DefaultFrontDomain = "www.google.com"
)

// Config represents user-configurable settings for the MHR relay subsystem
type Config struct {
	Enabled       bool   `json:"enabled"`
	Variant       string `json:"variant"` // "mhr" or "mhr-cfw"
	ScriptID      string `json:"script_id"`
	AuthKey       string `json:"auth_key"`
	HTTPPort      int    `json:"http_port"`
	Socks5Port    int    `json:"socks5_port"`
	GoogleIP      string `json:"google_ip"`
	FrontDomain   string `json:"front_domain"`
	SystemProxy   bool   `json:"system_proxy"`
	LANSharing    bool   `json:"lan_sharing"`
	CFWWorkerURL  string `json:"cfw_worker_url,omitempty"`
}

// Status represents runtime health and metrics of the MHR relay
type Status struct {
	Running     bool      `json:"running"`
	State       string    `json:"state"` // "stopped", "starting", "running", "error"
	PID         int       `json:"pid"`
	Variant     string    `json:"variant"`
	HTTPPort    int       `json:"http_port"`
	Socks5Port  int       `json:"socks5_port"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	UptimeSec   int64     `json:"uptime_sec"`
	SystemProxy bool      `json:"system_proxy"`
	ExitIP      string    `json:"exit_ip,omitempty"`
	Latency     int       `json:"latency"`
	Error       string    `json:"error,omitempty"`
}

// GoogleIPResult stores probe measurements for Google fronting IPs
type GoogleIPResult struct {
	IP        string `json:"ip"`
	LatencyMs int    `json:"latency_ms"`
	Reachable bool   `json:"reachable"`
	Error     string `json:"error,omitempty"`
}

// Manager controls the MHR relay lifecycle
type Manager struct {
	mu          sync.RWMutex
	db          *database.DB
	proxyMgr    *proxy.Manager
	config      Config
	status      Status
	process     *os.Process
	stopChan    chan struct{}
	pythonPath  string
	runtimeDir  string
	dataDir     string
}

// NewManager creates an MHR manager instance
func NewManager(db *database.DB, proxyMgr *proxy.Manager, dataDir string) *Manager {
	if dataDir == "" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			dataDir = filepath.Join(appData, "Conective")
		} else {
			dataDir = "./data"
		}
	}

	m := &Manager{
		db:       db,
		proxyMgr: proxyMgr,
		dataDir:  dataDir,
		status: Status{
			State: "stopped",
		},
	}
	m.config = m.LoadConfig()
	return m
}

// LoadConfig reads MHR configuration from SQLite settings table
func (m *Manager) LoadConfig() Config {
	m.mu.Lock()
	defer m.mu.Unlock()

	var cfg Config
	cfg.Variant = m.getSetting("mhr_variant", VariantMHR)
	if cfg.Variant != VariantMHR && cfg.Variant != VariantMHRCFW {
		cfg.Variant = VariantMHR
	}

	cfg.Enabled = m.getSetting("mhr_enabled", "false") == "true"
	cfg.ScriptID = strings.TrimSpace(m.getSetting("mhr_script_id", ""))
	cfg.AuthKey = strings.TrimSpace(m.getSetting("mhr_auth_key", ""))

	httpPort, _ := strconv.Atoi(m.getSetting("mhr_http_port", strconv.Itoa(DefaultHTTPPort)))
	if httpPort <= 0 || httpPort > 65535 {
		httpPort = DefaultHTTPPort
	}
	cfg.HTTPPort = httpPort

	socksPort, _ := strconv.Atoi(m.getSetting("mhr_socks5_port", strconv.Itoa(DefaultSocks5Port)))
	if socksPort <= 0 || socksPort > 65535 {
		socksPort = DefaultSocks5Port
	}
	cfg.Socks5Port = socksPort

	cfg.GoogleIP = m.getSetting("mhr_google_ip", DefaultGoogleIP)
	cfg.FrontDomain = m.getSetting("mhr_front_domain", DefaultFrontDomain)
	cfg.SystemProxy = m.getSetting("mhr_system_proxy", "true") == "true"
	cfg.LANSharing = m.getSetting("mhr_lan_sharing", "false") == "true"
	cfg.CFWWorkerURL = m.getSetting("mhr_cfw_worker_url", "")

	m.config = cfg
	return cfg
}

// SaveConfig persists MHR configuration to the database
func (m *Manager) SaveConfig(cfg Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.Variant != VariantMHR && cfg.Variant != VariantMHRCFW {
		cfg.Variant = VariantMHR
	}
	if cfg.HTTPPort <= 0 || cfg.HTTPPort > 65535 {
		cfg.HTTPPort = DefaultHTTPPort
	}
	if cfg.Socks5Port <= 0 || cfg.Socks5Port > 65535 {
		cfg.Socks5Port = DefaultSocks5Port
	}
	if cfg.GoogleIP == "" {
		cfg.GoogleIP = DefaultGoogleIP
	}
	if cfg.FrontDomain == "" {
		cfg.FrontDomain = DefaultFrontDomain
	}

	m.setSetting("mhr_variant", cfg.Variant)
	m.setSetting("mhr_enabled", strconv.FormatBool(cfg.Enabled))
	m.setSetting("mhr_script_id", strings.TrimSpace(cfg.ScriptID))
	m.setSetting("mhr_auth_key", strings.TrimSpace(cfg.AuthKey))
	m.setSetting("mhr_http_port", strconv.Itoa(cfg.HTTPPort))
	m.setSetting("mhr_socks5_port", strconv.Itoa(cfg.Socks5Port))
	m.setSetting("mhr_google_ip", strings.TrimSpace(cfg.GoogleIP))
	m.setSetting("mhr_front_domain", strings.TrimSpace(cfg.FrontDomain))
	m.setSetting("mhr_system_proxy", strconv.FormatBool(cfg.SystemProxy))
	m.setSetting("mhr_lan_sharing", strconv.FormatBool(cfg.LANSharing))
	m.setSetting("mhr_cfw_worker_url", strings.TrimSpace(cfg.CFWWorkerURL))

	m.config = cfg
	return nil
}

// GetConfig returns the active MHR configuration
func (m *Manager) GetConfig() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetStatus returns the live status
func (m *Manager) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	st := m.status
	if st.Running && !st.StartedAt.IsZero() {
		st.UptimeSec = int64(time.Since(st.StartedAt).Seconds())
	}
	return st
}

// OnExternalProxyActive is called when Xray or Psiphon activates system proxy
func (m *Manager) OnExternalProxyActive() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.status.SystemProxy {
		m.status.SystemProxy = false
	}
}

func (m *Manager) getSetting(key, def string) string {
	if m.db == nil {
		return def
	}
	return m.db.GetSetting(key, def)
}

func (m *Manager) setSetting(key, val string) {
	if m.db != nil {
		_ = m.db.SetSetting(key, val)
	}
}

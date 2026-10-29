package mhr

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"freenode/pkg/logger"
	"freenode/pkg/models"
)

// Start prepares the configuration, ensures runtimes, and launches MHR
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg := m.config
	if strings.TrimSpace(cfg.ScriptID) == "" {
		return fmt.Errorf("Google Apps Script Deployment ID is required")
	}
	if strings.TrimSpace(cfg.AuthKey) == "" {
		return fmt.Errorf("AUTH_KEY is required")
	}

	// 1. Check for port collisions with Conective Xray/Psiphon core
	if err := m.checkPortCollisionLocked(cfg.HTTPPort, cfg.Socks5Port); err != nil {
		return err
	}

	// 2. Stop any existing running instance
	m.stopProcessLocked()

	// 3. Ensure runtime files are ready
	runtimeDir, err := m.EnsureRuntime(cfg.Variant)
	if err != nil {
		m.status.State = "error"
		m.status.Error = err.Error()
		return fmt.Errorf("runtime setup failed: %w", err)
	}
	m.runtimeDir = runtimeDir

	// 4. Locate Python executable
	pyExe, err := FindPythonExecutable()
	if err != nil {
		m.status.State = "error"
		m.status.Error = err.Error()
		return err
	}
	m.pythonPath = pyExe

	// 5. Check if ports are available
	if !isPortFree(cfg.HTTPPort) {
		return fmt.Errorf("HTTP port %d is already in use by another application", cfg.HTTPPort)
	}
	if !isPortFree(cfg.Socks5Port) {
		return fmt.Errorf("SOCKS5 port %d is already in use by another application", cfg.Socks5Port)
	}

	// 6. Write config.json
	if err := m.writeConfigFileLocked(runtimeDir, cfg); err != nil {
		m.status.State = "error"
		m.status.Error = err.Error()
		return fmt.Errorf("failed writing config.json: %w", err)
	}

	// 7. Launch Python process
	args := []string{"main.py", "--config", "config.json", "--no-cert-check"}
	cmd := exec.Command(pyExe, args...)
	cmd.Dir = runtimeDir
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed creating stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		m.status.State = "error"
		m.status.Error = err.Error()
		return fmt.Errorf("failed starting Python MHR process: %w", err)
	}

	m.process = cmd.Process
	m.stopChan = make(chan struct{})
	pid := cmd.Process.Pid

	logger.Info("mhr", "Started MHR relay process (PID: %d, Variant: %s, HTTP: %d, SOCKS5: %d)",
		pid, cfg.Variant, cfg.HTTPPort, cfg.Socks5Port)

	// Stream stdout & stderr into Conective logger
	go m.streamLogs(stdoutPipe, false)
	go m.streamLogs(stderrPipe, true)

	// Monitor process termination in background
	go func() {
		_ = cmd.Wait()
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.process != nil && m.process.Pid == pid {
			m.process = nil
			m.status.Running = false
			m.status.State = "stopped"
			logger.Warn("mhr", "MHR relay process (PID: %d) exited", pid)
		}
	}()

	// 8. Poll for port binding (up to 30 seconds to allow python startup / network warmup)
	portsBound := false
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(250 * time.Millisecond)
		if isPortListening(cfg.HTTPPort) || isPortListening(cfg.Socks5Port) {
			portsBound = true
			break
		}
		// If process died immediately
		if m.process == nil {
			break
		}
	}

	if !portsBound {
		m.stopProcessLocked()
		return fmt.Errorf("MHR relay failed to bind ports %d/%d within timeout (check logs for Python dependencies)",
			cfg.HTTPPort, cfg.Socks5Port)
	}

	m.status.Running = true
	m.status.State = "running"
	m.status.PID = pid
	m.status.Variant = cfg.Variant
	m.status.HTTPPort = cfg.HTTPPort
	m.status.Socks5Port = cfg.Socks5Port
	m.status.StartedAt = time.Now()
	m.status.Error = ""
	cfg.Enabled = true
	m.config = cfg
	m.setSetting("mhr_enabled", "true")

	// 9. Update / create local SOCKS node in database
	go m.registerLocalNode(cfg)

	// 10. Enable Windows System Proxy if requested
	if cfg.SystemProxy && m.proxyMgr != nil {
		if err := m.proxyMgr.Enable(cfg.HTTPPort); err != nil {
			logger.Error("mhr", "Failed enabling system proxy on port %d: %v", cfg.HTTPPort, err)
		} else {
			m.status.SystemProxy = true
			logger.Info("mhr", "Windows system proxy routed to MHR on port %d", cfg.HTTPPort)
		}
	}

	// Auto-install MHR Root CA into Windows trust store so HTTPS & YouTube MITM interception works
	go func() {
		if out, err := m.InstallCA(); err != nil {
			logger.Warn("mhr", "CA verification note: %v (%s)", err, strings.TrimSpace(out))
		} else {
			logger.Info("mhr", "MHR Root CA verified / installed into Windows trust store")
		}
	}()

	// 11. Run background connectivity check
	go func() {
		time.Sleep(1 * time.Second)
		lat, exitIP, err := m.VerifyTunnel(10 * time.Second)
		if err == nil {
			m.mu.Lock()
			m.status.Latency = lat
			m.status.ExitIP = exitIP
			m.mu.Unlock()
			logger.Info("mhr", "MHR tunnel active (Latency: %dms, Exit IP: %s)", lat, exitIP)
		}
	}()

	return nil
}

// Stop cleanly terminates the MHR process and restores system proxy
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopProcessLocked()
}

func silentCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
	}
	return cmd
}

func (m *Manager) stopProcessLocked() error {
	if m.process != nil {
		pid := m.process.Pid
		logger.Info("mhr", "Stopping MHR relay process (PID: %d)...", pid)

		if runtime.GOOS == "windows" {
			_ = silentCmd("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run()
		} else {
			_ = m.process.Kill()
		}
		m.process = nil
	}

	if m.stopChan != nil {
		close(m.stopChan)
		m.stopChan = nil
	}

	// Restore system proxy if MHR held it
	if m.status.SystemProxy && m.proxyMgr != nil {
		_ = m.proxyMgr.Disable()
		m.status.SystemProxy = false
		logger.Info("mhr", "Windows system proxy cleared")
	}

	m.status.Running = false
	m.status.State = "stopped"
	m.status.PID = 0
	m.status.Latency = 0
	m.status.ExitIP = ""
	m.config.Enabled = false
	m.setSetting("mhr_enabled", "false")

	// Release lingering sockets
	ensurePortsReleased(m.config.HTTPPort, m.config.Socks5Port)
	return nil
}

// IsRunning reports whether MHR is actively running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status.Running && m.process != nil
}

// SetSystemProxy toggles system proxy for MHR
func (m *Manager) SetSystemProxy(enable bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.status.Running {
		return fmt.Errorf("cannot toggle system proxy when MHR is not running")
	}

	if m.proxyMgr == nil {
		return fmt.Errorf("system proxy manager not available")
	}

	if enable {
		if err := m.proxyMgr.Enable(m.status.HTTPPort); err != nil {
			return err
		}
		m.status.SystemProxy = true
		m.config.SystemProxy = true
		m.setSetting("mhr_system_proxy", "true")
	} else {
		if err := m.proxyMgr.Disable(); err != nil {
			return err
		}
		m.status.SystemProxy = false
		m.config.SystemProxy = false
		m.setSetting("mhr_system_proxy", "false")
	}
	return nil
}

// VerifyTunnel probes outbound internet access via MHR proxy
func (m *Manager) VerifyTunnel(timeout time.Duration) (int, string, error) {
	m.mu.RLock()
	httpPort := m.config.HTTPPort
	socksPort := m.config.Socks5Port
	m.mu.RUnlock()

	if httpPort <= 0 && socksPort <= 0 {
		return -1, "", fmt.Errorf("MHR ports not configured")
	}

	proxyURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", httpPort))
	if err != nil {
		return -1, "", err
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy:             http.ProxyURL(proxyURL),
			DisableKeepAlives: true,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: timeout,
	}

	testEndpoints := []string{
		"http://cp.cloudflare.com/generate_204",
		"https://www.google.com/generate_204",
	}

	start := time.Now()
	var success bool
	for _, ep := range testEndpoints {
		resp, err := client.Get(ep)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 200 || resp.StatusCode == 204 {
				success = true
				break
			}
		}
	}

	if !success {
		return -1, "", fmt.Errorf("unable to reach test endpoint through MHR relay")
	}
	elapsed := int(time.Since(start).Milliseconds())

	// Fetch public exit IP
	exitIP := ""
	ipResp, err := client.Get("https://api.ipify.org")
	if err == nil {
		defer ipResp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(ipResp.Body, 64))
		exitIP = strings.TrimSpace(string(body))
	}

	return elapsed, exitIP, nil
}

func (m *Manager) checkPortCollisionLocked(httpPort, socks5Port int) error {
	xrayHttp := 10809
	xraySocks := 10808
	if m.db != nil {
		if p, err := strconv.Atoi(m.db.GetSetting("http_port", "10809")); err == nil && p > 0 {
			xrayHttp = p
		}
		if p, err := strconv.Atoi(m.db.GetSetting("socks_port", "10808")); err == nil && p > 0 {
			xraySocks = p
		}
	}

	if httpPort == xrayHttp || httpPort == xraySocks {
		return fmt.Errorf("MHR HTTP port (%d) collides with Conective core proxy port (%d/%d)", httpPort, xrayHttp, xraySocks)
	}
	if socks5Port == xrayHttp || socks5Port == xraySocks {
		return fmt.Errorf("MHR SOCKS5 port (%d) collides with Conective core proxy port (%d/%d)", socks5Port, xrayHttp, xraySocks)
	}
	if httpPort == socks5Port {
		return fmt.Errorf("MHR HTTP port and SOCKS5 port cannot be identical (%d)", httpPort)
	}

	return nil
}

func (m *Manager) writeConfigFileLocked(dir string, cfg Config) error {
	examplePath := filepath.Join(dir, "config.example.json")
	targetPath := filepath.Join(dir, "config.json")

	var data map[string]any
	if content, err := os.ReadFile(examplePath); err == nil {
		_ = json.Unmarshal(content, &data)
	}
	if data == nil {
		data = make(map[string]any)
	}

	data["script_id"] = strings.TrimSpace(cfg.ScriptID)
	data["auth_key"] = strings.TrimSpace(cfg.AuthKey)
	data["http_port"] = cfg.HTTPPort
	data["listen_port"] = cfg.HTTPPort
	data["socks5_port"] = cfg.Socks5Port
	data["socks5_enabled"] = true
	data["verify_ssl"] = true
	data["warmup_before_listen"] = false

	if cfg.GoogleIP != "" {
		data["google_ip"] = strings.TrimSpace(cfg.GoogleIP)
	}
	if cfg.FrontDomain != "" {
		data["front_domain"] = strings.TrimSpace(cfg.FrontDomain)
	}

	if cfg.LANSharing {
		data["lan_sharing"] = true
		data["listen_host"] = "0.0.0.0"
	} else {
		data["lan_sharing"] = false
		data["listen_host"] = "127.0.0.1"
	}

	marshaled, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, marshaled, 0644)
}

func (m *Manager) streamLogs(pipe io.Reader, isErr bool) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		upper := strings.ToUpper(line)
		if strings.Contains(upper, " ERROR ") || strings.Contains(upper, "CRITICAL") || strings.Contains(upper, "TRACEBACK") || strings.Contains(upper, "EXCEPTION") {
			logger.Error("mhr", "%s", line)
		} else if strings.Contains(upper, " WARN") || strings.Contains(upper, "WARNING") {
			logger.Warn("mhr", "%s", line)
		} else {
			logger.Info("mhr", "%s", line)
		}
	}
}

func (m *Manager) registerLocalNode(cfg Config) {
	if m.db == nil {
		return
	}
	name := fmt.Sprintf("MHR Local Relay (%s)", strings.ToUpper(cfg.Variant))
	identity := fmt.Sprintf("mhr-local-%s", cfg.Variant)

	node := &models.Config{
		Identity:    identity,
		Protocol:    "socks",
		Name:        name,
		Server:      "127.0.0.1",
		Port:        cfg.Socks5Port,
		Transport:   "tcp",
		TLS:         "none",
		Country:     "US",
		CountryName: "MHR Relay",
		Source:      "MHR Local Relay",
		Status:      "working",
		Tags:        "mhr,relay,local",
		Score:       99,
		FirstSeen:   time.Now(),
		LastSeen:    time.Now(),
		LastTested:  time.Now(),
	}

	_, _ = m.db.UpsertConfig(node)
}

// InstallCA runs python main.py --install-cert
func (m *Manager) InstallCA() (string, error) {
	m.mu.RLock()
	runtimeDir := m.runtimeDir
	pyExe := m.pythonPath
	m.mu.RUnlock()

	if runtimeDir == "" {
		var err error
		runtimeDir, err = m.EnsureRuntime(VariantMHR)
		if err != nil {
			return "", err
		}
	}
	if pyExe == "" {
		var err error
		pyExe, err = FindPythonExecutable()
		if err != nil {
			return "", err
		}
	}

	cmd := silentCmd(pyExe, "main.py", "--install-cert")
	cmd.Dir = runtimeDir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// UninstallCA runs python main.py --uninstall-cert
func (m *Manager) UninstallCA() (string, error) {
	m.mu.RLock()
	runtimeDir := m.runtimeDir
	pyExe := m.pythonPath
	m.mu.RUnlock()

	if runtimeDir == "" {
		var err error
		runtimeDir, err = m.EnsureRuntime(VariantMHR)
		if err != nil {
			return "", err
		}
	}
	if pyExe == "" {
		var err error
		pyExe, err = FindPythonExecutable()
		if err != nil {
			return "", err
		}
	}

	cmd := silentCmd(pyExe, "main.py", "--uninstall-cert")
	cmd.Dir = runtimeDir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func isPortFree(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

func isPortListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 150*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func ensurePortsReleased(httpPort, socks5Port int) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		hFree := isPortFree(httpPort)
		sFree := isPortFree(socks5Port)
		if hFree && sFree {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

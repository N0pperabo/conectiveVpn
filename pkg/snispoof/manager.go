package snispoof

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"

	"freenode/pkg/logger"
	"freenode/pkg/tun"
)

// Manager coordinates the lifecycle of the SNI Spoofing forwarder and WinDivert packet filter
type Manager struct {
	mu           sync.RWMutex
	cfg          Config
	status       Status
	cmd          *exec.Cmd
	binDir       string
	activeTarget string
	activePort   int
	activeEngine string
}

// NewManager initializes a new SNI Spoofing Manager
func NewManager(binDir string) *Manager {
	m := &Manager{
		cfg: Config{
			Enabled:     false,
			Engine:      EngineAuto,
			ListenHost:  DefaultListenHost,
			ListenPort:  DefaultListenPort,
			ConnectPort: DefaultConnectPort,
			FakeSNI:     DefaultFakeSNI,
		},
		binDir: binDir,
	}
	m.cfg.Normalize()
	m.status = Status{
		Enabled:    false,
		Running:    false,
		ListenHost: m.cfg.ListenHost,
		ListenPort: m.cfg.ListenPort,
		FakeSNI:    m.cfg.FakeSNI,
	}
	return m
}

// SetConfig updates the SNI Spoofing configuration
func (m *Manager) SetConfig(cfg Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cfg.Normalize()
	m.cfg = cfg
	m.status.Enabled = cfg.Enabled
	m.status.ListenHost = cfg.ListenHost
	m.status.ListenPort = cfg.ListenPort
	m.status.FakeSNI = cfg.FakeSNI
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

// GetStatus returns the current status
func (m *Manager) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	st := m.status
	if m.cmd != nil && m.cmd.Process != nil && m.status.Running {
		// Verify process is still alive
		if !isProcessAlive(m.cmd) {
			st.Running = false
			st.ActiveEngine = "None"
			st.Error = "Process exited unexpectedly"
		}
	}
	return st
}

// IsRunning reports whether the SNI spoofer is active
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status.Running && isProcessAlive(m.cmd)
}

// IsEnabled reports whether SNI Spoofing is enabled in configuration
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.Enabled
}

// GetListenAddress returns the active loopback listening address and port
func (m *Manager) GetListenAddress() (string, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	host := m.cfg.ListenHost
	if host == "" {
		host = DefaultListenHost
	}
	port := m.cfg.ListenPort
	if port <= 0 {
		port = DefaultListenPort
	}
	return host, port
}

// Start launches the SNI Spoofing forwarder and configures WinDivert
func (m *Manager) Start(targetIP string, targetPort int, fakeSNI string, nodeID int64, nodeName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if runtime.GOOS != "windows" {
		return fmt.Errorf("SNI Spoofing is currently only supported on Windows (requires WinDivert)")
	}

	if !tun.IsAdmin() {
		return fmt.Errorf("SNI Spoofing requires Administrator privileges to load the WinDivert kernel driver")
	}

	// Resolve target IP
	cleanTarget := strings.TrimSpace(targetIP)
	if m.cfg.ConnectIP != "" {
		cleanTarget = strings.TrimSpace(m.cfg.ConnectIP)
	}
	resolvedIP, err := resolveIPv4(cleanTarget)
	if err != nil || resolvedIP == "" {
		if cleanTarget != "" {
			resolvedIP = cleanTarget
		} else {
			resolvedIP = "188.114.98.0" // Cloudflare default fallback
		}
	}

	port := targetPort
	if m.cfg.ConnectPort > 0 {
		port = m.cfg.ConnectPort
	}
	if port <= 0 {
		port = DefaultConnectPort
	}

	sni := strings.TrimSpace(fakeSNI)
	if m.cfg.FakeSNI != "" {
		sni = strings.TrimSpace(m.cfg.FakeSNI)
	}
	if sni == "" {
		sni = DefaultFakeSNI
	}

	// If already running with identical parameters, keep existing instance
	if m.status.Running && isProcessAlive(m.cmd) && m.activeTarget == resolvedIP && m.activePort == port {
		m.status.ActiveNodeID = nodeID
		m.status.ActiveNodeName = nodeName
		return nil
	}

	// Stop any existing instance
	m.stopLocked()

	// Locate engine and dependencies
	folder, exePath, isRust, isPython, err := m.locateEngine()
	if err != nil {
		m.status.Error = err.Error()
		return fmt.Errorf("locating SNI Spoofing engine: %w", err)
	}

	// Ensure WinDivert driver is ready
	m.ensureDriverInstalled(folder)

	// Write dual-format config.json
	configPath := filepath.Join(folder, "config.json")
	if err := m.writeEngineConfig(configPath, resolvedIP, port, sni); err != nil {
		m.status.Error = err.Error()
		return fmt.Errorf("writing engine config: %w", err)
	}

	// Prepare execution command
	var cmd *exec.Cmd
	var engineName string
	if isRust {
		engineName = "Rust"
		cmd = exec.Command(exePath, configPath)
	} else if isPython {
		engineName = "Python"
		pyExe := getPythonExecutable()
		cmd = exec.Command(pyExe, "-X", "utf8", "main.py")
	} else {
		engineName = "Binary"
		cmd = exec.Command(exePath)
	}

	cmd.Dir = folder
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	// Pipe logs through Conective logger
	logWriter := logger.Writer("snispoof", "info")
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	logger.Info("snispoof", "Starting %s SNI Spoofing engine: %s:%d -> %s:%d (Fake SNI: %s)",
		engineName, m.cfg.ListenHost, m.cfg.ListenPort, resolvedIP, port, sni)

	if err := cmd.Start(); err != nil {
		m.status.Error = err.Error()
		return fmt.Errorf("starting %s engine: %w", engineName, err)
	}

	// Grace period: check if process died immediately
	time.Sleep(350 * time.Millisecond)
	if !isProcessAlive(cmd) {
		_ = cmd.Process.Kill()
		m.cleanupDriver()
		errStr := fmt.Sprintf("SNI Spoofing %s engine terminated immediately after launch", engineName)
		m.status.Error = errStr
		return fmt.Errorf("%s", errStr)
	}

	now := time.Now()
	m.cmd = cmd
	m.activeTarget = resolvedIP
	m.activePort = port
	m.activeEngine = engineName
	m.status = Status{
		Enabled:        true,
		Running:        true,
		ActiveEngine:   engineName,
		ListenHost:     m.cfg.ListenHost,
		ListenPort:     m.cfg.ListenPort,
		TargetIP:       resolvedIP,
		TargetPort:     port,
		FakeSNI:        sni,
		ActiveNodeID:   nodeID,
		ActiveNodeName: nodeName,
		StartedAt:      &now,
		Error:          "",
	}

	logger.Info("snispoof", "SNI Spoofing active and listening on %s:%d", m.cfg.ListenHost, m.cfg.ListenPort)
	return nil
}

// Stop terminates the SNI Spoofing forwarder and unloads WinDivert
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopLocked()
}

func (m *Manager) stopLocked() error {
	m.status.Running = false
	m.status.ActiveEngine = "None"
	m.activeTarget = ""
	m.activePort = 0
	m.activeEngine = ""

	if m.cmd != nil && m.cmd.Process != nil {
		logger.Info("snispoof", "Stopping SNI Spoofing engine (PID %d)...", m.cmd.Process.Pid)
		_ = m.cmd.Process.Kill()
		_ = m.cmd.Wait()
		m.cmd = nil
	}

	killLingeringProcesses()
	m.cleanupDriver()
	return nil
}

// writeEngineConfig writes config.json compatible with both Python/Binary and Rust engines
func (m *Manager) writeEngineConfig(path, targetIP string, targetPort int, fakeSNI string) error {
	listenHost := m.cfg.ListenHost
	if listenHost == "" {
		listenHost = DefaultListenHost
	}
	listenPort := m.cfg.ListenPort
	if listenPort <= 0 {
		listenPort = DefaultListenPort
	}

	cfgMap := map[string]any{
		"LISTEN_HOST":  listenHost,
		"LISTEN_PORT":  listenPort,
		"CONNECT_IP":   targetIP,
		"CONNECT_PORT": targetPort,
		"FAKE_SNI":     fakeSNI,

		// Rust engine fields
		"graceful_shutdown_sec": 0,
		"listeners": []map[string]any{
			{
				"listen":                fmt.Sprintf("%s:%d", listenHost, listenPort),
				"connect":               fmt.Sprintf("%s:%d", targetIP, targetPort),
				"fake_sni":              fakeSNI,
				"conn_timeout_sec":      5,
				"handshake_timeout_sec": 2,
				"keepalive_time_sec":    11,
				"keepalive_interval_sec": 2,
			},
		},
	}

	data, err := json.MarshalIndent(cfgMap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// locateEngine finds the engine directory and executable/script based on preferred engine
func (m *Manager) locateEngine() (folder string, exePath string, isRust bool, isPython bool, err error) {
	searchDirs := make([]string, 0, 8)
	if m.binDir != "" {
		searchDirs = append(searchDirs, m.binDir)
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		searchDirs = append(searchDirs,
			filepath.Join(exeDir, "bin", "sni-spoofing"),
			filepath.Join(exeDir, "sni-spoofing"),
		)
	}

	searchDirs = append(searchDirs,
		filepath.Join("bin", "sni-spoofing"),
		filepath.Join("assets", "bin", "sni-spoofing"),
		`D:\conective\Conective\bin\sni-spoofing`,
	)

	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		searchDirs = append(searchDirs, filepath.Join(localApp, "Programs", "Conective", "bin", "sni-spoofing"))
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		searchDirs = append(searchDirs, filepath.Join(appData, "Conective", "bin", "sni-spoofing"))
	}

	pref := strings.ToLower(m.cfg.Engine)
	for _, dir := range searchDirs {
		if fi, statErr := os.Stat(dir); statErr != nil || !fi.IsDir() {
			continue
		}

		rustExe := filepath.Join(dir, "sni-spoof-rs.exe")
		binExe := filepath.Join(dir, "sni-spoofing.exe")
		scriptPy := filepath.Join(dir, "main.py")

		// 1. Rust engine requested or preferred
		if (pref == EngineRust || pref == EngineAuto) && fileExists(rustExe) {
			return dir, rustExe, true, false, nil
		}

		// 2. Standalone compiled binary
		if (pref == EngineBinary || pref == EngineAuto) && fileExists(binExe) {
			return dir, binExe, false, false, nil
		}

		// 3. Python script
		if fileExists(scriptPy) {
			return dir, scriptPy, false, true, nil
		}
	}

	return "", "", false, false, fmt.Errorf("no SNI Spoofing engine found in search directories (expected sni-spoofing.exe, sni-spoof-rs.exe, or main.py)")
}

func (m *Manager) ensureDriverInstalled(engineFolder string) {
	if runtime.GOOS != "windows" {
		return
	}

	candidates := []string{
		filepath.Join(engineFolder, "WinDivert64.sys"),
		filepath.Join(engineFolder, "_internal", "pydivert", "windivert_dll", "WinDivert64.sys"),
		filepath.Join(engineFolder, "WinDivert.sys"),
	}

	var sourceSys string
	for _, c := range candidates {
		if fileExists(c) {
			sourceSys = c
			break
		}
	}

	if sourceSys == "" {
		return
	}

	sysRoot := os.Getenv("SystemRoot")
	if sysRoot == "" {
		sysRoot = `C:\Windows`
	}
	driversDir := filepath.Join(sysRoot, "System32", "drivers")
	destSys := filepath.Join(driversDir, "WinDivert64.sys")

	// If drivers dir is writable, copy WinDivert64.sys there for persistent kernel loading
	driverPath := sourceSys
	if fileExists(driversDir) {
		if !fileExists(destSys) || getFileSize(sourceSys) != getFileSize(destSys) {
			if data, err := os.ReadFile(sourceSys); err == nil {
				_ = os.WriteFile(destSys, data, 0644)
			}
		}
		if fileExists(destSys) {
			driverPath = destSys
		}
	}

	scPath := filepath.Join(sysRoot, "System32", "sc.exe")
	if fileExists(scPath) {
		quoted := fmt.Sprintf(`"%s"`, driverPath)
		_ = runHidden(scPath, "create", "WinDivert", "binPath=", quoted, "type=", "kernel")
		_ = runHidden(scPath, "config", "WinDivert", "binPath=", quoted, "type=", "kernel")
		_ = runHidden(scPath, "start", "WinDivert")
	}
}

func (m *Manager) cleanupDriver() {
	if runtime.GOOS != "windows" {
		return
	}
	sysRoot := os.Getenv("SystemRoot")
	if sysRoot == "" {
		sysRoot = `C:\Windows`
	}
	scPath := filepath.Join(sysRoot, "System32", "sc.exe")
	if fileExists(scPath) {
		_ = runHidden(scPath, "stop", "WinDivert")
	}
}

func killLingeringProcesses() {
	if runtime.GOOS != "windows" {
		return
	}
	_ = runHidden("taskkill", "/F", "/IM", "sni-spoofing.exe")
	_ = runHidden("taskkill", "/F", "/IM", "sni-spoof-rs.exe")
}

func runHidden(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd.Run()
}

func isProcessAlive(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return false
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(cmd.Process.Pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}
	return exitCode == 259 // STILL_ACTIVE
}

func resolveIPv4(addr string) (string, error) {
	addr = strings.TrimSpace(addr)
	if parsed := net.ParseIP(addr); parsed != nil {
		if ipv4 := parsed.To4(); ipv4 != nil {
			return ipv4.String(), nil
		}
		return addr, nil
	}

	ips, err := net.LookupIP(addr)
	if err != nil {
		return "", err
	}
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String(), nil
		}
	}
	return "", fmt.Errorf("no IPv4 address found for %s", addr)
}

func getPythonExecutable() string {
	candidates := []string{}
	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		candidates = append(candidates,
			filepath.Join(localApp, "Programs", "Python", "Python312", "python.exe"),
			filepath.Join(localApp, "Programs", "Python", "Python311", "python.exe"),
			filepath.Join(localApp, "Programs", "Python", "Python310", "python.exe"),
		)
	}
	for _, c := range candidates {
		if fileExists(c) {
			return c
		}
	}
	return "python"
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

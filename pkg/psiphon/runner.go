package psiphon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"freenode/pkg/cleanip"
	"freenode/pkg/logger"
)

// Runner manages the lifecycle of the Psiphon ShiroKhorshid circumvention engine
type Runner struct {
	mu             sync.Mutex
	cmd            *exec.Cmd
	running        bool
	cleanIP        string
	egressRegion   string
	socksPort      int
	httpPort       int
	connectedAt    time.Time
	lastLatency    int
	exitIP         string
	activeTunnels  int
	candidateCount int
	tunnelReadyCh  chan struct{}
}

// NewRunner creates a new Psiphon runner instance
func NewRunner() *Runner {
	return &Runner{}
}

// FindBinaries locates aether.exe and psiphon-tunnel-core.exe
func FindBinaries() (aetherPath, psiphonPath string, err error) {
	searchDirs := make([]string, 0, 10)

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		searchDirs = append(searchDirs,
			filepath.Join(exeDir, "bin"),
			filepath.Join(exeDir, "assets", "bin"),
			exeDir,
		)
	}

	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		searchDirs = append(searchDirs,
			filepath.Join(localApp, "Programs", "Conective", "bin"),
			filepath.Join(localApp, "Programs", "Conective"),
		)
	}

	if appData := os.Getenv("APPDATA"); appData != "" {
		searchDirs = append(searchDirs,
			filepath.Join(appData, "Conective", "bin"),
			filepath.Join(appData, "Conective"),
		)
	}

	// Development and workspace fallbacks
	searchDirs = append(searchDirs,
		filepath.Join("assets", "bin"),
		"bin",
		".",
	)

	for _, dir := range searchDirs {
		aPath := filepath.Join(dir, "aether.exe")
		if _, err := os.Stat(aPath); err == nil && aetherPath == "" {
			aetherPath = aPath
		}
		pPath := filepath.Join(dir, "psiphon-tunnel-core.exe")
		if _, err := os.Stat(pPath); err == nil && psiphonPath == "" {
			psiphonPath = pPath
		}
	}

	if aetherPath == "" && psiphonPath == "" {
		return "", "", fmt.Errorf("neither psiphon-tunnel-core.exe nor aether.exe found in search paths")
	}

	return aetherPath, psiphonPath, nil
}

func detectIPDefaults(ip string) (sni, cdnSet string) {
	_, detSNI, detSet := cleanip.DetectCDN(ip)
	if detSNI != "" {
		return detSNI, detSet
	}
	return "a248.e.akamai.net", "psiphon-akamai"
}

// ShiroCdnFrontingConfigWithRegion generates fronting configuration matching ShirOKhorshid with optional egress region
func ShiroCdnFrontingConfigWithRegion(cleanIP, sni, egressRegion string) ([]byte, error) {
	if sni == "" {
		sni = "a248.e.akamai.net"
	}

	cfg := map[string]any{
		"LimitTunnelProtocols": []string{
			"FRONTED-MEEK-CDN-OSSH",
			"FRONTED-MEEK-CDN-HTTP-OSSH",
			"FRONTED-MEEK-CDN-QUIC-OSSH",
		},
		"DisableTactics":                      true,
		"FrontedMeekDialOverridesProbability": 1.0,
		"FrontedMeekCDNScanUseBuiltInSpec":    true,
		"FrontedMeekDialOverrides": []map[string]any{
			{
				"OverrideID":                     "fastly-provider",
				"MatchFrontingProviderIDRegexes": []string{"(?i)fastly"},
				"DialAddresses":                  []string{"pypi.org"},
				"SNIServerName":                  "pypi.org",
				"VerifyServerNames": []string{
					"www.python.org", "pypi.org", "fastly.com", "www.fastly.com",
					"developer.fastly.com", "githubassets.com", "github.com",
					"github.io", "githubusercontent.com",
				},
				"ALPNProtocols": []string{"h2", "http/1.1"},
				"TLSProfile":    "Chrome-83",
			},
			{
				"OverrideID":              "fastly-address",
				"MatchDialAddressRegexes": []string{"(?i)(fastly|pypi|python|github)"},
				"DialAddresses":           []string{"pypi.org"},
				"SNIServerName":           "pypi.org",
				"VerifyServerNames": []string{
					"www.python.org", "pypi.org", "fastly.com", "www.fastly.com",
					"developer.fastly.com", "githubassets.com", "github.com",
					"github.io", "githubusercontent.com",
				},
				"ALPNProtocols": []string{"h2", "http/1.1"},
				"TLSProfile":    "Chrome-83",
			},
			{
				"OverrideID":              "edge-custom",
				"MatchDialAddressRegexes": []string{".*"},
				"DialAddresses":           []string{cleanIP},
				"SNIServerName":           sni,
				"VerifyServerNames": []string{
					sni,
					cleanIP,
					"a248.e.akamai.net",
					"a.akamaized.net",
					"a.akamaized-staging.net",
					"a.akamaihd.net",
					"a.akamaihd-staging.net",
					"www.akamai.com",
				},
				"ALPNProtocols": []string{"http/1.1"},
				"TLSProfile":    "Chrome-83",
			},
		},
		"FrontedMeekCDNScanSpec": map[string]any{
			"IPCandidates":   []string{cleanIP},
			"SNIServerNames": []string{sni},
		},
	}

	reg := strings.ToUpper(strings.TrimSpace(egressRegion))
	if reg != "" && reg != "AUTO" && reg != "ANY" {
		cfg["EgressRegion"] = reg
	}

	return json.MarshalIndent(cfg, "", "  ")
}

// ShiroCdnFrontingConfig generates the exact configuration matching ShirOKhorshid Android client
func ShiroCdnFrontingConfig(cleanIP, sni string) ([]byte, error) {
	return ShiroCdnFrontingConfigWithRegion(cleanIP, sni, "")
}

// Start launches Psiphon in default mode with the selected Clean CDN IP
func (r *Runner) Start(cleanIP string, socksPort, httpPort int) error {
	return r.StartWithRegion(cleanIP, "", "", "", socksPort, httpPort)
}

func ensurePortsReleased(socksPort, httpPort int) {
	if socksPort <= 0 || httpPort <= 0 {
		return
	}
	deadline := time.Now().Add(2500 * time.Millisecond)
	for time.Now().Before(deadline) {
		sConn, sErr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", socksPort), 60*time.Millisecond)
		hConn, hErr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", httpPort), 60*time.Millisecond)
		if sConn != nil {
			_ = sConn.Close()
		}
		if hConn != nil {
			_ = hConn.Close()
		}
		if sErr != nil && hErr != nil {
			// Both ports are cleanly unhooked and free
			return
		}
		// Still held: force kill engine processes again and wait briefly
		_ = silentCmd("taskkill", "/F", "/IM", "aether.exe").Run()
		_ = silentCmd("taskkill", "/F", "/IM", "psiphon-tunnel-core.exe").Run()
		time.Sleep(120 * time.Millisecond)
	}
}

func getSafePsiphonDir(egressRegion string) string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = "."
	}
	baseDir := filepath.Join(appData, "Conective", "psiphon")
	_ = os.MkdirAll(baseDir, 0755)

	targetDir := filepath.Join(baseDir, "data")
	_ = os.MkdirAll(targetDir, 0755)

	coreDir := filepath.Join(targetDir, "ca.psiphon.PsiphonTunnel.tunnel-core")
	_ = os.MkdirAll(coreDir, 0755)
	dataStoreDir := filepath.Join(coreDir, "datastore")
	_ = os.MkdirAll(dataStoreDir, 0755)

	// Clean up stale locks and temporary files
	_ = os.Remove(filepath.Join(targetDir, "psiphon.boltdb.lock"))
	_ = os.Remove(filepath.Join(coreDir, "psiphon.boltdb.lock"))
	_ = os.Remove(filepath.Join(dataStoreDir, "psiphon.boltdb.lock"))
	_ = os.Remove(filepath.Join(coreDir, "remote_server_list.part"))

	return targetDir
}

// StartWithSNI launches Psiphon mode with the selected Clean CDN IP, custom SNI and CDN set
func (r *Runner) StartWithSNI(cleanIP, sni, cdnSet string, socksPort, httpPort int) error {
	return r.StartWithRegion(cleanIP, sni, cdnSet, "", socksPort, httpPort)
}

// BuildArgs constructs the command line arguments for launching Aether in Psiphon mode.
// Preserved for backward-compatibility and tests.
func BuildArgs(psiphonBin, psiphonDir, cleanIP, sni, cdnSet, egressRegion string, socksPort, httpPort int) []string {
	if socksPort <= 0 {
		socksPort = 10808
	}
	if httpPort <= 0 {
		httpPort = 10809
	}

	cleanIP = strings.TrimSpace(cleanIP)
	egressRegion = strings.ToUpper(strings.TrimSpace(egressRegion))
	if egressRegion == "AUTO" || egressRegion == "ANY" || egressRegion == "NONE" || egressRegion == "DEFAULT" {
		egressRegion = ""
	}

	if psiphonDir == "" {
		psiphonDir = getSafePsiphonDir(egressRegion)
	}

	args := []string{
		"--psiphon-only",
		"--psiphon-bin", psiphonBin,
		"--bind", fmt.Sprintf("127.0.0.1:%d", socksPort),
		"--psiphon-http", fmt.Sprintf("127.0.0.1:%d", httpPort),
		"--psiphon-dir", psiphonDir,
		"--log-level", "info",
	}

	if egressRegion != "" {
		args = append(args, "--psiphon-region", egressRegion)
	}

	isDirect := cleanIP == "" || cleanIP == "auto" || cleanIP == "direct"

	if !isDirect {
		if sni == "" || cdnSet == "" {
			detSNI, detSet := detectIPDefaults(cleanIP)
			if sni == "" {
				sni = detSNI
			}
			if cdnSet == "" {
				cdnSet = detSet
			}
		}

		args = append(args, "--psiphon-cdn-ips", cleanIP)
		if sni != "" {
			args = append(args, "--psiphon-cdn-sni", sni)
		}
		if cdnSet != "" {
			args = append(args, "--psiphon-cdn-sets", cdnSet)
		}
	}

	return args
}

// StartWithRegion launches the ShiroKhorshid Psiphon engine with the selected Clean CDN IP,
// custom SNI, CDN set and egress region.
// It directly invokes psiphon-tunnel-core with full ShiroKhorshid CDN fronting specs and
// real-time notice streaming, falling back to aether if necessary.
func (r *Runner) StartWithRegion(cleanIP, sni, cdnSet, egressRegion string, socksPort, httpPort int) error {
	r.mu.Lock()
	r.stopLocked()

	aetherBin, psiphonBin, err := FindBinaries()
	if err != nil {
		r.mu.Unlock()
		return fmt.Errorf("locating psiphon binaries: %w", err)
	}

	if socksPort <= 0 {
		socksPort = 10808
	}
	if httpPort <= 0 {
		httpPort = socksPort + 1
	}

	r.socksPort = socksPort
	r.httpPort = httpPort
	r.cleanIP = strings.TrimSpace(cleanIP)
	r.egressRegion = strings.ToUpper(strings.TrimSpace(egressRegion))
	if r.egressRegion == "AUTO" || r.egressRegion == "ANY" || r.egressRegion == "NONE" || r.egressRegion == "DEFAULT" {
		r.egressRegion = ""
	}

	psiphonDir := getSafePsiphonDir(r.egressRegion)
	isDirect := r.cleanIP == "" || r.cleanIP == "auto" || r.cleanIP == "direct"

	if !isDirect && sni == "" {
		detSNI, _ := detectIPDefaults(r.cleanIP)
		sni = detSNI
	}

	// 1. Ensure server_list.dat is seeded in target data directory
	serverListPath, errSList := EnsurePsiphonServerList(psiphonDir)
	if errSList != nil {
		logger.Warn("psiphon", "Warning seeding server list: %v", errSList)
	}

	// 2. Generate ShiroKhorshid Psiphon JSON configuration
	edges := ParseCdnEdges(r.cleanIP)
	if isDirect {
		edges = nil
	}

	shiroCfgBytes, errCfg := BuildShiroKhorshidConfig(ShiroConfigOptions{
		SocksPort:      socksPort,
		HTTPPort:       httpPort,
		DataDir:        psiphonDir,
		EgressRegion:   r.egressRegion,
		TunnelPoolSize: 1,
		CdnFronting:    !isDirect && len(edges) > 0,
		CdnEdges:       edges,
		SNI:            sni,
	})
	if errCfg != nil {
		r.mu.Unlock()
		return fmt.Errorf("generating ShiroKhorshid config: %w", errCfg)
	}

	configFilePath := filepath.Join(psiphonDir, "config.json")
	if err := os.WriteFile(configFilePath, shiroCfgBytes, 0644); err != nil {
		r.mu.Unlock()
		return fmt.Errorf("writing psiphon config file: %w", err)
	}

	ensurePortsReleased(socksPort, httpPort)

	r.tunnelReadyCh = make(chan struct{})
	var cmd *exec.Cmd

	// Prefer direct native psiphon-tunnel-core with full ShiroKhorshid notice streaming
	if psiphonBin != "" {
		args := []string{
			"-config", configFilePath,
			"-dataRootDirectory", psiphonDir,
			"-formatNotices",
		}
		if serverListPath != "" {
			args = append(args, "-serverList", serverListPath)
		}

		cmd = exec.Command(psiphonBin, args...)
		cmd.Dir = psiphonDir
		logger.Info("psiphon", "Launching native ShiroKhorshid Psiphon engine: %s (socks=%d, http=%d)", psiphonBin, socksPort, httpPort)
	} else {
		// Fallback to aether wrapping psiphon
		args := BuildArgs(psiphonBin, psiphonDir, r.cleanIP, sni, cdnSet, r.egressRegion, socksPort, httpPort)
		args = append(args, "--psiphon-config", configFilePath)
		if serverListPath != "" {
			args = append(args, "--psiphon-server-entries", serverListPath)
		}
		cmd = exec.Command(aetherBin, args...)
		logger.Info("psiphon", "Launching Aether-wrapped Psiphon engine: %s", aetherBin)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	stdoutPipe, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		r.mu.Unlock()
		return fmt.Errorf("creating stdout pipe: %w", pipeErr)
	}
	cmd.Stderr = logger.Writer("psiphon", "warn")

	regLog := r.egressRegion
	if regLog == "" {
		regLog = "auto (any available)"
	}
	if isDirect {
		logger.Info("psiphon", "Starting direct Psiphon engine: region=%s, socks=%d, http=%d", regLog, socksPort, httpPort)
	} else {
		logger.Info("psiphon", "Starting ShiroKhorshid CDN engine: cleanIP=%s, sni=%s, region=%s, socks=%d, http=%d", r.cleanIP, sni, regLog, socksPort, httpPort)
	}

	if err := cmd.Start(); err != nil {
		r.mu.Unlock()
		return fmt.Errorf("starting psiphon process: %w", err)
	}

	r.cmd = cmd
	r.running = true
	r.connectedAt = time.Now()

	// Stream notices and monitor tunnel state in background
	go r.streamNotices(stdoutPipe, cmd)

	r.mu.Unlock()

	// Step 1: Wait up to 15 seconds for local ports to bind
	logger.Info("psiphon", "Waiting for local SOCKS5 proxy 127.0.0.1:%d to bind...", socksPort)
	bindDeadline := time.Now().Add(15 * time.Second)
	portsBound := false
	for time.Now().Before(bindDeadline) {
		if !r.IsRunning() {
			_ = r.Stop()
			return fmt.Errorf("psiphon process terminated before ports bound")
		}
		conn, dialErr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", socksPort), 300*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			portsBound = true
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	if !portsBound {
		logger.Error("psiphon", "Psiphon tunnel failed to bind ports %d/%d within 15s", socksPort, httpPort)
		_ = r.Stop()
		return fmt.Errorf("psiphon tunnel failed to bind ports %d/%d within 15s", socksPort, httpPort)
	}

	logger.Info("psiphon", "Local proxy ports bound (SOCKS5:%d, HTTP:%d). Waiting for tunnel handshake...", socksPort, httpPort)

	// Step 2: Wait for tunnel establishment (signaled by Notice "Tunnels" count > 0 or verified via probe)
	tunnelReady := false
	tunnelDeadline := time.Now().Add(45 * time.Second)

	tunnelClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("127.0.0.1:%d", httpPort),
			}),
			DisableKeepAlives: true,
		},
		Timeout: 2500 * time.Millisecond,
	}

	for time.Now().Before(tunnelDeadline) {
		if !r.IsRunning() {
			_ = r.Stop()
			return fmt.Errorf("psiphon process terminated unexpectedly during tunnel establishment")
		}

		// Check if notice streamer already detected active tunnels
		r.mu.Lock()
		tCount := r.activeTunnels
		r.mu.Unlock()

		if tCount > 0 {
			tunnelReady = true
			// Non-blocking background probe to log initial connectivity
			go func() {
				resp, err := tunnelClient.Get("http://cp.cloudflare.com/generate_204")
				if err == nil {
					_ = resp.Body.Close()
				}
			}()
			break
		}

		// Periodic probe attempt before Tunnels notice
		resp, err := tunnelClient.Get("http://cp.cloudflare.com/generate_204")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 204 || resp.StatusCode == 200 {
				tunnelReady = true
				break
			}
		}

		time.Sleep(600 * time.Millisecond)
	}

	if !tunnelReady {
		_ = r.Stop()
		if isDirect {
			return fmt.Errorf("direct Psiphon tunnel failed to establish active tunnel within 45s")
		}
		return fmt.Errorf("psiphon tunnel failed to establish connection with CDN node %s within 45s", r.cleanIP)
	}

	logger.Info("psiphon", "ShiroKhorshid tunnel successfully active on SOCKS5 127.0.0.1:%d and HTTP 127.0.0.1:%d", socksPort, httpPort)
	log.Printf("[Psiphon] ShiroKhorshid tunnel active on SOCKS5 127.0.0.1:%d and HTTP 127.0.0.1:%d", socksPort, httpPort)
	return nil
}

// streamNotices reads and logs Psiphon notices from core output
func (r *Runner) streamNotices(reader io.Reader, cmd *exec.Cmd) {
	scanner := bufio.NewScanner(reader)
	// Support long notice lines (e.g. CandidateServers or diagnostics)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		notice := ParseNotice(line)
		if notice != nil {
			if logLine := FormatNoticeLog(notice); logLine != "" {
				logger.Info("psiphon", "%s", logLine)
			}

			r.mu.Lock()
			switch notice.NoticeType {
			case "CandidateServers":
				if c, ok := notice.Data["count"].(float64); ok {
					r.candidateCount = int(c)
				}
			case "Tunnels":
				if c, ok := notice.Data["count"].(float64); ok {
					r.activeTunnels = int(c)
					if r.activeTunnels > 0 && r.tunnelReadyCh != nil {
						select {
						case <-r.tunnelReadyCh:
						default:
							close(r.tunnelReadyCh)
						}
					}
				}
			}
			r.mu.Unlock()
		} else if strings.TrimSpace(line) != "" {
			logger.Info("psiphon", "%s", line)
		}
	}

	_ = cmd.Wait()
	r.mu.Lock()
	if r.cmd == cmd {
		r.running = false
		r.cmd = nil
	}
	r.mu.Unlock()
}

// Stop terminates the Psiphon/Aether process and kills any lingering instances
func (r *Runner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopLocked()
	return nil
}

func silentCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd
}

func (r *Runner) stopLocked() {
	wasRunning := r.running || (r.cmd != nil && r.cmd.Process != nil)
	socksPort := r.socksPort
	httpPort := r.httpPort

	if r.tunnelReadyCh != nil {
		select {
		case <-r.tunnelReadyCh:
		default:
			close(r.tunnelReadyCh)
		}
		r.tunnelReadyCh = nil
	}

	if r.cmd != nil && r.cmd.Process != nil {
		pid := r.cmd.Process.Pid
		log.Printf("[Psiphon] Stopping engine (PID %d)...", pid)

		_ = silentCmd("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run()
		_ = r.cmd.Process.Kill()
		r.cmd = nil
	}

	if wasRunning {
		_ = silentCmd("taskkill", "/F", "/IM", "aether.exe").Run()
		_ = silentCmd("taskkill", "/F", "/IM", "psiphon-tunnel-core.exe").Run()

		if socksPort > 0 && httpPort > 0 {
			ensurePortsReleased(socksPort, httpPort)
		}
	}

	appData := os.Getenv("APPDATA")
	if appData != "" {
		psiphonBase := filepath.Join(appData, "Conective", "psiphon")
		_ = filepath.Walk(psiphonBase, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && (strings.HasSuffix(info.Name(), ".lock") || strings.HasSuffix(info.Name(), ".part")) {
				_ = os.Remove(path)
			}
			return nil
		})
	}

	r.running = false
	r.cleanIP = ""
	r.egressRegion = ""
	r.lastLatency = 0
	r.exitIP = ""
	r.activeTunnels = 0
	r.candidateCount = 0

	runtime.GC()
	debug.FreeOSMemory()
	logger.Info("psiphon", "Engine stopped cleanly")
}

// IsRunning reports whether Psiphon is currently executing
func (r *Runner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// GetEgressRegion returns the active egress country/region code
func (r *Runner) GetEgressRegion() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.egressRegion
}

// GetActiveTunnels returns the number of active tunnels reported by core notices
func (r *Runner) GetActiveTunnels() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.activeTunnels
}

// GetCandidateCount returns the number of candidate servers loaded by core
func (r *Runner) GetCandidateCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.candidateCount
}

// VerifyTunnel performs a real outbound HTTP probe through the local SOCKS/HTTP proxy
func (r *Runner) VerifyTunnel(wait time.Duration) (int, string, error) {
	deadline := time.Now().Add(wait)
	var lastErr error

	for time.Now().Before(deadline) {
		lat, exitIP, err := r.probeOnce()
		if err == nil {
			r.mu.Lock()
			if lat > 0 {
				r.lastLatency = lat
			}
			if exitIP != "" {
				r.exitIP = exitIP
			}
			r.mu.Unlock()
			return lat, exitIP, nil
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("verification timed out")
	}
	return -1, "", lastErr
}

func (r *Runner) probeOnce() (int, string, error) {
	r.mu.Lock()
	sp := r.socksPort
	hp := r.httpPort
	r.mu.Unlock()

	if sp <= 0 && hp <= 0 {
		return -1, "", fmt.Errorf("no active proxy ports")
	}

	var proxyURL *url.URL
	if hp > 0 {
		proxyURL, _ = url.Parse(fmt.Sprintf("http://127.0.0.1:%d", hp))
	} else {
		proxyURL, _ = url.Parse(fmt.Sprintf("socks5://127.0.0.1:%d", sp))
	}

	tr := &http.Transport{
		Proxy:             http.ProxyURL(proxyURL),
		DisableKeepAlives: true,
	}
	defer tr.CloseIdleConnections()

	client := &http.Client{
		Transport: tr,
		Timeout:   5 * time.Second,
	}
	defer client.CloseIdleConnections()

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "https://www.cloudflare.com/cdn-cgi/trace", nil)
	resp, err := client.Do(req)
	if err != nil {
		req2, _ := http.NewRequestWithContext(ctx, "GET", "http://cp.cloudflare.com/generate_204", nil)
		resp2, err2 := client.Do(req2)
		if err2 != nil {
			return -1, "", err
		}
		defer resp2.Body.Close()
		latency := int(time.Since(start).Milliseconds())
		return latency, "", nil
	}
	defer resp.Body.Close()

	latency := int(time.Since(start).Milliseconds())

	buf := make([]byte, 2048)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])
	exitIP := ""
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "ip=") {
			exitIP = strings.TrimPrefix(line, "ip=")
			break
		}
	}

	return latency, exitIP, nil
}

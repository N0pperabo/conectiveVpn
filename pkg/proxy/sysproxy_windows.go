//go:build windows

package proxy

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

var (
	modWinInet            = syscall.NewLazyDLL("wininet.dll")
	procInternetSetOption = modWinInet.NewProc("InternetSetOptionW")
)

const (
	internetOptionSettingsChanged = 39
	internetOptionRefresh         = 37
	internetSettingsKeyPath       = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`
)

type Manager struct {
	mu           sync.Mutex
	backupEnable uint32
	backupServer string
	hasBackup    bool
}

func NewManager() *Manager {
	return &Manager{}
}

// BackupCurrentState records the existing system proxy settings
func (m *Manager) BackupCurrentState() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	enable, _, err := k.GetIntegerValue("ProxyEnable")
	if err == nil {
		m.backupEnable = uint32(enable)
	}

	server, _, err := k.GetStringValue("ProxyServer")
	if err == nil {
		m.backupServer = server
	}

	m.hasBackup = true
	return nil
}

// BuildProxyOverride formats the Windows WinINet ProxyOverride string with default private ranges
// and user-configured bypass domains/IPs.
func BuildProxyOverride(bypassDomains []string) string {
	baseEntries := []string{
		"<local>",
		"localhost",
		"127.*",
		"10.*",
		"172.16.*",
		"172.17.*",
		"172.18.*",
		"172.19.*",
		"172.20.*",
		"172.21.*",
		"172.22.*",
		"172.23.*",
		"172.24.*",
		"172.25.*",
		"172.26.*",
		"172.27.*",
		"172.28.*",
		"172.29.*",
		"172.30.*",
		"172.31.*",
		"192.168.*",
	}

	seen := make(map[string]bool)
	var result []string
	for _, b := range baseEntries {
		key := strings.ToLower(b)
		if !seen[key] {
			seen[key] = true
			result = append(result, b)
		}
	}

	for _, raw := range bypassDomains {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, ".ir$") || lower == "domain:ir" || lower == "*.ir" || lower == "ir" || lower == ".ir" {
			if !seen["*.ir"] {
				seen["*.ir"] = true
				result = append(result, "*.ir")
			}
			continue
		}

		// Strip protocol
		if idx := strings.Index(line, "://"); idx != -1 {
			line = line[idx+3:]
		}
		// Strip path and query parameters
		if idx := strings.IndexAny(line, "/?#"); idx != -1 {
			line = line[:idx]
		}
		// Strip port if host:port
		if !strings.Contains(line, "]") && strings.Count(line, ":") == 1 {
			if h, _, err := net.SplitHostPort(line); err == nil {
				line = h
			}
		}
		// Strip rule prefixes
		line = strings.TrimPrefix(line, "domain:")
		line = strings.TrimPrefix(line, "full:")
		line = strings.TrimPrefix(line, "keyword:")
		line = strings.TrimPrefix(line, "*.")
		line = strings.TrimPrefix(line, ".")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// In WinINet ProxyOverride:
		// *example.com matches example.com, www.example.com, and any subdomain
		// *.example.com matches subdomains explicitly
		// example.com matches the exact host
		for _, p := range []string{"*" + line, "*." + line, line} {
			pLower := strings.ToLower(p)
			if !seen[pLower] {
				seen[pLower] = true
				result = append(result, p)
			}
		}
	}

	return strings.Join(result, ";")
}

// Enable sets the Windows system proxy to 127.0.0.1:port with standard local bypass
func (m *Manager) Enable(httpPort int) error {
	return m.EnableWithBypass(httpPort, nil)
}

// EnableWithBypass sets the Windows system proxy to 127.0.0.1:port and applies user bypass domains
func (m *Manager) EnableWithBypass(httpPort int, bypassDomains []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Backup if not backed up yet
	if !m.hasBackup {
		_ = m.backupCurrentStateLocked()
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening registry key for write: %w", err)
	}
	defer k.Close()

	proxyAddress := fmt.Sprintf("127.0.0.1:%d", httpPort)
	if err := k.SetStringValue("ProxyServer", proxyAddress); err != nil {
		return fmt.Errorf("setting ProxyServer: %w", err)
	}

	if err := k.SetDWordValue("ProxyEnable", 1); err != nil {
		return fmt.Errorf("setting ProxyEnable: %w", err)
	}

	overrideStr := BuildProxyOverride(bypassDomains)
	if err := k.SetStringValue("ProxyOverride", overrideStr); err != nil {
		return fmt.Errorf("setting ProxyOverride: %w", err)
	}

	m.notifyWinINet()
	return nil
}

// Disable restores proxy to disabled or the original backup state
func (m *Manager) Disable() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if m.hasBackup && m.backupEnable == 1 {
		_ = k.SetStringValue("ProxyServer", m.backupServer)
		_ = k.SetDWordValue("ProxyEnable", 1)
	} else {
		_ = k.SetDWordValue("ProxyEnable", 0)
	}

	m.notifyWinINet()
	return nil
}

// RecoverOrphanedProxy checks if proxy was left pointing to local freenode and resets it on crash recovery
func (m *Manager) RecoverOrphanedProxy(httpPort int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()

	enable, _, err := k.GetIntegerValue("ProxyEnable")
	if err != nil || enable != 1 {
		return
	}

	server, _, err := k.GetStringValue("ProxyServer")
	if err != nil {
		return
	}

	// If it pointed to 127.0.0.1 from a previous crashed run, turn it off
	expected := fmt.Sprintf("127.0.0.1:%d", httpPort)
	if server == expected || server == "127.0.0.1:10809" {
		_ = k.SetDWordValue("ProxyEnable", 0)
		m.notifyWinINet()
	}
}

func (m *Manager) backupCurrentStateLocked() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	enable, _, err := k.GetIntegerValue("ProxyEnable")
	if err == nil {
		m.backupEnable = uint32(enable)
	}

	server, _, err := k.GetStringValue("ProxyServer")
	if err == nil {
		m.backupServer = server
	}

	m.hasBackup = true
	return nil
}

func (m *Manager) notifyWinINet() {
	go func() {
		defer func() {
			_ = recover()
		}()
		_, _, _ = procInternetSetOption.Call(0, uintptr(internetOptionSettingsChanged), 0, 0)
		_, _, _ = procInternetSetOption.Call(0, uintptr(internetOptionRefresh), 0, 0)
	}()
}

//go:build !windows

package proxy

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) BackupCurrentState() error {
	return nil
}

func (m *Manager) Enable(httpPort int) error {
	return nil
}

func (m *Manager) EnableWithBypass(httpPort int, bypassDomains []string) error {
	return nil
}

func (m *Manager) Disable() error {
	return nil
}

func (m *Manager) RecoverOrphanedProxy(httpPort int) {}

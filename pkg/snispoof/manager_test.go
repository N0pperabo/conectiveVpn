package snispoof

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigNormalize(t *testing.T) {
	cfg := Config{}
	cfg.Normalize()

	if cfg.ListenHost != DefaultListenHost {
		t.Errorf("expected ListenHost %s, got %s", DefaultListenHost, cfg.ListenHost)
	}
	if cfg.ListenPort != DefaultListenPort {
		t.Errorf("expected ListenPort %d, got %d", DefaultListenPort, cfg.ListenPort)
	}
	if cfg.ConnectPort != DefaultConnectPort {
		t.Errorf("expected ConnectPort %d, got %d", DefaultConnectPort, cfg.ConnectPort)
	}
	if cfg.FakeSNI != DefaultFakeSNI {
		t.Errorf("expected FakeSNI %s, got %s", DefaultFakeSNI, cfg.FakeSNI)
	}
	if cfg.Engine != EngineAuto {
		t.Errorf("expected Engine %s, got %s", EngineAuto, cfg.Engine)
	}
}

func TestManagerLifecycleGetters(t *testing.T) {
	mgr := NewManager("")

	if mgr.IsRunning() {
		t.Error("new manager should not be running")
	}

	cfg := Config{
		Enabled:    true,
		Engine:     "Rust",
		ListenHost: "127.0.0.1",
		ListenPort: 40443,
		FakeSNI:    "auth.vercel.com",
	}
	mgr.SetConfig(cfg)

	if !mgr.IsEnabled() {
		t.Error("expected IsEnabled to be true")
	}

	host, port := mgr.GetListenAddress()
	if host != "127.0.0.1" || port != 40443 {
		t.Errorf("unexpected listen address: %s:%d", host, port)
	}

	st := mgr.GetStatus()
	if !st.Enabled {
		t.Error("expected status.Enabled to be true")
	}
	if st.Running {
		t.Error("expected status.Running to be false before start")
	}
}

func TestWriteEngineConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	mgr := NewManager(tmpDir)
	err := mgr.writeEngineConfig(cfgPath, "188.114.98.0", 443, "auth.vercel.com")
	if err != nil {
		t.Fatalf("writeEngineConfig failed: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed reading config.json: %v", err)
	}

	content := string(data)
	if !contains(content, "LISTEN_HOST") || !contains(content, "listeners") || !contains(content, "auth.vercel.com") {
		t.Errorf("config.json missing expected fields: %s", content)
	}
}

func contains(s, substr string) bool {
	return filepath.Clean(s) != "" && len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

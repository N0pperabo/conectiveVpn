package mhr

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"freenode/pkg/database"
)

func TestManagerConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mhr_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open failed: %v", err)
	}
	defer db.Close()

	mgr := NewManager(db, nil, tmpDir)

	cfg := mgr.GetConfig()
	if cfg.Variant != VariantMHR {
		t.Errorf("expected default variant %s, got %s", VariantMHR, cfg.Variant)
	}
	if cfg.HTTPPort != DefaultHTTPPort {
		t.Errorf("expected default http port %d, got %d", DefaultHTTPPort, cfg.HTTPPort)
	}
	if cfg.Socks5Port != DefaultSocks5Port {
		t.Errorf("expected default socks5 port %d, got %d", DefaultSocks5Port, cfg.Socks5Port)
	}

	// Update configuration
	cfg.Variant = VariantMHRCFW
	cfg.ScriptID = "AKfycb_test_script_id"
	cfg.AuthKey = "super_secret_auth_key_123"
	cfg.HTTPPort = 8090
	cfg.Socks5Port = 1090
	cfg.GoogleIP = "216.239.38.120"
	cfg.FrontDomain = "www.google.com"
	cfg.SystemProxy = true
	cfg.LANSharing = true

	if err := mgr.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded := mgr.LoadConfig()
	if loaded.Variant != VariantMHRCFW {
		t.Errorf("expected variant %s, got %s", VariantMHRCFW, loaded.Variant)
	}
	if loaded.ScriptID != "AKfycb_test_script_id" {
		t.Errorf("expected ScriptID 'AKfycb_test_script_id', got '%s'", loaded.ScriptID)
	}
	if loaded.AuthKey != "super_secret_auth_key_123" {
		t.Errorf("expected AuthKey 'super_secret_auth_key_123', got '%s'", loaded.AuthKey)
	}
	if loaded.HTTPPort != 8090 {
		t.Errorf("expected HTTPPort 8090, got %d", loaded.HTTPPort)
	}
	if loaded.Socks5Port != 1090 {
		t.Errorf("expected Socks5Port 1090, got %d", loaded.Socks5Port)
	}
	if !loaded.SystemProxy {
		t.Errorf("expected SystemProxy true")
	}
	if !loaded.LANSharing {
		t.Errorf("expected LANSharing true")
	}
}

func TestPortCollisionCheck(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mhr_port_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open failed: %v", err)
	}
	defer db.Close()

	_ = db.SetSetting("http_port", "10809")
	_ = db.SetSetting("socks_port", "10808")

	mgr := NewManager(db, nil, tmpDir)

	// Collide with Xray HTTP port
	if err := mgr.checkPortCollisionLocked(10809, 8080); err == nil {
		t.Errorf("expected error when colliding with Xray HTTP port 10809")
	}

	// Collide with Xray SOCKS port
	if err := mgr.checkPortCollisionLocked(8080, 10808); err == nil {
		t.Errorf("expected error when colliding with Xray SOCKS port 10808")
	}

	// HTTP and SOCKS identical
	if err := mgr.checkPortCollisionLocked(8085, 8085); err == nil {
		t.Errorf("expected error when HTTP and SOCKS ports are identical")
	}

	// Non-colliding ports
	if err := mgr.checkPortCollisionLocked(8085, 1080); err != nil {
		t.Errorf("unexpected error for non-colliding ports: %v", err)
	}
}

func TestEnsureRuntime(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mhr_extract_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewManager(nil, nil, tmpDir)

	dir, err := mgr.EnsureRuntime(VariantMHR)
	if err != nil {
		t.Fatalf("EnsureRuntime(mhr) failed: %v", err)
	}

	mainPy := filepath.Join(dir, "main.py")
	if _, err := os.Stat(mainPy); err != nil {
		t.Errorf("main.py not found in extracted runtime: %v", err)
	}

	dirCFW, err := mgr.EnsureRuntime(VariantMHRCFW)
	if err != nil {
		t.Fatalf("EnsureRuntime(mhr-cfw) failed: %v", err)
	}
	mainPyCFW := filepath.Join(dirCFW, "main.py")
	if _, err := os.Stat(mainPyCFW); err != nil {
		t.Errorf("main.py not found in extracted CFW runtime: %v", err)
	}
}

func TestFindPythonExecutable(t *testing.T) {
	py, err := FindPythonExecutable()
	if err != nil {
		t.Logf("Python not found in test environment: %v", err)
		return
	}
	if py == "" {
		t.Errorf("expected non-empty python path")
	}
	t.Logf("Found python at: %s", py)
}

func TestGetDeploymentScripts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mhr_script_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewManager(nil, nil, tmpDir)
	_ = mgr.SaveConfig(Config{
		Variant:      VariantMHR,
		AuthKey:      "MY_TEST_KEY_456",
		CFWWorkerURL: "https://custom-worker.workers.dev",
	})

	scripts, err := mgr.GetDeploymentScripts()
	if err != nil {
		t.Fatalf("GetDeploymentScripts failed: %v", err)
	}

	if scripts.AppsScriptCode == "" {
		t.Errorf("AppsScriptCode is empty")
	}
	if scripts.WorkerCode == "" {
		t.Errorf("WorkerCode is empty")
	}
}

func TestScanGoogleIPs(t *testing.T) {
	mgr := NewManager(nil, nil, "")
	// Probe with short timeout to ensure goroutines and logic execute properly
	results, err := mgr.ScanGoogleIPs("www.google.com", 800*time.Millisecond)
	if err != nil {
		t.Fatalf("ScanGoogleIPs failed: %v", err)
	}
	if len(results) != len(CandidateGoogleIPs) {
		t.Errorf("expected %d results, got %d", len(CandidateGoogleIPs), len(results))
	}
}

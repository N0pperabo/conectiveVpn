package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"freenode/pkg/database"
	"freenode/pkg/proxy"
	"freenode/pkg/snispoof"
	"freenode/pkg/xray"
)

func TestSNISpoofAPI(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := database.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	runner := xray.NewRunner(10808, 10809, "http://cp.cloudflare.com/generate_204")
	proxyMgr := proxy.NewManager()
	server := NewServer(db, nil, runner, proxyMgr, nil)

	// 1. Test GET /api/snispoof/status
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/snispoof/status", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var statusResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("failed unmarshaling status response: %v", err)
	}

	// 2. Test POST /api/snispoof/settings
	newCfg := snispoof.Config{
		Enabled:     true,
		Engine:      "auto",
		ListenHost:  "127.0.0.1",
		ListenPort:  40443,
		ConnectIP:   "1.2.3.4",
		ConnectPort: 8443,
		FakeSNI:     "mci.ir",
	}
	body, _ := json.Marshal(newCfg)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/snispoof/settings", bytes.NewReader(body))
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on saving settings, got %d", rec.Code)
	}

	// Verify persistence in DB
	if db.GetSetting("sni_spoof_fake_sni", "") != "mci.ir" {
		t.Errorf("expected DB to store fake_sni 'mci.ir', got %s", db.GetSetting("sni_spoof_fake_sni", ""))
	}
	if db.GetSetting("sni_spoof_connect_ip", "") != "1.2.3.4" {
		t.Errorf("expected DB to store connect_ip '1.2.3.4', got %s", db.GetSetting("sni_spoof_connect_ip", ""))
	}

	// 3. Test POST /api/snispoof/toggle
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/snispoof/toggle", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on toggle, got %d", rec.Code)
	}

	var toggleResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &toggleResp)
	if toggleResp["enabled"] != false {
		t.Errorf("expected enabled to toggle to false, got %v", toggleResp["enabled"])
	}
}

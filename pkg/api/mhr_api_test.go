package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"freenode/pkg/database"
	"freenode/pkg/mhr"
)

func TestMHRAPIEndpoints(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mhr_api_test_*")
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

	srv := NewServer(db, nil, nil, nil, nil)
	mgr := mhr.NewManager(db, nil, tmpDir)
	srv.SetMHRManager(mgr)

	// 1. GET /api/mhr/config
	req := httptest.NewRequest(http.MethodGet, "/api/mhr/config", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/mhr/config status = %d; want 200", w.Code)
	}
	var cfg mhr.Config
	if err := json.NewDecoder(w.Body).Decode(&cfg); err != nil {
		t.Fatalf("failed decoding config: %v", err)
	}
	if cfg.Variant != mhr.VariantMHR {
		t.Errorf("expected variant %s, got %s", mhr.VariantMHR, cfg.Variant)
	}

	// 2. POST /api/mhr/config
	cfg.ScriptID = "AKfycb_sample_script_id"
	cfg.AuthKey = "sample_secret_key"
	cfg.HTTPPort = 8086
	cfg.Socks5Port = 1086
	bodyBytes, _ := json.Marshal(cfg)

	req = httptest.NewRequest(http.MethodPost, "/api/mhr/config", bytes.NewReader(bodyBytes))
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/mhr/config status = %d; want 200", w.Code)
	}

	// 3. GET /api/mhr/status
	req = httptest.NewRequest(http.MethodGet, "/api/mhr/status", nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/mhr/status status = %d; want 200", w.Code)
	}
	var st mhr.Status
	if err := json.NewDecoder(w.Body).Decode(&st); err != nil {
		t.Fatalf("failed decoding status: %v", err)
	}
	if st.Running {
		t.Errorf("expected status.Running false before start")
	}

	// 4. GET /api/mhr/scripts
	req = httptest.NewRequest(http.MethodGet, "/api/mhr/scripts", nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/mhr/scripts status = %d; want 200", w.Code)
	}
	var scripts mhr.ScriptTemplates
	if err := json.NewDecoder(w.Body).Decode(&scripts); err != nil {
		t.Fatalf("failed decoding scripts: %v", err)
	}
	if scripts.AppsScriptCode == "" {
		t.Errorf("expected non-empty AppsScriptCode")
	}

	// 5. POST /api/mhr/stop (should succeed even if not running)
	req = httptest.NewRequest(http.MethodPost, "/api/mhr/stop", nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/mhr/stop status = %d; want 200", w.Code)
	}
}

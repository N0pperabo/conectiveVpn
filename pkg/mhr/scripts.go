package mhr

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ScriptTemplates returns the deployment scripts for Google Apps Script and Cloudflare Worker
type ScriptTemplates struct {
	AppsScriptCode string `json:"apps_script_code"`
	WorkerCode     string `json:"worker_code"`
	Variant        string `json:"variant"`
}

// GetDeploymentScripts returns deployment code customized with user auth_key & worker_url
func (m *Manager) GetDeploymentScripts() (ScriptTemplates, error) {
	m.mu.RLock()
	variant := m.config.Variant
	authKey := m.config.AuthKey
	workerURL := m.config.CFWWorkerURL
	runtimeDir := m.runtimeDir
	m.mu.RUnlock()

	if authKey == "" {
		authKey = "CHANGE_ME_TO_A_STRONG_SECRET"
	}
	if workerURL == "" {
		workerURL = "https://your-worker.workers.dev"
	}

	var res ScriptTemplates
	res.Variant = variant

	// Read Apps Script
	gasCode := ""
	workerCode := ""

	if variant == VariantMHRCFW {
		gasPath := filepath.Join(runtimeDir, "deploy", "gas", "Code.gs")
		if data, err := os.ReadFile(gasPath); err == nil {
			gasCode = string(data)
		} else {
			gasCode = readEmbeddedFile("runtime/mhr-cfw/deploy/gas/Code.gs")
		}

		cwPath := filepath.Join(runtimeDir, "deploy", "cloudflare-worker", "worker.js")
		if data, err := os.ReadFile(cwPath); err == nil {
			workerCode = string(data)
		} else {
			workerCode = readEmbeddedFile("runtime/mhr-cfw/deploy/cloudflare-worker/worker.js")
		}
	} else {
		gasPath := filepath.Join(runtimeDir, "apps_script", "Code.gs")
		if data, err := os.ReadFile(gasPath); err == nil {
			gasCode = string(data)
		} else {
			gasCode = readEmbeddedFile("runtime/mhr/apps_script/Code.gs")
		}

		cwPath := filepath.Join(runtimeDir, "apps_script", "cloudflare_worker.js")
		if data, err := os.ReadFile(cwPath); err == nil {
			workerCode = string(data)
		} else {
			workerCode = readEmbeddedFile("runtime/mhr/apps_script/cloudflare_worker.js")
		}
	}

	if gasCode != "" {
		gasCode = strings.ReplaceAll(gasCode, `"CHANGE_ME_TO_A_STRONG_SECRET"`, fmt.Sprintf("%q", authKey))
		gasCode = strings.ReplaceAll(gasCode, `"STRONG_SECRET_KEY"`, fmt.Sprintf("%q", authKey))
		gasCode = strings.ReplaceAll(gasCode, `"https://example.workers.dev"`, fmt.Sprintf("%q", workerURL))
	}

	if workerCode != "" {
		workerHost := strings.TrimPrefix(strings.TrimPrefix(workerURL, "https://"), "http://")
		workerCode = strings.ReplaceAll(workerCode, `"myworker.workers.dev"`, fmt.Sprintf("%q", workerHost))
	}

	res.AppsScriptCode = gasCode
	res.WorkerCode = workerCode
	return res, nil
}

func readEmbeddedFile(path string) string {
	f, err := runtimeFS.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return ""
	}
	return string(data)
}

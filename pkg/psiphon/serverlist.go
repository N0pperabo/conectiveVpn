package psiphon

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed psiphon_server_list.dat
var embeddedServerList []byte

var (
	serverListMu sync.Mutex
)

// EnsurePsiphonServerList guarantees that a valid, populated server_list.dat
// is available in the target data directory for ShiroKhorshid Psiphon core.
//
// Search order (matching MehrON CoreConfigHandler.EnsurePsiphonServerList):
// 1. Existing server_list.dat in target dataDir (if size > 0)
// 2. Application bin / assets/bin directory
// 3. %APPDATA%\Psiphon3\server_list.dat
// 4. %LOCALAPPDATA%\Psiphon3\server_list.dat
// 5. Existing remote_server_list in Conective profile cache
// 6. Embedded official ShiroKhorshid server list (273 KB pre-vetted server entries)
func EnsurePsiphonServerList(dataDir string) (string, error) {
	serverListMu.Lock()
	defer serverListMu.Unlock()

	if dataDir == "" {
		dataDir = getSafePsiphonDir("")
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("creating data directory %q: %w", dataDir, err)
	}

	targetPath := filepath.Join(dataDir, "server_list.dat")

	// 1. Existing valid server_list.dat in target directory
	if fi, err := os.Stat(targetPath); err == nil && fi.Size() > 0 {
		return targetPath, nil
	}

	// Helper to copy file
	copyFile := func(src string) bool {
		sInfo, sErr := os.Stat(src)
		if sErr != nil || sInfo.Size() == 0 {
			return false
		}
		in, err := os.Open(src)
		if err != nil {
			return false
		}
		defer in.Close()

		out, err := os.Create(targetPath)
		if err != nil {
			return false
		}
		defer out.Close()

		if _, err := io.Copy(out, in); err != nil {
			_ = os.Remove(targetPath)
			return false
		}
		return true
	}

	// 2. Look in bin directories
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		for _, dir := range []string{
			filepath.Join(exeDir, "bin"),
			filepath.Join(exeDir, "assets", "bin"),
			exeDir,
		} {
			c := filepath.Join(dir, "server_list.dat")
			if copyFile(c) {
				return targetPath, nil
			}
		}
	}

	// Fallback dev search paths
	for _, dir := range []string{"bin", filepath.Join("assets", "bin"), "."} {
		c := filepath.Join(dir, "server_list.dat")
		if copyFile(c) {
			return targetPath, nil
		}
	}

	// 3. Look in %APPDATA%\Psiphon3
	if appData := os.Getenv("APPDATA"); appData != "" {
		c := filepath.Join(appData, "Psiphon3", "server_list.dat")
		if copyFile(c) {
			return targetPath, nil
		}
	}

	// 4. Look in %LOCALAPPDATA%\Psiphon3
	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		c := filepath.Join(localApp, "Psiphon3", "server_list.dat")
		if copyFile(c) {
			return targetPath, nil
		}
	}

	// 5. Look in Conective cached remote_server_list
	if appData := os.Getenv("APPDATA"); appData != "" {
		conectivePsiphon := filepath.Join(appData, "Conective", "psiphon")
		candidates := []string{
			filepath.Join(conectivePsiphon, "data", "ca.psiphon.PsiphonTunnel.tunnel-core", "remote_server_list"),
			filepath.Join(conectivePsiphon, "profile_auto", "ca.psiphon.PsiphonTunnel.tunnel-core", "remote_server_list"),
			filepath.Join(conectivePsiphon, "profile_us", "ca.psiphon.PsiphonTunnel.tunnel-core", "remote_server_list"),
			filepath.Join(conectivePsiphon, "profile_de", "ca.psiphon.PsiphonTunnel.tunnel-core", "remote_server_list"),
			filepath.Join(conectivePsiphon, "ca.psiphon.PsiphonTunnel.tunnel-core", "remote_server_list"),
		}
		for _, cand := range candidates {
			if copyFile(cand) {
				return targetPath, nil
			}
		}
	}

	// 6. Write embedded ShiroKhorshid server list
	if len(embeddedServerList) > 0 {
		if err := os.WriteFile(targetPath, embeddedServerList, 0644); err == nil {
			return targetPath, nil
		}
	}

	return targetPath, fmt.Errorf("no server list source found and failed to write embedded list")
}

// GetEmbeddedServerList returns the raw embedded server list bytes
func GetEmbeddedServerList() []byte {
	return embeddedServerList
}

// ParseCdnEdges parses comma-, newline-, or space-separated edge IPs/hostnames
func ParseCdnEdges(raw string) []string {
	var edges []string
	seen := make(map[string]bool)

	tokens := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ';'
	})

	for _, token := range tokens {
		item := strings.TrimSpace(token)
		if item == "" {
			continue
		}
		// In case user pasted with scheme or trailing comment
		if idx := strings.Index(item, "#"); idx >= 0 {
			item = strings.TrimSpace(item[:idx])
		}
		if item != "" && !seen[item] {
			seen[item] = true
			edges = append(edges, item)
		}
	}
	return edges
}

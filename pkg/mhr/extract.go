package mhr

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// EnsureRuntime prepares the runtime directory on disk for the given variant
func (m *Manager) EnsureRuntime(variant string) (string, error) {
	if variant != VariantMHR && variant != VariantMHRCFW {
		variant = VariantMHR
	}

	// 1. Look for pre-existing directory on disk
	candidates := make([]string, 0, 5)

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "bin", "mhr", variant),
			filepath.Join(exeDir, "mhr", variant),
		)
	}

	candidates = append(candidates,
		filepath.Join("bin", "mhr", variant),
		filepath.Join(m.dataDir, "mhr", variant),
	)

	for _, dir := range candidates {
		mainPy := filepath.Join(dir, "main.py")
		cfgEx := filepath.Join(dir, "config.example.json")
		if _, errMain := os.Stat(mainPy); errMain == nil {
			if _, errCfg := os.Stat(cfgEx); errCfg == nil {
				return dir, nil
			}
		}
	}

	// 2. Not found on disk: extract from embedded RuntimeFS into dataDir
	targetDir := filepath.Join(m.dataDir, "mhr", variant)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed creating target MHR directory: %w", err)
	}

	embedSubPath := "runtime/" + variant
	subFS, err := fs.Sub(runtimeFS, embedSubPath)
	if err != nil {
		return "", fmt.Errorf("failed accessing embedded %s files: %w", variant, err)
	}

	err = fs.WalkDir(subFS, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		destPath := filepath.Join(targetDir, path)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		srcFile, err := subFS.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		return err
	})

	if err != nil {
		return "", fmt.Errorf("failed extracting embedded MHR files: %w", err)
	}

	return targetDir, nil
}

// FindPythonExecutable resolves the path to Python 3.10+
func FindPythonExecutable() (string, error) {
	// 1. Check LocalAppData Programs Python
	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		pythonRoot := filepath.Join(localApp, "Programs", "Python")
		if entries, err := os.ReadDir(pythonRoot); err == nil {
			dirs := make([]string, 0, len(entries))
			for _, e := range entries {
				if e.IsDir() && strings.HasPrefix(strings.ToLower(e.Name()), "python3") {
					dirs = append(dirs, filepath.Join(pythonRoot, e.Name(), "python.exe"))
				}
			}
			sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
			for _, candidate := range dirs {
				if _, err := os.Stat(candidate); err == nil {
					return candidate, nil
				}
			}
		}
	}

	// 2. Check System Drives (e.g. C:\Python312\python.exe)
	for _, root := range []string{`C:\Python312`, `C:\Python311`, `C:\Python310`} {
		p := filepath.Join(root, "python.exe")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// 3. Fall back to PATH
	if p, err := exec.LookPath("python.exe"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("python"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("python3"); err == nil {
		return p, nil
	}

	return "", fmt.Errorf("Python 3.10+ not found. Please install Python from https://www.python.org or Microsoft Store")
}
